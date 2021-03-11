module mgkj

go 1.15

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/cloudwebrtc/go-protoo v0.0.0-20200324091126-61fe57ffd18f
	github.com/cloudwebrtc/nats-protoo v0.0.0-20200328144814-d3c1c848d442
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/nats-io/nats-server/v2 v2.1.9 // indirect
	github.com/notedit/sdp v0.0.4
	github.com/pion/ion-log v1.0.0
	github.com/pion/rtcp v1.2.6
	github.com/pion/rtp v1.6.2
	github.com/pion/webrtc/v2 v2.2.26
	github.com/spf13/viper v1.7.1
	go.etcd.io/etcd v3.3.25+incompatible
	sigs.k8s.io/yaml v1.2.0 // indirect
)
