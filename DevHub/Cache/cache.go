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
	"fmt"
	"github.com/flynn/json5"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"../OutsideInterface"
	"../RFModel"
)

type Cache struct {
	rf          *RFModel.RFModel
	out         OutsideInterface.Interface
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
	DeviceName   string
	UnitName     string
	FunctionName string
	Readable     bool
	Writeable    bool
}

type DeviceState struct {
	State State
}

func Init(self *Cache, rf *RFModel.RFModel, output OutsideInterface.Interface, devicesFile string) {
	self.log = logrus.New()
	self.log.Formatter = new(logrus.TextFormatter)
	self.log.Level = logrus.TraceLevel
	self.log.Out = os.Stdout
	self.rf = rf
	self.out = output
	self.deviceCache = make(map[DeviceKey]*DeviceState)
	self.cache = make(map[Key]*Value)
	// now read the devices file and register the devices functions enlisted in it
	jsonData, err := ioutil.ReadFile(devicesFile)
	if nil != err {
		panic(fmt.Errorf("Cache.Init: ioutil.ReadFile: %v; ", err.Error()))
	}
	var data map[string]interface{}
	err = json5.Unmarshal(jsonData, &data)
	if nil != err {
		panic(fmt.Errorf("Cache.Init: json5.Unmarshal: %v; ", err.Error()))
	}
	self.registerItems(data)
	// and now run goroutine to periodically update cache values
	go self.updateLoop()
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
	c.cache[key].LastRequest = time.Now()
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
			LastRequest:  time.Now(),
			AccessPeriod: time.Second,
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

func (c *Cache) registerItems(data map[string]interface{}) {
	for deviceName, deviceInterface := range data {
		device := deviceInterface.(map[string]interface{})
		units := device["units"].(map[string]interface{})
		for unitName, unitInterface := range units {
			unit := unitInterface.(map[string]interface{})
			functions := unit["functions"].(map[string]interface{})
			for functionName, functionInterface := range functions {
				function := functionInterface.(map[string]interface{})
				uid := RFModel.UID{
					Address: RFModel.ParseAddress(device["address"].(string)),
					Unit:    byte(unit["address"].(float64)),
				}
				key := Key{UID: uid, FNo: RFModel.FuncNo(byte(function["function"].(float64)))}
				if function["read"].(bool) {
					c.registerJsonItem(key, function, deviceName, unitName, functionName)
					c.cache[key].Readable = true
				}
				if function["write"].(bool) {
					key.FNo += 1
					c.registerJsonItem(key, function, deviceName, unitName, functionName)
					c.cache[key].Writeable = true
					go func(channel <-chan OutsideInterface.SubMessage) {
						for m := range channel {
							c.writeRequest(key, m.Value)
						}
					}(c.out.RegisterWritableComponent(c.outputKey(key)))
				}
			}
		}
	}
}

func (c *Cache) registerJsonItem(key Key, function map[string]interface{}, deviceName string, unitName string, functionName string) {
	c.RegisterItem(key.UID, key.FNo)
	if apInterface, ok := function["access period"]; ok {
		c.cache[key].AccessPeriod = time.Duration(apInterface.(float64) * float64(time.Second))
	}
	c.cache[key].DeviceName = deviceName
	c.cache[key].UnitName = unitName
	c.cache[key].FunctionName = functionName
}

func (c *Cache) outputKey(key Key) string {
	unitAddress := fmt.Sprintf("%X", key.UID.Unit)
	if 2 > len(unitAddress) {
		unitAddress = "0" + unitAddress
	}
	return RFModel.AddressToString(key.UID.Address) + ":" + unitAddress + "|" + fmt.Sprintf("%X", key.FNo)
}
