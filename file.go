package twitch

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	dumpFilePattern = "./data/dump_%s_%d.bin"
)

var (
	regexChatLogFileMatch = regexp.MustCompile("([[:word:]]*)_chat.log")
	regexDumpFileMatch    = regexp.MustCompile("dump_([[:word:]]*)_([0-9]*).bin")
	regexChatNewLog       = regexp.MustCompile("[\\+\\-]* New Log \\[([[:word:]]*)\\] [\\+\\-]* ([0-9].*)")
)

// DumpState - Dump the Internal State to File
func (ah *Client) DumpState() error {
	f, err := os.Create(fmt.Sprintf(dumpFilePattern, ah.RoomName, time.Now().Unix()))
	if err != nil {
		return err
	}

	enc := gob.NewEncoder(f)
	for _, v := range ah.Viewers {
		err = enc.Encode(v)
		if err != nil {
			f.Close()
			return err
		}
	}

	fmt.Printf("Dumped data to file: %s", f.Name())
	f.Close()
	return nil
}

// GetDumpListing - Listing of All Fumps in this folder
func GetDumpListing(chanName string) [][]string {
	sList := [][]string{}

	files, err := ioutil.ReadDir("./data/")
	if err != nil {
		log.Printf("Unable to ReadDir: %s", err)
		return nil
	}

	ignoreNameMatch := len(chanName) == 0

	for _, file := range files {
		res := regexDumpFileMatch.FindStringSubmatch(file.Name())
		if len(res) == 3 {
			if ignoreNameMatch || chanName == res[1] {
				sList = append(sList, res)
			}
		}
	}

	return sList
}

// GetChatLogListing - Listing of All Chat Logs in this folder
func GetChatLogListing() []string {
	sList := []string{}

	files, err := ioutil.ReadDir("./data/")
	if err != nil {
		log.Printf("Unable to ReadDir: %s", err)
		return nil
	}

	for _, file := range files {
		res := regexChatLogFileMatch.FindStringSubmatch(file.Name())
		if len(res) == 2 {
			sList = append(sList, res[1])
		}
	}

	return sList
}

// LoadDumpForAnalysis - Load Viewer Dump for Analysis
func LoadDumpForAnalysis(filename string) (*HistoricViewerData, error) {
	var hvd HistoricViewerData

	res := regexDumpFileMatch.FindStringSubmatch(filename)
	if len(res) != 3 {
		return nil, fmt.Errorf("Filename is invalid format lazy I know: [%s]", filename)
	}

	// Name
	hvd.Name = IrcNick(res[1])

	// Time
	unixTime, err := strconv.ParseInt(res[2], 10, 64)
	if err != nil {
		return nil, err
	}
	hvd.Timestamp = time.Unix(unixTime, 0)
	hvd.ViewerData = make(map[ID]Viewer)

	// Open file for Decoding
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)

	// Pull out all Viewers
	v := Viewer{}
	for err := dec.Decode(&v); err != io.EOF; err = dec.Decode(&v) {
		if err != nil {
			return nil, err
		}

		hvd.ViewerData[v.TwitchID] = v
	}

	return &hvd, nil
}

// LoadChatForAnalysis - Load Chat Log for Analysis
func LoadChatForAnalysis(filename string) (*HistoricChatLog, error) {
	var hc HistoricChatLog

	res := regexChatLogFileMatch.FindStringSubmatch(filename)
	if len(res) != 2 {
		return nil, fmt.Errorf("Filename is invalid format lazy I know: [%s]", filename)
	}

	// Name
	hc.Name = IrcNick(res[1])
	hc.LogLinesByDay = make(map[time.Time][]LogLineParsed)

	// Open file for Decoding
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	currT := time.Time{}
	ls := bufio.NewScanner(f)
	currDay := []LogLineParsed{}
	for ls.Scan() {
		line := ls.Text()
		llp, err := ParseLogLine(line)
		if err != nil {
			return nil, fmt.Errorf("Unabe to parse line\n%s\n%s", line, err)
		}

		subs := regexChatNewLog.FindStringSubmatch(llp.Body)
		if len(subs) == 3 {
			if len(currDay) > 0 {
				hc.LogLinesByDay[currT] = currDay
				currDay = []LogLineParsed{}
			}

			currT, err = time.Parse(time.RFC822Z, subs[2])
			if err != nil {
				return nil, fmt.Errorf("Unable to parse date [%s]\n%s", subs[2], err)
			}
		}

		currDay = append(currDay, *llp)
	}

	if len(currDay) > 0 {
		hc.LogLinesByDay[currT] = currDay
	}

	return &hc, nil
}
