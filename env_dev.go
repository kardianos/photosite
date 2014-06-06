// +build !PROD

package main

import (
	"time"
)

const (
	siteName = "Photo Site"
	domain   = "photosite.com"
	root     = "/home/daniel/src/bitbucket.org/kardianos/photosite"

	diskSession      = true
	secureConnection = false
	plainAddr        = ":8080"
	tlsAddr          = ":8081"

	checkExpireTime   = time.Minute
	expireSessionTime = 2 * time.Hour
	maxSessionTime    = 24 * time.Hour

	minUsernameLength = 8
	minPasswordLength = 6
)
