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
type LogCat string

//
const (
	LogCatSystem   LogCat = "*"
	LogCatSilent   LogCat = "_"
	LogCatFiltered LogCat = "~"
	LogCatMsg      LogCat = "#"
	LogCatUnknown  LogCat = "!!?"
)

// LogLineParsed - Useful for Parsing Log Lines
type LogLineParsed struct {
	StampSeconds int
	Cat          LogCat
	Filtered     bool
	Body         string

	UserID ID
	Nick   IrcNick
	Bits   int
}

//
var (
	regexLogMsg = regexp.MustCompile("^IRC: *([ 0-9][0-9]):([ 0-9][0-9]):([ 0-9][0-9]) *([\\*\\_]*) +(.*)")
	// # TwitchID badge nick {emoteString}? [bitString]? : body
	regexPrivMsg = regexp.MustCompile("#([[:word:]]+) ([[:word:]]+) ([[:word:]]+) (\\{[[:word:]]+\\})? (\\[[[:word:]]+\\])? : (.*)")
)

// Log - Log to internal message logger
func (c *Chat) Log(lvl LogCat, s string) {
	s = strings.Replace(strings.Replace(s, "\\", "\\\\", -1), "\n", "\\n", -1)
	c.msgLogger.Print(string(lvl) + s)
}

// Logf - FMT interface
func (c *Chat) Logf(lvl LogCat, s string, v ...interface{}) {
	c.Log(lvl, fmt.Sprintf(s, v...))
}

// ParseLog - Parse a Log Line useful for inspection
func (c *Chat) ParseLog(fullS string) (*LogLineParsed, error) {
	llp := LogLineParsed{}

	// First Parse
	sBits := regexLogMsg.FindStringSubmatch(fullS)
	if len(sBits) != 5 {
		return nil, fmt.Errorf("Failed basic parse")
	}

	// Convert Time stamp
	llp.StampSeconds = 0
	mult := []int{1, 60, 60 * 60}
	for i := 0; i < 3; i++ {
		v, e := strconv.Atoi(sBits[i+1])
		if e != nil {
			return nil, fmt.Errorf("Problem processing timestamp [%s] : %s", sBits[i], e.Error())
		}
		llp.StampSeconds += v * mult[i]
	}

	bodyText := sBits[4]

Reprocess:
	switch LogCat(bodyText[0:1]) {
	case LogCatSystem:
		llp.Cat = LogCatSystem
		llp.Body = bodyText[2:]

	case LogCatSilent:
		llp.Cat = LogCatSystem
		llp.Body = bodyText[2:]

	case LogCatFiltered:
		llp.Filtered = true
		bodyText = bodyText[2:]
		goto Reprocess

	case LogCatMsg:
		llp.Cat = LogCatMsg
		subStrings := regexPrivMsg.FindStringSubmatch(bodyText)
		if len(subStrings) == 7 {
			return nil, fmt.Errorf("Failed basic message parse: %s", bodyText)
		}

	default:
		llp.Cat = LogCatUnknown
		llp.Body = bodyText[2:]
	}

	return &llp, nil
}

// SetupLogWriter - Set where the log is written to
func (c *Chat) SetupLogWriter(newTarget ...io.Writer) {
	c.logBuffer.Reset()
	if newTarget != nil {
		writeList := append(newTarget, &c.logBuffer)
		mw := io.MultiWriter(writeList...)
		c.msgLogger = log.New(mw, "IRC: ", log.Ltime)
	} else {
		c.msgLogger = log.New(&c.logBuffer, "IRC: ", log.Ltime)
	}

	if c.msgLogger == nil {
		log.Fatalln("Log shouldn't be null")
	}

	ts := time.Now().Format(time.RFC822Z)
	c.Logf("+------------ New Log [%s] ------------+ %s",
		c.Room, ts)
}
