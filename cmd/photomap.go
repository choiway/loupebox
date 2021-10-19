package cmd

import (
	"sync"
)

// PhotoMap is a struct for the photos hash map.
// The abstraction on the map is required because of locks that occur
// when checking for existing Photos on one channel while inserting a
// photo on another channel.
type PhotoMap struct {
	sync.RWMutex
	photos map[string]Photo
}

func (pm *PhotoMap) Insert(p Photo) {
	pm.Lock()
	defer pm.Unlock()
	pm.photos[p.ShaHash] = p
}

func (pm *PhotoMap) Get(sha string) (Photo, bool) {
	pm.RLock()
	defer pm.RUnlock()
	if p, ok := pm.photos[sha]; ok {
		return p, true
	}
	return Photo{}, false
}
