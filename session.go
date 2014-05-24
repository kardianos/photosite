package main

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type sessionItem struct {
	username string
	update   time.Time
	create   time.Time
}

type SessionList struct {
	sync.Mutex
	list map[string]*sessionItem
}

func NewSessionList(persistPath string) *SessionList {
	return &SessionList{
		list: make(map[string]*sessionItem, 10),
	}
}

func (s *SessionList) HasKey(key string) (username string, err error) {
	s.Lock()
	defer s.Unlock()

	item, found := s.list[key]
	if !found {
		return "", nil
	}
	item.update = time.Now()

	return item.username, nil
}
func (s *SessionList) Insert(username string) (key string, err error) {
	s.Lock()
	defer s.Unlock()

	b := make([]byte, keyByteLength)
	_, err = rand.Read(b)
	if err != nil {
		return "", err
	}

	now := time.Now()

	key = base64.StdEncoding.EncodeToString(b)
	s.list[key] = &sessionItem{
		username: username,
		update:   now,
		create:   now,
	}

	return key, nil
}
func (s *SessionList) Delete(username string) (err error) {
	s.Lock()
	defer s.Unlock()

	keys := []string{}
	for k, item := range s.list {
		if item.username == username {
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		delete(s.list, k)
	}

	return nil
}
func (s *SessionList) ExpireBefore(update time.Time, create time.Time) (err error) {
	s.Lock()
	defer s.Unlock()

	keys := []string{}
	for k, item := range s.list {
		if item.update.Before(update) || item.create.Before(create) {
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		delete(s.list, k)
	}

	return nil
}
