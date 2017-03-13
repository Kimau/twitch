package twitch

import (
	"fmt"
	"log"
	"sync"
)

/*  v5 User Calls
GetMe       | Get User                           | Gets a user object based on the OAuth token provided.
Get         | Get User by ID                     | Gets a specified user object.
EmoteList   | Get User Emotes                    | Gets a list of the emojis and emoticons that the specified user can use in chat
IsSubTo     | Check User Subscription by Channel | Checks if a specified user is subscribed to a specified channel. Intended for viewers.
FollowList  | Get User Follows                   | Gets a list of all channels followed by a specified user, sorted by the date when they started following each channel.
IsFollowing | Check User Follows by Channel      | Checks if a specified user follows a specified channel. If the user is following the channel, a follow object is returned.
Follow      | Follow Channel                     | Adds a specified user to the followers of a specified channel.
Unfollow    | Unfollow Channel                   | Deletes a specified user from the followers of a specified channel.
BlockList   | Get User Block List                | Gets a user’s block list.
Block       | Block User                         | Blocks the target user.
Unblock     | Unblock User                       | Unblocks the target user.
*/

// User - Twitch User
type User struct {
	ID          ID      `json:"_id"`
	Name        IrcNick `json:"name"`
	DisplayName string  `json:"display_name,omitempty"`
	Bio         string  `json:"bio,omitempty"` // "Just a gamer playing games and chatting. :)"

	Logo     string `json:"logo,omitempty"` // "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",
	UserType string `json:"type,omitempty"` // staff

	CreatedAtString string `json:"created_at"` // 2013-06-03T19:12:02Z
	UpdatedAtStr    string `json:"updated_at"` // 2016-12-14T01:01:44Z
}

// UserNotification - What Notifications the user has turned on - only for you
type UserNotification struct {
	Email bool `json:"email,omitempty"`
	Push  bool `json:"push,omitempty"`
}

// UserPersonal - Personal data can only be got for AuthUser
type UserPersonal struct {
	Email            string           `json:"email,omitempty"`
	EmailIsVerified  bool             `json:"email_verified,omitempty"`
	Partnered        bool             `json:"partnered,omitempty"`
	TwitterConnected bool             `json:"twitter_connected,omitempty"`
	Notification     UserNotification `json:"notifications,omitempty"`
}

// UserFull can only be fetched for OAuth User
type UserFull struct {
	*User
	*UserPersonal
}

// UsersMethod - Contains all the user functions
type UsersMethod struct {
	client *Client
	au     *UserAuth
}

// GetMe - Get OAuth User Details
func (u *UsersMethod) GetMe() (*UserFull, error) {
	err := u.au.checkScope(scopeUserRead)
	if err != nil {
		return nil, err
	}

	var user UserFull
	_, err = u.client.Get(u.au, "user", &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Get - Get User by ID
func (u *UsersMethod) Get(id ID) (*User, error) {
	var user User
	_, err := u.client.Get(u.au, "users/"+string(id), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UsersMethod) getByNameSmall(names []IrcNick) ([]User, error) {

	uList := struct {
		Total    int    `json:"_total"`
		UserList []User `json:"users"`
	}{}

	nameList := JoinNickComma(names)
	reqStr := fmt.Sprintf("users?login=%s", nameList)

	_, err := u.client.Get(u.au, reqStr, &uList)
	if err != nil {
		return nil, err
	}

	if uList.Total != len(names) {
		log.Printf("Total Number of Users was: %d / %d", uList.Total, len(names))
	}

	return uList.UserList, nil
}

// GetByName - Get User by v3 Name
func (u *UsersMethod) GetByName(names []IrcNick) ([]User, error) {
	numUsersPerGroup := 25
	if len(names) < 25 {
		return u.getByNameSmall(names)
	}

	uList := make([]struct {
		Total    int    `json:"_total"`
		UserList []User `json:"users"`
	}, len(names)/numUsersPerGroup+1, len(names)/numUsersPerGroup+2)

	lNum := 0
	errChannel := make(chan error, len(names)/numUsersPerGroup+1)
	wg := sync.WaitGroup{}

	for i := 0; i < len(names); i += numUsersPerGroup {
		endPointI := i + numUsersPerGroup
		if endPointI > len(names) {
			endPointI = len(names)
		}

		wg.Add(1)
		go func(gnum int, is int, ie int) {
			nameList := JoinNickComma(names[is:ie])

			reqStr := fmt.Sprintf("users?login=%s", nameList)

			_, err := u.client.Get(u.au, reqStr, &uList[gnum])
			if err != nil {
				errChannel <- err
			}
			wg.Done()
		}(lNum, i, endPointI)

		lNum++
	}

	wg.Wait()
	errChannel <- nil

	err := <-errChannel
	if err != nil {
		return nil, err
	}

	finalList := []User{}
	totalCount := 0
	for _, v := range uList {
		finalList = append(finalList, v.UserList...)
		totalCount += v.Total
	}

	if totalCount != len(names) {
		log.Printf("Total Number of Users was: %d / %d", totalCount, len(names))
	}

	return finalList, nil
}

// EmoteList - Get User Emotes
func (u *UsersMethod) EmoteList(id string) (*[]Emote, error) {
	err := u.au.checkScope(scopeUserSubscriptions)
	if err != nil {
		return nil, err
	}

	var eList []Emote
	_, err = u.client.Get(u.au, "users/"+id+"/emotes", &eList)
	if err != nil {
		return nil, err
	}

	return &eList, nil
}

// IsFollowing - Check User Follows by Channel
func (u *UsersMethod) IsFollowing(uid string, cid string) (*ChannelFollow, error) {
	var fAns ChannelFollow

	_, err := u.client.Get(u.au,
		fmt.Sprintf("/users/%s/follows/channels/%s", uid, cid), &fAns)
	if err != nil {
		return nil, err
	}

	return &fAns, nil
}

// IsSubscribed - Check User Subscription by Channel
func (u *UsersMethod) IsSubscribed(uid string, cid string) (*ChannelSub, error) {
	err := u.au.checkScope(scopeUserSubscriptions)
	if err != nil {
		return nil, err
	}

	var fAns ChannelSub

	_, err = u.client.Get(u.au,
		fmt.Sprintf("/users/%s/subscriptions/channels/%s", uid, cid), &fAns)
	if err != nil {
		return nil, err
	}

	return &fAns, nil
}

/*

// FollowList - Get User Follows
func (u *UsersMethod) FollowList(id string) error {}

// Follow - Follow Channel
func (u *UsersMethod) Follow(id string) error {}

// Unfollow - Unfollow Channel
func (u *UsersMethod) Unfollow(id string) error {}

// BlockList - Get User Block List
func (u *UsersMethod) BlockList(id string) error {}

// Block - Block User
func (u *UsersMethod) Block(id string) error {}

// Unblock - Unblock User
func (u *UsersMethod) Unblock(id string) error {}
*/
