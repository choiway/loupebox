package cmd

import (
	"sync"
)

// Set is a struct for the set hash map which just tracks whether
// the index exists. The mutex abstraction is required.
type Set struct {
	sync.RWMutex
	set map[string]bool
}

func (pm *Set) Insert(s string) {
	pm.Lock()
	defer pm.Unlock()
	pm.set[s] = true
}

func (pm *Set) Check(s string) bool {
	pm.RLock()
	defer pm.RUnlock()
	if _, ok := pm.set[s]; ok {
		return true
	}
	return false
}
