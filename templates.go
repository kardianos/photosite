package main

import (
	"html/template"
	"log"
)

func loadTemplates() {
	var err error
	allTemplates, err = template.ParseGlob("template/*.template")
	if err != nil {
		log.Fatalf("Failed to parse template: %v")
	}
}
