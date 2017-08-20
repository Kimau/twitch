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
	viewers       map[ID]*Viewer
	followerCache map[ID]time.Time
}

// CreateViewerMethod - Create VM with client
func CreateViewerMethod(c *Client) *ViewerMethod {
	return &ViewerMethod{
		client: c,

		viewers:       make(map[ID]*Viewer),
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

// GetRoomName - Get Room Name
func (vm *ViewerMethod) GetRoomName() IrcNick {
	return vm.client.RoomName
}

// Set - Set New Viewer Value
func (vm *ViewerMethod) Set(v *Viewer) {
	vm.lockmap()
	defer vm.unlockmap()

	newV := vm.allocViewer(v.TwitchID)

	newV.User = v.User
	newV.Auth = v.Auth
	newV.Chatter = v.Chatter
	newV.Follower = v.Follower

	if newV.Follower == nil {
		delete(vm.followerCache, newV.TwitchID)
	} else {
		vm.followerCache[newV.TwitchID] = ChannelRelationship(*newV.Follower).CreatedAt()
	}
}

//SetFollower - Sets the new value in with lock
func (vm *ViewerMethod) SetFollower(newVal ChannelFollow) {
	v := vm.GetFromUser(*newVal.User)
	v.Lockme()
	v.Follower = &newVal
	v.Unlockme()

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
		v.Lockme()
		v.Follower = nil
		v.Unlockme()
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

// GetCopy - Get Copy of Viewer
func (vm *ViewerMethod) GetCopy(twitchID ID) (Viewer, error) {
	var v Viewer
	src := vm.GetPtr(twitchID)
	if src != nil {
		src.CopyTo(&v)
		return v, nil
	}

	err := fmt.Errorf("Unable to Find Viewer")
	return v, err
}

// GetPtr - Get Viewer by ID
func (vm *ViewerMethod) GetPtr(twitchID ID) *Viewer {
	vm.lockmap()
	v, ok := vm.viewers[twitchID]
	vm.unlockmap()
	if ok {
		return v
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

func (vm *ViewerMethod) allocViewer(tid ID) *Viewer {
	v := new(Viewer)
	v.TwitchID = tid
	v.client = vm.client

	vm.viewers[tid] = v

	return v
}

// GetFromUser - Get Viewer from User
func (vm *ViewerMethod) GetFromUser(usr User) *Viewer {
	vm.lockmap()
	defer vm.unlockmap()

	v, ok := vm.viewers[usr.ID]

	if ok {
		v.SetUser(usr)
	} else {
		v = vm.allocViewer(usr.ID)
		v.User = &usr
	}

	v.CreateChatter()

	return v
}

func (vm *ViewerMethod) findViewerByName(nick IrcNick) *Viewer {
	vm.lockmap()
	defer vm.unlockmap()

	nick = IrcNick(strings.ToLower(string(nick)))
	for k := range vm.viewers {
		v := vm.viewers[k]
		if v.User.Name == nick {
			return v
		}
	}
	return nil
}

// Find - Find Viewer Method
func (vm *ViewerMethod) Find(nick IrcNick) (*Viewer, error) {
	if nick.IsValid() == false {
		return nil, fmt.Errorf("Invalid Nick")
	}

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

		v.Lockme()
		v.Follower = &f
		v.Unlockme()
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
				vRes[c] = v
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

	for k := range vm.viewers {
		v := vm.viewers[k]

		if v.User == nil {
			continue
		}

		updatedTime := v.User.UpdatedAt()
		if updatedTime.After(oldTime) {
			oldTime = updatedTime
			mostRecentViewer = v
		}
	}

	return mostRecentViewer, oldTime
}

// SanityScan - Checks Viewer Data for Issues
func (vm *ViewerMethod) SanityScan() error {
	vm.lockmap()
	defer vm.unlockmap()

	for k := range vm.viewers {
		v := vm.viewers[k]

		if k != v.TwitchID {
			return fmt.Errorf("Twitch ID doesn't match %s\n\t %s != %s", v.GetNick(), k, v.TwitchID)
		}

		if v.User == nil {
			return fmt.Errorf("User is nil: %s", k)
		}

		if k != v.User.ID {
			return fmt.Errorf("User ID doesn't match %s\n\t %s != %s", v.GetNick(), k, v.User.ID)
		}
	}

	return nil
}
