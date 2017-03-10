package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	rootURL       = "https://api.twitch.tv/kraken/"
	ircServerAddr = "irc.chat.twitch.tv:6667"
	baseURL       = "%soauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s"
	clientID      = "qhaf2djfhvkohczx08oyqra51hjasn"
	clientSecret  = "u5jj3g6qtcj8fut5yx2sj50u525i3a"
	cookieDoman   = "localhost:30006"
	redirURL      = "http://localhost:30006/twitch/after_signin/" //"https://twitch.otg-gt.xyz/twitch/after_signin/"

	// ListenRoot - Http Listen Root
	ListenRoot = "/twitch/"

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
func CreateTwitchClient(reqScopes []string) (*Client, error) {
	urlParsed, _ := url.Parse(rootURL)

	kb := Client{
		url:          urlParsed,
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

// AdminHTTP for backoffice requests
func (ah *Client) AdminHTTP(w http.ResponseWriter, req *http.Request) {
	// Get Relative Path
	relPath := req.URL.Path[strings.Index(req.URL.Path, ListenRoot)+len(ListenRoot):]
	log.Println("Twitch ADMIN: ", relPath)

	// Force Auth
	if ah.AdminAuth.token == nil {
		ah.handleOAuthAdminStart(w, req)
		return
	}

	switch {
	case strings.HasPrefix(relPath, "me"):
		uf, err := ah.User.GetMe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%#v", uf)

	case strings.HasPrefix(relPath, "user"):
		userName := regexp.MustCompile("username/([\\w]+)/*")
		r := userName.FindStringSubmatch(relPath)
		nameList := []IrcNick{IrcNick(r[1])}
		uf, err := ah.User.GetByName(nameList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%#v", uf)

	case debugOptions && strings.HasPrefix(relPath, "debug/"):
		splitD := strings.Split(req.RequestURI, "debug/")
		log.Println("Debug: " + splitD[1])
		body, err := ah.Get(ah.AdminAuth, splitD[1], nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, body)

	default:
		http.Error(w, fmt.Sprintf("Invalid Endpoint: %s", req.URL.Path), 404)
	}
}

func (ah *Client) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Get Relative Path
	relPath := req.URL.Path[strings.Index(req.URL.Path, ListenRoot)+len(ListenRoot):]

	if strings.HasPrefix(relPath, "after_signin") {
		ah.handlePublicOAuthResult(w, req)
		return
	}

	// User isn't Auth start login
	c, err := req.Cookie(UserAuthSessionCookieName)
	if err != nil {
		log.Println("Session Error", err)
		ah.handleOAuthStart(w, req)
		return
	}

	cList := strings.Split(c.Value, ":")
	tid := ID(cList[0])

	// Try Find User
	u, ok := ah.Viewers[tid]
	if !ok {
		http.Error(w, fmt.Sprintf("Couldn't find user %s", tid), http.StatusUnauthorized)
		return
	}

	// User isn't Auth start login
	if u.Auth == nil || (u.Auth.checkCookie(c) == false) {
		log.Println("Cookie Failed", u.Auth, c)
		ah.handleOAuthStart(w, req)
		return
	}

	http.SetCookie(w, u.Auth.sessionCookie)
	fmt.Fprintf(w, "You are logged in %s", u.GetNick())
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
