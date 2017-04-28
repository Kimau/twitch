package twitch

type viewerProvider interface {
	GetRoomID() ID
	GetRoomName() IrcNick
	AllKeys() []ID

	GetPtr(ID) *Viewer
	GetCopy(ID) (Viewer, error)
	GetFromUser(User) *Viewer

	Find(IrcNick) (*Viewer, error)
	UpdateViewers([]IrcNick) []*Viewer
}
