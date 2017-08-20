package twitch

import "time"

// HistoricViewerData - Is a Snapshot of Viewer data at a given point in time
type HistoricViewerData struct {
	Name       IrcNick
	Timestamp  time.Time
	RoomID     ID
	ViewerData map[ID]ViewerData
}

// HistoricChatLog - Old Chat Logs used for Analysis
type HistoricChatLog struct {
	Name          IrcNick
	LogLinesByDay map[time.Time][]LogLineParsed
}

// GetRoomID - Room Twitch ID
func (hvd *HistoricViewerData) GetRoomID() ID { return hvd.RoomID }

// GetRoomName - Room Twitch Nick
func (hvd *HistoricViewerData) GetRoomName() IrcNick { return hvd.Name }

// GetNick - Get Nick of Auth User
func (hvd *HistoricViewerData) GetNick() IrcNick { return hvd.Name }

// AllKeys - Get All Viewer IDs slower than a direct range over
func (hvd *HistoricViewerData) AllKeys() []ID {
	myKeys := make([]ID, len(hvd.ViewerData))
	i := 0
	for k := range hvd.ViewerData {
		myKeys[i] = k
		i++
	}
	return myKeys
}
