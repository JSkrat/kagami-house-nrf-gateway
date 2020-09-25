package Cache

import (
	"time"

	"../RFModel"
)

type State byte

const (
	SOffline State = 0
	SOnline        = 1
)

type Key RFModel.UnitFunctionKey

// CacheValue — either read value, or write value. One item per function
type CacheValue struct {
	Value           string
	State           State
	LastUpdate      time.Time
	LastRequest     time.Time
	AccessFrequency time.Time
}

var cache = map[Key]*CacheValue{}

func ensureKeyInCache(key Key) {
	_, ok := cache[key]
	if ok {
		// check if it is actual
		cache[key].LastRequest = time.Now()
	} else {
		value := CacheValue{
			State:       SOffline,
			LastRequest: time.Now(),
		}
		cache[key] = &value
	}
}

func RegisterItem(uid RFModel.UID, fno RFModel.FuncNo) {
	key := Key{UID: uid, FNo: fno}
	ensureKeyInCache(key)
}

// GetCached return data immediately, no async operations before answer
// should never panic
func GetCached(uid RFModel.UID, fno RFModel.FuncNo) (value string, state State, timestamp time.Time) {
	key := Key{UID: uid, FNo: fno}
	ensureKeyInCache(key)
	value = cache[key].Value
	state = cache[key].State
	if SOffline == state {
		value = ""
	}
	return value, state, cache[key].LastUpdate
}
