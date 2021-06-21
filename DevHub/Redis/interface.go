// Redis translates components to redis database
// we're using database 0, key is stringified UID+FNo, value is plain value for now, no json yet
package Redis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"os"
	//"github.com/flynn/json5"
	"../OutsideInterface"
)

var log = logrus.New()

type Interface struct {
	db          *redis.Client
	ctx         context.Context
	databaseNum int
}

func Init(self *Interface, address string, db int) {
	log.Formatter = new(logrus.TextFormatter)
	log.Level = logrus.DebugLevel
	log.Out = os.Stdout
	self.db = redis.NewClient(&redis.Options{Addr: address})
	self.ctx = context.Background()
	self.databaseNum = db
}

func (i *Interface) UpdateComponent(key string, value string) {
	i.db.Set(i.ctx, key, value, 0)
}

func (i *Interface) RegisterWritableComponent(key string) <-chan OutsideInterface.SubMessage {
	redisChannel := fmt.Sprintf("__keyspace@%d__:%s", i.databaseNum, key)
	log.Debug(fmt.Sprintf("Redis.RegisterWritableComponent(%s): subscribing to channel %s", key, redisChannel))
	pubsub := i.db.Subscribe(i.ctx, redisChannel)
	// Wait for confirmation that subscription is created before publishing anything.
	_, err := pubsub.Receive(i.ctx)
	if err != nil {
		panic(err)
	}
	// Go channel which receives messages.
	ch := pubsub.Channel()
	// buffered since we might push initial data into it before returning it
	ret := make(chan OutsideInterface.SubMessage, 2)
	go func() {
		for message := range ch {
			log.Debug(fmt.Sprintf("Redis.RegisterWritableComponent(%s) goroutine: chan <%s>, payload <%s>, payload slice <%v>", key, message.Channel, message.Payload, message.PayloadSlice))
			if "set" == message.Payload {
				value, err := i.db.Get(i.ctx, key).Result()
				if nil != err {
					panic(err)
				}
				log.Debug(fmt.Sprintf("Redis.RegisterWritableComponent(%s) goroutine: value is <%s>", key, value))
				ret <- OutsideInterface.SubMessage{
					Value: value,
					Key:   key,
				}
			}
		}
	}()
	// make sure such key exists in redis, but if it does, preserve the value
	// we update the values in the device from it (so empty redis db will initialize all writable devices functions to 0)
	value, rerr := i.db.Get(i.ctx, key).Result()
	if nil != rerr {
		log.Debug(fmt.Sprintf("Redis.RegisterWritableComponent(%s): there was no initial value, creating", key))
		i.db.Set(i.ctx, key, "", 0)
	} else {
		log.Debug(fmt.Sprintf("Redis.RegisterWritableComponent(%s): initial value is <%s>", key, value))
		ret <- OutsideInterface.SubMessage{
			Value: value,
			Key:   key,
		}
	}
	return ret
}
