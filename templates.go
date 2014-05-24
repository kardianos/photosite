package main

import (
	"html/template"
	"log"
)

var allTemplates *template.Template

func init() {
	var err error
	allTemplates, err = template.ParseGlob("template/*.template")
	if err != nil {
		log.Fatalf("Failed to parse template: %v")
	}
}

// var t = template.Must(template.New("name").Parse("html"))
