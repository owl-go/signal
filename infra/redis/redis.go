package redis

import (
	"time"

	"log"

	db "github.com/go-redis/redis"
)

// Config Redis配置对象
type Config struct {
	Addrs []string
	Pwd   string
	DB    int
}

// Redis Redis对象
type Redis struct {
	cluster     *db.ClusterClient
	single      *db.Client
	clusterMode bool
}

// NewRedis 创建Redis对象
func NewRedis(c Config) *Redis {
	if len(c.Addrs) == 0 {
		return nil
	}

	r := &Redis{}
	if len(c.Addrs) == 1 {
		r.single = db.NewClient(
			&db.Options{
				Addr:         c.Addrs[0], // use default Addr
				Password:     c.Pwd,      // no password set
				DB:           c.DB,       // use default DB
				DialTimeout:  3 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			})
		if err := r.single.Ping().Err(); err != nil {
			log.Printf(err.Error())
			return nil
		}
		r.clusterMode = false
		return r
	}

	// 集群对象赋值
	r.cluster = db.NewClusterClient(
		&db.ClusterOptions{
			Addrs:        c.Addrs,
			Password:     c.Pwd,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		})
	if err := r.cluster.Ping().Err(); err != nil {
		log.Printf(err.Error())
	}

	r.clusterMode = true
	return r
}

// Keys redis查找所有符合给定模式的所有key
func (r *Redis) Keys(k string) []string {
	if r.clusterMode {
		return r.cluster.Keys(k).Val()
	}
	return r.single.Keys(k).Val()
}

// Del redis删除指定key的所有数据
func (r *Redis) Del(k string) error {
	if r.clusterMode {
		return r.cluster.Del(k).Err()
	}
	return r.single.Del(k).Err()
}

// Expire redis设置key过期时间
func (r *Redis) Expire(k string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.Expire(k, t).Err()
	}
	return r.single.Expire(k, t).Err()
}

// Set redis以字符串方式存储key值
func (r *Redis) Set(k, v string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.Set(k, v, t).Err()
	}
	return r.single.Set(k, v, t).Err()
}

// Get redis以字符串方式存储,获取key值
func (r *Redis) Get(k string) string {
	if r.clusterMode {
		return r.cluster.Get(k).Val()
	}
	return r.single.Get(k).Val()
}

// SetNx 不存在则写入
func (r *Redis) SetNx(k, v string, t time.Duration) bool {
	if r.clusterMode {
		return r.cluster.SetNX(k, v, t).Val()
	}
	return r.single.SetNX(k, v, t).Val()
}

// LPop
func (r *Redis) LPop(k string) string {
	if r.clusterMode {
		return r.cluster.LPop(k).Val()
	}
	return r.single.LPop(k).Val()
}

// RPush
func (r *Redis) RPush(k string, v ...interface{}) error {
	if r.clusterMode {
		return r.cluster.RPush(k, v).Err()
	}
	return r.single.RPush(k, v).Err()
}

// LLen
func (r *Redis) LLen(k string) int64 {
	if r.clusterMode {
		return r.cluster.LLen(k).Val()
	}
	return r.single.LLen(k).Val()
}

// HSet redis以hash散列表方式存储key的field字段的值
func (r *Redis) HSet(k, field string, value interface{}) error {
	if r.clusterMode {
		return r.cluster.HSet(k, field, value).Err()
	}
	return r.single.HSet(k, field, value).Err()
}

// HSet redis以hash散列表方式存储key的field字段的值
func (r *Redis) HMSet(k string, fields map[string]interface{}) error {
	if r.clusterMode {
		return r.cluster.HMSet(k, fields).Err()
	}
	return r.single.HMSet(k, fields).Err()
}

// HGet redis读取hash散列表key的field字段的值
func (r *Redis) HGet(k, field string) string {
	if r.clusterMode {
		return r.cluster.HGet(k, field).Val()
	}
	return r.single.HGet(k, field).Val()
}

// HDel redis删除hash散列表key的field字段
func (r *Redis) HDel(k, field string) error {
	if r.clusterMode {
		return r.cluster.HDel(k, field).Err()
	}
	return r.single.HDel(k, field).Err()
}

// HGetAll redis读取hash散列表key值对应的全部字段数据
func (r *Redis) HGetAll(k string) map[string]string {
	if r.clusterMode {
		return r.cluster.HGetAll(k).Val()
	}
	return r.single.HGetAll(k).Val()
}
