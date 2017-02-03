package twitch

import (
	"errors"
	"html/template"
	"regexp"

	"strings"

	"github.com/kimau/kbot/web"
)

var (
	authTemp        *template.Template
	authSignRegEx   = regexp.MustCompile("/twitch/signin/([\\w]+)/*")
	authRegEx       = regexp.MustCompile("/twitch/after_signin/\\?code=([\\w-]+)&scope=([\\w-+]+)&state=([\\w-]+)")
	authCancelRegEx = regexp.MustCompile("/twitch/after_signin/*")

	// ValidScopes is a list of valid scopes your allowed
	// "channel_check_subscription" - Read whether a user is subscribed to your channel.
	// "channel_commercial"         - Trigger commercials on channel.
	// "channel_editor"             - Write channel metadata (game, status, etc).
	// "channel_feed_edit"          - Add posts and reactions to a channel feed.
	// "channel_feed_read"          - View a channel feed.
	// "channel_read"               - Read nonpublic channel information, including email address and stream key.
	// "channel_stream"             - Reset a channel’s stream key.
	// "channel_subscriptions"      - Read all subscribers to your channel.
	// "chat_login"                 - Log into chat and send messages.
	// "user_blocks_edit"           - Turn on/off ignoring a user. Ignoring a user means you cannot see them type, receive messages from them, etc.
	// "user_blocks_read"           - Read a user’s list of ignored users.
	// "user_follows_edit"          - Manage a user’s followed channels.
	// "user_read"                  - Read nonpublic user information, like email address.
	// "user_subscriptions"         - Read a user’s subscriptions.
	ValidScopes = []string{
		"channel_check_subscription",
		"channel_commercial",
		"channel_editor",
		"channel_feed_edit",
		"channel_feed_read",
		"channel_read",
		"channel_stream",
		"channel_subscriptions",
		"chat_login",
		"user_blocks_edit",
		"user_blocks_read",
		"user_follows_edit",
		"user_read",
		"user_subscriptions",
	}

	// DefaultStreamerScope - Good set of scopes for Streamer Login
	DefaultStreamerScope = []string{
		"channel_check_subscription",
		"channel_editor",
		"channel_feed_edit",
		"channel_feed_read",
		"channel_read",
		"channel_subscriptions",
		"user_blocks_edit",
		"user_blocks_read",
		"user_follows_edit",
		"user_read",
		"user_subscriptions",
	}
)

const (
	baseURL  = "https://api.twitch.tv/kraken/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s"
	clientID = "qhaf2djfhvkohczx08oyqra51hjasn"
	redirURL = "http://localhost:30006/twitch/after_signin/" //"https://twitch.otg-gt.xyz/twitch/after_signin/"
)

// Client - Twitch OAuth Client
type Client struct {
	WebFace *web.WebFace

	scopes map[string]bool
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

	kb := Client{WebFace: wf, scopes: newScope}
	wf.Router.Handle("/twitch/signin/", &kb)
	wf.Router.Handle("/twitch/after_signin/", &kb)

	return &kb, nil
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
