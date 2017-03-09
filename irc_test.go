package twitch

import (
	"testing"

	"github.com/go-irc/irc"
)

var (
	msgList = []string{
		":tmi.twitch.tv 001 kimau :Welcome, GLHF!",
		":tmi.twitch.tv 002 kimau :Your host is tmi.twitch.tv",
		":tmi.twitch.tv 003 kimau :This server is rather new",
		":tmi.twitch.tv 004 kimau :-",
		":tmi.twitch.tv 375 kimau :-",
		":tmi.twitch.tv 372 kimau :You are in a maze of twisty passages, all alike.",
		":tmi.twitch.tv 376 kimau :>",
		":tmi.twitch.tv CAP * ACK :twitch.tv/membership",
		":tmi.twitch.tv CAP * ACK :twitch.tv/tags",
		":tmi.twitch.tv CAP * ACK :twitch.tv/commands",
		":kimau!kimau@kimau.tmi.twitch.tv JOIN #kimau",
		":kimau.tmi.twitch.tv 353 kimau = #kimau :kimau ronni fred wilma",
		":kimau.tmi.twitch.tv 353 kimau = #kimau :pickles",
		":kimau.tmi.twitch.tv 366 kimau #kimau :End of /NAMES list",
		"@badges=broadcaster/1,premium/1;color=#C705C0;display-name=Kimau;emote-sets=0,42,451,2178,16010,19194,21297;mod=0;subscriber=0;user-type= :tmi.twitch.tv USERSTATE #kimau",
		"@broadcaster-lang=;emote-only=0;followers-only=-1;r9k=0;slow=0;subs-only=0 :tmi.twitch.tv ROOMSTATE #kimau",
		":tmi.twitch.tv HOSTTARGET #kimau :obezianka -",
		"@msg-id=host_on :tmi.twitch.tv NOTICE #kimau :Now hosting obezianka.",
	}
)

///////////////////////////////////////////////////////////////////////////////
type DummyAuth struct {
}

func (da *DummyAuth) GetIrcAuth() (hasauth bool, name string, pass string, addr string) {
	return true, "kimau", "pass", "irc.server.com:6667"
}

type DummyWriteRead struct {
}

func (dr *DummyWriteRead) Read(p []byte) (n int, err error)  { return len(p), nil }
func (dr *DummyWriteRead) Write(p []byte) (n int, err error) { return len(p), nil }

type DummyHandler struct {
}

func (dh *DummyHandler) Handle(c *irc.Client, m *irc.Message) {

}

type DummyViewProvider struct {
	Viewers map[ID]*Viewer
}

func (dvp *DummyViewProvider) GetNick() IrcNick { return "kimau" }
func (dvp *DummyViewProvider) GetViewer(id ID) *Viewer {
	v, ok := dvp.Viewers[id]
	if !ok {
		v = &Viewer{
			TwitchID: id,
			User: &User{
				ID:          id,
				Name:        IrcNick("DummyName" + GenerateRandomString(4)),
				DisplayName: "Name" + GenerateRandomString(6),
			},
		}
	}

	return v
}
func (dvp *DummyViewProvider) FindViewer(nick IrcNick) *Viewer {
	for _, v := range dvp.Viewers {
		if v.User.Name == nick {
			return v
		}
	}

	id := ID(GenerateRandomString(10))
	v := &Viewer{
		TwitchID: id,
		User: &User{
			ID:          id,
			Name:        nick,
			DisplayName: string(nick),
		},
	}
	return v
}
func (dvp *DummyViewProvider) UpdateViewers(nickList []IrcNick) []*Viewer {
	vList := []*Viewer{}
	for _, name := range nickList {
		vList = append(vList, dvp.FindViewer(name))
	}

	return vList
}

///////////////////////////////////////////////////////////////////////////////

func TestIrcMessage(t *testing.T) {

	_, nick, pass, serverAddr := (&DummyAuth{}).GetIrcAuth()

	chat := &Chat{
		Server:  serverAddr,
		verbose: *flagIrcVerbose,
		config: irc.ClientConfig{
			Nick: nick,
			Pass: pass,
			User: "Username",
			Name: "Full Name",
		},
		viewers: &DummyViewProvider{},
	}

	chat.SetupLogWriter(&chat.msgLog)
	chat.log.Println("+------------ New Log ------------+")
	chat.config.Handler = chat

	ircClient := irc.NewClient(&DummyWriteRead{}, chat.config)

	for _, v := range msgList {
		m, err := irc.ParseMessage(v)
		if err != nil {
			t.Log(err)
			t.Fail()
		}

		chat.Handle(ircClient, m)
	}
}
