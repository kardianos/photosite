package main

import (
	"net/http"
	"path/filepath"
	"sync"
	"time"
)

var (
	sessions Session

	auth *AuthHandler
)

type User struct {
	Username string
	Password string

	Groups []string
}

type UserList struct {
	Order      []*User
	ByUsername map[string]*User
}

type Context struct {
	http.ResponseWriter

	Username string
	Groups   []string
}

func (c *Context) InGroup(group string) bool {
	for _, g := range c.Groups {
		if g == group {
			return true
		}
	}
	return false
}

type AuthHandler struct {
	Authorized   http.Handler
	Unauthorized http.Handler

	sync.RWMutex
	AuthorizedList *UserList
}

func (auth *AuthHandler) isValid(username, password string) bool {
	auth.RLock()
	defer auth.RUnlock()

	if auth.AuthorizedList == nil {
		return false
	}

	u, found := auth.AuthorizedList.ByUsername[username]
	if !found {
		return false
	}
	if u.Username != username || u.Password != password {
		return false
	}

	return true
}
func (auth *AuthHandler) groups(username string) []string {
	auth.RLock()
	defer auth.RUnlock()

	if auth.AuthorizedList == nil {
		return nil
	}

	u, found := auth.AuthorizedList.ByUsername[username]
	if !found {
		return nil
	}

	return u.Groups
}

func (auth *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		http.ServeFile(w, r, filepath.Join(root, "lib/favicon.ico"))
		return
	}
	username, in := authCheck(r)
	if !in {
		auth.Unauthorized.ServeHTTP(w, r)
		return
	}
	groups := auth.groups(username)

	c := &Context{
		ResponseWriter: w,
		Username:       username,
		Groups:         groups,
	}
	auth.Authorized.ServeHTTP(c, r)
}

func startExpire() {
	// Do not make longer then one minute.
	ticker := time.NewTicker(checkExpireTime)
	for {
		<-ticker.C
		now := time.Now()
		sessions.ExpireBefore(now.Add(-expireSessionTime), now.Add(-maxSessionTime))
	}
}

// Checks request for auth cookie. Validates auth cookie.
func authCheck(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(cookieKeyName)
	if err != nil || cookie == nil {
		return "", false
	}
	username, err := sessions.HasKey(cookie.Value)
	if err != nil {
		log.Error("Error checking session key: %v", err)
		return "", false
	}
	if len(username) == 0 {
		return "", false
	}
	return username, true
}

func authLogin(w http.ResponseWriter, r *http.Request) bool {
	err := r.ParseForm()
	if err != nil {
		log.Error("Error parsing form: %v", err)
		return false
	}
	u := r.Form.Get("username")
	p := r.Form.Get("password")
	valid := auth.isValid(u, p)
	if !valid {
		return false
	}
	key, err := sessions.Insert(u)
	if err != nil {
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieKeyName,
		Value:    key,
		Expires:  time.Now().Add(maxSessionTime),
		Path:     "/",
		HttpOnly: true,
		Secure:   secureConnection,
	})
	return true
}

func authLogout(w http.ResponseWriter, r *http.Request) error {
	c := w.(*Context)
	http.SetCookie(w, &http.Cookie{
		Name:   cookieKeyName,
		Path:   "/",
		MaxAge: -1,
	})
	return sessions.Delete(c.Username)
}
