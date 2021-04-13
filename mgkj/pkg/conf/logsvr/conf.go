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
	// Log 日志级别设置
	Log = &cfg.Log
	// Etcd Etcd设置
	Etcd = &cfg.Etcd
	//日志收集服
	LogSvr = &cfg.LogSvr
	Es     = &cfg.Es
	// nats-server 消息中间件设置
	Nats  = &cfg.Nats
	Mysql = &cfg.Mysql
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

type log struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"maxsize"`
	MaxAge     int    `mapstructure:"maxage"`
	Maxbackups int    `mapstructure:"maxbackups"`
	Encoder    string `mapstructure:"encoder"`
}

type elasticsearch struct {
	Url string `mapstructure:"url"`
}

type etcd struct {
	Addrs string `mapstructure:"addrs"`
}

type logsvr struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type nats struct {
	URL     string `mapstructure:"url"`
	NatsLog string `mapstructure:"natslog"`
}
type mysql struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

type config struct {
	Global  global        `mapstructure:"global"`
	Log     log           `mapstructure:"log"`
	Es      elasticsearch `mapstructure:"elasticsearch"`
	Etcd    etcd          `mapstructure:"etcd"`
	LogSvr  logsvr        `mapstructure:"logsvr"`
	Nats    nats          `mapstructure:"nats"`
	Mysql   mysql         `mapstructure:"mysql"`
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
