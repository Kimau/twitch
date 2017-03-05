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

type ircNick string

type chatter struct {
	nick        ircNick
	displayName string
	emoteSets   map[int]int
	bits        int

	mod      bool
	sub      bool
	userType string
	badges   map[string]int
	color    string

	lastActive time.Time
}

type chatMode struct {
	subsOnly      bool
	lang          string
	emoteOnly     bool
	followersOnly bool
	slowMode      bool
	r9k           bool
}

type Chat struct {
	Server  string
	verbose bool
	config  irc.ClientConfig
	limiter *rate.Limiter

	self *chatter
	mode     *chatMode

	messageOfTheDay []string
	chatterNames    []ircNick
	activeChat map[TwitchID]ircNick 

	log    *log.Logger
	msgLog bytes.Buffer

	client *Client
}

// IrcAuthProvider - Provides Auth normally expects UserAuth
type IrcAuthProvider interface {
	GetIrcAuth() (hasauth bool, name string, pass string, addr string)
}

func init() {

}

func createIrcClient(auth IrcAuthProvider, client *Client) (*Chat, error) {

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
		activeChat: make(map[TwitchID]ircNick),

		client: client,
	}

	chat.setupLog(&chat.msgLog)
	chat.log.Println("+------------ New Log ------------+")
	chat.config.Handler = chat
	chat.limiter = rate.NewLimiter(rate.Every(limitIrcMessageTime), limitIrcMessageNum)

	return chat, nil
}

func (c *Chat) setupLog(newTarget io.Writer) {
	c.log = log.New(newTarget, "IRC: ", log.Ltime)
	if c.log == nil {
		log.Fatalln("Log shouldn't be null")
	}
}

func (c *Chat) GetLog() *bytes.Buffer {
	return &c.msgLog
}

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
	c.activeChat = make(map[TwitchID]ircNick)
	c.chatterNames = []ircNick{}

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

func (c *Chat) UpdateChatterFromTags(chatterName ircNick, m *irc.Message) *chatter {
	var cu *chatter
	id, ok := m.Tags[TwitchTagUserId]
	if ok {
		vwr := c.client.GetViewer(TwitchID(id))

		if vwr.Chatter == nil {
			vwr.Chatter = &chatter{
				nick: chatterName,
			}
		}
		cu = vwr.Chatter
	} else {
		if c.self == nil {
			c.self = &chatter{
				nick: ircNick( c.client.GetNick()),
			}
		}

		cu = c.self
	}
	cu.lastActive = time.Now()

	for tagName, tagVal := range m.Tags {
		switch tagName {

		case TwitchTagUserTurbo:
			if cu.badges == nil {
				cu.badges = make(map[string]int)
			}
			cu.badges[TwitchTagUserTurbo] = 1

		case TwitchTagUserBadge:
			cu.badges = make(map[string]int)
			for _, badgeStr := range strings.Split(string(tagVal), ",") {
				iVal := 0
				t := strings.Split(badgeStr, "/")
				testVal, err := strconv.Atoi(t[1])
				if err != nil {
					log.Println(tagName, badgeStr, err)
				} else {
					iVal = testVal
				}
				cu.badges[t[0]] = iVal
			}

		case TwitchTagUserColor:
			cu.color = string(tagVal)
		case TwitchTagUserDisplayName:
			cu.displayName = string(tagVal)
		case TwitchTagUserEmoteSet:
			emoteStrings := strings.Split(string(tagVal), ",")
			cu.emoteSets = make(map[int]int)
			for _, v := range emoteStrings {
				vInt, err := strconv.Atoi(v)
				if err != nil {
					log.Println(tagName, tagVal, err)
				} else {
					cu.emoteSets[vInt] = 1
				}
			}
		case TwitchTagUserMod:
			intVal, err := strconv.Atoi(string(tagVal))
			if err != nil {
				log.Println(tagName, tagVal, err)
			} else {
				cu.mod = (intVal > 0)
			}
		case TwitchTagUserSub:
			intVal, err := strconv.Atoi(string(tagVal))
			if err != nil {
				log.Println(tagName, tagVal, err)
			} else {
				cu.sub = (intVal > 0)
			}
		case TwitchTagUserType:
			cu.userType = string(tagVal)
		}
	}

	return cu
}

func (c *Chat) ProcessNameList(printNames bool) {
	uList, err := c.client.User.GetByName(c.chatterNames)
	if(err != nil) {
		log.Println(err)
		return
	}

	for _, u :=range uList {
		c.activeChat[u.ID] = u.Name

		v, ok := c.client.Viewers[u.ID]
		if(!ok) {
			c.client.Viewers[u.ID] := &Viewer{
				TwitchID: u.ID,
				User: &u,
				client:c.client, 
			}
		}
	}
	
	if printNames {
		nickformated := ""
			nickLength := 15
			
			for i, v := range c.chatterNames {

				vStr := string(v)
		
				if len(vStr) > nickLength {
					vStr = vStr[0:nickLength]
				}
				for len(vStr) < nickLength {
					vStr += " "
				}
				if i > 0 && (i%4) == 0 {
					nickformated += "\n\t" + vStr
				} else {
					nickformated += "\t" + vStr
				}
			}
			c.log.Printf("--- Names ---\n%s\n-------------", nickformated)		
	}
}

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
	for _, in := range strings.Split(m.Trailing(), " ") {
		c.chatterNames = append(c.chatterNames, ircNick(in))
	}

	case IrcReplyEndofnames:
		c.ProcessNameList(true)
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
		nick := ircNick(m.Name)
		v := c.client.FindViewerIdByName(nick)
		if(v != nil) {
			c.activeChat[v.TwitchID] = v.Nick()
			return
		}
			uList, err := c.client.User.GetByName([]ircNick{nick})
			if(err != nil) {
				log.Printf("Unable to find %s", nick)
				return
			}
			
		c.activeChat[uList[0].ID] = nick

	case IrcCmdPart: // User Parted Channel
		c.log.Printf("PART %s", m.Name)
		nick := ircNick(m.Name)
		v := c.client.FindViewerIdByName(nick)
		if(v != nil) {

		delete(c.activeChat, v.TwitchID)

			return 
		}

			uList, err := c.client.User.GetByName([]ircNick{nick})
			if(err != nil) {
				log.Printf("Unable to find %s", nick)
				return
			}
			
		c.activeChat[uList[0].ID] = nick

		
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
		chatterName := ircNick(irc.CurrentNick())

		c.log.Printf("User State updated from %s in %s", chatterName, chatChanName)
		c.UpdateChatterFromTags(chatterName, m)

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
		v := c.UpdateChatterFromTags(ircNick(m.Name), m)
		bits, ok := m.Tags[TwitchTagBits]

		if ok {
			bVal, err := strconv.Atoi(string(bits))
			if err != nil {
				log.Print("Bits error -", m, err)
			} else {
				v.bits += bVal
				c.log.Printf("BITS %d \t %s: \t%s", 
				bVal, v.NameWithBadge(), m.Trailing())
			}

		} else {
			c.log.Printf("%s: \t%s", v.NameWithBadge(), m.Trailing())
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
	for n,v := vwr.badges {
		r += n[0] + v
 	}
	r += string(vwr.nick)
	return r
}
