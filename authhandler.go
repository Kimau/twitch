package twitch

import (
	"fmt"
	"net/http"
)

func (ah *Client) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if req.Method == "GET" {

		var match []string

		match = authSignRegEx.FindStringSubmatch(req.URL.Path)
		if match != nil {
			fullRedirStr := fmt.Sprintf(baseURL, clientID, redirURL, ah.getScopeString(), match[1])
			http.Redirect(w, req, fullRedirStr, http.StatusSeeOther)
			return
		}

		match = authRegEx.FindStringSubmatch(req.URL.Path)
		if match != nil {
			fmt.Fprintf(w, "Hello your code is %s", match[1])
			return
		}

		match = authCancelRegEx.FindStringSubmatch(req.URL.Path)
		if match != nil {
			fmt.Fprintf(w, "Hello your auth was cancelled")
			return
		}
	}

	http.Error(w, fmt.Sprintf("Invalid Endpoint: %s", req.URL.Path), 404)

}
