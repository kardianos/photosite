// photosite
/*
URL Root:
	/
		.users < username:password@groupA,groupB <newline> username2:password@groupB
		groupA/
			album1/
				.cache/
					imgA@200.jpg
					imgA@640.jpg
					imgB@200.jpg
					imgB@640.jpg
				text.txt < title <newline><newline> body
				imgA.jpg
				imgB.jpg

*/
package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"

	"bitbucket.org/kardianos/service/config"
)

// Input: www root, TLS certs.

var root = flag.String("root", "", "Application Root")

var (
	minUsernameLength = 8
	minPasswordLength = 6
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

type AuthHandler struct {
	Authorized http.Handler

	sync.RWMutex
	AuthorizedList *UserList
}

func (auth *AuthHandler) groups(username, password string) []string {
	auth.RLock()
	defer auth.RUnlock()

	if auth.AuthorizedList == nil {
		return nil
	}

	u, found := auth.AuthorizedList.ByUsername[username]
	if !found {
		return nil
	}
	if u.Username != username || u.Password != password {
		return nil
	}

	return u.Groups
}

func (auth *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, p, tried := getAuth(r)
	if !tried {
		sendUnAuth(w, "Photo Site")
		return
	}
	groups := auth.groups(u, p)
	if len(groups) == 0 {
		sendUnAuth(w, "Photo Site")
		return
	}

	c := &Context{
		ResponseWriter: w,
		Username:       u,
		Groups:         groups,
	}
	auth.Authorized.ServeHTTP(c, r)
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

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	auth := &AuthHandler{Authorized: setupRouter()}

	usersFile := filepath.Join(*root, ".users")
	sampleUsers := &UserList{
		Order: []*User{
			{
				Username: "usernameA",
				Password: "letmein",
				Groups:   []string{"g1", "g2"},
			},
		},
	}
	watch, err := config.NewWatchConfig(usersFile, UserDecode, sampleUsers, UserEncode)
	if err != nil {
		log.Fatalf("Failed to start watch: %s", err)
	}
	defer watch.Close()
	go func() {
		for {
			select {
			case <-watch.C:
				userList := &UserList{}
				err := watch.Load(userList)
				if err != nil {
					log.Printf("Failed to load user list: %v", err)
					continue
				}
				auth.Lock()
				auth.AuthorizedList = userList
				auth.Unlock()
				log.Print("Users loaded.")
			}
		}
	}()
	watch.TriggerC()

	log.Fatal(http.ListenAndServe(":9080", auth))
}
