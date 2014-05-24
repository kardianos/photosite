package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

func setupRouter() *httprouter.Router {
	router := httprouter.New()
	router.NotFound = notFound

	router.GET("/", rootHandler)
	router.GET("/u/:group/", checkGroup(groupHandler))
	router.GET("/u/:group/:album/", checkGroup(albumHandler))
	router.GET("/u/:group/:album/:res/:image", checkGroup(imageHandler))

	router.ServeFiles("/lib/*filepath", http.Dir("lib"))

	return router
}

func notFound(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", 302)
}

func checkGroup(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		c := w.(*Context)
		group := vars["group"]
		if !c.InGroup(group) {
			notFound(w, r)
			return
		}
		h(w, r, vars)
	}
}

// /
func rootHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	// List authorized groups available from context.
	c := w.(*Context)
	err := allTemplates.ExecuteTemplate(w, "root.template", c)
	if err != nil {
		log.Printf("Error running template: %v", err)
		return
	}
}

// /:group
func groupHandler(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	// List albums in groups (list folders in group that don't start with a ".").
	// Fetch list of folders in group.
	group := vars["group"]
	albums, err := getAlbums(group)
	if err != nil {
		log.Printf("Error getting albums: %v", err)
		notFound(w, r)
		return
	}
	err = allTemplates.ExecuteTemplate(w, "group.template", struct {
		Group  string
		Albums []string
	}{
		Group:  group,
		Albums: albums,
	})
	if err != nil {
		log.Printf("Error running template: %v", err)
		return
	}
}

// /:group/:album
func albumHandler(w http.ResponseWriter, r *http.Request, vars map[string]string) {
	// List images from files in directory. Will reference images (below).
	album := vars["album"]
	desc, images, err := getImages(vars["group"], album)
	if err != nil {
		log.Printf("Error getting images: %v", err)
		notFound(w, r)
		return
	}
	desc = strings.Trim(desc, " \n\r\t")
	titleAt := strings.Index(desc, "\n\n")
	title := desc[:titleAt]
	desc = desc[titleAt+2:]

	err = allTemplates.ExecuteTemplate(w, "album.template", struct {
		Album       string
		Title, Desc string
		Images      []string
	}{
		Album:  album,
		Images: images,
		Title:  title,
		Desc:   desc,
	})
	if err != nil {
		log.Printf("Error running template: %v", err)
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
		log.Printf("Error getting images: %v", err)
		notFound(w, r)
		return
	}
	http.ServeFile(w, r, filename)
}
