package regdb

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"log"
)

type Identity struct {
	Name    string
	ID      string
	ExtInfo string //新增个备注信息
}

func Setup(dataport string, passwd string, database int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:" + dataport, // use default Addr
		Password: passwd,                  // no password set
		DB:       database,                // use default DB
	})
	_, err := client.Ping().Result()
	return client, err
}

func Set(regDb *redis.Client, key string, value *Identity) {
	//有效期为0表示不设置有效期，非0表示经过该时间后键值对失效
	var valueM []byte
	valueM, _ = json.Marshal(value)
	result, err := regDb.Set(key, valueM, 0).Result()

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}

func Get(regDb *redis.Client, key string) string {
	result, err := regDb.Get(key).Result()

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	// raw 为反序列化后的Identity结构体
	//raw := new(Identity)
	//err = json.Unmarshal([]byte(result),&raw)
	//fmt.Println(raw)
	return result
}

func Exists(regDb *redis.Client, key string) bool {
	//返回1表示存在，0表示不存在
	isExists, err := regDb.Exists(key).Result()

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(isExists)

	if isExists == 1 {
		return true
	}
	return false
}

func Del(regDb *redis.Client, key string) {
	result, err := regDb.Del(key).Result()

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
