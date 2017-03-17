package twitch

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	twitchBase     = "https://api.twitch.tv/kraken/"
	twitchAuthURL  = "https://api.twitch.tv/kraken/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s"
	redirStringURL = "http://%safter_signin/"

	pageLimit    = 100
	debugOptions = true
)

var (
	// ValidScopes is a list of valid scopes your allowed
	ValidScopes = []string{
		scopeViewingActivityRead,
		scopeChannelCheckSubscription,
		scopeChannelCommercial,
		scopeChannelEditor,
		scopeChannelFeedEdit,
		scopeChannelFeedRead,
		scopeChannelRead,
		scopeChannelStream,
		scopeChannelSubscriptions,
		scopeChatLogin,
		scopeUserBlocksEdit,
		scopeUserBlocksRead,
		scopeUserFollowsEdit,
		scopeUserRead,
		scopeUserSubscriptions,
	}

	// DefaultStreamerScope - Good set of scopes for Streamer Login
	DefaultStreamerScope = []string{
		scopeViewingActivityRead,
		scopeChannelCheckSubscription,
		scopeChannelEditor,
		scopeChannelFeedEdit,
		scopeChannelFeedRead,
		scopeChannelRead,
		scopeChannelSubscriptions,
		scopeUserBlocksEdit,
		scopeUserBlocksRead,
		scopeUserFollowsEdit,
		scopeUserRead,
		scopeUserSubscriptions,
		scopeChatLogin,
	}

	// DefaultViewerScope - Good set of scopes for Viewer Login
	DefaultViewerScope = []string{
		scopeViewingActivityRead,
	}
)

// authInternalState - OAuth State token for security
type authInternalState string

// Client - Twitch OAuth Client
type Client struct {
	*tokenData

	httpClient WebClient
	domain     string
	servePath  string

	AdminUser    *User
	AdminAuth    *UserAuth
	AdminChannel chan int

	MyID     ID
	MyStream *StreamBody

	ViewCount []int

	Viewers       map[ID]*Viewer
	PendingLogins map[authInternalState]time.Time

	Chat    *Chat
	User    *UsersMethod
	Channel *ChannelsMethod
	Stream  *StreamsMethod
}

// CreateTwitchClient -
func CreateTwitchClient(servingFromDomain string, reqScopes []string) (*Client, error) {
	kb := Client{
		domain:       servingFromDomain,
		servePath:    servingFromDomain[strings.Index(servingFromDomain, "/"):],
		httpClient:   &http.Client{},
		AdminChannel: make(chan int, 3),

		Viewers:       make(map[ID]*Viewer),
		PendingLogins: make(map[authInternalState]time.Time),
	}

	kb.loadSecrets()

	kb.AdminAuth = &UserAuth{
		token:      nil,
		oauthState: authInternalState(GenerateRandomString(16)),
		Scopes:     make(map[string]bool),
	}
	kb.AdminAuth.updateScope(reqScopes)

	kb.User = &UsersMethod{client: &kb, au: kb.AdminAuth}
	kb.Channel = &ChannelsMethod{client: &kb, au: kb.AdminAuth}
	kb.Stream = &StreamsMethod{client: &kb, au: kb.AdminAuth}

	return &kb, nil
}

// GetAuth - Returns Auth Code not sure if this is okay but I need it for twitch interaction
func (ah *Client) GetAuth() string {
	b, c := ah.AdminAuth.GetAuth()
	if b {
		return c
	}
	return ""
}

// GetNick - Returns the name of the streamer account
func (ah *Client) GetNick() IrcNick {
	if ah.AdminAuth != nil && ah.AdminAuth.token != nil {
		return ah.AdminAuth.token.Username
	}

	return ""
}

func (ah *Client) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Get Relative Path
	relPath := req.URL.Path[strings.Index(req.URL.Path, ah.servePath)+len(ah.servePath):]

	if strings.HasPrefix(relPath, "after_signin") {
		ah.handlePublicOAuthResult(w, req)
		return
	}

	// User isn't Auth start login
	c, err := req.Cookie(UserAuthSessionCookieName)
	if err != nil {
		log.Printf("Session Error: %s - %s", err, req.URL)
		ah.handleOAuthStart(w, req)
		return
	}

	cList := strings.Split(c.Value, ":")
	tid := ID(cList[0])

	// Try Find User
	vwr := ah.GetViewer(tid)

	// User isn't Auth start login
	if vwr.Auth == nil || (vwr.Auth.checkCookie(c) == false) {
		log.Println("Cookie Failed", vwr.Auth, c)
		ah.handleOAuthStart(w, req)
		return
	}

	http.SetCookie(w, vwr.Auth.SessionCookie)
	fmt.Fprintf(w, "You are logged in %s", vwr.GetNick())
}

// Get will make Twitch API request with correct headers then attempt to decode JSON into jsonStruct
// If you call it with a nil user or user without a token it will do request without auth
func (ah *Client) Get(au *UserAuth, path string, jsonStruct interface{}) (string, error) {
	log.Printf("Twitch Get: %s", path)
	path = twitchBase + path

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("Client-ID", ah.ClientID)
	if au != nil && au.token != nil {
		req.Header.Add("Authorization", "OAuth "+au.AuthCode)
	}

	resp, err := ah.httpClient.Do(req)

	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified {
		return "", errors.New("api error, response code: " + strconv.Itoa(resp.StatusCode))
	}

	defer resp.Body.Close()

	if jsonStruct != nil {
		err = json.NewDecoder(resp.Body).Decode(jsonStruct)
		return "", err
	}

	if b, err := ioutil.ReadAll(resp.Body); err == nil {
		return string(b), nil
	}

	return "", err
}

func (ah *Client) adminHasAuthed() {
	ah.MyID = ah.AdminAuth.token.UserID

	if ah.AdminAuth.Scopes[scopeChatLogin] {
		go ah.startNewChat()
	}

	go ah.monitorHeartbeat()

	ah.AdminChannel <- 1
}

func (ah *Client) monitorHeartbeat() {
	ircRoomToJoin := IrcNick(*flagIrcChannel)
	if len(ircRoomToJoin) < 2 {
		ircRoomToJoin = ah.GetNick()
	}

	v, _ := ah.FindViewer(ircRoomToJoin)

	oldHosts := make(map[ID]bool)

	updateBeat := func(t time.Time) {
		// Stream Viewer Count
		sb, err := ah.Stream.GetStreamByUser(v.TwitchID)
		if err != nil || sb == nil {
			fmt.Printf("Heartbeart %s: NOT LIVE %s\n", t.Format(time.UnixDate), v.GetNick())
		} else {
			fmt.Printf("Heartbeat %s: %5d   %s\n", t.Format(time.UnixDate), sb.Viewers, sb.Game)
			ah.ViewCount = append(ah.ViewCount, sb.Viewers)
			ah.MyStream = sb
		}

		// List of Hosts
		hostList, err := ah.Stream.GetHostsByUser(v.TwitchID)
		if err != nil {
			fmt.Printf("Host Check failed: %s\n", err.Error())
		} else {
			for i := range oldHosts {
				oldHosts[i] = false
			}

			for _, h := range hostList {
				srcID := IDFromInt(h.HostID)

				_, ok := oldHosts[srcID]
				if !ok {

					hostStr, err := ah.Stream.GetStreamByUser(srcID)
					if err == nil {
						fmt.Printf("Hosting Started [%d]: %s %s\n", hostStr.Viewers, h.HostLogin, srcID)
					} else {
						fmt.Printf("Hosting Started: %s %s\n", h.HostLogin, srcID)
					}

				}
				oldHosts[srcID] = true
			}

			for i, isFresh := range oldHosts {
				if isFresh == false {
					fmt.Printf("Hosting Stopped: %s\n", i)
					delete(oldHosts, i)
				}
			}

			hlNum := len(hostList)
			fmt.Printf("Hosts: %d\n", hlNum)
		}

	}

	updateBeat(time.Now())

	timeSinceDump := time.Duration(0)

	tBeat := time.NewTicker(time.Minute)
	for ts := range tBeat.C {
		updateBeat(ts)

		// Dumping to File
		timeSinceDump += time.Minute * 1
		if timeSinceDump > time.Hour {
			timeSinceDump = 0
			f, err := os.Create(fmt.Sprintf("dump_%s_%d.bin", ah.Chat.Room, time.Now().Unix()))
			if err != nil {
			}
			enc := gob.NewEncoder(f)
			for _, v := range ah.Viewers {
				err = enc.Encode(v)
				if err != nil {
					f.Close()
					continue
				}
			}

			fmt.Printf("Dumped data to file: %s", f.Name())
			f.Close()
		}
	}
}

func (ah *Client) startNewChat() {
	ircRoomToJoin := *flagIrcChannel
	if len(ircRoomToJoin) < 2 {
		ircRoomToJoin = string(ah.AdminAuth.token.Username)
	}

	logFile, err := os.OpenFile(
		fmt.Sprintf("%s_chat.log", ircRoomToJoin),
		os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatal("Shouldn't fail to create chat log")
	}
	defer logFile.Close()

	c, err := createIrcClient(ah.AdminAuth, ah, ah.IrcServerAddr)
	if err != nil {
		log.Printf("Failed to Start New Chat %s", err.Error())
		return
	}
	ah.Chat = c
	ah.Chat.SetupLogWriter(logFile)

	err = ah.Chat.StartRunLoop()
	if err != nil {
		log.Printf("Chat Shutdown %s", err.Error())
		return
	}
}
