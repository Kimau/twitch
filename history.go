package twitch

import "time"
import "fmt"

const ()

var ()

// HistoricViewerData - Is a Snapshot of Viewer data at a given point in time
type HistoricViewerData struct {
	Name       IrcNick
	Timestamp  time.Time
	ViewerData map[ID]Viewer
}

// HistoricChatLog - Old Chat Logs used for Analysis
type HistoricChatLog struct {
	Name          IrcNick
	LogLinesByDay map[time.Time][]LogLineParsed
}

// GetAuthViewer - Returns Nothing because Historic Data
func (hvd *HistoricViewerData) GetAuthViewer() *Viewer {
	return nil
}

// GetRoom - Returns nothing because Historic
func (hvd *HistoricViewerData) GetRoom() *Viewer {
	return nil
}

// GetAllViewerIDs - Get All Viewer IDs slower than a direct range over
func (hvd *HistoricViewerData) GetAllViewerIDs() []ID {
	myKeys := make([]ID, len(hvd.ViewerData))
	i := 0
	for k := range hvd.ViewerData {
		myKeys[i] = k
		i++
	}
	return myKeys
}

// GetViewer - Get Viewer by ID
func (hvd *HistoricViewerData) GetViewer(id ID) *Viewer {
	v, ok := hvd.ViewerData[id]
	if ok {
		return &v
	}
	return nil
}

// GetViewerFromChatter  - Get Viewer from Chatter ID
func (hvd *HistoricViewerData) GetViewerFromChatter(cu Chatter) *Viewer {
	return hvd.GetViewer(cu.id)
}

// GetViewerFromUser - Get Viewer from User ID (no update)
func (hvd *HistoricViewerData) GetViewerFromUser(u User) *Viewer {
	return hvd.GetViewer(u.ID)
}

// FindViewer - Find viewer by Nick
func (hvd *HistoricViewerData) FindViewer(nick IrcNick) (*Viewer, error) {
	for _, v := range hvd.ViewerData {
		if v.GetNick() == nick {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("Not captured")
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
