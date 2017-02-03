package twitch

import (
	"regexp"
	"testing"
)

func TestClient_Regex(t *testing.T) {

	testCases := []struct {
		rexp *regexp.Regexp
		url  string
		res  []string
	}{
		{
			authSignRegEx,
			"/twitch/signin/kimau/",
			[]string{
				"/twitch/signin/kimau/",
				"kimau",
			},
		},
		{
			authSignRegEx,
			"/twitch/signin/kimau",
			[]string{
				"/twitch/signin/kimau",
				"kimau",
			},
		},

		{
			authRegEx,
			"/twitch/after_signin/?code=fk4hx4pt6pe5mogmncy8c2f4zwlxor&scope=channel_subscriptions+channel_feed_read+user_blocks_read+user_follows_edit+channel_feed_edit+user_subscriptions+channel_read+user_blocks_edit+user_read+channel_check_subscription+channel_editor&state=kimau",
			[]string{
				"/twitch/after_signin/?code=fk4hx4pt6pe5mogmncy8c2f4zwlxor&scope=channel_subscriptions+channel_feed_read+user_blocks_read+user_follows_edit+channel_feed_edit+user_subscriptions+channel_read+user_blocks_edit+user_read+channel_check_subscription+channel_editor&state=kimau",
				"fk4hx4pt6pe5mogmncy8c2f4zwlxor",
				"channel_subscriptions+channel_feed_read+user_blocks_read+user_follows_edit+channel_feed_edit+user_subscriptions+channel_read+user_blocks_edit+user_read+channel_check_subscription+channel_editor",
				"kimau",
			},
		},

		{
			authCancelRegEx,
			"/twitch/after_signin/",
			[]string{"/twitch/after_signin/"},
		},
	}

SubStringLoop:
	for i, test := range testCases {

		res := test.rexp.FindStringSubmatch(test.url)
		l1 := len(test.res)
		l2 := len(res)
		if l1 != l2 {
			t.Logf("Failed %d because lengths didn't match [%d:%d]", i, l1, l2)
			t.Fail()
			continue
		}

		for i, v := range test.res {
			if res[i] != v {
				t.Logf("%d : %s != %s", i, res[i], v)
				t.Fail()
				continue SubStringLoop
			}
		}
	}

}
