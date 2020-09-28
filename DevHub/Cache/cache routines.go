package Cache

import (
	"../RFModel"
	"time"
)

// updateRoutine is a single cycle, which is synchronously:
// write all pending values
// update access frequency
// update all read values according to their access frequency
func (c *Cache) updateRoutine() {
	// update device states first by pinging unit 0 function 0
	for key, _ := range c.deviceCache {
		if c.probeDevice(key) {
			c.deviceCache[key].State = SOnline
		} else {
			c.deviceCache[key].State = SOffline
		}
	}
	// and then perform update cycle
	for key, value := range c.cache {
		if SOnline == c.deviceCache[DeviceKey(key.UID.Address)].State {
			if WSPending == value.WriteState {
				c.performWrite(key)
			}
			c.updateAccessPeriod(key)
			if time.Now().After(value.LastUpdate.Add(value.AccessPeriod)) {
				c.performRead(key)
			}
		}
	}
}

func (c *Cache) performWrite(key Key) {
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()
	defer func() {
		if r := recover(); r != nil {
			c.deviceCache[DeviceKey(key.UID.Address)].State = SOffline
			// we should generate event here
		}
	}()
	c.rf.WriteFunction(key.UID, key.FNo, c.cache[key].WriteValue)
	c.cache[key].WriteState = WSWritten
}

func (c *Cache) performRead(key Key) {
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()

}

func (c *Cache) updateAccessPeriod(key Key) {
	c.cache[key].lock.Lock()
	defer c.cache[key].lock.Unlock()

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
	c.rf.CallFunction(RFModel.UID{Address: RFModel.DeviceAddress(key), Unit: 0}, RFModel.FGetListOfUnitFunctions, []byte{})
	return true
}
