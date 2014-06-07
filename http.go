package main

import (
	"math/rand"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
)

func setupAuthRouter() *httprouter.Router {
	router := httprouter.New()
	router.NotFound = notFoundAuth
	router.PanicHandler = httpPanic

	router.GET("/u/", rootHandler)
	router.GET("/u/:group/", checkGroup(groupHandler))
	router.GET("/u/:group/:album/", checkGroup(albumHandler))
	router.GET("/u/:group/:album/:res/:image", checkGroup(imageHandler))

	router.GET("/api/logout", logout)

	router.ServeFiles("/lib/*filepath", http.Dir(filepath.Join(root, "lib")))

	return router
}

func setupUnauthRouter() *httprouter.Router {
	router := httprouter.New()
	router.NotFound = notFoundUnauth
	router.PanicHandler = httpPanic

	router.GET("/l/", loginPage)

	router.POST("/api/login", doLogin)

	return router
}

func httpPanic(w http.ResponseWriter, r *http.Request, err interface{}) {
	log.Error("Panic in code: %v", err)
	http.Error(w, "SITE ERROR", 500)
}

// Begin unauthenticated handlers.

func loginPage(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	err := allTemplates.ExecuteTemplate(w, "login.template", struct {
		SiteName string
	}{
		SiteName: siteName,
	})
	if err != nil {
		log.Error("Error running template: %v", err)
		return
	}
}
func doLogin(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	ok := authLogin(w, r)
	if ok {
		http.Error(w, "/u/", 200)
		return
	}
	http.Error(w, "Login Failed", 403)
}
func notFoundUnauth(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/l/", 302)
}

// End unauthenticated handlers.

func notFoundAuth(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/u/", 302)
}
func logout(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	err := authLogout(w, r)
	if err != nil {
		log.Error("Failed to logout: %v", err)
		return
	}
	http.Redirect(w, r, "/", 302)
}

func checkGroup(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		c := w.(*Context)
		group := vars["group"]
		if !c.InGroup(group) {
			notFoundAuth(w, r)
			return
		}
		h(w, r, vars)
	}
}

// /
func rootHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	// List authorized groups available from context.
	c := w.(*Context)
	if len(c.Groups) == 1 {
		http.Redirect(w, r, path.Join("/u/", c.Groups[0]), 302)
		return
	}
	err := allTemplates.ExecuteTemplate(w, "root.template", struct {
		Rand     int64
		SiteName string
		C        *Context
	}{
		Rand:     rand.Int63(),
		SiteName: siteName,
		C:        c,
	})
	if err != nil {
		log.Error("Error running template: %v", err)
		return
	}
}

// /:group
func groupHandler(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	// List albums in groups (list folders in group that don't start with a ".").
	// Fetch list of folders in group.
	group := vars["group"]
	c := w.(*Context)
	albums, err := getAlbums(group)
	if err != nil {
		log.Error("Error getting albums: %v", err)
		notFoundAuth(w, r)
		return
	}
	err = allTemplates.ExecuteTemplate(w, "group.template", struct {
		Rand     int64
		SiteName string
		Group    string
		Albums   []string

		ManyGroup bool
	}{
		Rand:     rand.Int63(),
		SiteName: siteName,
		Group:    group,
		Albums:   albums,

		ManyGroup: (len(c.Groups) != 1),
	})
	if err != nil {
		log.Error("Error running template: %v", err)
		return
	}
}

// /:group/:album
func albumHandler(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	// List images from files in directory. Will reference images (below).
	album := vars["album"]
	desc, images, err := getImages(vars["group"], album)
	if err != nil {
		log.Error("Error getting images: %v", err)
		notFoundAuth(w, r)
		return
	}
	desc = strings.Trim(desc, " \n\r\t")
	titleAt := strings.Index(desc, "\n\n")
	title := ""
	if titleAt > 0 {
		title = desc[:titleAt]
		desc = desc[titleAt+2:]
	}

	err = allTemplates.ExecuteTemplate(w, "album.template", struct {
		Rand        int64
		SiteName    string
		Album       string
		Title, Desc string
		Images      []string
	}{
		Rand:     rand.Int63(),
		SiteName: siteName,
		Album:    album,
		Images:   images,
		Title:    title,
		Desc:     desc,
	})
	if err != nil {
		log.Error("Error running template: %v", err)
		return
	}
}

// /:group/:album/:res/:image
func imageHandler(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	// Serve image from Group/Album/img, cache in Group/Album/.cache/img@res.
	var (
		group = vars["group"]
		album = vars["album"]
		res   = vars["res"]
		image = vars["image"]
	)
	filename, err := getSingleImage(group, album, res, image)
	if err != nil {
		log.Error("Error getting images: %v", err)
		notFoundAuth(w, r)
		return
	}
	http.ServeFile(w, r, filename)
}
