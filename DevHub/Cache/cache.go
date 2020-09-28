// there are three main commands to the cache module:
// read, write and command
// first two are states
// commands are events, so we need a queue for them instead of a single value
//
// First component needs to be registered so cache will pull its value at the next update cycle
// Reading is just returns last value of the component
// Reading affects update frequency: more often reads increase frequency, no reads decrease frequency
// Writing is just store required value to a cache, it will be written at the next update cycle
//
package Cache

import (
	"sync"
	"time"

	"../RFModel"
)

type State byte

const (
	SOffline State = 0
	SOnline        = 1
)

type WriteState byte

const (
	WSUninitialized WriteState = 0
	WSPending                  = 1
	WSWritten                  = 2
	WSFailed                   = 3
)

type Key RFModel.UnitFunctionKey
type DeviceKey RFModel.DeviceKey

// Value — either read value, or write value. One item per function
type Value struct {
	ReadValue       string
	LastUpdate      time.Time // when it was updated from the unit last time
	LastRequest     time.Time // when it was read from cache last time
	AccessFrequency time.Time
	WriteValue      string
	WriteState      WriteState
	lock            sync.Mutex
}

type DeviceState struct {
	State State
}

var cache = map[Key]*Value{}
var deviceCache = map[DeviceKey]*DeviceState{}

func ensureDeviceInCache(key DeviceKey) {
	_, ok := deviceCache[key]
	if !ok {
		value := DeviceState{State: SOffline}
		deviceCache[key] = &value
	}
}

func ensureKeyInCache(key Key, isRead bool) {
	ensureDeviceInCache(DeviceKey(key.UID.Address))
	_, ok := cache[key]
	if ok {
		if isRead {
			// check if it is actual
			// update read access time
			cache[key].LastRequest = time.Now()
		}
	} else {
		value := Value{
			LastRequest: time.Now(),
		}
		cache[key] = &value
	}
}

// RegisterItem put requested uid/fno pair for read update routine
func RegisterItem(uid RFModel.UID, fno RFModel.FuncNo) {
	key := Key{UID: uid, FNo: fno}
	ensureKeyInCache(key, true)
}

// GetCached return data immediately, no async operations before answer
// should never panic
// in case device is offline function will return default value ("")
func GetCached(uid RFModel.UID, fno RFModel.FuncNo) (value string, state State, timestamp time.Time) {
	key := Key{UID: uid, FNo: fno}
	ensureKeyInCache(key, true)
	cache[key].lock.Lock()
	defer cache[key].lock.Unlock()
	value = cache[key].ReadValue
	state = deviceCache[DeviceKey(key.UID.Address)].State
	if SOffline == state {
		value = ""
	}
	return value, state, cache[key].LastUpdate
}

// SetCached return immediately
func SetCached(uid RFModel.UID, fno RFModel.FuncNo, value string) {
	key := Key{UID: uid, FNo: fno}
	ensureKeyInCache(key, false)
	cache[key].lock.Lock()
	defer cache[key].lock.Unlock()
	cache[key].WriteValue = value
	cache[key].WriteState = WSPending
}
