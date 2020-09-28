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
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"

	"../RFModel"
)

type Cache struct {
	rf          *RFModel.RFModel
	log         *logrus.Logger
	cache       map[Key]*Value
	deviceCache map[DeviceKey]*DeviceState
}

type State byte

const (
	SOffline State = 0
	SOnline        = 1
	SError         = 2 // should be treated the same way as offline, but it is when slave firmware behaves incorrectly
)

type WriteState byte

const (
	WSUninitialized WriteState = 0
	WSPending                  = 1
	WSWritten                  = 2
	WSFailed                   = 3
)

type Key RFModel.UnitFunctionKey
type DeviceKey RFModel.DeviceAddress

// Value — either read value, or write value. One item per function
type Value struct {
	ReadValue    string
	LastUpdate   time.Time     // when it was updated from the unit last time
	LastRequest  time.Time     // when it was read from cache last time
	AccessPeriod time.Duration // update if LastUpdate + AccessPeriod is bigger than time.Now()
	WriteValue   string
	WriteState   WriteState
	lock         sync.Mutex
}

type DeviceState struct {
	State State
}

func Init(self *Cache, rf *RFModel.RFModel) {
	self.log = logrus.New()
	self.log.Formatter = new(logrus.TextFormatter)
	self.log.Level = logrus.TraceLevel
	self.log.Out = os.Stdout
	self.rf = rf
}

// RegisterItem put requested uid/fno pair for read update routine
func (c *Cache) RegisterItem(uid RFModel.UID, fno RFModel.FuncNo) {
	key := Key{UID: uid, FNo: fno}
	c.ensureKeyExists(key, true)
}

// GetCached return data immediately, no async operations before answer
// should never panic
// in case device is offline function will return default value ("")
func (c *Cache) GetCached(uid RFModel.UID, fno RFModel.FuncNo) (value string, state State, timestamp time.Time) {
	key := Key{UID: uid, FNo: fno}
	c.ensureKeyExists(key, true)
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()
	value = c.cache[key].ReadValue
	state = c.deviceCache[DeviceKey(key.UID.Address)].State
	if SOnline != state {
		value = ""
	}
	return value, state, c.cache[key].LastUpdate
}

// SetCached return immediately
func (c *Cache) SetCached(uid RFModel.UID, fno RFModel.FuncNo, value string) {
	key := Key{UID: uid, FNo: fno}
	c.ensureKeyExists(key, false)
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()
	c.cache[key].WriteValue = value
	c.cache[key].WriteState = WSPending
}

func (c *Cache) ensureKeyExists(key Key, isRead bool) {
	c.ensureDeviceExists(DeviceKey(key.UID.Address))
	_, ok := c.cache[key]
	if ok {
		if isRead {
			// check if it is actual
			// update read access time
			c.cache[key].LastRequest = time.Now()
		}
	} else {
		value := Value{
			LastRequest: time.Now(),
		}
		c.cache[key] = &value
	}
}

func (c *Cache) ensureDeviceExists(key DeviceKey) {
	_, ok := c.deviceCache[key]
	if !ok {
		value := DeviceState{State: SOffline}
		c.deviceCache[key] = &value
	}
}
