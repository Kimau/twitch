package twitch

import (
	"fmt"
)

type chatMode struct {
	subsOnly      bool
	emoteOnly     bool
	followersOnly bool
	slowMode      bool
	r9k           bool
	lang          string
	hosting       *Viewer
}

func (cm chatMode) String() string {
	if cm.hosting != nil {
		return fmt.Sprintf("Hosting %s", cm.hosting.GetNick())
	}
	s := ""

	if len(cm.lang) > 0 {
		s += fmt.Sprintf("[%s]", cm.lang)
	}

	if cm.subsOnly {
		s += " Subs only"
	}
	if cm.emoteOnly {
		s += " Emotes only"
	}
	if cm.followersOnly {
		s += " Followers only"
	}
	if cm.slowMode {
		s += " Slow mode"
	}
	if cm.r9k {
		s += " r9k"
	}

	if len(s) < 1 {
		return "default"
	}

	return s
}
