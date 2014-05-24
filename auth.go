package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

func getAuth(r *http.Request) (username, password string, tried bool) {
	a := r.Header.Get("Authorization")
	if a == "" {
		return
	}
	tried = true

	basic := "Basic "
	index := strings.Index(a, basic)
	if index < 0 {
		return
	}

	upString, err := base64.StdEncoding.DecodeString(a[index+len(basic):])
	if err != nil {
		return
	}
	up := strings.SplitN(string(upString), ":", 2)
	if len(up) != 2 {
		return
	}

	username = up[0]
	password = up[1]

	return
}
func sendUnAuth(w http.ResponseWriter, realm string) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", realm))
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("unauthorized"))
}
