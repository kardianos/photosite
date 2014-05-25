package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

func UserDecode(r io.Reader, v interface{}) error {
	userList, isValue := v.(*UserList)
	if !isValue {
		return fmt.Errorf("Incoming value is not of type: *UserList")
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	b = bytes.Trim(b, " \t\n\r")
	lines := bytes.Split(b, []byte("\n"))

	userList.Order = make([]*User, len(lines))
	userList.ByUsername = make(map[string]*User, len(lines))

	i := 0
	for _, line := range lines {
		if len(line) == 0 || line[0] == byte('#') {
			continue
		}
		passwordIndex := bytes.IndexRune(line, ':')
		groupsIndex := bytes.IndexRune(line, '@')
		if passwordIndex <= 0 {
			continue
		}
		if groupsIndex <= 0 {
			continue
		}
		if groupsIndex < passwordIndex {
			continue
		}
		username := line[:passwordIndex]
		password := line[passwordIndex+1 : groupsIndex]
		groups := line[groupsIndex+1:]

		groupList := bytes.Split(groups, []byte(","))

		if len(groupList) == 0 {
			continue
		}
		if len(username) < minUsernameLength {
			log.Error("Username must at least %d letters", minUsernameLength)
			continue
		}
		if len(password) < minPasswordLength {
			log.Error("Password must at least %d letters", minPasswordLength)
			continue
		}

		u := &User{
			Username: string(username),
			Password: string(password),
			Groups:   make([]string, len(groupList)),
		}
		for gi, g := range groupList {
			u.Groups[gi] = string(g)
		}
		userList.Order[i] = u
		userList.ByUsername[u.Username] = u

		i++
	}
	userList.Order = userList.Order[:i]

	return nil
}

func UserEncode(w io.Writer, v interface{}) error {
	userList, isValue := v.(*UserList)
	if !isValue {
		return fmt.Errorf("Incoming value is not of type: *UserList")
	}
	for _, user := range userList.Order {
		line := fmt.Sprintf("%s:%s@%s\n", user.Username, user.Password, strings.Join(user.Groups, ","))
		_, err := io.WriteString(w, line)
		if err != nil {
			return err
		}
	}
	return nil
}
