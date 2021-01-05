package discovery

import (
	"strconv"
	"time"

	"mgkj/pkg/log"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var (
	nodeIP   string
	nodePort int
	etcdBase string
	etcdRoom string
	etcdRtp  string
	etcdNode string
	etcd     *Etcd
	quit     chan struct{}
)

// init 初始化包
func init() {
	quit = make(chan struct{})
}

// Init 初始化对象
func Init(etcds []string) {
	var err error
	etcd, err = newEtcd(etcds)
	if err != nil {
		panic(err)
	}
}

// UpdateLoad 测试功能，暂时没用
func UpdateLoad(ip string, port int) {
	nodeIP = ip
	nodePort = port
	etcdBase = "ion://"
	etcdRoom = etcdBase + "room"
	etcdRtp = etcdBase + "rtp"
	etcdNode = etcdBase + "node/" + nodeIP + ":" + strconv.Itoa(nodePort)

	go func() {
		etcd.keep(etcdNode, "")
		for {
			select {
			case <-quit:
				return
			case <-time.After(time.Second):
				etcd.update(etcdNode, getScore())
			}
		}
	}()
}

// getScore 获取服务器信息
func getScore() string {
	var score float64
	p, err := cpu.Percent(time.Second, false)
	if len(p) != 1 {
		log.Errorf("cpu.Percent err => %v", err)
		return "0"
	}

	cpuScore := 100 - p[0]

	v, _ := mem.VirtualMemory()
	memScore := 100 - v.UsedPercent

	// test net by etcd
	var netScore float64
	baseTime := time.Now()
	_, err = etcd.get("netscore")
	costTime := time.Since(baseTime).Nanoseconds() / 1e6

	if err != nil {
		netScore = 0
	} else if costTime < 300 {
		netScore = float64(300-costTime) / 300 * 100
	} else if costTime >= 300 && costTime <= 1000 {
		netScore = float64(1000-costTime) / float64(1000) * 50
	} else {
		netScore = 0
	}

	score = cpuScore*0.4 + memScore*0.2 + netScore*0.4
	if cpuScore < 10 || memScore < 10 || netScore < 10 {
		score = 0
	}
	return strconv.Itoa(int(score))
}

// Close 退出
func Close() {
	close(quit)
}

// Keep 写入key-value并保存key，保活
func Keep(key, val string) {
	log.Infof("discovery.Keep etcd=%v", etcd)
	if etcd != nil {
		etcd.keep(key, val)
	}
}

// Watch 监控对应的key改变
func Watch(key string, watchFunc WatchCallback, prefix bool) {
	if etcd != nil {
		etcd.watch(key, watchFunc, prefix)
	}
}

// Del etcd删除key
func Del(key string, prefix bool) {
	if etcd != nil {
		etcd.del(key, prefix)
	}
}

// GetByPrefix etcd查询key对应的value
func GetByPrefix(key string) map[string]string {
	if etcd == nil {
		return nil
	}
	m, err := etcd.getByPrefix(key)
	if err != nil {
		return nil
	}
	return m
}
