package twitch

import (
	"fmt"
	"strings"
)

type DummyViewProvider struct {
	Viewers map[ID]*Viewer
}

func (dvp *DummyViewProvider) AllKeys() []ID {
	myKeys := make([]ID, len(dvp.Viewers))
	i := 0
	for k := range dvp.Viewers {
		myKeys[i] = k
		i++
	}
	return myKeys
}

func (dvp *DummyViewProvider) GetRoomID() ID {
	return ID(0)
}

func (dvp *DummyViewProvider) GetNick() IrcNick { return "kimau" }
func (dvp *DummyViewProvider) Get(id ID) *Viewer {
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
func (dvp *DummyViewProvider) Find(nick IrcNick) (*Viewer, error) {
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
		v, _ := dvp.Find(name)
		vList = append(vList, v)
	}

	return vList
}

func (dvp *DummyViewProvider) GetFromUser(u User) *Viewer {
	v := dvp.Get(u.ID)
	v.SetUser(u)
	return v
}

// GetFromChatter - Get Viewer from Chatter
func (dvp *DummyViewProvider) GetFromChatter(cu Chatter) *Viewer {
	if cu.id != "" {
		v := dvp.Get(cu.id)
		v.SetChatter(cu)
		return v
	} else if cu.Nick != "" {
		v, _ := dvp.Find(cu.Nick)
		v.SetChatter(cu)
		return v
	} else if cu.DisplayName != "" {
		v, _ := dvp.Find(IrcNick(cu.DisplayName))
		v.SetChatter(cu)
		return v
	}

	fmt.Printf("GetFromChatter ERROR \n %s",
		strings.Replace(fmt.Sprintf("%#v", cu), ",", ",\n", -1))
	return nil
}

///////////////////////////////////////////////////////////////////////////////
