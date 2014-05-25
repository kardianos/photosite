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
	"html/template"
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

	minUsernameLength = 8
	minPasswordLength = 6

	usersFileName   = "users.txt"
	sessionFileName = "sessions.bolt"

	cacheDir        = ".cache"
	descriptionFile = "Description.txt"
)

var (
	sizes = []int{200, 1280}

	allTemplates *template.Template

	root = flag.String("root", "", "Application Root")

	sampleUsers = &UserList{
		Order: []*User{
			{
				Username: "usernameA",
				Password: "letmein",
				Groups:   []string{"g1", "g2"},
			},
		},
	}

	watch *config.WatchConfig
)

// TODO: Load setting from config file.
// TODO: Run as service.
// TODO: Add TLS server.
// TODO: Add in-line large photo viewer.
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer stop()
	start()
}

func start() {
	auth = &AuthHandler{
		Authorized:   setupAuthRouter(),
		Unauthorized: setupUnauthRouter(),
	}
	var err error

	sessions, err = NewDiskSessionList(filepath.Join(*root, sessionFileName))
	if err != nil {
		log.Fatalf("Failed to start disk sessions: %v", err)
	}

	go startExpire()
	loadTemplates()

	usersFile := filepath.Join(*root, usersFileName)
	watch, err = config.NewWatchConfig(usersFile, UserDecode, sampleUsers, UserEncode)
	if err != nil {
		log.Fatalf("Failed to start watch: %s", err)
	}
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

func stop() {
	if watch != nil {
		watch.Close()
	}
	if sessions != nil {
		sessions.Close()
	}
}
