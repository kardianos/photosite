package session

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type Session interface {
	HasKey(key string) (username string, err error)
	Insert(username string) (key string, err error)
	Delete(username string) (err error)
	DeleteKey(key string) (err error)
	ExpireBefore(update time.Time, create time.Time) (err error)
	Close() error
}

type Length struct {
	Username string
	Start    time.Time
	Duration time.Duration
}

type sessionItem struct {
	username string
	update   time.Time
	create   time.Time
}

type MemorySessionList struct {
	keyLength int
	length    chan<- Length
	sync.Mutex
	list map[string]*sessionItem
}

func NewMemorySessionList(path string, keyLength int, length chan<- Length) (Session, error) {
	return &MemorySessionList{
		keyLength: keyLength,
		list:      make(map[string]*sessionItem, 10),
	}, nil
}
func (s *MemorySessionList) sendLength(item *sessionItem) {
	if s.length == nil {
		return
	}
	s.length <- Length{
		Username: item.username,
		Start:    item.create,
		Duration: item.update.Sub(item.create),
	}
}

func (s *MemorySessionList) HasKey(key string) (username string, err error) {
	s.Lock()
	defer s.Unlock()

	item, found := s.list[key]
	if !found {
		return "", nil
	}
	item.update = time.Now()

	return item.username, nil
}
func (s *MemorySessionList) Insert(username string) (key string, err error) {
	s.Lock()
	defer s.Unlock()

	b := make([]byte, s.keyLength)
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
func (s *MemorySessionList) Delete(username string) (err error) {
	s.Lock()
	defer s.Unlock()

	keys := []string{}
	for k, item := range s.list {
		if item.username == username {
			s.sendLength(item)
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		delete(s.list, k)
	}

	return nil
}
func (s *MemorySessionList) DeleteKey(key string) (err error) {
	s.Lock()
	defer s.Unlock()

	if item, found := s.list[key]; found {
		s.sendLength(item)
		delete(s.list, key)
	}

	return nil
}
func (s *MemorySessionList) ExpireBefore(update time.Time, create time.Time) (err error) {
	s.Lock()
	defer s.Unlock()

	keys := []string{}
	for k, item := range s.list {
		if item.update.Before(update) || item.create.Before(create) {
			s.sendLength(item)
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		delete(s.list, k)
	}

	return nil
}
func (s *MemorySessionList) Close() error {
	return nil
}
