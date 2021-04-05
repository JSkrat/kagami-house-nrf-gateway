// Redis translates components to redis database
// we're using database 0, key is stringified UID+FNo, value is plain value for now, no json yet
package Redis

import (
	"context"
	"github.com/go-redis/redis"
)

type Interface struct {
	db *redis.Client
	ctx context.Context
}

func Init(self *Interface, address string) {
	self.db = redis.NewClient(&redis.Options{Addr: address})
	self.ctx = context.Background()
}

func (i *Interface) UpdateComponent(key string, value string) {
	i.db.Set(i.ctx, key, value, 0)
}
