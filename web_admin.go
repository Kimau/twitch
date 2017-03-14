package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// AdminHTTP for backoffice requests
func (ah *Client) AdminHTTP(w http.ResponseWriter, req *http.Request) {
	// Get Relative Path

	relPath := req.URL.Path[strings.Index(req.URL.Path, ah.servePath)+len(ah.servePath):]
	log.Println("Twitch ADMIN: ", relPath)

	// Force Auth
	if ah.AdminAuth.token == nil {
		ah.handleOAuthAdminStart(w, req)
		return
	}

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
		io.Copy(w, &ah.Chat.logBuffer)

	case strings.HasPrefix(relPath, "me"):
		uf, err := ah.User.GetMe()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%#v", uf)

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

	default:
		b, err := json.Marshal(ah.AdminUser)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid Endpoint: %s", req.URL.Path), 404)
		} else {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, string(b))
		}
	}
}
