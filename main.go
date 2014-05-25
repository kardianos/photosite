// photosite
/*
URL Root:
	/
		users.txt < username:password@groupA,groupB <newline> username2:password@groupB
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
	"crypto/tls"
	"html/template"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"bitbucket.org/kardianos/service"
	"bitbucket.org/kardianos/service/config"
	srv "bitbucket.org/kardianos/service/stdservice"
)

// Input: www root, TLS certs.

const (
	siteName = "Photo Site"
	domain   = "photosite.com"
	root     = "/srv/photosite"

	plainAddr = ":http"
	tlsAddr   = ":https"

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

	sampleUsers = &UserList{
		Order: []*User{
			{
				Username: "usernameA",
				Password: "letmein",
				Groups:   []string{"g1", "g2"},
			},
		},
	}

	userWatch *config.WatchConfig

	log service.Logger

	plainListen net.Listener
	plainServer *http.Server

	tlsListen net.Listener
	tlsServer *http.Server
)

// TODO: Load setting from config file.
// TODO: Add in-line large photo viewer.
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	sc := &srv.Config{
		Name:            "photosite",
		DisplayName:     "Photo Site",
		LongDescription: "Photo viewing website",

		Start: start,
		Stop:  stop,

		Init: httpInit,
	}
	sc.Run()

	defer stop(nil)
	start(nil)
}

type redirectToDomain string

func (rt redirectToDomain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://www."+string(rt), 301)
}

type redirectToAuth struct {
	auth http.Handler
}

func (rt redirectToAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Info("Request URL: %#v", r.Host)
	if strings.HasPrefix(r.Host, "www.") == false {
		http.Redirect(w, r, "https://www."+r.Host, 301)
		return
	}
	rt.auth.ServeHTTP(w, r)

}

func httpInit(c *srv.Config) error {
	var err error
	log = c.Logger()

	err = loadTemplates()
	if err != nil {
		return err
	}

	auth = &AuthHandler{
		Authorized:   setupAuthRouter(),
		Unauthorized: setupUnauthRouter(),
	}

	sessions, err = NewDiskSessionList(filepath.Join(root, sessionFileName))
	if err != nil {
		log.Error("Failed to start disk sessions: %v", err)
		return err
	}

	usersFile := filepath.Join(root, usersFileName)
	userWatch, err = config.NewWatchConfig(usersFile, UserDecode, sampleUsers, UserEncode)
	if err != nil {
		log.Error("Failed to start userWatch: %s", err)
		return err
	}
	err = listen(plainAddr, redirectToDomain(domain))
	if err != nil {
		return err
	}
	err = listenSecure(tlsAddr, domain, redirectToAuth{auth})
	if err != nil {
		return err
	}

	return nil
}
func start(c *srv.Config) {
	var err error

	go startExpire()

	go func() {
		for {
			select {
			case <-userWatch.C:
				userList := &UserList{}
				err := userWatch.Load(userList)
				if err != nil {
					log.Error("Failed to load user list: %v", err)
					continue
				}
				auth.Lock()
				auth.AuthorizedList = userList
				auth.Unlock()
				log.Info("Users loaded.")
			}
		}
	}()
	userWatch.TriggerC()

	go func() {
		err := tlsServer.Serve(tlsListen)
		log.Error("Failed to serve: %v", err)
	}()
	err = plainServer.Serve(plainListen)
	log.Error("Failed to serve: %v", err)
}

func stop(c *srv.Config) {
	if userWatch != nil {
		userWatch.Close()
	}
	if sessions != nil {
		sessions.Close()
	}
}

func listenSecure(addr, domain string, h http.Handler) error {
	config := &tls.Config{
		NextProtos: []string{"http/1.1"},
		MinVersion: tls.VersionTLS10,
	}
	var err error

	tlsServer = &http.Server{
		Addr:      addr,
		Handler:   h,
		TLSConfig: config,
	}

	cert_com, err := tls.LoadX509KeyPair(filepath.Join(root, "cert", "cert_com.pem"), filepath.Join(root, "cert", "key_com.pem"))
	if err != nil {
		return err
	}
	config.Certificates = []tls.Certificate{
		cert_com,
	}

	config.NameToCertificate = map[string]*tls.Certificate{
		domain:          &cert_com,
		"www." + domain: &cert_com,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListen = tls.NewListener(ln, config)
	return nil
}

func listen(addr string, handler http.Handler) error {
	var err error
	plainServer = &http.Server{Addr: addr, Handler: handler}
	plainListen, err = net.Listen("tcp", addr)
	return err
}
