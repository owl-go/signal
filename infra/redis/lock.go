package redis

import "time"

//Lock 加锁
func (r *Redis) Lock(lockKey string) bool {
	for {
		ok := r.SetNx(lockKey, "lock", 1000*time.Millisecond) //设置锁过期时间
		if ok {
			return ok //直到获得这个锁
		}
		time.Sleep(2 * time.Millisecond)
	}
}

//Unlock 解锁
func (r *Redis) Unlock(lockKey string) {
	r.Del(lockKey)
}
