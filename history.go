package twitch

import "time"
import "fmt"

// HistoricViewerData - Is a Snapshot of Viewer data at a given point in time
type HistoricViewerData struct {
	Name       IrcNick
	Timestamp  time.Time
	RoomID     ID
	ViewerData map[ID]Viewer
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

// GetPtr - Get Viewer by ID
func (hvd *HistoricViewerData) GetPtr(id ID) *Viewer {
	v, ok := hvd.ViewerData[id]
	if ok {
		return &v
	}
	return nil
}

// GetCopy - Get Copy of Viewer
func (hvd *HistoricViewerData) GetCopy(twitchID ID) (Viewer, error) {
	var v Viewer
	src := hvd.GetPtr(twitchID)
	if src != nil {
		src.CopyTo(&v)
		return v, nil
	}

	err := fmt.Errorf("Unable to Find Viewer")
	return v, err
}

// GetFromUser - Get Viewer from User ID (no update)
func (hvd *HistoricViewerData) GetFromUser(u User) *Viewer {
	return hvd.GetPtr(u.ID)
}

// Find - Find viewer by Nick
func (hvd *HistoricViewerData) Find(nick IrcNick) (*Viewer, error) {
	for _, v := range hvd.ViewerData {
		if v.GetNick() == nick {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("Not captured: " + string(nick))
}

// UpdateViewers - Get viewers from nicks no updating because historic
func (hvd *HistoricViewerData) UpdateViewers(nList []IrcNick) []*Viewer {
	vList := []*Viewer{}

Outer:
	for _, v := range hvd.ViewerData {
		for i, nick := range nList {
			if v.GetNick() == nick {
				vSave := v
				vList = append(vList, &vSave)
				if len(nList) > (i + 1) {
					nList = append(nList[:i], nList[i+1:]...)
				} else {
					nList = nList[:i]
				}
				continue Outer
			}
		}
	}

	return vList
}
