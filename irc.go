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
	limitIrcMessageNum  = 20
	limitIrcMessageTime = time.Second * 30
)

// needs to bind to real address for VPN
var (
	flagIrcAddr        = flag.String("irc", "irc.afternet.org:6667", "irc server to connect to")
	flagIrcVerbose     = flag.Bool("ircVerbose", false, "Should IRC logging be verbose")
	flagIrcPerformFile = flag.String("performfile", "", "Load File and perform on load")

	regexHostName = regexp.MustCompile(" ([a-z_]+)\\.$")

	ignoreMsgCmd = map[string]bool{
		IrcReplyYourhost: false,
		IrcReplyCreated:  false,
		IrcReplyMyinfo:   false,
	}
)

type chatMode struct {
	subsOnly      bool
	lang          string
	emoteOnly     bool
	followersOnly bool
	slowMode      bool
	r9k           bool
	hosting       *Viewer
}

// ircAuthProvider - Provides Auth normally expects UserAuth
type ircAuthProvider interface {
	GetIrcAuth() (hasauth bool, name string, pass string, addr string)
}

type viewerProvider interface {
	GetNick() IrcNick
	GetViewer(ID) *Viewer
	FindViewer(IrcNick) *Viewer
	UpdateViewers([]IrcNick) []*Viewer
}

// Chat - IRC Chat interface
type Chat struct {
	Server  string
	verbose bool
	config  irc.ClientConfig
	limiter *rate.Limiter

	self   *chatter
	mode   *chatMode
	InRoom map[IrcNick]*Viewer

	messageOfTheDay []string
	nameReplyList   []IrcNick

	log    *log.Logger
	msgLog bytes.Buffer

	viewers viewerProvider
	irc     *irc.Client
}

func init() {

}

func createIrcClient(auth ircAuthProvider, vp viewerProvider) (*Chat, error) {

	log.Println("Creating IRC Client")

	hasAuth, nick, pass, serverAddr := auth.GetIrcAuth()
	if !hasAuth {
		return nil, fmt.Errorf("Associated user has no valid Auth")
	}

	chat := &Chat{
		Server:  serverAddr,
		verbose: *flagIrcVerbose,
		config: irc.ClientConfig{
			Nick: nick,
			Pass: pass,
			User: "Username",
			Name: "Full Name",
		},
		viewers: vp,
	}

	chat.SetupLogWriter(&chat.msgLog)
	chat.log.Println("+------------ New Log ------------+")
	chat.config.Handler = chat
	chat.limiter = rate.NewLimiter(rate.Every(limitIrcMessageTime), limitIrcMessageNum)

	return chat, nil
}

// SetupLogWriter - Set where the log is written to
func (c *Chat) SetupLogWriter(newTarget io.Writer) {
	c.log = log.New(newTarget, "IRC: ", log.Ltime)
	if c.log == nil {
		log.Fatalln("Log shouldn't be null")
	}
}

// GetLog - Getting Chat Log
func (c *Chat) GetLog() *bytes.Buffer {
	return &c.msgLog
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

func (c *Chat) respondToWelcome(m *irc.Message) {
	c.InRoom = make(map[IrcNick]*Viewer)

	c.WriteRawIrcMsg("CAP REQ :twitch.tv/membership")
	c.WriteRawIrcMsg("CAP REQ :twitch.tv/tags")
	c.WriteRawIrcMsg("CAP REQ :twitch.tv/commands")
	c.WriteRawIrcMsg(fmt.Sprintf("JOIN #%s", c.config.Nick))
}

func (c *Chat) nowHosting(target *Viewer) {
	c.mode.hosting = target

	if target == nil {
		c.log.Printf("* No longer hosting.")
	} else {
		c.log.Printf("* Now Hosting %s", target.User.DisplayName)
	}
}

func printDebugTag(m *irc.Message) {
	tags := ""
	for k, v := range m.Tags {
		tags += fmt.Sprintf("%s:\t %s\n", k, v)
	}

	log.Printf("IRC NOT DEBUG[%s] %s \n%s \n%s", m.Command, m.Trailing(), strings.Join(m.Params, " -\t "), tags)

}

// JoinNicks - Join a list of Nicks into a string
func JoinNicks(nl []IrcNick, columns int, nickPadLength int) string {
	nickformated := ""
	for i, v := range nl {
		vStr := string(v)

		if len(vStr) > nickPadLength {
			vStr = vStr[0:nickPadLength]
		}
		for len(vStr) < nickPadLength {
			vStr += " "
		}
		if i > 0 && (i%columns) == 0 {
			nickformated += "\n\t" + vStr
		} else {
			nickformated += "\t" + vStr
		}
	}

	return nickformated
}

func (c *Chat) processNameList() {
	vList := c.viewers.UpdateViewers(c.nameReplyList)

	for _, v := range vList {
		nick := v.getNick()
		c.InRoom[nick] = v
	}
}

// Handle - IRC Message
func (c *Chat) Handle(irc *irc.Client, m *irc.Message) {
	printOut, ok := ignoreMsgCmd[m.Command]
	if ok {
		if printOut {
			c.log.Printf("IRC [%s] \t %s", m.Command, m.Trailing())
		}
		return
	}

	switch m.Command {
	case IrcReplyWelcome: // 001 is a welcome event, so we join channels there
		c.log.Println("Respond to Welcome")
		c.respondToWelcome(m)

		// Message of the Day
	case IrcReplyMotdstart:
		c.messageOfTheDay = []string{}

	case IrcReplyMotd:
		c.messageOfTheDay = append(c.messageOfTheDay, m.Trailing())

	case IrcReplyEndofmotd:
		c.log.Printf(
			"--- Message of the Day ---\n\t%s \n-----------------------------",
			strings.Join(c.messageOfTheDay, "\n\t"))
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
		nickformated := JoinNicks(c.nameReplyList, 4, 18)
		c.log.Printf("--- Names ---\n%s\n-------------", nickformated)
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
		c.log.Printf("JOIN %s", m.Name)
		nick := IrcNick(m.Name)

		v := c.viewers.FindViewer(nick)
		c.InRoom[nick] = v

	case IrcCmdPart: // User Parted Channel
		c.log.Printf("PART %s", m.Name)
		delete(c.InRoom, IrcNick(m.Name))

	case TwitchCmdClearChat:
		printDebugTag(m)

	case TwitchCmdGlobalUserState:
		printDebugTag(m)

	case TwitchCmdRoomState:
		chatChanName := m.Trailing()
		c.log.Printf("* Room State updated %s", chatChanName)

		c.mode = &chatMode{}
		for tagName, tagVal := range m.Tags {
			switch tagName {
			case TwitchTagRoomFollowersOnly:
				c.mode.followersOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.log.Println(tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.followersOnly = true
				}

			case TwitchTagRoomR9K:
				c.mode.r9k = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.log.Println(tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.r9k = true
				}
			case TwitchTagRoomSlow:
				c.mode.slowMode = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.log.Println(tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.slowMode = true
				}
			case TwitchTagRoomSubOnly:
				c.mode.subsOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.log.Println(tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.subsOnly = true
				}

			case TwitchTagRoomLang:
				c.mode.lang = string(tagVal)

			case TwitchTagRoomEmote:
				c.mode.emoteOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					c.log.Println(tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.emoteOnly = true
				}

			}
		}

	case TwitchCmdUserNotice:
		printDebugTag(m)

	case TwitchCmdUserState:
		chatChanName := m.Trailing()
		if c.self == nil {
			c.self = &chatter{
				nick: IrcNick(c.config.Nick),
			}
		}
		c.self.UpdateChatterFromTags(m)
		c.log.Printf("* User State updated from %s in %s", c.self.nick, chatChanName)

	case TwitchCmdHostTarget:
		match := regexHostName.FindStringSubmatch(m.Trailing())
		if match != nil {
			v := c.viewers.FindViewer(IrcNick(match[1]))
			c.nowHosting(v)
		}

	case TwitchCmdReconnect:
		printDebugTag(m)

	case IrcCmdPrivmsg:
		// < PRIVMSG #<channel> :This is a sample message
		// > :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :This is a sample message
		// > @badges=<badges>;bits=<bits>;color=<color>;display-name=<display-name>;emotes=<emotes>;id=<id>;mod=<mod>;room-id=<room-id>;subscriber=<subscriber>;turbo=<turbo>;user-id=<user-id>;user-type=<user-type> :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :<message>
		// Trailing = <message>
		// Param[0] channel
		v := c.viewers.FindViewer(IrcNick(m.Name))
		if v.Chatter == nil {
			v.Chatter = &chatter{
				nick: v.getNick(),
			}
		}
		v.Chatter.UpdateChatterFromTags(m)

		bits, ok := m.Tags[TwitchTagBits]

		if ok {
			bVal, err := strconv.Atoi(string(bits))
			if err != nil {
				log.Print("Bits error -", m, err)
			} else {
				v.Chatter.bits += bVal
				c.log.Printf("BITS %d \t %s: \t%s",
					bVal, v.Chatter.NameWithBadge(), m.Trailing())
			}

		} else {
			c.log.Printf("%s: \t%s", v.Chatter.NameWithBadge(), m.Trailing())
		}

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
			v := c.viewers.FindViewer(IrcNick(m.Trailing()))
			c.nowHosting(v)

		case TwitchMsgR9kOff:
			c.mode.r9k = false
			c.log.Println("* " + m.Trailing())
		case TwitchMsgR9kOn:
			c.mode.r9k = true
			c.log.Println("* " + m.Trailing())
		case TwitchMsgSlowOff:
			c.mode.slowMode = false
			c.log.Println("* " + m.Trailing())
		case TwitchMsgSlowOn:
			c.mode.slowMode = true
			c.log.Println("* " + m.Trailing())
		case TwitchMsgSubsOff:
			c.mode.subsOnly = false
			c.log.Println("* " + m.Trailing())
		case TwitchMsgSubsOn:
			c.mode.subsOnly = true
			c.log.Println("* " + m.Trailing())

		case TwitchMsgEmoteOnlyOff:
			c.mode.emoteOnly = false
			c.log.Println("* " + m.Trailing())
		case TwitchMsgEmoteOnlyOn:
			c.mode.emoteOnly = true
			c.log.Println("* " + m.Trailing())

		case TwitchMsgAlreadyEmoteOnlyOff:
			c.log.Println("* MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadyEmoteOnlyOn:
			c.log.Println("* MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadyR9kOff:
			c.log.Println("* MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadyR9kOn:
			c.log.Println("* MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadySubsOff:
			c.log.Println("* MODE ALREADY SET: " + m.Trailing())
		case TwitchMsgAlreadySubsOn:
			c.log.Println("* MODE ALREADY SET: " + m.Trailing())

		case TwitchMsgUnrecognizedCmd:
			c.log.Println("* " + m.Trailing())

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
