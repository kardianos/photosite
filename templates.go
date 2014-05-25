package main

import (
	"html/template"
	"path/filepath"
)

func loadTemplates() error {
	var err error
	allTemplates, err = template.ParseGlob(filepath.Join(root, "template", "*.template"))
	return err
}
