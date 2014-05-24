package main

import (
	"reflect"
	"testing"
	"time"
)

func TestDiskSession(t *testing.T) {
	itemA := &diskSessionItem{
		username: "Bob",
		update:   time.Now(),
		create:   time.Now().Add(time.Hour),
	}
	itemB := &diskSessionItem{}

	b, err := diskEncode(itemA)
	if err != nil {
		t.Errorf("Encode error: %v", err)
	}
	t.Logf("len(b): %d", len(b))
	err = diskDecode(b, itemB)
	if err != nil {
		t.Errorf("Decode error: %v", err)
	}

	if reflect.DeepEqual(itemA, itemB) == false {
		t.Errorf("Items not equal")
	}
}
