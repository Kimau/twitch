package twitch

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func (ah *Client) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if req.Method == "GET" {

		var match []string

		log.Println("Twitch: ", req.URL.Path)

		match = authSignRegEx.FindStringSubmatch(req.URL.Path)
		if match != nil {
			fullRedirStr := fmt.Sprintf(baseURL, clientID, redirURL, ah.getScopeString(), match[1])
			http.Redirect(w, req, fullRedirStr, http.StatusSeeOther)
			return
		}

		match = authRegEx.FindStringSubmatch(req.URL.Path)
		if match != nil {
			fmt.Fprintf(w, "Hello\n code: %s \n name: %s \n scope: %s", match[1], match[3], match[2])
			return
		}

		match = authCancelRegEx.FindStringSubmatch(req.URL.Path)
		if match != nil {
			ah.handleOAuthResult(w, req)
			return

		}
	}

	http.Error(w, fmt.Sprintf("Invalid Endpoint: %s", req.URL.Path), 404)

}

func (ah *Client) handleOAuthResult(w http.ResponseWriter, req *http.Request) {
	qList := req.URL.Query()

	c, ok := qList["code"]
	if !ok {
		fmt.Fprintf(w, "Hello your auth was cancelled")

		for k, v := range qList {
			fmt.Fprintf(w, "%s:%s\n", k, v)
		}
		return
	}

	nameList, ok := qList["state"]
	if !ok {
		fmt.Fprintf(w, "Invalid State")
		return
	}
	name := nameList[0]

	scopeList, ok := qList["scope"]
	if !ok {
		fmt.Fprintf(w, "Invalid Scope")
		return
	}
	scopeList = strings.Split(scopeList[0], " ")

	fmt.Fprintf(w,
		"Hello %s your code is [%s] and you have been allowed these scopes\n",
		name, c)

	for _, s := range scopeList {
		fmt.Fprintf(w, "* %s \n", s)
	}
}
