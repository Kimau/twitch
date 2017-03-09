package twitch

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
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

	irc := irc.NewClient(conn, c.config)

	if c.verbose {
		// In Verbose mode log all messages
		irc.Reader.DebugCallback = func(m string) {
			log.Printf("IRC (V) >> %s", m)
		}

		irc.Writer.DebugCallback = func(m string) {
			c.limiter.Limit()
			log.Printf("IRC (V) << %s", m)
		}

	} else {
		// Just Rate Limit
		irc.Writer.DebugCallback = func(string) {
			c.limiter.Limit()
		}
	}

	log.Println("IRC Connected")

	err = irc.Run()
	return err
}

func (c *Chat) respondToWelcome(irc *irc.Client, m *irc.Message) {
	c.InRoom = make(map[IrcNick]*Viewer)

	irc.Write("CAP REQ :twitch.tv/membership")
	irc.Write("CAP REQ :twitch.tv/tags")
	irc.Write("CAP REQ :twitch.tv/commands")
	irc.Write(fmt.Sprintf("JOIN #%s", irc.CurrentNick()))
}

func printDebugTag(m *irc.Message) {
	tags := ""
	for k, v := range m.Tags {
		tags += fmt.Sprintf("%s:\t %s\n", k, v)
	}

	log.Printf("IRC NOT[%s] %s \n%s \n%s", m.Command, m.Trailing(), strings.Join(m.Params, " -\t "), tags)

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
		log.Println("Respondto Welcome")
		c.respondToWelcome(irc, m)

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
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case TwitchCmdGlobalUserState:
		printDebugTag(m)

	case TwitchCmdRoomState:
		chatChanName := m.Trailing()
		c.log.Printf("Room State updated %s", chatChanName)

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
				nick: IrcNick(irc.CurrentNick()),
			}
		}
		c.self.UpdateChatterFromTags(m)
		c.log.Printf("User State updated from %s in %s", c.self.nick, chatChanName)

	case TwitchCmdHostTarget:
		printDebugTag(m)

	case TwitchCmdReconnect:
		printDebugTag(m)

	case IrcCmdPrivmsg:
		// < PRIVMSG #<channel> :This is a sample message
		// > :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :This is a sample message
		// > @badges=<badges>;bits=<bits>;color=<color>;display-name=<display-name>;emotes=<emotes>;id=<id>;mod=<mod>;room-id=<room-id>;subscriber=<subscriber>;turbo=<turbo>;user-id=<user-id>;user-type=<user-type> :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :<message>
		// Trailing = <message>
		// Param[0] channel
		v := c.viewers.FindViewer(IrcNick(m.Name))
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
		printDebugTag(m)

	case IrcCmdPing:

	default:
		log.Printf("IRC ???[%s] \t %+v", m.Command, m)
	}
}

func (vwr *chatter) NameWithBadge() string {
	r := ""
	for n, v := range vwr.badges {
		r += fmt.Sprintf("%s%d", n[0], v)
	}
	r += string(vwr.nick)
	return r
}
