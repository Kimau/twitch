package twitch

import (
	"fmt"
	"testing"

	"os"

	"io"

	"bytes"

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
		":ronni!ronni@ronni.tmi.twitch.tv PRIVMSG #dallas :Kappa Keepo Kappa",
		":reallylongname!reallylongname@ronni.tmi.twitch.tv PRIVMSG #dallas :Annoyingly long message for you to parse Professor X was in a wheelchair in “Logan,” which means Sir Patrick Stewart spent half the movie being carried up stairs and into bathrooms by Hugh Jackman.",
		"@badges=staff/1,bits/1000;bits=100;color=;display-name=dallas;emotes=;id=b34ccfc7-4977-403a-8a94-33c6bac34fb8;mod=0;room-id=1337;subscriber=0;turbo=1;user-id=1337;user-type=staff :ronni!ronni@ronni.tmi.twitch.tv PRIVMSG #dallas :cheer100",
		":tmi.twitch.tv CLEARCHAT #kimau :ronni",
		"@ban-reason=Follow\\sthe\\srules :tmi.twitch.tv CLEARCHAT #kimau :ronni",
		"@badges=staff/1,broadcaster/1,turbo/1;color=#008000;display-name=ronni;emotes=;mod=0;msg-id=resub;msg-param-months=6;room-id=1337;subscriber=1;system-msg=ronni\\shas\\ssubscribed\\sfor\\s6\\smonths!;login=ronni;turbo=1;user-id=1337;user-type=staff :tmi.twitch.tv USERNOTICE #dallas :Great stream -- keep it up!",
		"@badges=global_mod/1,turbo/1;color=#0D4200;display-name=dallas;emotes=25:0-4,12-16/1902:6-10;mod=0;room-id=1337;subscriber=0;turbo=1;user-id=1337;user-type=global_mod :ronni!ronni@ronni.tmi.twitch.tv PRIVMSG #dallas :Kappa Keepo Kappa",
		`@badges=moderator/1,subscriber/12,bits/100000;color=#FF4500;display-name=VishtheMexican;emotes=;id=bc737831-427f-4005-8de6-901bf837fcf5;mod=1;room-id=100101057;sent-ts=1489350022460;subscriber=1;tmi-sent-ts=1489350020017;turbo=0;user-id=103759705;user-type=mod :vishthemexican!vishthemexican@vishthemexican.tmi.twitch.tv PRIVMSG #elvenaimee :oh, "L U L " is that face`,
		"@badges=moderator/1,subscriber/12,bits/100000;color=#FF4500;display-name=VishtheMexican;emotes=;id=6ec55675-9218-441f-87fd-344456198ad7;mod=1;room-id=100101057;sent-ts=1489350024378;subscriber=1;tmi-sent-ts=1489350021930;turbo=0;user-id=103759705;user-type=mod :vishthemexican!vishthemexican@vishthemexican.tmi.twitch.tv PRIVMSG #elvenaimee :i never knew",
		"@badges=;color=;display-name=charlipiccolina;emotes=;id=9af0482a-635b-4b17-b4b5-b88d470ac804;mod=0;room-id=100101057;subscriber=0;tmi-sent-ts=1489350025535;turbo=0;user-id=106846183;user-type= :charlipiccolina!charlipiccolina@charlipiccolina.tmi.twitch.tv PRIVMSG #elvenaimee :it's similar to Harvest Moon!",
		":tmi.twitch.tv HOSTTARGET #kimau :obezianka 3",
		"@msg-id=host_on :tmi.twitch.tv NOTICE #kimau :Now hosting obezianka.",
		":twitchnotify!twitchnotify@twitchnotify.tmi.twitch.tv PRIVMSG #kimau :PurinaMooseChow just subscribed to obezianka!",
		":tmi.twitch.tv HOSTTARGET #kimau :- 500000",
		"@msg-id=host_target_went_offline :tmi.twitch.tv NOTICE #kimau :obezianka has gone offline. Exiting host mode.",
		":nightbot!nightbot@nightbot.tmi.twitch.tv PART #kimau",
		":jtv MODE #kimau -o nightbot",
		":nightbot!nightbot@nightbot.tmi.twitch.tv JOIN #kimau",
		":jtv MODE #kimau +o nightbot",
		":tmi.twitch.tv HOSTTARGET #kimau :willowmvp 1",
		"@msg-id=host_on :tmi.twitch.tv NOTICE #kimau :Now hosting willowmvp.",
		":tmi.twitch.tv HOSTTARGET #kimau :- 0",
		"@msg-id=host_target_went_offline :tmi.twitch.tv NOTICE #kimau :willowmvp has gone offline. Exiting host mode.",
		":tmi.twitch.tv HOSTTARGET #kimau :maharunn 1",
		"@msg-id=host_on :tmi.twitch.tv NOTICE #kimau :Now hosting Maharunn.",
		":skintrader_pl!skintrader_pl@skintrader_pl.tmi.twitch.tv PART #kimau",
		":skintrader_pl!skintrader_pl@skintrader_pl.tmi.twitch.tv JOIN #kimau",
		":tmi.twitch.tv HOSTTARGET #kimau :- 0",
		"@msg-id=host_target_went_offline :tmi.twitch.tv NOTICE #kimau :maharunn has gone offline. Exiting host mode.",
		":skintrader_pl!skintrader_pl@skintrader_pl.tmi.twitch.tv PART #kimau",
		":skintrader_pl!skintrader_pl@skintrader_pl.tmi.twitch.tv JOIN #kimau",
		":skintrader_pl!skintrader_pl@skintrader_pl.tmi.twitch.tv PART #kimau",
		"@display-name=CohhKittenBot;emotes;mod=1;room-id=26610234;tmi-sent-ts=1489766625363;user-id=91400118;user-type=mod;badges=moderator/1,subscriber/12;color=#FF0000;id=5060a3e9-2d44-42c2-897a-07feb6db6edc;subscriber=1;turbo=0 :cohhkittenbot!cohhkittenbot@cohhkittenbot.tmi.twitch.tv CTCP #cohhcarnage :ACTION curls up for a nap",
		"@id=d08d3654-8129-4cec-91e4-1ca18625ebe1;login=mcgerr;tmi-sent-ts=1494426407321;emotes;turbo=0;display-name=McGerr;msg-id=resub;msg-param-months=2;msg-param-sub-plan-name=Channel\\sSubscription\\s(CohhCarnage);msg-param-sub-plan=1000;system-msg=McGerr\\sjust\\ssubscribed\\swith\\sa\\s$4.99\\ssub.\\sMcGerr\\ssubscribed\\sfor\\s2\\smonths\\sin\\sa\\srow!;user-id=90548818;badges=subscriber/0;mod=0;room-id=26610234;subscriber=1;user-type;color :tmi.twitch.tv USERNOTICE #cohhcarnage :My second month! Love your stream, you're the best!",
	}
)

func init() {
	os.RemoveAll("./test")
	err := os.MkdirAll("./test/data", os.ModePerm)
	if err != nil {
		fmt.Println("Creating Folder", err)
	}
	os.Chdir("./test")
}

///////////////////////////////////////////////////////////////////////////////
type DummyAuth struct {
}

func (da *DummyAuth) GetIrcAuth() (hasauth bool, name string, pass string) {
	return true, "kimau", "pass"
}

type DummyWriteRead struct {
	Input  io.Reader
	Output io.Writer
}

func (dr *DummyWriteRead) Read(p []byte) (n int, err error)  { return dr.Input.Read(p) }
func (dr *DummyWriteRead) Write(p []byte) (n int, err error) { return dr.Output.Write(p) }

type DummyHandler struct {
}

func (dh *DummyHandler) Handle(c *irc.Client, m *irc.Message) {

}

///////////////////////////////////////////////////////////////////////////////

func TestIrcMessage(t *testing.T) {

	chat, err := createIrcClient(&DummyAuth{}, &DummyViewProvider{}, "")
	chat.config.Handler = chat

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	testBuffer := bytes.NewBufferString("")

	chat.StartRunLoop(&DummyWriteRead{
		Input:  testBuffer,
		Output: os.Stdout,
	})

	// Last M
	lastM := msgList[0]
	defer func() {
		if lastM != "" {
			t.Logf("PANIC\n_____\n%s\n________", lastM)
		}
	}()

	for _, v := range msgList {
		lastM = v
		_, err := testBuffer.WriteString(v)
		if err != nil {
			t.Logf("Write Msg Error %s", err.Error())
		}
	}
	lastM = ""

	t.Log("___________________________")

	for i, llp := range chat.logger.ChatLines {
		pup, err := ParseLogLine(llp.String())

		if err != nil {
			t.Logf("LOG LINE PARSE FAIL [%d]: \n%s\n%s", i, llp.String(), err.Error())
			t.Fail()
			continue
		}

		if pup.Msg != nil && llp.Msg != nil &&
			pup.StampSeconds == llp.StampSeconds &&
			pup.Cat == llp.Cat &&
			pup.Body == llp.Body &&

			pup.Msg.UserID == llp.Msg.UserID &&
			pup.Msg.Nick == llp.Msg.Nick &&
			pup.Msg.Bits == llp.Msg.Bits &&
			pup.Msg.Badge == llp.Msg.Badge &&
			pup.Msg.Content == llp.Msg.Content &&
			len(pup.Msg.Emotes) == len(llp.Msg.Emotes) {

			for i := range pup.Msg.Emotes {
				if pup.Msg.Emotes[i] != llp.Msg.Emotes[i] {
					t.Logf("LOG LINE No match [%d] EMOTE:\n%s%s", i, pup.Msg.Emotes, llp.Msg.Emotes)
					t.Fail()
				}
			}

			continue
		}

		if *pup != llp {
			t.Logf("LOG LINE No match [%d]: \n%s%s", i, llp.String(), pup.String())
			t.Fail()
		}
	}
}

func TestEmoteTagProcessor(t *testing.T) {

	emoteTagTest := []struct {
		input irc.TagValue
		res   EmoteReplaceListFromBack
		err   error
	}{
		{"", EmoteReplaceListFromBack{}, nil},
		{"25:0-4,12-16/1902:6-10", EmoteReplaceListFromBack{
			{25, 12, 16}, {1902, 6, 10}, {25, 0, 4},
		}, nil},
	}

	for i, ett := range emoteTagTest {
		x, err := emoteTagToList(ett.input)
		if len(x) != len(ett.res) {
			t.Logf("%d TestEmoteTagProcessor RES FAIL \n%v\n!=\n%v", i, x, ett.res)
			t.Fail()
		}

		for subI := 0; subI < len(x); subI++ {
			if x[subI] != ett.res[subI] {
				t.Logf("%d %d TestEmoteTagProcessor RES FAIL \n%v\n!=\n%v", i, subI, x[subI], ett.res[subI])
				t.Fail()
			}
		}

		if err != ett.err {
			t.Logf("%d TestEmoteTagProcessor ERROR FAIL \n%v\n!=\n%v", i, err, ett.err)
			t.Fail()
		}
	}

}

func TestParser(t *testing.T) {

	for i, s := range []string{
		"IRC: 00:00:00 *Blank Message",
		"IRC: 00:00:00 _Blank Message",
		"IRC: 00:00:00 ~Blank Message",
		"IRC: 00:00:00 ?Blank Message",
		"IRC: 00:00:00 #59727914 . morbiddezirez : fine",
		"IRC: 00:00:00 #59727914 S6 morbiddezirez : fine",
		"IRC: 00:00:00 !31527093 P buttrot : Good in West US",
	} {
		_, err := ParseLogLine(s)
		if err != nil {
			t.Logf("PARSE FAIL: %d - %s", i, err)
			t.Fail()
		}
	}
}
