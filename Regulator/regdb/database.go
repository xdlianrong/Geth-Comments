package regdb

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"regulator/utils"
	"regulator/utils/ElGamal"
)

type Identity struct {
	Name    string
	ID      string
	Hashky  string
	ExtInfo string //新增个备注信息
}

func ConnectToDB(dataport string, passwd string, database int) *redis.Client {
	Db, err := Setup(dataport, passwd, database)
	if err != nil {
		utils.Fatalf("Failed to connect to redis: %v", err)
	}
	return Db
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

func Set(regDb *redis.Client, key string, value interface{}) error {
	//有效期为0表示不设置有效期，非0表示经过该时间后键值对失效
	var valueM []byte
	valueM, _ = json.Marshal(value)
	_, err := regDb.Set(key, valueM, 0).Result()
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func Get(regDb *redis.Client, key string) interface{} {
	result, err := regDb.Get(key).Result()

	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(result)
	// raw 为反序列化后的Identity结构体
	switch key {
	case "key":
		{
			raw := new(ElGamal.PrivateKey)
			if err := json.Unmarshal([]byte(result), &raw); err != nil {
				utils.Fatalf("Failed to Unmarshal: %v", err)
			}
			return raw
		}
	default:
		{
			raw := new(Identity)
			if err := json.Unmarshal([]byte(result), &raw); err != nil {
				utils.Fatalf("Failed to Unmarshal: %v", err)
			}
			return raw
		}
	}
}

func Exists(regDb *redis.Client, key string) bool {
	//返回1表示存在，0表示不存在
	isExists, err := regDb.Exists(key).Result()
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(isExists)
	return isExists == 1
}

func Del(regDb *redis.Client, key string) {
	result, err := regDb.Del(key).Result()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
