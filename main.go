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
	"time"

	"bitbucket.org/kardianos/service/config"
)

// Input: www root, TLS certs.

const (
	siteName = "Photo Site"

	cookieKeyName = "sk"
	keyByteLength = 2048 / 8

	secureConnection  = false
	expireSessionTime = 2 * time.Hour
	maxSessionTime    = 24 * time.Hour
)

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

var auth *AuthHandler

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	auth = &AuthHandler{
		Authorized:   setupAuthRouter(),
		Unauthorized: setupUnauthRouter(),
	}

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
