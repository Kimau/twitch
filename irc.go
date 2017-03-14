package twitch

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-irc/irc"
	"golang.org/x/time/rate"
)

const (
	limitIrcMessageNum   = 20
	limitIrcMessageTime  = time.Second * 30
	defaultNickPadLength = 14
)

// needs to bind to real address for VPN
var (
	flagIrcVerbose     = flag.Bool("ircVerbose", false, "Should IRC logging be verbose")
	flagIrcChannel     = flag.String("ircRoom", "", "Which Twitch chat room to Join")
	flagIrcPerformFile = flag.String("performfile", "", "Load File and perform on load")

	regexHostName = regexp.MustCompile(" ([a-z_]+)\\.$")
	regexLogMsg   = regexp.MustCompile("^IRC: ([ 0-9][0-9]):([ 0-9][0-9]):([ 0-9][0-9]) ([\\*\\_]*) (.*)")
	regexPrivMsg  = regexp.MustCompile("([^ ]*) ([[:word:]]+) (B([0-9]+))?(\\[([0-9]+)\\])?(\\{(([0-9]+),([0-9]+),([0-9]+)|)+\\})?: (.*)")
	regexHostMsg  = regexp.MustCompile("([[:word:]]+)\\.")

	ignoreMsgCmd = map[string]bool{
		IrcReplyYourhost: false,
		IrcReplyCreated:  false,
		IrcReplyMyinfo:   false,
	}
)

// Chat - IRC Chat interface
type Chat struct {
	Server  string
	Room    string
	verbose bool
	config  irc.ClientConfig
	limiter *rate.Limiter

	self   *Chatter
	mode   chatMode
	InRoom map[IrcNick]*Viewer

	messageOfTheDay []string
	nameReplyList   []IrcNick

	msgLogger *log.Logger
	logBuffer bytes.Buffer

	viewers viewerProvider
	irc     *irc.Client
}

func init() {

}

func createIrcClient(auth ircAuthProvider, vp viewerProvider, serverAddr string) (*Chat, error) {

	log.Println("Creating IRC Client")

	hasAuth, nick, pass := auth.GetIrcAuth()
	if !hasAuth {
		return nil, fmt.Errorf("Associated user has no valid Auth")
	}

	ircRoomToJoin := *flagIrcChannel
	if len(ircRoomToJoin) < 2 {
		ircRoomToJoin = nick
	}

	chat := &Chat{
		Server:  serverAddr,
		Room:    ircRoomToJoin,
		verbose: *flagIrcVerbose,
		config: irc.ClientConfig{
			Nick: nick,
			Pass: pass,
			User: "Username",
			Name: "Full Name",
		},
		viewers: vp,
	}

	chat.SetupLogWriter()
	chat.config.Handler = chat
	chat.limiter = rate.NewLimiter(rate.Every(limitIrcMessageTime), limitIrcMessageNum)

	return chat, nil
}

// StartRunLoop - Start Run Loop
func (c *Chat) StartRunLoop() error {
	conn, err := net.Dial("tcp", c.Server)
	if err != nil {
		return err
	}

	c.irc = irc.NewClient(conn, c.config)

	if c.verbose {
		// In Verbose mode log all messages
		c.irc.Reader.DebugCallback = func(m string) {
			log.Printf("IRC (V) >> %s", m)
		}

		c.irc.Writer.DebugCallback = func(m string) {
			c.limiter.Limit()
			log.Printf("IRC (V) << %s", m)
		}

	} else {
		// Just Rate Limit
		c.irc.Writer.DebugCallback = func(string) {
			c.limiter.Limit()
		}
	}

	log.Println("IRC Connected")

	err = c.irc.Run()
	if err != nil {
		c.irc = nil
	}

	return err
}

// WriteRawIrcMsg - Writes a raw IRC message
func (c *Chat) WriteRawIrcMsg(msg string) {
	if c.irc == nil {
		log.Printf("Cannot Write to IRC as it's NIL: %s", msg)
		return
	}

	err := c.irc.Write(msg)
	if err != nil {
		log.Printf("Write Raw Failed: %s\n %s", msg, err.Error())
	}
}

// Log - Log to internal message logger
func (c *Chat) Log(s string) {
	s = strings.Replace(strings.Replace(s, "\\", "\\\\", -1), "\n", "\\n", -1)
	c.msgLogger.Print(s)
}

// Logf - FMT interface
func (c *Chat) Logf(s string, v ...interface{}) {
	c.Log(fmt.Sprintf(s, v...))
}

// SetupLogWriter - Set where the log is written to
func (c *Chat) SetupLogWriter(newTarget ...io.Writer) {
	c.logBuffer.Reset()
	if newTarget != nil {
		writeList := append(newTarget, &c.logBuffer)
		mw := io.MultiWriter(writeList...)
		c.msgLogger = log.New(mw, "IRC: ", log.Ltime)
	} else {
		c.msgLogger = log.New(&c.logBuffer, "IRC: ", log.Ltime)
	}

	if c.msgLogger == nil {
		log.Fatalln("Log shouldn't be null")
	}

	c.Logf("+------------ New Log ------------+ %s", time.Now())
}

func (c *Chat) respondToWelcome(m *irc.Message) {
	c.InRoom = make(map[IrcNick]*Viewer)

	c.WriteRawIrcMsg("CAP REQ :twitch.tv/membership")
	c.WriteRawIrcMsg("CAP REQ :twitch.tv/tags")
	c.WriteRawIrcMsg("CAP REQ :twitch.tv/commands")
	c.WriteRawIrcMsg(fmt.Sprintf("JOIN #%s", c.Room))
}

func (c *Chat) nowHosting(target *Viewer) {
	c.mode.hosting = target

	if target == nil {
		c.Logf("* No longer hosting.")
	} else {
		c.Logf("* Now Hosting %s", target.User.DisplayName)
	}
}

func (c *Chat) clearChat(m *irc.Message) {
	// TODO :: Check it's this channel

	nickToClear := m.Trailing()

	newMsg := bytes.Buffer{}
	for fullS, e := c.logBuffer.ReadString('\n'); e == nil; fullS, e = c.logBuffer.ReadString('\n') {
		if !strings.HasPrefix(fullS, "IRC:") {
			fmt.Fprint(&newMsg, fullS)
			continue
		}

		t := fullS[:14]
		s := fullS[14:]

		switch s[0:1] {
		case "*":
			fmt.Fprint(&newMsg, fullS)
		case "_":
			fmt.Fprint(&newMsg, fullS)
		default:
			if len(s) < 4 {
				fmt.Fprint(&newMsg, fullS)
				continue
			}

			nick := s[2 : strings.Index(s[3:], " ")+3]
			if nick == nickToClear {
				fmt.Fprintf(&newMsg, "%s_ %s", t, s)
			} else {
				fmt.Fprint(&newMsg, fullS)
			}
		}
	}

	c.logBuffer.Reset()
	c.logBuffer = newMsg

	// Log Ban Reason
	r, ok := m.Tags[TwitchTagBanReason]
	if ok {
		c.Logf("_ Cleared %s from chat: %s", nickToClear, r)
	} else {
		c.Logf("_ Cleared %s from chat", nickToClear)
	}
}

func printDebugTag(m *irc.Message) {
	tags := ""
	for k, v := range m.Tags {
		tags += fmt.Sprintf("|%s:\t %s|\n", k, v)
	}

	log.Printf("IRC NOT DEBUG [%s]\n%s\n[%s]\n%s", m.Command, m.Trailing(), strings.Join(m.Params, " \t|"), tags)

}

// JoinNickComma - Join a list of Nicks into a string with comma
func JoinNickComma(nl []IrcNick) string {
	nickformated := ""
	for i, v := range nl {
		if i == 0 {
			nickformated = string(v)
		} else {
			nickformated += fmt.Sprintf(",%s", v)
		}
	}

	return nickformated
}

// JoinNicks - Join a list of Nicks into text columns
func JoinNicks(nl []IrcNick, columns int, nickPadLength int) string {
	nickformated := ""
	for i, v := range nl {
		vStr := string(v)

		if nickPadLength > 0 {
			if len(vStr) > nickPadLength {
				vStr = vStr[0:nickPadLength]
			}
			for len(vStr) < nickPadLength {
				vStr += " "
			}
		}

		// Column Formatting
		if columns > 0 {
			if i > 0 && (i%columns) == 0 {
				nickformated += "\n\t" + vStr
			} else {
				nickformated += "\t" + vStr
			}
		} else {
			nickformated += " " + vStr
		}
	}

	return nickformated
}

func (c *Chat) processNameList() {
	vList := c.viewers.UpdateViewers(c.nameReplyList)

	for _, v := range vList {
		nick := v.GetNick()
		c.InRoom[nick] = v
	}
}

// Handle - IRC Message
func (c *Chat) Handle(irc *irc.Client, m *irc.Message) {
	printOut, ok := ignoreMsgCmd[m.Command]
	if ok {
		if printOut {
			c.Logf("IRC [%s] \t %s", m.Command, m.Trailing())
		}
		return
	}

	switch m.Command {
	case IrcReplyWelcome: // 001 is a welcome event, so we join channels there
		c.Log("Respond to Welcome")
		c.respondToWelcome(m)

		// Message of the Day
	case IrcReplyMotdstart:
		c.messageOfTheDay = []string{}

	case IrcReplyMotd:
		c.messageOfTheDay = append(c.messageOfTheDay, m.Trailing())

	case IrcReplyEndofmotd:
		c.Logf("_ MOTD %s", strings.Join(c.messageOfTheDay, "\n"))
		// End of Message of the Day

	// Name List
	case IrcReplyNamreply:
		if c.nameReplyList == nil {
			c.nameReplyList = []IrcNick{}
		}
		for _, in := range strings.Split(m.Trailing(), " ") {
			c.nameReplyList = append(c.nameReplyList, IrcNick(in))
		}

	case IrcReplyEndofnames:
		c.processNameList()
		nickformated := JoinNickComma(c.nameReplyList)
		c.Logf("* Names: %s", nickformated)
		c.nameReplyList = nil
		// End of Name List

	case IrcCap:
		if m.Params[0] == "*" && m.Params[1] == "ACK" {
			return
		}

		log.Printf("IRC Unhandled CAP MSG [%s] %s",
			strings.Join(m.Params, "]["),
			m.Trailing())

	case IrcCmdJoin: // User Joined Channel
		c.Logf("* Join %s", m.Name)
		nick := IrcNick(m.Name)

		v, err := c.viewers.FindViewer(nick)
		if err != nil {
			log.Printf("JOIN ERROR [%s] not found\n%s", nick, err)
			return
		}

		c.InRoom[nick] = v

	case IrcCmdPart: // User Parted Channel
		c.Logf("* Part %s", m.Name)
		delete(c.InRoom, IrcNick(m.Name))

	case TwitchCmdClearChat:
		c.clearChat(m)

	case TwitchCmdGlobalUserState:
		cu := &Chatter{}
		cu.updateChatterFromTags(m)
		v := c.viewers.GetViewerFromChatter(cu)
		c.Logf("_ Global User State for %s", v.GetNick())

	case TwitchCmdRoomState:
		for tagName, tagVal := range m.Tags {
			switch tagName {
			case TwitchTagRoomFollowersOnly:
				c.mode.followersOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.Logf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.followersOnly = true
				}

			case TwitchTagRoomR9K:
				c.mode.r9k = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.Logf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.r9k = true
				}
			case TwitchTagRoomSlow:
				c.mode.slowMode = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.Logf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.slowMode = true
				}
			case TwitchTagRoomSubOnly:
				c.mode.subsOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.Logf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.subsOnly = true
				}

			case TwitchTagRoomLang:
				c.mode.lang = string(tagVal)

			case TwitchTagRoomEmote:
				c.mode.emoteOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.Logf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.emoteOnly = true
				}

			}
		}

		chatChanName := m.Trailing()
		c.Logf("* %s updated: %s", chatChanName, c.mode)

	case TwitchCmdUserNotice:
		cu := &Chatter{}
		cu.updateChatterFromTags(m)
		c.viewers.GetViewerFromChatter(cu)
		c.Logf("* %s", m.Tags[TwitchTagSystemMsg])

	case TwitchCmdUserState:
		cu := &Chatter{}
		cu.updateChatterFromTags(m)
		v := c.viewers.GetViewerFromChatter(cu)

		c.InRoom[v.GetNick()] = v
		c.Logf("* User State updated from %s in %s", v.GetNick(), m.Trailing())

	case TwitchCmdHostTarget:
		match := regexHostName.FindStringSubmatch(m.Trailing())
		if match != nil {
			nick := IrcNick(match[1])
			v, err := c.viewers.FindViewer(nick)
			if err != nil {
				log.Printf("HOST ERROR [%s] not found\n%s", nick, err)
				return
			}
			c.nowHosting(v)
		}

	case TwitchCmdReconnect:
		printDebugTag(m)

	case IrcCmdMode:
		// printDebugTag(m)
		// TODO :: Handle Mode

	case IrcCmdPrivmsg:
		// < PRIVMSG #<channel> :This is a sample message
		// > :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :This is a sample message
		// > @badges=<badges>;bits=<bits>;color=<color>;display-name=<display-name>;emotes=<emotes>;id=<id>;mod=<mod>;room-id=<room-id>;subscriber=<subscriber>;turbo=<turbo>;user-id=<user-id>;user-type=<user-type> :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :<message>
		// Trailing = <message>
		// Param[0] channel
		v, err := c.viewers.FindViewer(IrcNick(m.Name))
		if err != nil {
			log.Printf("PRIV MSG ERROR [%s] not found\n%s", m.Name, err)
			return
		}

		if v.Chatter == nil {
			v.Chatter = &Chatter{
				Nick: v.GetNick(),
			}
		}
		v.Chatter.updateChatterFromTags(m)

		finalText := v.Chatter.NameWithBadge()

		// Handle Bits
		bits, ok := m.Tags[TwitchTagBits]
		if ok {
			bVal, err := strconv.Atoi(string(bits))
			if err != nil {
				log.Print("Bits error -", m, err)
			} else {
				finalText += fmt.Sprintf("[%d]", bVal)
			}
		}

		// Handle Emotes
		emoteList, ok := m.Tags[TwitchTagMsgEmotes]
		if ok {
			el, err := emoteTagToList(emoteList)
			if err != nil {
				log.Printf("Unable to parse Emote Tag [%s]\n%s", emoteList, err)
			} else if len(el) > 0 {
				finalText += fmt.Sprintf("{%s}", el)
			}
		}

		// Padding before
		finalText = fmt.Sprintf("%-14s", finalText)

		// Add Text
		finalText += ": " + m.Trailing()

		// Output
		c.Log(finalText)

	case IrcCmdNotice:
		msgID, ok := m.Tags[TwitchTagMsgID]
		if !ok {
			printDebugTag(m)
			return
		}
		switch msgID {
		case TwitchMsgHostOff:
			c.nowHosting(nil)

		case TwitchMsgHostOn:
			nick := regexHostMsg.FindStringSubmatch(m.Trailing())
			v, err := c.viewers.FindViewer(IrcNick(nick[1]))
			if err != nil {
				log.Printf("HOST MSG ON ERROR [%s] not found\n%s", nick[1], err)
				return
			}

			c.nowHosting(v)

		case TwitchMsgR9kOff:
			c.mode.r9k = false
			c.Log("* " + m.Trailing())
		case TwitchMsgR9kOn:
			c.mode.r9k = true
			c.Log("* " + m.Trailing())
		case TwitchMsgSlowOff:
			c.mode.slowMode = false
			c.Log("* " + m.Trailing())
		case TwitchMsgSlowOn:
			c.mode.slowMode = true
			c.Log("* " + m.Trailing())
		case TwitchMsgSubsOff:
			c.mode.subsOnly = false
			c.Log("* " + m.Trailing())
		case TwitchMsgSubsOn:
			c.mode.subsOnly = true
			c.Log("* " + m.Trailing())

		case TwitchMsgEmoteOnlyOff:
			c.mode.emoteOnly = false
			c.Log("* " + m.Trailing())
		case TwitchMsgEmoteOnlyOn:
			c.mode.emoteOnly = true
			c.Log("* " + m.Trailing())

		case TwitchMsgAlreadyEmoteOnlyOff:
			c.Log("_ MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadyEmoteOnlyOn:
			c.Log("_ MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadyR9kOff:
			c.Log("_ MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadyR9kOn:
			c.Log("_ MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadySubsOff:
			c.Log("_ MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadySubsOn:
			c.Log("_ MODE ALREADY SET: " + m.Trailing())

		case TwitchMsgUnrecognizedCmd:
			c.Log("_ " + m.Trailing())

		/*
			case 		 TwitchMsgAlreadyBanned      :
			case TwitchMsgBadHostHosting     :
			case TwitchMsgBadUnbanNoBan      :
			case TwitchMsgBanSuccess         :

			case TwitchMsgHostsRemaining     :
			case TwitchMsgMsgChannelSuspended:

			case TwitchMsgTimeoutSuccess     :
			case TwitchMsgUnbanSuccess       :

		*/
		default:
			log.Printf("UNKNOWN MSG ID: [%s]", msgID)
			printDebugTag(m)
		}

	case IrcCmdPing:

	default:
		log.Printf("IRC ???[%s] \t %+v", m.Command, m)
	}
}
