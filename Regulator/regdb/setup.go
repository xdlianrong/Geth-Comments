package regdb

import (
	"github.com/go-redis/redis"
)

func Setup(dataport string, passwd string, database int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:" + dataport, // use default Addr
		Password: passwd,                  // no password set
		DB:       database,                // use default DB
	})
	_, err := client.Ping().Result()
	return client, err
}
