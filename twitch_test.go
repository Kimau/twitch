package twitch

import "fmt"

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

func (dvp *DummyViewProvider) GetRoomID() ID        { return ID(0) }
func (dvp *DummyViewProvider) GetRoomName() IrcNick { return "kimau" }
func (dvp *DummyViewProvider) GetNick() IrcNick     { return "kimbot" }

func (dvp *DummyViewProvider) GetCopy(twitchID ID) (Viewer, error) {
	var v Viewer
	src := dvp.GetPtr(twitchID)
	if src != nil {
		src.CopyTo(&v)
		return v, nil
	}

	err := fmt.Errorf("Unable to Find Viewer")
	return v, err
}

func (dvp *DummyViewProvider) GetPtr(id ID) *Viewer {
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
	v := dvp.GetPtr(u.ID)
	v.SetUser(u)
	return v
}

///////////////////////////////////////////////////////////////////////////////
