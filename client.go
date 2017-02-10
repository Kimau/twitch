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

	"github.com/kimau/kbot/web"
)

const (
	rootURL      = "https://api.twitch.tv/kraken/"
	baseURL      = "%soauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s"
	clientID     = "qhaf2djfhvkohczx08oyqra51hjasn"
	clientSecret = "u5jj3g6qtcj8fut5yx2sj50u525i3a"
	redirURL     = "http://localhost:30006/twitch/after_signin/" //"https://twitch.otg-gt.xyz/twitch/after_signin/"

	listenRoot = "/twitch/"

	pageLimit = 100

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
	}
)

// Client - Twitch OAuth Client
type Client struct {
	WebFace *web.WebFace

	httpClient *http.Client
	url        *url.URL

	hasAuth    bool
	oauthState string
	authcode   string
	scopes     map[string]bool

	User *UsersMethod
}

// CreateTwitchClient -
func CreateTwitchClient(wf *web.WebFace, reqScopes []string) (*Client, error) {

	if wf == nil {
		return nil, errors.New("WebFace must be valid")
	}

	newScope := make(map[string]bool)
	for _, ov := range ValidScopes {
		newScope[ov] = false
		for _, v := range reqScopes {
			if v == ov {
				newScope[ov] = true
			}
		}
	}

	urlParsed, _ := url.Parse(rootURL)

	kb := Client{
		WebFace:    wf,
		scopes:     newScope,
		hasAuth:    false,
		url:        urlParsed,
		httpClient: &http.Client{},
	}

	kb.User = &UsersMethod{client: &kb}
	kb.oauthState = GenerateRandomString(32)
	wf.Router.Handle(listenRoot, &kb)

	return &kb, nil
}

func (ah *Client) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	// Get Relative Path
	relPath := req.URL.Path[strings.Index(req.URL.Path, listenRoot)+len(listenRoot):]
	log.Println("Twitch: ", relPath)

	// Force Auth
	if ah.hasAuth == false {
		if strings.HasPrefix(relPath, "after_signin") {
			ah.handleOAuthResult(w, req)
		} else {
			ah.handleOAuthStart(w, req)
		}

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
		uf, err := ah.User.GetByName(r[1])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%#v", uf)
	default:
		http.Error(w, fmt.Sprintf("Invalid Endpoint: %s", req.URL.Path), 404)
	}
}

func (ah *Client) getScopeString() string {
	s := ""
	for k, v := range ah.scopes {
		if v {
			s += k + "+"
		}
	}

	s = strings.TrimRight(s, "+")

	return s
}

// Get will make Twitch API request with correct headers then attempt to decode JSON into jsonStruct
func (ah *Client) Get(path string, jsonStruct interface{}) (string, error) {
	if !ah.hasAuth {
		return "", fmt.Errorf("Client doesn't have auth. Cannot perform [%s]", path)
	}

	rel, err := url.Parse(path)

	if err != nil {
		return "", err
	}

	subURL := ah.url.ResolveReference(rel)

	req, err := http.NewRequest("GET", subURL.String(), nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("Client-ID", clientID)
	req.Header.Add("Authorization", "OAuth "+ah.authcode)

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
