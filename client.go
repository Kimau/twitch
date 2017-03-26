package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

	chatWriters []ChatLogger

	AdminID      ID
	AdminAuth    *UserAuth
	AdminChannel chan int

	RoomName   IrcNick
	RoomID     ID
	RoomStream *StreamBody

	Viewers       map[ID]*Viewer
	PendingLogins map[authInternalState]time.Time

	Chat    *Chat
	User    *UsersMethod
	Channel *ChannelsMethod
	Stream  *StreamsMethod
	Heart   *Heartbeat
}

// CreateTwitchClient -
func CreateTwitchClient(servingFromDomain string, reqScopes []string, roomToJoin string, chatWriterList []ChatLogger) (*Client, error) {
	kb := Client{
		domain:    servingFromDomain,
		servePath: servingFromDomain[strings.Index(servingFromDomain, "/"):],

		RoomName:    IrcNick(roomToJoin),
		chatWriters: chatWriterList,

		httpClient:   &http.Client{},
		AdminChannel: make(chan int, 3),

		Viewers:       make(map[ID]*Viewer),
		PendingLogins: make(map[authInternalState]time.Time),
	}

	kb.loadSecrets()

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

	kb.loadToken()

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
	ah.GetViewer(ah.AdminID)

	// Get Room we are Watching
	roomViewer, err := ah.FindViewer(ah.RoomName)
	if err != nil {
		log.Fatalf("Unable to find room [%s]\n%s", ah.RoomName, err)
	}
	ah.RoomID = roomViewer.TwitchID

	// Start up IRC Chat
	if ah.AdminAuth.Scopes[scopeChatLogin] {
		go ah.startNewChat()
	}

	go ah.Heart.StartBeat()

	ah.AdminChannel <- 1
}

func (ah *Client) startNewChat() {

	c, err := createIrcClient(ah.AdminAuth, ah, ah.IrcServerAddr, ah.chatWriters)
	if err != nil {
		log.Printf("Failed to Start New Chat %s", err.Error())
		return
	}
	ah.Chat = c

	err = ah.Chat.StartRunLoop()
	if err != nil {
		log.Printf("Chat Shutdown %s", err.Error())
		return
	}
}
