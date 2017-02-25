package twitch

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"strings"

	"io"

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

type chatViewer struct {
	ID          string
	nick        string
	displayName string
	emoteSets   []int

	mod      bool
	sub      bool
	userType string
	badges   []string
	color    string

	lastActive time.Time
}

type chatMode struct {
	subsOnly      bool
	lang          string
	emoteOnly     bool
	followersOnly bool
	slowMode      bool
}

type Chat struct {
	Server  string
	verbose bool
	config  irc.ClientConfig
	limiter *rate.Limiter

	viewers map[string]*chatViewer
	mode    *chatMode

	messageOfTheDay []string
	viewerNames     []string

	log    *log.Logger
	msgLog bytes.Buffer
}

// IrcAuthProvider - Provides Auth normally expects UserAuth
type IrcAuthProvider interface {
	GetIrcAuth() (hasauth bool, name string, pass string, addr string)
}

func init() {

}

func createIrcClient(auth IrcAuthProvider) (*Chat, error) {

	log.Println("Creating IRC Client")

	hasAuth, nick, pass, serverAddr := auth.GetIrcAuth()
	if !hasAuth {
		return nil, fmt.Errorf("Associated user has no valid Auth")
	}

	client := &Chat{
		Server:  serverAddr,
		verbose: *flagIrcVerbose,
		config: irc.ClientConfig{
			Nick: nick,
			Pass: pass,
			User: "Username",
			Name: "Full Name",
		},
	}

	client.viewers = make(map[string]*chatViewer)
	client.setupLog(&client.msgLog)
	client.log.Println("+------------ New Log ------------+")
	client.config.Handler = client
	client.limiter = rate.NewLimiter(rate.Every(limitIrcMessageTime), limitIrcMessageNum)

	return client, nil
}

func (client *Chat) setupLog(newTarget io.Writer) {
	client.log = log.New(newTarget, "IRC: ", log.Ltime)
	if client.log == nil {
		log.Fatalln("Log shouldn't be null")
	}
}

func (client *Chat) GetLog() *bytes.Buffer {
	return &client.msgLog
}

func (client *Chat) StartRunLoop() error {
	conn, err := net.Dial("tcp", client.Server)
	if err != nil {
		return err
	}

	c := irc.NewClient(conn, client.config)

	if client.verbose {
		// In Verbose mode log all messages
		c.Reader.DebugCallback = func(m string) {
			log.Printf("IRC (V) >> %s", m)
		}

		c.Writer.DebugCallback = func(m string) {
			client.limiter.Limit()
			log.Printf("IRC (V) << %s", m)
		}

	} else {
		// Just Rate Limit
		c.Writer.DebugCallback = func(string) {
			client.limiter.Limit()
		}
	}

	log.Println("IRC Connected")

	err = c.Run()
	return err
}

func (client *Chat) respondToWelcome(c *irc.Client, m *irc.Message) {
	c.Write("CAP REQ :twitch.tv/membership")
	c.Write("CAP REQ :twitch.tv/tags")
	c.Write("CAP REQ :twitch.tv/commands")
	c.Write(fmt.Sprintf("JOIN #%s", c.CurrentNick()))
}

func (client *Chat) Handle(c *irc.Client, m *irc.Message) {
	printOut, ok := ignoreMsgCmd[m.Command]
	if ok {
		if printOut {
			client.log.Printf("IRC [%s] \t %s", m.Command, m.Trailing())
		}
		return
	}

	switch m.Command {
	case IrcReplyWelcome: // 001 is a welcome event, so we join channels there
		log.Println("Respondto Welcome")
		client.respondToWelcome(c, m)

		// Message of the Day
	case IrcReplyMotdstart:
		client.messageOfTheDay = []string{}

	case IrcReplyMotd:
		client.messageOfTheDay = append(client.messageOfTheDay, m.Trailing())

	case IrcReplyEndofmotd:
		client.log.Printf(
			"--- Message of the Day ---\n\t%s \n-----------------------------",
			strings.Join(client.messageOfTheDay, "\n\t"))
		// End of Message of the Day

	// Name List
	case IrcReplyNamreply:
		client.viewerNames = append(client.viewerNames, strings.Split(m.Trailing(), " ")...)

	case IrcReplyEndofnames:
		nickformated := ""
		nickLength := 15
		for i, v := range client.viewerNames {
			_, ok := client.viewers[v]
			if !ok {
				client.viewers[v] = nil
			}

			if len(v) > nickLength {
				v = v[0:nickLength]
			}
			for len(v) < nickLength {
				v += " "
			}
			if i > 0 && (i%4) == 0 {
				nickformated += "\n\t" + v
			} else {
				nickformated += "\t" + v
			}
		}
		client.log.Printf("--- Names ---\n%s\n-------------", nickformated)

		// End of Name List

	case IrcCap:
		if m.Params[0] == "*" && m.Params[1] == "ACK" {
			return
		}

		log.Printf("IRC Unhandled CAP MSG [%s] %s",
			strings.Join(m.Params, "]["),
			m.Trailing())

	case IrcCmdJoin: // User Joined Channel
		client.log.Printf("JOIN %s", m.Name)
		client.viewers[m.Name] = nil

	case IrcCmdPart: // User Parted Channel
		client.log.Printf("PART %s", m.Name)
		delete(client.viewers, m.Name)

	case TwitchCmdClearChat:
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case TwitchCmdGlobalUserState:
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case TwitchCmdRoomState:
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case TwitchCmdUserNotice:
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case TwitchCmdUserState:
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case TwitchCmdHostTarget:
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case TwitchCmdReconnect:
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case IrcCmdPrivmsg:
		// < PRIVMSG #<channel> :This is a sample message
		// > :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :This is a sample message
		// > @badges=<badges>;bits=<bits>;color=<color>;display-name=<display-name>;emotes=<emotes>;id=<id>;mod=<mod>;room-id=<room-id>;subscriber=<subscriber>;turbo=<turbo>;user-id=<user-id>;user-type=<user-type> :<user>!<user>@<user>.tmi.twitch.tv PRIVMSG #<channel> :<message>
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case IrcCmdNotice:
		log.Printf("IRC NOT[%s] \t %+v", m.Command, m)

	case IrcCmdPing:

	default:
		log.Printf("IRC ???[%s] \t %+v", m.Command, m)
	}
}
