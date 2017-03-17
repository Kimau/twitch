package twitch

import "net/http"
import "fmt"

// ID - Numberic Identifier of Twitch Identity
type ID string

// IrcNick - Irc Nick all lowercase identifier
type IrcNick string

// Currency use to track viewer Value
type Currency int

// WebClient - Provides basic Request Poster
type WebClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ircAuthProvider - Provides Auth normally expects UserAuth
type ircAuthProvider interface {
	GetIrcAuth() (hasauth bool, name string, pass string)
}

type viewerProvider interface {
	GetNick() IrcNick
	GetViewer(ID) *Viewer
	FindViewer(IrcNick) (*Viewer, error)
	UpdateViewers([]IrcNick) []*Viewer
	GetViewerFromChatter(*Chatter) *Viewer
}

// Twitch Scopes
const (
	scopeViewingActivityRead      = "viewing_activity_read"      // "viewing_activity_read"      - Turn on Viewer Heartbeat Service ability to record user data.
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

// IDFromInt - Convert ID from int to string ID
// Some older API return a number
func IDFromInt(id int) ID {
	return ID(fmt.Sprintf("%d", id))
}
