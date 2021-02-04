package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	cfg = config{}
	// Global 全局设置
	Global = &cfg.Global
	// Plugins 插件参数
	Plugins = &cfg.Plugins
	// WebRTC rtc参数
	WebRTC = &cfg.WebRTC
	// Log 日志级别设置
	Log = &cfg.Log
	// Etcd Etcd设置
	Etcd = &cfg.Etcd
	// Amqp 消息中间件设置
	Amqp = &cfg.Amqp
)

func init() {
	if !cfg.parse() {
		showHelp()
		os.Exit(-1)
	}
}

type global struct {
	Pprof string `mapstructure:"pprof"`
	Ndc   string `mapstructure:"dc"`
	Name  string `mapstructure:"name"`
	Nid   string `mapstructure:"nid"`
	Nip   string `mapstructure:"nip"`
}

type jitterBuffer struct {
	On            bool `mapstructure:"on"`
	REMBCycle     int  `mapstructure:"rembcycle"`
	PLICycle      int  `mapstructure:"plicycle"`
	MaxBandwidth  int  `mapstructure:"maxbandwidth"`
	MaxBufferTime int  `mapstructure:"maxbuffertime"`
}

type plugins struct {
	On           bool         `mapstructure:"on"`
	JitterBuffer jitterBuffer `mapstructure:"jitterbuffer"`
}

type log struct {
	Level string `mapstructure:"level"`
}

type etcd struct {
	Addrs []string `mapstructure:"addrs"`
}

type iceserver struct {
	URLs       []string `mapstructure:"urls"`
	Username   string   `mapstructure:"username"`
	Credential string   `mapstructure:"credential"`
}

type webrtc struct {
	ICEPortRange []uint16    `mapstructure:"portrange"`
	ICEServers   []iceserver `mapstructure:"iceserver"`
}

type amqp struct {
	URL string `mapstructure:"url"`
}

type config struct {
	Global  global  `mapstructure:"global"`
	Plugins plugins `mapstructure:"plugins"`
	WebRTC  webrtc  `mapstructure:"webrtc"`
	Log     log     `mapstructure:"log"`
	Etcd    etcd    `mapstructure:"etcd"`
	Amqp    amqp    `mapstructure:"amqp"`
	CfgFile string
}

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func (c *config) load() bool {
	_, err := os.Stat(c.CfgFile)
	if err != nil {
		return false
	}

	viper.SetConfigFile(c.CfgFile)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", c.CfgFile, err)
		return false
	}
	err = viper.GetViper().UnmarshalExact(c)
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", c.CfgFile, err)
		return false
	}

	if len(c.WebRTC.ICEPortRange) > 2 {
		fmt.Printf("config file %s loaded failed. range port must be [min,max]\n", c.CfgFile)
		return false
	}

	if len(c.WebRTC.ICEPortRange) != 0 && c.WebRTC.ICEPortRange[1]-c.WebRTC.ICEPortRange[0] <= 100 {
		fmt.Printf("config file %s loaded failed. range port must be [min, max] and max - min >= %d\n", c.CfgFile, 100)
		return false
	}

	fmt.Printf("config %s load ok!\n", c.CfgFile)
	return true
}

func (c *config) parse() bool {
	flag.StringVar(&c.CfgFile, "c", "conf/conf.toml", "config file")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !c.load() {
		return false
	}

	if *help {
		showHelp()
		return false
	}
	return true
}
