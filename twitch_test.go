package twitch

import (
	"fmt"
	"strings"
)

type DummyViewProvider struct {
	Viewers map[ID]*Viewer
}

func (dvp *DummyViewProvider) GetAuthViewer() *Viewer {
	v := dvp.GetViewer(ID(0))
	v.User.Name = "kimau"
	return v
}

func (dvp *DummyViewProvider) GetRoom() *Viewer {
	v := dvp.GetViewer(ID(0))
	v.User.Name = "kimau"
	return v
}

func (dvp *DummyViewProvider) GetNick() IrcNick { return "kimau" }
func (dvp *DummyViewProvider) GetViewer(id ID) *Viewer {
	v, ok := dvp.Viewers[id]
	if !ok {
		v = &Viewer{
			TwitchID: id,
			User: &User{
				ID:          id,
				Name:        IrcNick("DummyName" + GenerateRandomString(4)),
				DisplayName: "Name" + GenerateRandomString(6),
			},
		}
	}

	return v
}
func (dvp *DummyViewProvider) FindViewer(nick IrcNick) (*Viewer, error) {
	for _, v := range dvp.Viewers {
		if v.User.Name == nick {
			return v, nil
		}
	}

	id := generateDummyID()
	v := &Viewer{
		TwitchID: id,
		User: &User{
			ID:          id,
			Name:        nick,
			DisplayName: string(nick),
		},
	}
	return v, nil
}
func (dvp *DummyViewProvider) UpdateViewers(nickList []IrcNick) []*Viewer {
	vList := []*Viewer{}
	for _, name := range nickList {
		v, _ := dvp.FindViewer(name)
		vList = append(vList, v)
	}

	return vList
}

func (dvp *DummyViewProvider) GetViewerFromUser(u User) *Viewer {
	v := dvp.GetViewer(u.ID)
	v.User = &u
	return v
}

// GetViewerFromChatter - Get Viewer from Chatter
func (dvp *DummyViewProvider) GetViewerFromChatter(cu Chatter) *Viewer {
	if cu.id != "" {
		v := dvp.GetViewer(cu.id)
		v.Chatter = &cu
		return v
	} else if cu.Nick != "" {
		v, _ := dvp.FindViewer(cu.Nick)
		v.Chatter = &cu
		return v
	} else if cu.DisplayName != "" {
		v, _ := dvp.FindViewer(IrcNick(cu.DisplayName))
		v.Chatter = &cu
		return v
	}

	fmt.Printf("GetViewerFromChatter ERROR \n %s",
		strings.Replace(fmt.Sprintf("%#v", cu), ",", ",\n", -1))
	return nil
}

///////////////////////////////////////////////////////////////////////////////
