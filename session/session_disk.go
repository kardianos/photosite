package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

type diskSessionItem struct {
	username string
	update   time.Time
	create   time.Time
}

var diskCantOpenBucket = errors.New("Can't open bucket")

func (i *diskSessionItem) String() string {
	return fmt.Sprintf("User: %s, Update: %s, Create: %s", i.username, i.update, i.create)
}

type DiskSessionList struct {
	keyLength int
	length    chan<- Length

	db *bolt.DB

	sync.Mutex
	updates map[string]time.Time
}

var (
	diskBucketName = []byte("session")
)

func diskEncode(item *diskSessionItem) ([]byte, error) {
	update, err := item.update.MarshalBinary()
	if err != nil {
		return nil, err
	}
	create, err := item.create.MarshalBinary()
	if err != nil {
		return nil, err
	}
	name := []byte(item.username)
	length := len(update) + len(create) + len(name)
	b := make([]byte, length)

	n := copy(b, update)
	n += copy(b[n:], create)
	copy(b[n:], name)

	return b, nil
}
func diskDecode(b []byte, item *diskSessionItem) (err error) {
	err = (&item.update).UnmarshalBinary(b[:15])
	if err != nil {
		return err
	}
	err = (&item.create).UnmarshalBinary(b[15:30])
	if err != nil {
		return err
	}
	item.username = string(b[30:])
	return nil
}

func NewDiskSessionList(persistPath string, keyLength int, length chan<- Length) (Session, error) {
	db, err := bolt.Open(persistPath, 0600)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(diskBucketName)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &DiskSessionList{
		keyLength: keyLength,
		length:    length,
		db:        db,
		updates:   make(map[string]time.Time, 10),
	}, nil
}
func (s *DiskSessionList) sendLength(item *diskSessionItem) {
	if s.length == nil {
		return
	}
	s.length <- Length{
		Username: item.username,
		Start:    item.create,
		Duration: item.update.Sub(item.create),
	}
}

func (s *DiskSessionList) HasKey(key string) (username string, err error) {
	bkey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	tx, err := s.db.Begin(false)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(diskBucketName)
	if bucket == nil {
		return "", diskCantOpenBucket
	}
	v := bucket.Get(bkey)
	if v == nil {
		return "", nil
	}
	item := &diskSessionItem{}
	err = diskDecode(v, item)
	if err != nil {
		return "", err
	}

	// Update in memeory to batch up for later.
	s.Lock()
	s.updates[key] = time.Now()
	s.Unlock()

	return item.username, nil
}
func (s *DiskSessionList) Insert(username string) (skey string, err error) {
	tx, err := s.db.Begin(true)
	if err != nil {
		return "", err
	}
	defer tx.Commit()

	bucket := tx.Bucket(diskBucketName)
	if bucket == nil {
		return "", nil
	}

	key := make([]byte, s.keyLength)
	_, err = rand.Read(key)
	if err != nil {
		return "", err
	}

	now := time.Now()

	skey = base64.StdEncoding.EncodeToString(key)
	item := &diskSessionItem{
		username: username,
		update:   now,
		create:   now,
	}
	v, err := diskEncode(item)
	if err != nil {
		return "", err
	}
	err = bucket.Put(key, v)
	if err != nil {
		return "", err
	}

	return skey, nil
}
func (s *DiskSessionList) Delete(username string) (err error) {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Commit()

	bucket := tx.Bucket(diskBucketName)
	if bucket == nil {
		return err
	}
	item := &diskSessionItem{}
	keys := [][]byte{}
	err = bucket.ForEach(func(k, v []byte) error {
		err = diskDecode(v, item)
		if err != nil {
			return err
		}
		if item.username == username {
			s.sendLength(item)
			keys = append(keys, k)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, k := range keys {
		err = bucket.Delete(k)
		if err != nil {
			return err
		}
	}

	return nil
}
func (s *DiskSessionList) DeleteKey(key string) (err error) {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Commit()

	bucket := tx.Bucket(diskBucketName)
	if bucket == nil {
		return err
	}
	bkey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	bb := bucket.Get(bkey)
	if bb != nil {
		item := &diskSessionItem{}
		err = diskDecode(bb, item)
		if err == nil {
			s.sendLength(item)
		}
	}

	bucket.Delete(bkey)

	return nil
}
func (s *DiskSessionList) ExpireBefore(update time.Time, create time.Time) (err error) {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Commit()

	bucket := tx.Bucket(diskBucketName)
	if bucket == nil {
		return err
	}
	item := &diskSessionItem{}

	// Update the update time before the expire time takes effect.
	err = func() error {
		s.Lock()
		defer s.Unlock()
		for skey, update := range s.updates {
			// Ignore error, already checked before.
			bkey, err := base64.StdEncoding.DecodeString(skey)
			if err != nil {
				return err
			}
			v := bucket.Get(bkey)
			if v == nil {
				continue
			}
			err = diskDecode(v, item)
			if err != nil {
				return err
			}
			item.update = update
			v, err = diskEncode(item)
			if err != nil {
				return err
			}
			err = bucket.Put(bkey, v)
			if err != nil {
				return err
			}
		}
		return nil
	}()
	if err != nil {
		return err
	}

	keys := [][]byte{}
	err = bucket.ForEach(func(k, v []byte) error {
		err = diskDecode(v, item)
		if err != nil {
			return err
		}
		if item.update.Before(update) || item.create.Before(create) {
			s.sendLength(item)
			keys = append(keys, k)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, k := range keys {
		err = bucket.Delete(k)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *DiskSessionList) Close() error {
	return s.db.Close()
}
