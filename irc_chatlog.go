package twitch

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	LogCatUnknown  LogCat = '?'
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
	default:
		return "Unknown"
	}
}

// LogLineParsed - Useful for Parsing Log Lines
type LogLineParsed struct {
	StampSeconds int
	Cat          LogCat
	Body         string

	Msg *LogLineParsedMsg
}

// LogLineParsedMsg - Extra Content for Msg Log Entries
type LogLineParsedMsg struct {
	UserID  ID
	Nick    IrcNick
	Bits    int
	Badge   string
	Content string
	Emotes  EmoteReplaceListFromBack
}

//
const (
	ChatLogFormatHTML = `<div class="chatline %s">
		<span class="time">%2d:%02d:%02d</span>
		<span class="content">%s</span>
		</div>`
	ChatLogFormatMsgHTML = `<div class="chatline %s">
		<span class="time">%2d:%02d:%02d</span>
		<span class="id">%s</span>
		<span class="badge %s"></span>
		<span class="nick">%s</span>
		<span class="content">%s</span>
		</div>`
	ChatLogFormatString = "IRC: %2d:%02d:%02d %c%s"
)

var (
	regexLogMsg = regexp.MustCompile("^IRC: *([ 0-9][0-9]):([ 0-9][0-9]):([ 0-9][0-9]) ([^ ])(.*)")
	// # TwitchID badge nick {emoteString}? [bitString]? : body
	regexPrivMsg = regexp.MustCompile("([[:word:]]+) ([[:graph:]]+) ([[:word:]]+)( \\{[0-9,\\|]+\\})?( \\[[[:word:]]+\\])? *: (.*)")
)

// Log - Log to internal message logger
func (c *Chat) Log(lvl LogCat, s string) {
	s = strings.Replace(strings.Replace(s, "\\", "\\\\", -1), "\n", "\\n", -1)

	hour, min, sec := time.Now().Clock()
	fmt.Fprintf(c.msgLogger, ChatLogFormatString, hour, min, sec, lvl, s)
}

// Logf - FMT interface
func (c *Chat) Logf(lvl LogCat, s string, v ...interface{}) {
	c.Log(lvl, fmt.Sprintf(s, v...))
}

// ReadChatFull - Dumps the full in memory buffer of chat
func (c *Chat) ReadChatFull() string {
	return c.logBuffer.String()
}

// ReadChatLine - Read next single Line from Chat
func (c *Chat) ReadChatLine() string {
	return c.logBuffer.NextLine()
}

// ResetChatCursor - Read next single Line from Chat
func (c *Chat) ResetChatCursor() {
	c.logBuffer.ResetCursor()
}

// SetupLogWriter - Set where the log is written to
func (c *Chat) setupLogWriter(newTarget ...io.Writer) {
	c.logBuffer = makeCircLineBuffer(1024 * 1024 * 8)
	c.logBuffer.Reset()
	if newTarget != nil {
		writeList := append(newTarget, c.logBuffer)
		c.msgLogger = io.MultiWriter(writeList...)
	} else {
		c.msgLogger = c.logBuffer
	}

	if c.msgLogger == nil {
		log.Fatalln("Log shouldn't be null")
	}

	ts := time.Now().Format(time.RFC822Z)
	c.Logf(LogCatSilent, "+------------ New Log [%s] ------------+ %s",
		c.viewers.GetRoom().GetNick(), ts)
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

	if (llp.Cat == LogCatAction) || (llp.Cat == LogCatMsg) {
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
		Badge:   subStrings[2],
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
		b, err := strconv.Atoi(subStrings[5][2 : len(subStrings[5])-1])
		if err != nil {
			return fmt.Errorf("Failed to Parse Bits: %s \n %s", subStrings[5], err.Error())
		}

		llp.Msg.Bits = b
	}

	return nil

}

// HTML - Produce HTML for Chat Line
func (llp *LogLineParsed) HTML() string {
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

	return fmt.Sprintf(ChatLogFormatMsgHTML,
		catStr,
		hour, minute, seconds,
		llp.Msg.UserID,
		llp.Msg.Badge,
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
	// TwitchID badge nick {emoteString}? [bitString]? : body
	if len(llp.Msg.Emotes) > 0 && llp.Msg.Bits > 0 {
		llp.Body = fmt.Sprintf("%s %s %s {%s} [%d] : %s", llp.Msg.UserID, llp.Msg.Badge, llp.Msg.Nick, llp.Msg.Emotes, llp.Msg.Bits, llp.Msg.Content)
	} else if len(llp.Msg.Emotes) > 0 {
		llp.Body = fmt.Sprintf("%s %s %s {%s} : %s", llp.Msg.UserID, llp.Msg.Badge, llp.Msg.Nick, llp.Msg.Emotes, llp.Msg.Content)
	} else if llp.Msg.Bits > 0 {
		llp.Body = fmt.Sprintf("%s %s %s [%d] : %s", llp.Msg.UserID, llp.Msg.Badge, llp.Msg.Nick, llp.Msg.Bits, llp.Msg.Content)
	} else {
		llp.Body = fmt.Sprintf("%s %s %s : %s", llp.Msg.UserID, llp.Msg.Badge, llp.Msg.Nick, llp.Msg.Content)
	}
}
