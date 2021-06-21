package Cache

import (
	"../RFModel"
	"fmt"
	"time"
)

func (c *Cache) updateLoop() {
	for {
		time.Sleep(time.Millisecond * 100)
		c.updateRoutine()
	}
}

// updateRoutine is a single cycle, which is synchronously:
// write all pending values
// update access frequency
// update all read values according to their access frequency
func (c *Cache) updateRoutine() {
	// update device states first by pinging unit 0 function 0
	for key := range c.deviceCache {
		if c.probeDevice(key) {
			c.deviceCache[key].State = SOnline
		} else {
			c.deviceCache[key].State = SOffline
		}
	}
	// and then perform update cycle
	for key, value := range c.cache {
		if SOnline == c.deviceCache[DeviceKey(key.UID.Address)].State {
			if value.Writeable && WSPending == value.WriteState {
				c.performWrite(key)
			}
			c.updateAccessPeriod(key)
			if value.Readable && time.Now().After(value.LastUpdate.Add(value.AccessPeriod)) {
				c.performRead(key)
			}
		}
	}
}

// writeRequest is entrypoint for writing values from outside interface
func (c *Cache) writeRequest(key Key, value string) {
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()
	c.cache[key].WriteValue = value
	c.cache[key].WriteState = WSPending
}

// performWrite is a routine to send write command to rf interface and update cache state
// for the update routine
func (c *Cache) performWrite(key Key) {
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()
	defer func() {
		if r := recover(); r != nil {
			switch r.(RFModel.Error).Type {
			case RFModel.EDeviceTimeout:
				c.deviceCache[DeviceKey(key.UID.Address)].State = SOffline
			default:
				c.deviceCache[DeviceKey(key.UID.Address)].State = SError
			}
			// we should generate event here
		}
	}()
	c.rf.WriteFunction(key.UID, key.FNo, c.cache[key].WriteValue)
	c.cache[key].WriteState = WSWritten
}

// performRead is a routine to send read command to rf interface, update cache values
// and send updates to outside interface
// for the update routine
func (c *Cache) performRead(key Key) {
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()
	defer func() {
		c.out.UpdateComponent(c.outputKey(key), c.cache[key].ReadValue)
	}()
	// this defer will run before unlock
	defer func() {
		if r := recover(); r != nil {
			// todo what would it do if it can't convert error into RFModel.Error?
			switch r.(RFModel.Error).Type {
			case RFModel.EBadCode:
				switch r.(RFModel.Error).Code {
				case RFModel.ERCBadUnitId, RFModel.ERCBadFunctionId:
					//panic(fmt.Errorf("Cache.performRead: incorrect mapping, return code is: %v; ", r.(RFModel.Error).Code))
					c.cache[key].ReadValue = fmt.Sprintf("Cache.performRead: incorrect mapping, return code is: %X; ", r.(RFModel.Error).Code)
				default:
					c.cache[key].ReadValue = fmt.Sprintf("Cache.performRead: return code is: %X; ", r.(RFModel.Error).Code)
				}
			default:
				c.cache[key].ReadValue = fmt.Sprintf("Cache.performRead: error type is: %v; ", r.(RFModel.Error).Type)
			}
		}
	}()
	value := c.rf.ReadFunction(key.UID, key.FNo)
	c.cache[key].ReadValue = fmt.Sprintf("%v", value)
	// todo move that into error handler
	c.cache[key].LastUpdate = time.Now()
}

func (c *Cache) updateAccessPeriod(key Key) {
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()
	// todo: implement update access period based on request frequency (LastRequest is being updated in GetCached)
}

func (c *Cache) probeDevice(key DeviceKey) (isOnline bool) {
	defer func() {
		if r := recover(); r != nil {
			isOnline = false
		}
	}()
	// it tries hard enough to conclude that if it failed, the device must be offline
	// that function of unit 0 should never fail if device is online
	// tons of noise will cause devices to be offline too, but can we do anything about that?
	// todo: consider obtain metrics here instead
	c.rf.CallFunction(RFModel.UID{Address: RFModel.DeviceAddress(key), Unit: 0}, RFModel.FGetListOfUnitFunctions, []byte{})
	return true
}
