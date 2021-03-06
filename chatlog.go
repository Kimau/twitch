package twitch

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//
const (
	ChatLogFormatHTML = `<div class="chatline %s">
 		<span class="time">%2d:%02d:%02d</span>
 		<span class="content">%s</span>
 		</div>`
	ChatLogFormatMsgHTML = `<div class="chatline %s">
 		<span class="time">%2d:%02d:%02d</span>
 		<span class="id">%s</span>
 		<span class="badge">%s</span>
 		<span class="nick">%s</span>
 		<span class="content">%s</span>
 		</div>`
	ChatLogFormatMsgExtraHTML = `<div class="chatline %s" style="background:%s;">
 		<span class="time">%2d:%02d:%02d</span>
 		<span class="id">%s</span>
 		<span class="badge">%s</span>
 		<span class="nick">%s</span>
 		<span class="content">%s</span>
 		</div>`
	ChatLogFormatBadgeHTML = `<span class="%s"></span>`
	ChatLogFormatString    = "CHAT: %2d:%02d:%02d %c%s\n"
)

var (
	regexLogMsg = regexp.MustCompile("^CHAT: *([ 0-9][0-9]):([ 0-9][0-9]):([ 0-9][0-9]) ([^ ])(.*)")
	// # TwitchID badge nick {emoteString}? [bitString]? : body
	regexPrivMsg = regexp.MustCompile("([[:word:]]+) \"([[:graph:]]+)\" ([[:word:]]+)( +\\{[0-9,\\|]+\\})?( +\\[[[:word:]]+\\])? *: (.*)")
)

// LogCat - The Type of Log Category message
type LogCat rune

//
const (
	LogCatSystem   LogCat = '*'
	LogCatSilent   LogCat = '_'
	LogCatFiltered LogCat = '~'
	LogCatMsg      LogCat = '#'
	LogCatAction   LogCat = '!'
	LogCatWhisper  LogCat = '>'
	LogCatUnknown  LogCat = '?'

	numInteralLogLines int = 1024
)

// FriendlyName - Produce a friendly name for Cat
func (lc LogCat) FriendlyName() string {
	switch lc {
	case LogCatSystem:
		return "System"
	case LogCatSilent:
		return "Silent"
	case LogCatFiltered:
		return "Filtered"
	case LogCatMsg:
		return "Msg"
	case LogCatAction:
		return "Action"
	case LogCatWhisper:
		return "Whisper"
	default:
		return "Unknown"
	}
}

// LogLineParsed - Useful for Parsing Log Lines
type LogLineParsed struct {
	StampSeconds int    `json:"time"`
	Cat          LogCat `json:"cat"`
	Body         string `json:"body"`

	Msg *LogLineParsedMsg `json:"msg"`
}

// LogLineParsedMsg - Extra Content for Msg Log Entries
type LogLineParsedMsg struct {
	UserID  ID                       `json:"userid"`
	Nick    IrcNick                  `json:"nick"`
	Bits    int                      `json:"bits"`
	Content string                   `json:"content"`
	Emotes  EmoteReplaceListFromBack `json:"emotes"`
}

type subToChatPump struct {
	Name   string
	Subbed []LogCat
	C      chan LogLineParsed
}

type chatLogInteral struct {
	subbedToChat []subToChatPump
	newSubs      chan subToChatPump
	killSubs     chan subToChatPump
	newLines     chan LogLineParsed

	ChatLines [numInteralLogLines]LogLineParsed
	ChatFile  io.Writer

	writeCursor int
	readCursor  int
	hasWrapped  bool
}

// LogLine - Log Line
func (cli *chatLogInteral) LogLine(llp LogLineParsed) {
	// Write to Subs
	cli.newLines <- llp

	// To avoid reusing memory
	safeLine := llp
	if safeLine.Msg != nil {
		msg := *llp.Msg
		safeLine.Msg = &msg
	}

	// Write Line
	cli.ChatLines[cli.writeCursor] = safeLine

	cli.writeCursor++
	if cli.writeCursor >= numInteralLogLines {
		cli.writeCursor -= numInteralLogLines
		cli.hasWrapped = true
	}
	if cli.writeCursor == cli.readCursor {
		cli.readCursor++
		if cli.readCursor > numInteralLogLines {
			cli.readCursor -= numInteralLogLines
		}
	}

	fmt.Fprint(cli.ChatFile, safeLine.String())
}

// LogLine - Log to internal message logger
func (c *Chat) LogLine(llp LogLineParsed) {
	c.logger.LogLine(llp)
}

// Log - Log to internal message logger
func (c *Chat) Log(lvl LogCat, s string) {
	s = strings.Replace(strings.Replace(s, "\\", "\\\\", -1), "\n", "\\n", -1)

	c.logger.LogLine(MakeLogLine(lvl, s))
}

// Logf - FMT interface
func (c *Chat) Logf(lvl LogCat, s string, v ...interface{}) {
	c.Log(lvl, fmt.Sprintf(s, v...))
}

// ReadChatFull - Dumps the full in memory buffer of chat
func (c *Chat) ReadChatFull() []LogLineParsed {
	if c.logger.hasWrapped {
		return append(
			c.logger.ChatLines[c.logger.writeCursor:],
			c.logger.ChatLines[:c.logger.writeCursor]...)
	}

	return c.logger.ChatLines[:c.logger.writeCursor]
}

// ReadChatLine - Read next single Line from Chat
func (c *Chat) ReadChatLine() *LogLineParsed {
	if c.logger.readCursor == c.logger.writeCursor {
		return nil
	}

	l := &c.logger.ChatLines[c.logger.readCursor]
	c.logger.readCursor++
	if c.logger.readCursor > numInteralLogLines {
		c.logger.readCursor -= numInteralLogLines
	}

	return l
}

// ResetChatCursor - Read next single Line from Chat
func (c *Chat) ResetChatCursor() {
	if c.logger.hasWrapped {
		c.logger.readCursor = c.logger.writeCursor + 1
		if c.logger.readCursor > numInteralLogLines {
			c.logger.readCursor -= numInteralLogLines
		}
	} else {
		c.logger.readCursor = 0
	}
}

// MakeLogLine - Make Log Line with current time stamped
func MakeLogLine(cat LogCat, body string) LogLineParsed {
	h, m, s := time.Now().Clock()
	return LogLineParsed{
		Cat:          cat,
		Body:         body,
		StampSeconds: h*60*60 + m*60 + s,
	}
}

// MakeLogLineMsg - Make Log Line Message with current time stamp
func MakeLogLineMsg(cat LogCat, msgData LogLineParsedMsg) LogLineParsed {
	h, m, s := time.Now().Clock()
	llp := LogLineParsed{
		Cat:          cat,
		Body:         "",
		Msg:          &msgData,
		StampSeconds: h*60*60 + m*60 + s,
	}

	llp.UpdateBody()
	return llp
}

// ParseLogLine - Parse a Log Line useful for inspection
func ParseLogLine(fullS string) (*LogLineParsed, error) {
	llp := LogLineParsed{}

	// First Parse
	sBits := regexLogMsg.FindStringSubmatch(fullS)
	if len(sBits) != 6 {
		d := len(sBits)
		return nil, fmt.Errorf("Failed basic parse [%d/6]: %s", d, fullS)
	}

	// Convert Time stamp
	llp.StampSeconds = 0
	mult := []int{0, 60 * 60, 60, 1}
	for i := 1; i < 4; i++ {
		v, e := strconv.Atoi(strings.Trim(sBits[i], " "))
		if e != nil {
			return nil, fmt.Errorf("Problem processing timestamp [%s] : %s", sBits[i], e.Error())
		}
		llp.StampSeconds += v * mult[i]
	}

	llp.Cat = LogCat(sBits[4][0])
	llp.Body = sBits[5]

	if (llp.Cat == LogCatAction) || (llp.Cat == LogCatMsg) || (llp.Cat == LogCatWhisper) {
		err := llp.parseMsgBody()
		return &llp, err
	}

	return &llp, nil
}

func (llp *LogLineParsed) parseMsgBody() error {

	// TwitchID badge nick {emoteString}? [bitString]? : body
	subStrings := regexPrivMsg.FindStringSubmatch(llp.Body)
	if len(subStrings) != 7 {
		d := len(subStrings)
		return fmt.Errorf("Failed basic message parse [%d/7]: \n%s\n%s", d, regexPrivMsg, llp.Body)
	}

	llp.Msg = &LogLineParsedMsg{
		UserID:  ID(subStrings[1]),
		Nick:    IrcNick(subStrings[3]),
		Bits:    0,
		Content: subStrings[6],
	}

	// Parse Emotes
	if len(subStrings[4]) > 2 {
		var err error
		llp.Msg.Emotes, err = ParseEmoteReplaceListFromBack(subStrings[4][2 : len(subStrings[4])-1])
		if err != nil {
			return err
		}
	}

	// Parse Bits
	if len(subStrings[5]) > 2 {
		b, err := strconv.Atoi(subStrings[5][strings.Index(subStrings[5], "[")+1 : strings.Index(subStrings[5], "]")])
		if err != nil {
			return fmt.Errorf("Failed to Parse Bits: %s %s \n %s",
				subStrings[5], subStrings[5][strings.Index(subStrings[5], "[")+1:strings.Index(subStrings[5], "]")], err.Error())
		}

		llp.Msg.Bits = b
	}

	return nil

}

// HTMLBody - Replaces Emotes with <img> tags
func (llp *LogLineParsed) HTMLBody(vp viewerProvider) string {
	if llp.Msg == nil {
		return llp.Body
	}

	return llp.Msg.Emotes.Replace(llp.Msg.Content)
}

// HTML - Produce HTML for Chat Line
func (llp *LogLineParsed) HTML(vp viewerProvider) string {
	seconds := llp.StampSeconds
	hour := seconds / (60 * 60)
	seconds -= hour * 60 * 60
	minute := seconds / 60
	seconds -= minute * 60

	catStr := llp.Cat.FriendlyName()
	if llp.Msg == nil {
		return fmt.Sprintf(ChatLogFormatHTML,
			catStr,
			hour, minute, seconds,
			llp.Body)
	}

	msgContent := llp.Msg.Emotes.Replace(llp.Msg.Content)

	// Multiple Badges
	badgeHTML := ""

	// Get Viewer Data
	v, err := vp.GetData(llp.Msg.UserID)
	if err != nil {
		return fmt.Sprintf(ChatLogFormatMsgHTML,
			catStr,
			hour, minute, seconds,
			llp.Msg.UserID,
			badgeHTML,
			llp.Msg.Nick,
			msgContent)
	}

	// View Lock
	if v.Follower != nil {
		catStr += " follow"
	}

	chatColor := "#DDD"
	if v.Chatter != nil {
		chatColor = v.Chatter.Color

		for badgeID, ver := range v.Chatter.Badges {
			badgeHTML += vp.Client().Badges.BadgeHTML(badgeID, ver)
		}

		return llp.Body
	}
	//

	return fmt.Sprintf(ChatLogFormatMsgExtraHTML,
		catStr,
		chatColor,
		hour, minute, seconds,
		llp.Msg.UserID,
		badgeHTML,
		llp.Msg.Nick,
		msgContent)
}

func (llp *LogLineParsed) String() string {

	seconds := llp.StampSeconds
	hour := seconds / (60 * 60)
	seconds -= hour * 60 * 60
	minute := seconds / 60
	seconds -= minute * 60

	if llp.Msg != nil {
		llp.UpdateBody()
	}

	return fmt.Sprintf(ChatLogFormatString,
		hour, minute, seconds, llp.Cat, llp.Body)
}

// SetTime - Set Time from timestamp
func (llp *LogLineParsed) SetTime(newTime time.Time) {
	hour, min, sec := newTime.Clock()
	llp.StampSeconds = hour*60*60 + min*60 + sec
}

// UpdateBody - Updates the Body string based on the current msg data
func (llp *LogLineParsed) UpdateBody() {
	llp.Body = fmt.Sprintf("%s %s %s %s : %s",
		llp.Msg.UserID, llp.Msg.Nick,
		func(em EmoteReplaceListFromBack) string { // Emote String
			if len(em) == 0 {
				return ""
			}

			return fmt.Sprintf("{%s}", em)
		}(llp.Msg.Emotes),
		func(bits int) string { // Bit String
			if bits <= 0 {
				return ""
			}

			return fmt.Sprintf("[%d]", bits)
		}(llp.Msg.Bits), llp.Msg.Content)

}

// startChatLogPump - Start the internal go routine and create the pump
func startChatLogPump(room IrcNick) *chatLogInteral {

	cli := chatLogInteral{
		newSubs:  make(chan subToChatPump, 10),
		killSubs: make(chan subToChatPump, 10),
		newLines: make(chan LogLineParsed, 10),
		ChatFile: getChatLogWriter(room),
	}

	go cli.run()

	return &cli
}

// run - The Goroutine loop that is close by closing the channel
func (cli *chatLogInteral) run() {

pumpLoop:
	for {
		select {
		case deadSub, ok := <-cli.killSubs:
			if !ok {
				break pumpLoop
			}

			subs := []subToChatPump{}
			for _, sub := range cli.subbedToChat {
				if sub.C == deadSub.C {
					close(deadSub.C)
				} else {
					subs = append(subs, sub)
				}
			}
			cli.subbedToChat = subs

		case newSub, ok := <-cli.newSubs:
			if !ok {
				break pumpLoop
			}

			cli.subbedToChat = append(cli.subbedToChat, newSub)

		case llp := <-cli.newLines:
			cli.postInternal(llp)
		}
	}

	// Close all alert Channels
	for _, sub := range cli.subbedToChat {
		close(sub.C)
	}
}

// postInternal - Manages the Actual Posting of the Line
func (cli *chatLogInteral) postInternal(llp LogLineParsed) {

	// Forward Alert
SubLoop:
	for _, subs := range cli.subbedToChat {
		// Check if Channel is Subbed to this Topic
		for _, t := range subs.Subbed {
			if t == llp.Cat {
				select {
				case subs.C <- llp:
				default: // Non blocking
				}
				continue SubLoop
			}
		}

		// Not Subbed to Topic
	}
}
