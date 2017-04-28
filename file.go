package twitch

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
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
	chatFilePattern = "./data/%s_chat.log"
)

var (
	regexChatLogFileMatch = regexp.MustCompile("([[:word:]]*)_chat.log")
	regexDumpFileMatch    = regexp.MustCompile("dump_([[:word:]]*)_([0-9]*).bin")
	regexChatNewLog       = regexp.MustCompile("[\\+\\-]* New Log \\[([[:word:]]*)\\] [\\+\\-]* ([0-9].*)")
)

type tokenData struct {
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	IrcServerAddr string `json:"irc_server"`
}

func (ah *Client) loadSecrets() {
	fileData, err := ioutil.ReadFile("./data/twitch_secret.json")
	if err != nil {
		panic(fmt.Sprintf("Failed to load token data from ./data/secret.json: \n%s", err))
	}

	sd := tokenData{}
	json.Unmarshal(fileData, &sd)

	ah.tokenData = &sd
}

func (ah *Client) loadToken() {
	fileData, err := ioutil.ReadFile("./data/twitch_secret_token.json")
	if err != nil {
		log.Printf("Failed to load saved auth token")
		return
	}

	userAuthTemp := UserAuth{}
	err = json.Unmarshal(fileData, &userAuthTemp)
	if err != nil {
		log.Printf("-------------------\nFailed to Unmarshall auth token.\n %s", err)
		return
	}

	// Check Token
	err = ah.getRootToken(&userAuthTemp)
	if err != nil {
		log.Printf("--------FAIL TOKEN-----------\n%s\n---\n%s", err, ah.AdminAuth)
		return
	}

	// Token is Valid
	if userAuthTemp.Token != nil {
		*ah.AdminAuth = userAuthTemp
		ah.adminHasAuthed()
	}
}

func (ah *Client) saveToken() error {
	b, err := json.Marshal(*ah.AdminAuth)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile("./data/twitch_secret_token.json", b, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// getChatLogWriter - Get Permant Chat Log File
func getChatLogWriter(roomName IrcNick) *os.File {
	// Messy that we don't close this

	filename := fmt.Sprintf("./data/%s_chat.log", roomName)

	w, e := os.OpenFile(
		filename,
		os.O_CREATE|os.O_APPEND, os.ModePerm)

	if e != nil {
		panic(fmt.Sprintf("Unable to create chat log: %s\n%s", filename, e))
	}

	return w
}

var localIrcMsgStoreFile *os.File

// localIrcMsgStore - Simply file Write for raw messages
func localIrcMsgStore() *os.File {
	if localIrcMsgStoreFile != nil {
		return localIrcMsgStoreFile
	}

	// Messy that we don't close this
	var err error
	localIrcMsgStoreFile, err = os.OpenFile(
		"./data/_irc.log",
		os.O_CREATE|os.O_APPEND, os.ModePerm)

	if err != nil {
		panic(fmt.Sprintf("Unable to create irc log: %s\n%s", "./data/_irc.log", err))
	}

	return localIrcMsgStoreFile
}

// DumpViewers - Dump the Internal State to File
func (ah *Client) DumpViewers() error {
	f, err := os.Create(fmt.Sprintf(dumpFilePattern, ah.RoomName, time.Now().Unix()))
	if err != nil {
		return err
	}

	enc := gob.NewEncoder(f)
	for _, vid := range ah.Viewers.AllKeys() {
		v, _ := ah.Viewers.GetCopy(vid)
		err = enc.Encode(v)
		if err != nil {
			f.Close()
			return err
		}
	}

	log.Printf("Dumped data to file: %s", f.Name())
	f.Close()
	return nil
}

// GetDumpListing - Listing of All Fumps in this folder
func GetDumpListing(chanName IrcNick) [][]string {
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
			if ignoreNameMatch || chanName == IrcNick(res[1]) {
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

// LoadMostRecentViewerDump - Load the most recent User Data for User
func LoadMostRecentViewerDump(chanName IrcNick) (*HistoricViewerData, error) {
	listings := GetDumpListing(chanName)
	bigNum := 0

	// Get Highest Num
	for _, item := range listings {
		v, err := strconv.Atoi(item[2])
		if err != nil {
			log.Printf("Cannot Convert: %s", err)
			continue
		}

		if v > bigNum {
			bigNum = v
		}
	}

	if bigNum == 0 {
		return nil, fmt.Errorf("Unable to find any listings")
	}

	fileName := fmt.Sprintf(dumpFilePattern, chanName, bigNum)
	return LoadViewerDumpForAnalysis(fileName)
}

// LoadViewerDumpForAnalysis - Load Viewer Dump for Analysis
func LoadViewerDumpForAnalysis(filename string) (*HistoricViewerData, error) {
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
	for {
		v := Viewer{}
		err := dec.Decode(&v)

		if err != nil {
			if err == io.EOF {
				return &hvd, nil
			}
			return nil, err
		}

		hvd.ViewerData[v.TwitchID] = v

	}
}

// LoadChatForAnalysis - Load Chat Log for Analysis
func LoadChatForAnalysis(room IrcNick) (*HistoricChatLog, error) {
	var hc HistoricChatLog

	filename := fmt.Sprintf(chatFilePattern, room)

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

	hc.LogLinesByDay[currT] = currDay

	return &hc, nil
}
