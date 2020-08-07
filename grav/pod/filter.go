package pod

import (
	"sync"

	"github.com/suborbital/grav/grav/message"
)

// messageFilter is a series of maps that associate things about a message (its UUID, type, etc) with a boolean value to say if
// It should be allowed or filtered out. For each of the maps, if an entry is included, the value of the boolean is respected (true = allow, false = deny)
// The an entry is not included in the map, it is defaulted to allow.
type messageFilter struct {
	UUIDMap map[string]bool

	sync.RWMutex
}

func newMessageFilter() *messageFilter {
	mf := &messageFilter{
		UUIDMap: map[string]bool{},
		RWMutex: sync.RWMutex{},
	}

	return mf
}

func (mf *messageFilter) allow(msg message.Message) bool {
	mf.RLock()
	defer mf.RUnlock()

	allow, exists := mf.UUIDMap[msg.UUID()]
	if !exists {
		return true
	}

	return allow
}

// FilterUUID likely should not be used in normal cases, it adds a message UUID to the pod's filter.
func (mf *messageFilter) FilterUUID(uuid string, allow bool) {
	mf.Lock()
	defer mf.Unlock()

	mf.UUIDMap[uuid] = allow
}
