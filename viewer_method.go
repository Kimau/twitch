package twitch

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ViewerMethod - Contains all the user functions
type ViewerMethod struct {
	client *Client

	mapLock       sync.Mutex
	viewers       map[ID]Viewer
	followerCache map[ID]time.Time
}

// CreateViewerMethod - Create VM with client
func CreateViewerMethod(c *Client) *ViewerMethod {
	return &ViewerMethod{
		client: c,

		viewers:       make(map[ID]Viewer),
		followerCache: make(map[ID]time.Time),
	}
}

func (vm *ViewerMethod) lockmap() {
	vm.mapLock.Lock()
}

func (vm *ViewerMethod) unlockmap() {
	vm.mapLock.Unlock()
}

// GetRoomID - Get ID of Room
func (vm *ViewerMethod) GetRoomID() ID {
	return vm.client.RoomID
}

// Set - Set New Viewer Value
func (vm *ViewerMethod) Set(v Viewer) {
	newV := Viewer{
		TwitchID: v.TwitchID,
		client:   vm.client,
	}
	newV = v

	vm.lockmap()
	defer vm.unlockmap()

	vm.viewers[v.TwitchID] = newV
	if newV.Follower == nil {
		delete(vm.followerCache, newV.TwitchID)
	} else {
		vm.followerCache[newV.TwitchID] = ChannelRelationship(*newV.Follower).CreatedAt()
	}
}

//SetFollower - Sets the new value in with lock
func (vm *ViewerMethod) SetFollower(newVal ChannelFollow) {
	v := vm.GetFromUser(*newVal.User)
	v.lockme()
	v.Follower = &newVal
	v.unlockme()

	vm.lockmap()
	vm.followerCache[newVal.User.ID] = ChannelRelationship(newVal).CreatedAt()
	vm.unlockmap()
}

//ClearFollower - Sets the new value in with lock
func (vm *ViewerMethod) ClearFollower(tid ID) {
	vm.lockmap()
	defer vm.unlockmap()

	v, ok := vm.viewers[tid]
	if ok {
		v.lockme()
		v.Follower = nil
		v.unlockme()
	}
	delete(vm.followerCache, tid)
}

// IsFollower - IS the person a follower
func (vm *ViewerMethod) IsFollower(tid ID) (bool, time.Time) {
	t, ok := vm.followerCache[tid]
	return ok, t
}

// AllKeys - Get All Viewer IDs slower than a direct range over
func (vm *ViewerMethod) AllKeys() []ID {
	vm.lockmap()
	defer vm.unlockmap()

	myKeys := make([]ID, len(vm.viewers))
	i := 0
	for k := range vm.viewers {
		myKeys[i] = k
		i++
	}

	return myKeys
}

// Get - Get Viewer by ID
func (vm *ViewerMethod) Get(twitchID ID) *Viewer {
	vm.lockmap()
	v, ok := vm.viewers[twitchID]
	vm.unlockmap()
	if ok {
		return &v
	}

	u, err := vm.client.User.Get(twitchID)
	if err != nil {
		log.Printf("Unable to get User %s\n%s", twitchID, err.Error())
		return nil
	}

	if u == nil {
		return nil
	}

	return vm.GetFromUser(*u)
}

// GetFromUser - Get Viewer from User
func (vm *ViewerMethod) GetFromUser(usr User) *Viewer {
	vm.lockmap()
	v, ok := vm.viewers[usr.ID]
	vm.unlockmap()

	if ok {
		v.SetUser(usr)
	} else {
		v = Viewer{
			TwitchID: usr.ID,
			client:   vm.client,
			User:     &usr,
		}

		vm.lockmap()
		vm.viewers[usr.ID] = v
		vm.unlockmap()
	}

	return &v
}

// GetFromChatter - Get Viewer from Chatter
func (vm *ViewerMethod) GetFromChatter(cu Chatter) *Viewer {
	if cu.id != "" {
		v := vm.Get(cu.id)
		v.SetChatter(cu)
		return v
	} else if cu.Nick != "" {
		v, err := vm.Find(cu.Nick)
		if err != nil {
			log.Printf("GetFromChatter - unable to get from nick [%s] \n%s", cu.Nick, err)
			return nil
		}
		v.SetChatter(cu)
		return v
	} else if cu.DisplayName != "" {
		v, err := vm.Find(IrcNick(cu.DisplayName))
		if err != nil {
			log.Printf("GetFromChatter - unable to get from display name [%s] \n%s", cu.DisplayName, err)
			return nil
		}
		v.SetChatter(cu)
		return v
	}

	fmt.Printf("GetFromChatter ERROR \n %#v", cu)
	return nil
}

func (vm *ViewerMethod) findViewerByName(nick IrcNick) *Viewer {
	vm.lockmap()
	defer vm.unlockmap()

	nick = IrcNick(strings.ToLower(string(nick)))
	for _, v := range vm.viewers {
		if v.User.Name == nick {
			return &v
		}
	}
	return nil
}

// Find - Find Viewer Method
func (vm *ViewerMethod) Find(nick IrcNick) (*Viewer, error) {
	v := vm.findViewerByName(nick)
	if v != nil {
		return v, nil
	}

	userList, err := vm.client.User.GetByName([]IrcNick{nick})
	if err != nil {
		return nil, err
	}

	if len(userList) == 0 {
		return nil, fmt.Errorf("No user found called: %s", nick)
	}

	return vm.GetFromUser(userList[0]), nil
}

// UpdateViewers - Update Viewers from list of Names
func (vm *ViewerMethod) UpdateViewers(nickList []IrcNick) []*Viewer {
	vList := []*Viewer{}

	unkownNicks := []IrcNick{}
	// Check if Anyone Unknown
	for _, nick := range nickList {
		ov := vm.findViewerByName(nick)
		if ov != nil {
			vList = append(vList, ov)
		} else {
			unkownNicks = append(unkownNicks, nick)
		}
	}

	if len(unkownNicks) == 0 {
		return vList
	}

	// Get Full List by Name
	userList, err := vm.client.User.GetByName(unkownNicks)
	if err != nil {
		log.Printf("Error in userList \n---\n%s\n---\n%s",
			JoinNicks(unkownNicks, 4, 18),
			err.Error())
		return nil
	}

	// Get Viewer
	for _, u := range userList {
		vList = append(vList, vm.GetFromUser(u))
	}

	return vList
}

func (vm *ViewerMethod) updateInteralFollowerCache(f ChannelFollow) {
	vm.lockmap()
	vm.followerCache[f.User.ID] = ChannelRelationship(f).CreatedAt()
	vm.unlockmap()
}

// UpdateFollowers - Update Followers values and stored cache
func (vm *ViewerMethod) UpdateFollowers(fList []ChannelFollow) {
	for _, f := range fList {
		v := vm.GetFromUser(*f.User)

		v.lockme()
		v.Follower = &f
		v.unlockme()
	}

	vm.lockmap()
	for _, f := range fList {
		vm.followerCache[f.User.ID] = ChannelRelationship(f).CreatedAt()
	}
	vm.unlockmap()
}

// GetRandomFollowers - Returns Random Follow for Channel
func (vm *ViewerMethod) GetRandomFollowers(numFollowers int) []*Viewer {

	// Lock
	vm.lockmap()
	defer vm.unlockmap()

	mapLength := len(vm.followerCache)
	listOfOffset := rand.Perm(mapLength)

	if numFollowers < mapLength {
		listOfOffset = listOfOffset[:numFollowers]
	}

	c := 0
	vRes := make([]*Viewer, numFollowers, numFollowers)

	offset := 0
	for i := range vm.followerCache {
		for _, x := range listOfOffset {
			if x == offset {
				v := vm.viewers[i]
				vRes[c] = &v
				c++
				break
			}
		}

		offset++
	}

	for i := 0; i < len(vRes); i++ {
		if vRes[i] == nil {
			fmt.Println(mapLength)
			fmt.Println(listOfOffset)
			fmt.Println(vRes)
			panic("Failed to Gen Random")
		}
	}

	return vRes
}

// MostUpToDateViewer - Get Viewer based on User recent update useful for judging how stale data is
func (vm *ViewerMethod) MostUpToDateViewer(numFollowers int) (*Viewer, time.Time) {
	var mostRecentViewer *Viewer
	oldTime := time.Unix(0, 0)

	// Lock
	vm.lockmap()
	defer vm.unlockmap()

	for _, v := range vm.viewers {
		if v.User == nil {
			continue
		}

		updatedTime := v.User.UpdatedAt()
		if updatedTime.After(oldTime) {
			oldTime = updatedTime
			mostRecentViewer = &v
		}
	}

	return mostRecentViewer, oldTime
}
