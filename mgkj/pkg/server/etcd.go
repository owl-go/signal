package server

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"mgkj/pkg/log"

	"go.etcd.io/etcd/clientv3"
)

const (
	defaultDialTimeout      = time.Second * 5
	defaultGrantTimeout     = 5
	defaultOperationTimeout = time.Second * 5
)

// WatchCallback watch回调
type WatchCallback func(clientv3.WatchChan)

// Etcd etcd对象
type Etcd struct {
	client        *clientv3.Client            // etcd客户端
	liveKeyID     map[string]clientv3.LeaseID // 租约map
	liveKeyIDLock sync.RWMutex                // map租约锁
	stop          bool                        // 停止开关
}

// NewEtcd 创建etcd对象
func NewEtcd(endpoints []string) (*Etcd, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: defaultDialTimeout,
	})

	if err != nil {
		log.Errorf("newEtcd err=%v", err)
		return nil, err
	}

	return &Etcd{
		stop:      false,
		client:    cli,
		liveKeyID: make(map[string]clientv3.LeaseID),
	}, nil
}

// Keep 写入key-value并保活
func (e *Etcd) Keep(key, value string) error {
	resp, err := e.client.Grant(context.TODO(), defaultGrantTimeout)
	if err != nil {
		log.Errorf("Etcd Grant err = %s %v", key, err)
		return err
	}
	_, err = e.client.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Errorf("Etcd Put err = %s %v", key, err)
		return err
	}
	ch, err := e.client.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		log.Errorf("Etcd KeepAlive err = %s %v", key, err)
		return err
	}
	go func() {
		for {
			if e.stop {
				return
			}
			<-ch
		}
	}()
	// 加入map
	e.liveKeyIDLock.Lock()
	e.liveKeyID[key] = resp.ID
	e.liveKeyIDLock.Unlock()
	log.Infof("Etcd keep ok = %s %v", key, value)
	return nil
}

// Update 更新key-value
func (e *Etcd) Update(key, value string) error {
	e.liveKeyIDLock.Lock()
	id := e.liveKeyID[key]
	e.liveKeyIDLock.Unlock()
	_, err := e.client.Put(context.TODO(), key, value, clientv3.WithLease(id))
	if err != nil {
		// 出错就重新来keep
		err = e.Keep(key, value)
		if err != nil {
			log.Errorf("Etcd Update err = %s %s %v", key, value, err)
		}
	}
	return err
}

// Delete 删除key，prefix是否前缀
func (e *Etcd) Delete(key string, prefix bool) error {
	var err error
	if prefix {
		for k := range e.liveKeyID {
			if strings.HasPrefix(k, key) {
				delete(e.liveKeyID, k)
			}
		}
		_, err = e.client.Delete(context.TODO(), key, clientv3.WithPrefix())
	} else {
		e.liveKeyIDLock.Lock()
		delete(e.liveKeyID, key)
		e.liveKeyIDLock.Unlock()
		_, err = e.client.Delete(context.TODO(), key)
	}
	return err
}

// Watch 观察指定的key,有改变通过watchFunc回调告知
func (e *Etcd) Watch(key string, watchFunc WatchCallback, prefix bool) error {
	if watchFunc == nil {
		return errors.New("watchFunc is nil")
	}
	if prefix {
		watchFunc(e.client.Watch(context.Background(), key, clientv3.WithPrefix()))
	} else {
		watchFunc(e.client.Watch(context.Background(), key))
	}
	return nil
}

// Close 关闭对象
func (e *Etcd) Close() error {
	if e.stop {
		return errors.New("Etcd already close")
	}
	e.stop = true
	e.liveKeyIDLock.Lock()
	for k := range e.liveKeyID {
		delete(e.liveKeyID, k)
		e.client.Delete(context.TODO(), k)
	}
	e.liveKeyIDLock.Unlock()
	return e.client.Close()
}

// GetValue 获取指定key对应的值,key不带前缀
func (e *Etcd) GetValue(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key)
	if err != nil {
		cancel()
		return "", err
	}
	var val string
	for _, ev := range resp.Kvs {
		val = string(ev.Value)
	}
	cancel()
	return val, err
}

// GetByPrefix 获取指定前缀的key对应的值,map格式返回
func (e *Etcd) GetByPrefix(key string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		cancel()
		return nil, err
	}
	data := make(map[string]string)
	for _, kv := range resp.Kvs {
		data[string(kv.Key)] = string(kv.Value)
	}
	cancel()
	return data, err
}

// GetResponseByPrefix 获取指定前缀的key对应的值
func (e *Etcd) GetResponseByPrefix(key string) (*clientv3.GetResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		cancel()
		return nil, err
	}
	cancel()
	return resp, nil
}
