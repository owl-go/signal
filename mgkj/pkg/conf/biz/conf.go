package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	cfg    = config{}
	Global = &cfg.Global
	WebRTC = &cfg.WebRTC
	Rtp    = &cfg.Rtp
	Log    = &cfg.Log
	Etcd   = &cfg.Etcd
	Signal = &cfg.Signal
	Amqp   = &cfg.Amqp
)

func init() {
	if !cfg.parse() {
		showHelp()
		os.Exit(-1)
	}
}

type global struct {
	Addr  string `mapstructure:"addr"`
	Pprof string `mapstructure:"pprof"`
	// TestIP []string `mapstructure:"testip"`
}

type log struct {
	Level string `mapstructure:"level"`
}

type etcd struct {
	Addrs []string `mapstructure:"addrs"`
}

type signal struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type amqp struct {
	URL string `mapstructure:"url"`
}

type webrtc struct {
	ICE []string `mapstructure:"ice"`
}

type rtp struct {
	Port int `mapstructure:"port"`
}

type config struct {
	Global  global `mapstructure:"global"`
	WebRTC  webrtc `mapstructure:"webrtc"`
	Rtp     rtp    `mapstructure:"rtp"`
	Log     log    `mapstructure:"log"`
	Etcd    etcd   `mapstructure:"etcd"`
	Signal  signal `mapstructure:"signal"`
	Amqp    amqp   `mapstructure:"amqp"`
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
