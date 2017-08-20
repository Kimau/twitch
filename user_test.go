package twitch

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// generateDummyID - Useful for testing
func generateDummyID() ID {
	x := rand.Intn(10000000) + 10000000
	return ID(fmt.Sprintf("%d", x))
}

func selectRand(inList []interface{}) interface{} {
	return inList[rand.Intn(len(inList))]
}

// User - Twitch User
func generateDummyUser() User {

	return User{
		ID:          generateDummyID(),
		Name:        IrcNick(GenerateRandomString(16)),
		DisplayName: GenerateRandomString(16),
		Bio:         GenerateRandomString(64),

		Logo:     "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",
		UserType: TwitchTypeMod,

		CreatedAtString: "2013-06-03T19:12:02Z",
		UpdatedAtString: "2016-12-14T01:01:44Z",
	}
}

func TestUserFull(t *testing.T) {

	xStr := `{
    "_id": "44322889",
    "bio": "Just a gamer playing games and chatting. :)",
    "created_at": "2013-06-03T19:12:02Z",
    "display_name": "dallas",
    "email": "email-address@provider.com",
    "email_verified": true,
    "logo": "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",
    "name": "dallas",
    "notifications": {
        "email": false,
        "push": true
    },
    "partnered": false,
    "twitter_connected": false,
    "type": "staff", 
    "updated_at": "2016-12-14T01:01:44Z"
}`

	testUser := UserFull{
		User: &User{
			ID:              "44322889",
			Name:            "dallas",
			DisplayName:     "dallas",
			Bio:             "Just a gamer playing games and chatting. :)",
			Logo:            "https://static-cdn.jtvnw.net/jtv_user_pictures/dallas-profile_image-1a2c906ee2c35f12-300x300.png",
			UserType:        "staff",
			CreatedAtString: "2013-06-03T19:12:02Z",
			UpdatedAtString: "2016-12-14T01:01:44Z",
		},

		UserPersonal: &UserPersonal{
			Email:            "email-address@provider.com",
			EmailIsVerified:  true,
			Partnered:        false,
			TwitterConnected: false,
			Notification: UserNotification{
				Email: false,
				Push:  true,
			},
		},
	}

	dec := json.NewDecoder(strings.NewReader(xStr))
	var user UserFull
	err := dec.Decode(&user)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if testUser.ID != user.ID {
		t.Fail()
		t.Logf("User [%s:%s] not equal", testUser.ID, user.ID)
	}

	if testUser.Name != user.Name {
		t.Fail()
		t.Logf("Name [%s:%s] not equal", testUser.Name, user.Name)
	}

	if testUser.DisplayName != user.DisplayName {
		t.Fail()
		t.Logf("DisplayName [%s:%s] not equal", testUser.DisplayName, user.DisplayName)
	}

	if testUser.Bio != user.Bio {
		t.Fail()
		t.Logf("Bio [%s:%s] not equal", testUser.Bio, user.Bio)
	}

	if testUser.Logo != user.Logo {
		t.Fail()
		t.Logf("Logo [%s:%s] not equal", testUser.Logo, user.Logo)
	}

	if testUser.UserType != user.UserType {
		t.Fail()
		t.Logf("UserType [%s:%s] not equal", testUser.UserType, user.UserType)
	}

	if testUser.CreatedAtString != user.CreatedAtString {
		t.Fail()
		t.Logf("CreatedAtString [%s:%s] not equal", testUser.CreatedAtString, user.CreatedAtString)
	}

	if testUser.UpdatedAtString != user.UpdatedAtString {
		t.Fail()
		t.Logf("UpdatedAtString [%s:%s] not equal", testUser.UpdatedAtString, user.UpdatedAtString)
	}
	if testUser.Email != user.Email {
		t.Fail()
		t.Logf("Email [%s:%s] not equal", testUser.Email, user.Email)
	}
	if testUser.EmailIsVerified != user.EmailIsVerified {
		t.Fail()
		t.Logf("EmailIsVerified [%t:%t] not equal", testUser.EmailIsVerified, user.EmailIsVerified)
	}
	if testUser.Partnered != user.Partnered {
		t.Fail()
		t.Logf("Partnered [%t:%t] not equal", testUser.Partnered, user.Partnered)
	}
	if testUser.TwitterConnected != user.TwitterConnected {
		t.Fail()
		t.Logf("TwitterConnected [%t:%t] not equal", testUser.TwitterConnected, user.TwitterConnected)
	}

	if testUser.Notification.Email != user.Notification.Email {
		t.Fail()
		t.Logf("Notification.Email [%t:%t] not equal", testUser.Notification.Email, user.Notification.Email)
	}
	if testUser.Notification.Push != user.Notification.Push {
		t.Fail()
		t.Logf("Notification.Push [%t:%t] not equal", testUser.Notification.Push, user.Notification.Push)
	}
}

func TestUsersMethod_GetMe(t *testing.T) {
	tests := []struct {
		name    string
		u       *UsersMethod
		want    *UserFull
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.u.GetMe()
			if (err != nil) != tt.wantErr {
				t.Errorf("UsersMethod.GetMe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UsersMethod.GetMe() = %v, want %v", got, tt.want)
			}
		})
	}
}
