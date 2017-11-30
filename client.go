package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	twitchBase     = "https://api.twitch.tv/kraken/"
	twitchAuthURL  = "https://api.twitch.tv/kraken/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s"
	redirStringURL = "https://%safter_signin/"

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

	AdminID      ID
	AdminAuth    *UserAuth
	AdminChannel chan int

	RoomName   IrcNick
	RoomID     ID
	RoomStream *StreamBody

	PendingLogins map[authInternalState]time.Time

	Alerts  *AlertPump
	Badges  *BadgeMethod
	Channel *ChannelsMethod
	Chat    *Chat
	Heart   *Heartbeat
	PubSub  *PubSubConn
	Stream  *StreamsMethod
	User    *UsersMethod
	Viewers *ViewerMethod
}

// CreateTwitchClient -
func CreateTwitchClient(servingFromDomain string, reqScopes []string, roomToJoin string, forceAuth bool) (*Client, error) {
	kb := Client{
		domain:    servingFromDomain,
		servePath: servingFromDomain[strings.Index(servingFromDomain, "/"):],

		RoomName: IrcNick(roomToJoin),

		httpClient:   &http.Client{},
		AdminChannel: make(chan int, 3),

		PendingLogins: make(map[authInternalState]time.Time),
	}

	kb.loadSecrets()

	kb.Viewers = CreateViewerMethod(&kb)
	hvd, err := LoadMostRecentViewerDump(kb.RoomName)
	if err == nil {
		if hvd != nil {
			for k := range hvd.ViewerData {
				kb.Viewers.Set(hvd.ViewerData[k])
			}
		}

		err = kb.Viewers.SanityScan()
		if err != nil {
			for k, v := range kb.Viewers.viewers {
				b, _ := json.Marshal(v)
				fmt.Println(k, string(b))
			}

			panic(err)
		}
		fmt.Println("===== LOADED VIEWERS FROM FILES ====")

	} else {
		log.Printf("Unable to load old user data: %s", err)
	}

	// Creat Admin Auth Temp
	kb.AdminAuth = &UserAuth{
		Token:        nil,
		InteralState: authInternalState(GenerateRandomString(16)),
		Scopes:       make(map[string]bool),
	}
	kb.AdminAuth.updateScope(reqScopes)

	kb.User = &UsersMethod{client: &kb, au: kb.AdminAuth}
	kb.Channel = &ChannelsMethod{client: &kb, au: kb.AdminAuth}
	kb.Stream = &StreamsMethod{client: &kb, au: kb.AdminAuth}
	kb.Heart = &Heartbeat{client: &kb}
	kb.Alerts = StartAlertPump(&kb)

	if err != nil {
		panic(err)
	}

	if !forceAuth {
		kb.loadToken()
	}

	return &kb, nil
}

// GetAuth - Returns Auth Code not sure if this is okay but I need it for twitch interaction
func (ah *Client) GetAuth() string {
	if ah.AdminAuth == nil {
		return ""
	}

	b, c := ah.AdminAuth.GetAuth()
	if b {
		return c
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
	vwr, err := ah.Viewers.GetData(tid)
	if err != nil {
		log.Println("Invalid User: ", tid, err)
		return
	}

	// User isn't Auth start login
	if vwr.Auth.checkCookie(c) == false {
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
	if au != nil && au.Token != nil {
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
	ah.AdminID = ah.AdminAuth.Token.UserID
	ah.Viewers.GetPtr(ah.AdminID) // Load up in Background

	// Get Room we are Watching
	roomViewer, err := ah.Viewers.Find(ah.RoomName)
	if err != nil {
		panic(fmt.Sprintf("Unable to find room [%s]\n%s", ah.RoomName, err))
	}
	ah.RoomID = roomViewer.GetData().TwitchID

	// Get Badges
	ah.Badges = CreateBadgeMethod(ah)

	// Get All Followers slowly
	// for a big channel with a million follows this will take 3 hours
	go func() {
		followChan := ah.Channel.GetAllFollowersSlow(ah.RoomID, time.Second, true)
		for fList, ok := <-followChan; ok; fList, ok = <-followChan {
			ah.Viewers.UpdateFollowers(fList)
		}
	}()

	// Start up IRC Chat
	if ah.AdminAuth.Scopes[scopeChatLogin] {
		go ah.startNewChat()
	}

	go ah.Heart.StartBeat()

	// PubSub
	ah.PubSub, err = CreatePubSub(ah, PubSubTopicList{
		//{Subject: psChanBits, Target: ah.RoomID},
		//{Subject: psChanSubs, Target: ah.RoomID},
		//{Subject: psVideoPlayback, Target: ah.RoomID},
		//{Subject: psChatModActions, Target: ah.RoomID},
		{Subject: psUserWhispers, Target: ah.AdminID},
	})

	go ah.PubSub.runningLoop()

	// HACK :: Filthy Hack
	// Allow a brief startup gap for responses ect...
	time.Sleep(time.Second * 1)

	ah.AdminChannel <- 1
}

func (ah *Client) startNewChat() {

	c, err := createIrcClient(ah.AdminAuth, ah.Viewers, ah.IrcServerAddr)
	if err != nil {
		log.Printf("Failed to Start New Chat %s", err.Error())
		return
	}
	ah.Chat = c
	ah.Chat.weakClientRef = ah

	// Make Connection
	conn, err := net.Dial("tcp", c.Server)
	if err != nil {
		log.Printf("Chat Shutdown %s", err.Error())
		return
	}

	err = ah.Chat.StartRunLoop(conn)
	if err != nil {
		log.Printf("Chat Shutdown %s", err.Error())
		return
	}
}

// SayMsg - Say IRC Message
func (ah *Client) SayMsg(line string) {
	ah.Chat.WriteSayMsg(line)
}
