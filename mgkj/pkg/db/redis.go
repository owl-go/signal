package db

import (
	"context"
	"fmt"
	"time"

	"mgkj/pkg/log"

	"github.com/go-redis/redis/v7"
)

// Config Redis配置对象
type Config struct {
	Addrs []string
	Pwd   string
	DB    int
}

// Redis Redis对象
type Redis struct {
	cluster     *redis.ClusterClient
	single      *redis.Client
	clusterMode bool
}

// NewRedis 创建Redis对象
func NewRedis(c Config) *Redis {
	if len(c.Addrs) == 0 {
		return nil
	}

	r := &Redis{}
	if len(c.Addrs) == 1 {
		r.single = redis.NewClient(
			&redis.Options{
				Addr:         c.Addrs[0], // use default Addr
				Password:     c.Pwd,      // no password set
				DB:           c.DB,       // use default DB
				DialTimeout:  3 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			})
		if err := r.single.Ping().Err(); err != nil {
			log.Errorf(err.Error())
			return nil
		}
		r.clusterMode = false
		return r
	}

	// 集群对象赋值
	r.cluster = redis.NewClusterClient(
		&redis.ClusterOptions{
			Addrs:        c.Addrs,
			Password:     c.Pwd,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		})
	if err := r.cluster.Ping().Err(); err != nil {
		log.Errorf(err.Error())
	}

	r.clusterMode = true
	return r
}

// Set redis以key-value方式存储
func (r *Redis) Set(k, v string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.Set(k, v, t).Err()
	}
	return r.single.Set(k, v, t).Err()
}

// Get redis获取key的值
func (r *Redis) Get(k string) string {
	if r.clusterMode {
		return r.cluster.Get(k).Val()
	}
	return r.single.Get(k).Val()
}

// Del redis删除指定key的值
func (r *Redis) Del(k string) error {
	if r.clusterMode {
		return r.cluster.Del(k).Err()
	}
	return r.single.Del(k).Err()
}

// HSet redis以hash散列表方式存储
func (r *Redis) HSet(k, field string, value interface{}) error {
	if r.clusterMode {
		return r.cluster.HSet(k, field, value).Err()
	}
	return r.single.HSet(k, field, value).Err()
}

// HGet redis读取hash散列表key值
func (r *Redis) HGet(k, field string) string {
	if r.clusterMode {
		return r.cluster.HGet(k, field).Val()
	}
	return r.single.HGet(k, field).Val()
}

// HGetAll redis读取全部hash散列表key值
func (r *Redis) HGetAll(k string) map[string]string {
	if r.clusterMode {
		return r.cluster.HGetAll(k).Val()
	}
	return r.single.HGetAll(k).Val()
}

// HDel redis删除hash散列表key值
func (r *Redis) HDel(k, field string) error {
	if r.clusterMode {
		return r.cluster.HDel(k, field).Err()
	}
	return r.single.HDel(k, field).Err()
}

// Expire redis设置key过期时间
func (r *Redis) Expire(k string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.Expire(k, t).Err()
	}
	return r.single.Expire(k, t).Err()
}

// Keys redis查找所有符合给定模式的key
func (r *Redis) Keys(k string) []string {
	if r.clusterMode {
		return r.cluster.Keys(k).Val()
	}
	return r.single.Keys(k).Val()
}

// HSetTTL redis设置给定key的值和剩余生存时间
func (r *Redis) HSetTTL(k, field string, value interface{}, t time.Duration) error {
	if r.clusterMode {
		if err := r.cluster.HSet(k, field, value).Err(); err != nil {
			return err
		}
		return r.cluster.Expire(k, t).Err()
	}
	if err := r.single.HSet(k, field, value).Err(); err != nil {
		return err
	}
	return r.single.Expire(k, t).Err()
}

// Watch http://redisdoc.com/topic/notification.html
func (r *Redis) Watch(ctx context.Context, key string) <-chan interface{} {
	var pubsub *redis.PubSub
	if r.clusterMode {
		pubsub = r.cluster.PSubscribe(fmt.Sprintf("__key*__:%s", key))
	} else {
		pubsub = r.single.PSubscribe(fmt.Sprintf("__key*__:%s", key))
	}

	res := make(chan interface{})
	go func() {
		for {
			select {
			case msg := <-pubsub.Channel():
				op := msg.Payload
				log.Infof("key => %s, op => %s", key, op)
				res <- op
			case <-ctx.Done():
				pubsub.Close()
				close(res)
				return
			}
		}
	}()
	return res
}
