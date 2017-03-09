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
		"@msg-id=subs_on :tmi.twitch.tv NOTICE #kimau :This room is now in subscribers-only mode.",
		"@msg-id=subs_off :tmi.twitch.tv NOTICE #kimau :This room is no longer in subscribers-only mode.",
		"@msg-id=r9k_on :tmi.twitch.tv NOTICE #kimau :This room is now in r9k mode.",
		"@msg-id=r9k_off :tmi.twitch.tv NOTICE #kimau :This room is no longer in r9k mode.",
		"@msg-id=slow_on :tmi.twitch.tv NOTICE #kimau :This room is now in slow mode. You may send messages every 120 seconds.",
		"@msg-id=slow_off :tmi.twitch.tv NOTICE #kimau :This room is no longer in slow mode.",
		"@msg-id=host_on :tmi.twitch.tv NOTICE #kimau :Now hosting GikkBot.",
		"@msg-id=host_off :tmi.twitch.tv NOTICE #kimau :Exited host mode.",
		"@msg-id=unrecognized_cmd :tmi.twitch.tv NOTICE #kimau :Unrecognized command: /do",
		"@color=#0D4200;display-name=dallas;emote-sets=0,33,50,237,793,2126,3517,4578,5569,9400,10337,12239;turbo=0;user-id=1337;user-type=admin :tmi.twitch.tv GLOBALUSERSTATE",
		"@badges=global_mod/1,turbo/1;color=#0D4200;display-name=dallas;emotes=25:0-4,12-16/1902:6-10;mod=0;room-id=1337;subscriber=0;turbo=1;user-id=1337;user-type=global_mod :ronni!ronni@ronni.tmi.twitch.tv PRIVMSG #dallas :Kappa Keepo Kappa",
		"@badges=staff/1,bits/1000;bits=100;color=;display-name=dallas;emotes=;id=b34ccfc7-4977-403a-8a94-33c6bac34fb8;mod=0;room-id=1337;subscriber=0;turbo=1;user-id=1337;user-type=staff :ronni!ronni@ronni.tmi.twitch.tv PRIVMSG #dallas :cheer100",
		":tmi.twitch.tv CLEARCHAT #kimau :ronni",
		"@ban-reason=Follow\\sthe\\srules :tmi.twitch.tv CLEARCHAT #kimau :ronni",
		"@badges=staff/1,broadcaster/1,turbo/1;color=#008000;display-name=ronni;emotes=;mod=0;msg-id=resub;msg-param-months=6;room-id=1337;subscriber=1;system-msg=ronni\\shas\\ssubscribed\\sfor\\s6\\smonths!;login=ronni;turbo=1;user-id=1337;user-type=staff :tmi.twitch.tv USERNOTICE #dallas :Great stream -- keep it up!",
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

	t.Log(&chat.msgLog)
}
