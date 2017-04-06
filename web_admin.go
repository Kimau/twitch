package twitch

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// AdminHTTP for backoffice requests
func (ah *Client) AdminHTTP(w http.ResponseWriter, req *http.Request) {

	// Force Auth
	if ah.AdminAuth.Token == nil {
		ah.handleOAuthAdminStart(w, req)
		return
	}

	// Get Relative Path
	relPath := req.URL.Path[strings.Index(req.URL.Path, ah.servePath)+len(ah.servePath):]
	log.Println("Twitch ADMIN: ", relPath)

	switch {
	case strings.HasPrefix(relPath, "utest"):
		nickRawList := strings.Split(relPath, "/")
		nickList := make([]IrcNick, len(nickRawList)-1, len(nickRawList)-1)
		for i := 1; i < len(nickRawList); i++ {
			nickList[i-1] = IrcNick(nickRawList[i])
		}

		v := ah.UpdateViewers(nickList)
		fmt.Fprintf(w, "%#v", v)

	case strings.HasPrefix(relPath, "chat"):
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, ah.Chat.ReadChatFull())

	case strings.HasPrefix(relPath, "me"):
		uf, err := ah.User.GetMe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%#v", uf)

	case strings.HasPrefix(relPath, "updateFollowers"):
		numFollowers, err := ah.ForceUpdateFollowers()
		if err != nil {
			fmt.Fprintf(w, "Followers: %d \n %s", numFollowers, err.Error())
		} else {
			fmt.Fprintf(w, "Followers: %d", numFollowers)
		}

	case strings.HasPrefix(relPath, "user"):
		userName := regexp.MustCompile("username/([\\w]+)/*")
		r := userName.FindStringSubmatch(relPath)
		nameList := []IrcNick{IrcNick(r[1])}
		uf, err := ah.User.GetByName(nameList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%#v", uf)

	case debugOptions && strings.HasPrefix(relPath, "debug/"):
		splitD := strings.Split(req.RequestURI, "debug/")
		log.Println("Debug: " + splitD[1])
		body, err := ah.Get(ah.AdminAuth, splitD[1], nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, body)

	case debugOptions && strings.HasPrefix(relPath, "savedump"):
		if err := ah.DumpViewers(); err != nil {
			fmt.Fprintf(w, "Failed to Dump: %s", err)
		} else {
			fmt.Fprintf(w, "Dumped data to file")
		}

	default:
		http.Error(w, fmt.Sprintf("Invalid Endpoint: %s", req.URL.Path), 404)
	}
}
