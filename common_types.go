package twitch

import "net/http"

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
