// +build !PROD

package main

const (
	siteName = "Photo Site"
	domain   = "photosite.com"
	root     = "/home/daniel/src/bitbucket.org/kardianos/photosite"

	diskSession      = false
	secureConnection = false
	plainAddr        = ":8080"
	tlsAddr          = ":8081"
)
