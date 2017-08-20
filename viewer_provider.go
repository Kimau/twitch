package twitch

type viewerProvider interface {
	GetRoomID() ID
	GetRoomName() IrcNick
	AllKeys() []ID

	Set(ViewerData)
	GetPtr(ID) *Viewer
	GetData(ID) (ViewerData, error)
	GetFromUser(User) *Viewer

	Find(IrcNick) (*Viewer, error)
	UpdateViewers([]IrcNick) []*Viewer

	Client() *Client
}
