package twitch

type viewerProvider interface {
	GetRoomID() ID
	AllKeys() []ID

	Get(ID) *Viewer
	GetFromChatter(Chatter) *Viewer
	GetFromUser(User) *Viewer

	Find(IrcNick) (*Viewer, error)
	UpdateViewers([]IrcNick) []*Viewer
}
