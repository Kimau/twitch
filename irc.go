package twitch

import (
	"fmt"
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
	IrcVerboseMode = false
	regexHostMatch = regexp.MustCompile("([[:word:]]+) ([0-9]+|-)")

	ignoreMsgCmd = map[string]bool{
		IrcReplyYourhost: false,
		IrcReplyCreated:  false,
		IrcReplyMyinfo:   false,
	}
)

// Chat - IRC Chat interface
type Chat struct {
	Server  string
	config  irc.ClientConfig
	limiter *rate.Limiter

	mode   chatMode
	InRoom map[IrcNick]*Viewer

	messageOfTheDay []string
	nameReplyList   []IrcNick

	logger *chatLogInteral

	viewers       viewerProvider
	irc           *irc.Client
	weakClientRef *Client
}

func init() {

}

func createIrcClient(auth ircAuthProvider, vp viewerProvider, serverAddr string) (*Chat, error) {

	log.Println("Creating IRC Client")

	hasAuth, nick, pass := auth.GetIrcAuth()
	if !hasAuth {
		return nil, fmt.Errorf("Associated user has no valid Auth")
	}

	roomViewer := vp.Get(vp.GetRoomID())

	chat := &Chat{
		Server: serverAddr,
		config: irc.ClientConfig{
			Nick: nick,
			Pass: pass,
			User: "Username",
			Name: "Full Name",
		},
		viewers: vp,
		InRoom:  make(map[IrcNick]*Viewer),

		logger: startChatLogPump(roomViewer.GetNick()),
	}

	chat.Logf(LogCatSilent, "+------------ New Log [%s] ------------+ %s",
		roomViewer.GetNick(), time.Now().Format(time.RFC822Z))

	chat.config.Handler = chat
	chat.limiter = rate.NewLimiter(rate.Every(limitIrcMessageTime), limitIrcMessageNum)

	return chat, nil
}

func (c *Chat) tickRoomActive() {
	log.Println("Tick room Active")
	for _, v := range c.InRoom {
		c.updateOrSetChatter(v, v.GetNick())
	}
}

// StartRunLoop - Start Run Loop
func (c *Chat) StartRunLoop() error {
	conn, err := net.Dial("tcp", c.Server)
	if err != nil {
		return err
	}

	c.irc = irc.NewClient(conn, c.config)

	if IrcVerboseMode {
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
	c.WriteRawIrcMsg("CAP REQ :twitch.tv/membership")
	c.WriteRawIrcMsg("CAP REQ :twitch.tv/tags")
	c.WriteRawIrcMsg("CAP REQ :twitch.tv/commands")
	c.WriteRawIrcMsg(fmt.Sprintf("JOIN #%s", c.weakClientRef.RoomName))
}

func (c *Chat) forwardAlert(aType AlertName, src IrcNick, extra int) error {
	if c.weakClientRef == nil {
		return fmt.Errorf("Weak Client Ref Missing")
	}

	c.weakClientRef.Alerts.Post(src, aType, extra)
	return nil
}

func (c *Chat) hostUpdate(src IrcNick, target IrcNick, numViewers int) error {
	roomViewer := c.viewers.Get(c.viewers.GetRoomID())
	roomNick := roomViewer.GetNick()

	if src == roomNick {
		// You are now hosting
		c.Logf(LogCatSystem, "You are hosting %s with %d viewers", target, numViewers)

	} else if target == roomNick {
		// You are being hosted by Src
		c.Logf(LogCatSystem, "%s is now hosting you with %d viewers", src, numViewers)
		c.forwardAlert(AlertHost, src, numViewers)

	} else if target == "-" {
		// Src is no longer hosting you
		c.Logf(LogCatSystem, "%s stopped hosting", src)
	} else {
		return fmt.Errorf("Something went wrong with this message")
	}

	return nil
}

func (c *Chat) nowHosting(target *Viewer) {
	c.mode.hosting = target

	if target == nil {
		c.Logf(LogCatSystem, "No longer hosting.")
	} else {
		c.Logf(LogCatSystem, "Now Hosting %s", target.User.DisplayName)
	}
}

func (c *Chat) clearChat(m *irc.Message) {
	// TODO :: Check it's this channel
	nickToClear := m.Trailing()

	// Log Ban Reason
	r, ok := m.Tags[TwitchTagBanReason]
	if ok {
		c.Logf(LogCatSilent, "Cleared %s from chat: %s", nickToClear, r)
	} else {
		c.Logf(LogCatSilent, "Cleared %s from chat", nickToClear)
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

func (c *Chat) updateOrSetChatter(v *Viewer, nick IrcNick) {
	v.CreateChatter(nick)

	v.lockme()
	v.Chatter.updateActive()
	v.unlockme()
}

func (c *Chat) processNameList() {
	vList := c.viewers.UpdateViewers(c.nameReplyList)

	for _, v := range vList {
		nick := v.GetNick()
		c.updateOrSetChatter(v, nick)
		c.InRoom[nick] = v
	}
}

// Handle - IRC Message
func (c *Chat) Handle(irc *irc.Client, m *irc.Message) {
	fmt.Fprintln(localIrcMsgStore(), m)

	printOut, ok := ignoreMsgCmd[m.Command]
	if ok {
		if printOut {
			c.Logf(LogCatSilent, "IGNORE %s \t %s", m.Command, m.Trailing())
		}
		return
	}

	switch m.Command {
	case IrcReplyWelcome: // 001 is a welcome event, so we join channels there
		c.Log(LogCatSilent, "Respond to Welcome")
		c.respondToWelcome(m)

		// Message of the Day
	case IrcReplyMotdstart:
		c.messageOfTheDay = []string{}

	case IrcReplyMotd:
		c.messageOfTheDay = append(c.messageOfTheDay, m.Trailing())

	case IrcReplyEndofmotd:
		c.Logf(LogCatSilent, "MOTD %s", strings.Join(c.messageOfTheDay, "\n"))
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
		c.Logf(LogCatSystem, "Names: %s", nickformated)
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
		nick := IrcNick(m.Name)
		c.Logf(LogCatSystem, "Join %s", nick)

		v, err := c.viewers.Find(nick)
		if err != nil {
			log.Printf("JOIN ERROR [%s] not found\n%s", nick, err)
			return
		}

		c.InRoom[nick] = v
		c.updateOrSetChatter(v, nick)

	case IrcCmdPart: // User Parted Channel
		c.Logf(LogCatSystem, "Part %s", m.Name)

		nick := IrcNick(m.Name)
		v, ok := c.InRoom[nick]
		if ok {
			c.updateOrSetChatter(v, nick)
			delete(c.InRoom, nick)
		}

	case TwitchCmdClearChat:
		c.clearChat(m)

	case TwitchCmdGlobalUserState:
		nick := IrcNick(m.Name)
		if nick.IsValid() == false {
			log.Printf("Global User State: Ignoring %s", nick)
			return
		}

		v, err := c.viewers.Find(nick)
		if err != nil {
			panic(err)
		}
		v.CreateChatter(nick)
		v.Chatter.updateChatterFromTags(m)

		c.Logf(LogCatSilent, "Global User State for %s", nick)

	case TwitchCmdRoomState:
		for tagName, tagVal := range m.Tags {
			switch tagName {
			case TwitchTagRoomFollowersOnly:
				c.mode.followersOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					log.Printf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.followersOnly = true
				}

			case TwitchTagRoomR9K:
				c.mode.r9k = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					log.Printf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.r9k = true
				}
			case TwitchTagRoomSlow:
				c.mode.slowMode = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					log.Printf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.slowMode = true
				}
			case TwitchTagRoomSubOnly:
				c.mode.subsOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					log.Printf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.subsOnly = true
				}

			case TwitchTagRoomLang:
				c.mode.lang = string(tagVal)

			case TwitchTagRoomEmote:
				c.mode.emoteOnly = false
				intVal, err := strconv.Atoi(string(tagVal))
				if err != nil {
					log.Printf("%s:%s \n%s", tagName, tagVal, err)
				} else if intVal > 0 {
					c.mode.emoteOnly = true
				}

			}
		}

		chatChanName := m.Trailing()
		c.Logf(LogCatSystem, "%s updated: %s", chatChanName, c.mode)

	case TwitchCmdUserNotice:
		nick := IrcNick(m.Name)
		if nick.IsValid() == false {
			log.Printf("User Notice: Ignoring %s", nick)
			return
		}

		v, err := c.viewers.Find(nick)
		if err != nil {
			panic(err)
		}
		v.CreateChatter(nick)
		v.lockme()
		v.Chatter.updateChatterFromTags(m)
		v.unlockme()

		c.Logf(LogCatSystem, "%s", m.Tags[TwitchTagSystemMsg])

	case TwitchCmdUserState:
		nick := IrcNick(m.Name)
		if nick.IsValid() == false {
			log.Printf("User State: Ignoring %s", nick)
			return
		}

		v, err := c.viewers.Find(nick)
		if err != nil {
			panic(err)
		}

		v.CreateChatter(nick)
		v.lockme()
		v.Chatter.updateChatterFromTags(m)
		v.unlockme()

		c.InRoom[nick] = v
		c.Logf(LogCatSystem, "User State updated from %s in %s", nick, m.Trailing())

	case TwitchCmdHostTarget:
		channelDoingTheHost := m.Params[0]
		subs := regexHostMatch.FindStringSubmatch(m.Trailing())
		if len(subs) != 3 {
			log.Printf("HOST ERROR 1 - Parsing [%s]\n%s", subs, m)
			return
		}

		viewerNum := 0
		if subs[2] != "-" {
			var err error
			viewerNum, err = strconv.Atoi(subs[2])
			if err != nil {
				log.Printf("HOST ERROR 2 - Parsing [%s]\n%s", subs[2], m)
			}
		}
		c.hostUpdate(IrcNick(channelDoingTheHost), IrcNick(subs[1]), viewerNum)

	case TwitchCmdReconnect:
		printDebugTag(m)

	case IrcCmdMode:
		// printDebugTag(m)
		// TODO :: Handle Mode

	case IrcCmdAction:
		fallthrough
	case IrcCmdPrivmsg:
		// < PRIVMSG #<channel> :This is a sample message
		// > :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :This is a sample message
		// > @badges=<badges>;bits=<bits>;color=<color>;display-name=<display-name>;emotes=<emotes>;id=<id>;mod=<mod>;room-id=<room-id>;subscriber=<subscriber>;turbo=<turbo>;user-id=<user-id>;user-type=<user-type> :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :<message>
		// Trailing = <message>
		// Param[0] channel

		nick := IrcNick(m.Name)

		// SPECIAL CASES because twitch
		switch nick {
		case "jtv":
			// TODO :: Add reaction hook
			c.Logf(LogCatSystem, "%s", m.Trailing())
			return
		case "twitchnotify":
			// TODO :: Add reaction hook
			// Sub notify [[:word:]] just subscribed to [[:word:]]
			c.Logf(LogCatSystem, "%s", m.Trailing())
			return
		}

		v, err := c.viewers.Find(nick)
		if err != nil {
			log.Printf("PRIV MSG ERROR [%s] not found\n%s", nick, err)
			return
		}

		v.CreateChatter(nick)
		v.lockme()
		v.Chatter.updateChatterFromTags(m)
		v.unlockme()

		// Priority Badge
		singleBadge := v.Chatter.SingleBadge()

		// Handle Bits
		bVal := 0
		bits, ok := m.Tags[TwitchTagBits]
		if ok {
			bVal, err = strconv.Atoi(string(bits))
			if err != nil {
				log.Print("Bits error -", m, err)
				bVal = 0
			}

			if bVal > 0 {
				c.forwardAlert(AlertBits, v.Chatter.Nick, bVal)
			}
		}

		// Handle Emotes
		var emoList EmoteReplaceListFromBack
		emoteList, ok := m.Tags[TwitchTagMsgEmotes]
		if ok {
			emoList, err = emoteTagToList(emoteList)
			if err != nil {
				log.Printf("Unable to parse Emote Tag [%s]\n%s", emoteList, err)
				emoList = []EmoteReplace{}
			}
		}

		// Output
		// # 111111 S10 nick emote
		msgBody := m.Trailing()
		if strings.HasPrefix(msgBody, "ACTION") {
			c.LogLine(
				MakeLogLineMsg(LogCatAction,
					LogLineParsedMsg{
						UserID:  v.TwitchID,
						Nick:    v.Chatter.Nick,
						Bits:    bVal,
						Badge:   singleBadge,
						Content: strings.TrimLeft(msgBody, "ACTION"),
						Emotes:  emoList,
					}))

		} else {
			c.LogLine(
				MakeLogLineMsg(LogCatMsg,
					LogLineParsedMsg{
						UserID:  v.TwitchID,
						Nick:    v.Chatter.Nick,
						Bits:    bVal,
						Badge:   singleBadge,
						Content: msgBody,
						Emotes:  emoList,
					}))

		}

	case IrcCmdNotice:
		msgID, ok := m.Tags[TwitchTagMsgID]
		if !ok {
			printDebugTag(m)
			return
		}
		switch msgID {
		// Notice Host Messages - Not reacting to these because they lack the viewer count
		case TwitchMsgHostOffline:
			c.Log(LogCatSystem, m.Trailing())
		case TwitchMsgHostOff:
			c.Log(LogCatSystem, m.Trailing())
		case TwitchMsgHostOn:
			c.Log(LogCatSystem, m.Trailing())
			/////////

		case TwitchMsgR9kOff:
			c.mode.r9k = false
			c.Log(LogCatSystem, m.Trailing())
		case TwitchMsgR9kOn:
			c.mode.r9k = true
			c.Log(LogCatSystem, m.Trailing())
		case TwitchMsgSlowOff:
			c.mode.slowMode = false
			c.Log(LogCatSystem, m.Trailing())
		case TwitchMsgSlowOn:
			c.mode.slowMode = true
			c.Log(LogCatSystem, m.Trailing())
		case TwitchMsgSubsOff:
			c.mode.subsOnly = false
			c.Log(LogCatSystem, m.Trailing())
		case TwitchMsgSubsOn:
			c.mode.subsOnly = true
			c.Log(LogCatSystem, m.Trailing())

		case TwitchMsgEmoteOnlyOff:
			c.mode.emoteOnly = false
			c.Log(LogCatSystem, m.Trailing())
		case TwitchMsgEmoteOnlyOn:
			c.mode.emoteOnly = true
			c.Log(LogCatSystem, m.Trailing())

		case TwitchMsgAlreadyEmoteOnlyOff:
			c.Log(LogCatSilent, "MODE ALREADY SET: "+m.Trailing())
		case TwitchMsgAlreadyEmoteOnlyOn:
			c.Log(LogCatSilent, "MODE ALREADY SET: "+m.Trailing())
		case TwitchMsgAlreadyR9kOff:
			c.Log(LogCatSilent, "MODE ALREADY SET: "+m.Trailing())
		case TwitchMsgAlreadyR9kOn:
			c.Log(LogCatSilent, "MODE ALREADY SET: "+m.Trailing())
		case TwitchMsgAlreadySubsOff:
			c.Log(LogCatSilent, "MODE ALREADY SET: "+m.Trailing())
		case TwitchMsgAlreadySubsOn:
			c.Log(LogCatSilent, "MODE ALREADY SET: "+m.Trailing())

		case TwitchMsgUnrecognizedCmd:
			c.Log(LogCatSilent, m.Trailing())

		case TwitchUserNoticeReSub:
			// TODO :: Add reaction hook
			c.Log(LogCatSystem, m.Trailing())

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
		c.tickRoomActive()

	default:
		log.Printf("IRC ???[%s] \t %+v", m.Command, m)
	}
}

// Sub - Create a Subsciption to Alerts
func (c *Chat) Sub(subName string, topics []LogCat) chan LogLineParsed {
	newSub := make(chan LogLineParsed, 10)
	c.logger.newSubs <- subToChatPump{
		Name:   subName,
		Subbed: topics,
		C:      newSub,
	}

	return newSub
}

// Unsub - Kill a Subscription Channel
func (c *Chat) Unsub(deadChannel chan LogLineParsed) {
	c.logger.killSubs <- subToChatPump{
		Name: "dead",
		C:    deadChannel,
	}
}
