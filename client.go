package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	rootURL       = "https://api.twitch.tv/kraken/"
	ircServerAddr = "irc.chat.twitch.tv:6667"
	ircRoomToJoin = "elvenaimee"
	clientID      = "qhaf2djfhvkohczx08oyqra51hjasn"
	clientSecret  = "u5jj3g6qtcj8fut5yx2sj50u525i3a"

	redirStringURL = "http://%safter_signin/"
	baseURL        = "%soauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s"

	pageLimit    = 100
	debugOptions = true

	scopeChannelCheckSubscription = "channel_check_subscription" // "channel_check_subscription" - Read whether a user is subscribed to your channel.
	scopeChannelCommercial        = "channel_commercial"         // "channel_commercial"         - Trigger commercials on channel.
	scopeChannelEditor            = "channel_editor"             // "channel_editor"             - Write channel metadata (game, status, etc).
	scopeChannelFeedEdit          = "channel_feed_edit"          // "channel_feed_edit"          - Add posts and reactions to a channel feed.
	scopeChannelFeedRead          = "channel_feed_read"          // "channel_feed_read"          - View a channel feed.
	scopeChannelRead              = "channel_read"               // "channel_read"               - Read nonpublic channel information, including email address and stream key.
	scopeChannelStream            = "channel_stream"             // "channel_stream"             - Reset a channel’s stream key.
	scopeChannelSubscriptions     = "channel_subscriptions"      // "channel_subscriptions"      - Read all subscribers to your channel.
	scopeChatLogin                = "chat_login"                 // "chat_login"                 - Log into chat and send messages.
	scopeUserBlocksEdit           = "user_blocks_edit"           // "user_blocks_edit"           - Turn on/off ignoring a user. Ignoring a user means you cannot see them type, receive messages from them, etc.
	scopeUserBlocksRead           = "user_blocks_read"           // "user_blocks_read"           - Read a user’s list of ignored users.
	scopeUserFollowsEdit          = "user_follows_edit"          // "user_follows_edit"          - Manage a user’s followed channels.
	scopeUserRead                 = "user_read"                  // "user_read"                  - Read nonpublic user information, like email address.
	scopeUserSubscriptions        = "user_subscriptions"         // "user_subscriptions"         - Read a user’s subscriptions.
)

var (
	// ValidScopes is a list of valid scopes your allowed
	ValidScopes = []string{
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
	DefaultViewerScope = []string{}
)

// authInternalState - OAuth State token for security
type authInternalState string

// Client - Twitch OAuth Client
type Client struct {
	httpClient WebClient
	url        *url.URL
	domain     string
	servePath  string

	AdminUser    *User
	AdminAuth    *UserAuth
	AdminChannel chan int

	Chat *Chat

	Viewers       map[ID]*Viewer
	PendingLogins map[authInternalState]time.Time

	User    *UsersMethod
	Channel *ChannelsMethod
}

// CreateTwitchClient -
func CreateTwitchClient(servingFromDomain string, reqScopes []string) (*Client, error) {
	urlParsed, _ := url.Parse(rootURL)

	kb := Client{
		url:          urlParsed,
		domain:       servingFromDomain,
		servePath:    servingFromDomain[strings.Index(servingFromDomain, "/"):],
		httpClient:   &http.Client{},
		AdminChannel: make(chan int, 3),

		Viewers:       make(map[ID]*Viewer),
		PendingLogins: make(map[authInternalState]time.Time),
	}

	kb.AdminAuth = &UserAuth{
		token:      nil,
		oauthState: authInternalState(GenerateRandomString(16)),
		scopes:     make(map[string]bool),
	}
	kb.AdminAuth.updateScope(reqScopes)

	kb.User = &UsersMethod{client: &kb, au: kb.AdminAuth}
	kb.Channel = &ChannelsMethod{client: &kb, au: kb.AdminAuth}

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

	http.SetCookie(w, vwr.Auth.sessionCookie)
	fmt.Fprintf(w, "You are logged in %s", vwr.GetNick())
}

// Get will make Twitch API request with correct headers then attempt to decode JSON into jsonStruct
// If you call it with a nil user or user without a token it will do request without auth
func (ah *Client) Get(au *UserAuth, path string, jsonStruct interface{}) (string, error) {

	urlString := "https://api.twitch.tv/kraken"
	if path != "" {

		rel, err := url.Parse(path)

		if err != nil {
			return "", err
		}

		subURL := ah.url.ResolveReference(rel)
		urlString = subURL.String()
	}

	log.Printf("Twitch Get: %s", urlString)
	//log.Printf("Twitch Get: %s \n---\n%s\n----\n", urlString, debug.Stack())

	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("Client-ID", clientID)
	if au != nil && au.token != nil {
		req.Header.Add("Authorization", "OAuth "+au.authcode)
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

func (ah *Client) startNewChat() {
	logFile, err := os.OpenFile("chat.log", os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatal("Shouldn't fail to create chat log")
	}
	defer logFile.Close()

	c, err := createIrcClient(ah.AdminAuth, ah)
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
