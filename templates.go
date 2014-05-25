package main

import (
	"html/template"
)

func loadTemplates() error {
	var err error
	allTemplates, err = template.ParseGlob("template/*.template")
	return err
}
