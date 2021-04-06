package rtc

import (
	"errors"
	"strings"
	"sync"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc/plugins"
	"mgkj/pkg/rtc/transport"
	"mgkj/pkg/util"

	sdps "github.com/gearghost/sdp/transform"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

const (
	maxWriteErr = 100
	liveCycle   = 6 * time.Second
)

//                                      +--->sub
//                                      |
// pub--->pubCh-->pluginChain-->subCh---+--->sub
//                                      |
//                                      +--->sub
// Router is rtp router
type Router struct {
	pub         transport.Transport
	subs        map[string]transport.Transport
	subLock     sync.RWMutex
	stop        bool
	liveTime    time.Time
	pluginChain *plugins.PluginChain
	tracks      []proto.TrackInfo
}

// NewRouter 新建一个Router对象
func NewRouter(id string) *Router {
	log.Infof("NewRouter id=%s", id)
	return &Router{
		subs:        make(map[string]transport.Transport),
		liveTime:    time.Now().Add(liveCycle),
		pluginChain: plugins.NewPluginChain(),
		tracks:      make([]proto.TrackInfo, 0),
	}
}

// InitPlugins 新建一个插件
func (r *Router) InitPlugins(config plugins.Config) error {
	log.Infof("Router.InitPlugins config=%+v", config)
	if r.pluginChain != nil {
		return r.pluginChain.Init(config)
	}
	return nil
}

func (r *Router) start() {
	go func() {
		defer util.Recover("[Router.start]")
		for {
			if r.stop {
				return
			}

			var pkt *rtp.Packet
			var err error
			// get rtp from pluginChain or pub
			if r.pluginChain != nil && r.pluginChain.On() {
				pkt = r.pluginChain.ReadRTP()
			} else {
				pkt, err = r.pub.ReadRTP()
				if err != nil {
					log.Errorf("r.pub.ReadRTP err=%v", err)
					continue
				}
			}
			// log.Infof("pkt := <-r.subCh %v", pkt)
			if pkt == nil {
				continue
			}
			r.liveTime = time.Now().Add(liveCycle)
			// nonblock sending
			go func() {
				for _, t := range r.GetSubs() {
					if t == nil {
						log.Errorf("Transport is nil")
						continue
					}

					//log.Infof(" WriteRTP %v:%v to %v ", pkt.SSRC, pkt.SequenceNumber, t.ID())
					if err := t.WriteRTP(pkt); err != nil {
						log.Errorf("wt.WriteRTP err=%v", err)
						// del sub when err is increasing
						if t.WriteErrTotal() > maxWriteErr {
							r.DelSub(t.ID())
						}
					}
					t.WriteErrReset()
				}
			}()
		}
	}()
}

// AddPub add a pub transport
func (r *Router) AddPub(sdp, id, ip string, options map[string]interface{}) (string, error) {
	pub := transport.NewWebRTCTransport(id, options, true)
	if pub == nil {
		return "", errors.New("pub is not create")
	}

	tracks, err := sdpTotracks(sdp)
	if err != nil {
		return "", err
	}

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := pub.Answer(offer, true)
	if err != nil {
		return "", err
	}

	r.tracks = tracks
	r.pub = pub
	r.pluginChain.AttachPub(pub)
	r.start()
	return answer.SDP, nil
}

// GetPub 获取pub对象
func (r *Router) GetPub() transport.Transport {
	return r.pub
}

// DelPub 删除pub对象
func (r *Router) DelPub() {
	log.Infof("Router.DelPub %v", r.pub)
	if r.pub != nil {
		r.pub.Close()
	}
	if r.pluginChain != nil {
		r.pluginChain.Close()
	}
	r.pub = nil
}

// MapRouter 遍历router处理
func MapRouter(fn func(id string, r *Router)) {
	routerLock.RLock()
	defer routerLock.RUnlock()
	for id, r := range routers {
		fn(id, r)
	}
}

// AddSub add a pub to router
func (r *Router) AddSub(sdp, id, ip string, options map[string]interface{}) (string, error) {
	bAudioSub := options["audio"].(bool)
	bVideoSub := options["video"].(bool)
	// 创建拉流peer
	sub := transport.NewWebRTCTransport(id, options, false)
	if sub == nil {
		return "", errors.New("sub is not create")
	}

	addTracks(r.tracks, sub, bAudioSub, bVideoSub)
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := sub.Answer(offer, false)
	if err != nil {
		return "", err
	}

	r.subLock.Lock()
	r.subs[id] = sub
	r.subLock.Unlock()
	go r.DoRtcp(id, sub)
	return answer.SDP, nil
}

// DoRtcp ...
func (r *Router) DoRtcp(id string, sub *transport.WebRTCTransport) {
	for {
		pkt := <-sub.GetRTCPChan()
		if r.stop {
			return
		}
		switch pkt.(type) {
		case *rtcp.PictureLossIndication:
			if r.GetPub() != nil {
				log.Infof("Router.AddSub got pli: %+v", pkt)
				r.GetPub().WriteRTCP(pkt)
			}
		case *rtcp.TransportLayerNack:
			nack := pkt.(*rtcp.TransportLayerNack)
			for _, nackPair := range nack.Nacks {
				if !r.ReSendRTP(id, nack.MediaSSRC, nackPair.PacketID) {
					n := &rtcp.TransportLayerNack{
						SenderSSRC: nack.SenderSSRC,
						MediaSSRC:  nack.MediaSSRC,
						Nacks:      []rtcp.NackPair{{PacketID: nackPair.PacketID}},
					}
					if r.pub != nil {
						r.GetPub().WriteRTCP(n)
					}
				}
			}
		default:
		}
	}
}

// GetSub get a sub by id
func (r *Router) GetSub(id string) transport.Transport {
	r.subLock.Lock()
	defer r.subLock.Unlock()
	return r.subs[id]
}

// GetSubs get all subs
func (r *Router) GetSubs() map[string]transport.Transport {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	return r.subs
}

// HasNoneSub check if sub == 0
func (r *Router) HasNoneSub() bool {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	isNoSub := (len(r.subs) == 0)
	return isNoSub
}

// DelSub del sub by id
func (r *Router) DelSub(id string) {
	log.Infof("Router.DelSub id=%s", id)
	r.subLock.Lock()
	defer r.subLock.Unlock()
	if r.subs[id] != nil {
		r.subs[id].Close()
	}
	delete(r.subs, id)
}

// DelSubs del all sub
func (r *Router) DelSubs() {
	log.Infof("Router.DelSubs")
	r.subLock.Lock()
	defer r.subLock.Unlock()
	for _, sub := range r.subs {
		if sub != nil {
			sub.Close()
		}
	}
	r.subs = nil
}

// Close release all
func (r *Router) Close() {
	if r.stop {
		return
	}
	log.Infof("Router.Close")
	r.stop = true
	r.DelPub()
	r.DelSubs()
}

func (r *Router) ReSendRTP(sid string, ssrc uint32, sn uint16) bool {
	if r.pub == nil {
		return false
	}
	hd := r.pluginChain.GetPlugin(plugins.TypeJitterBuffer)
	if hd != nil {
		jb := hd.(*plugins.JitterBuffer)
		pkt := jb.GetPacket(ssrc, sn)
		if pkt == nil {
			// log.Infof("Router.ReSendRTP pkt not found sid=%s ssrc=%d sn=%d pkt=%v", sid, ssrc, sn, pkt)
			return false
		}
		sub := r.GetSub(sid)
		if sub != nil {
			err := sub.WriteRTP(pkt)
			if err != nil {
				log.Errorf("router.ReSendRTP err=%v", err)
			}
			// log.Infof("Router.ReSendRTP sid=%s ssrc=%d sn=%d", sid, ssrc, sn)
			return true
		}
	}
	return false
}

// Alive return router status
func (r *Router) Alive() bool {
	return !r.liveTime.Before(time.Now())
}

// sdp转换成tracks
func sdpTotracks(sdp string) ([]proto.TrackInfo, error) {
	sdpObj, err := sdps.Parse(sdp)
	if err != nil {
		return nil, errors.New("offer sdp is err")
	}

	var infos []proto.TrackInfo
	for _, media := range sdpObj.Media {
		for _, rtp := range media.Rtp {
			if media.Type == "audio" {
				if strings.ToUpper(rtp.Codec) == strings.ToUpper(webrtc.Opus) {
					track := proto.TrackInfo{}
					track.ID = media.Mid
					track.Codec = rtp.Codec
					track.Type = media.Type
					track.Payload = rtp.Payload
					// 查询fmtp数据
					for _, fmtp := range media.Fmtp {
						if fmtp.Payload == rtp.Payload {
							track.Fmtp = fmtp.Config
							break
						}
					}
					// 查询ssrc
					for _, ssrc := range media.Ssrcs {
						track.Ssrc = ssrc.Id
						break
					}
					// 增加到数组中
					infos = append(infos, track)
					break
				}
			}
			if media.Type == "video" {
				if strings.ToUpper(rtp.Codec) == strings.ToUpper(webrtc.H264) {
					for _, fmtp := range media.Fmtp {
						if fmtp.Payload == rtp.Payload {
							parameters := sdps.ParseParams(fmtp.Config)
							profile := parameters["profile-level-id"]
							if strings.HasPrefix(profile, "42") {
								track := proto.TrackInfo{}
								track.ID = media.Mid
								track.Codec = rtp.Codec
								track.Type = media.Type
								track.Payload = rtp.Payload
								track.Fmtp = fmtp.Config
								// 查询ssrc
								for _, ssrc := range media.Ssrcs {
									track.Ssrc = ssrc.Id
									break
								}
								// 增加到数组中
								infos = append(infos, track)
								break
							}
						}
					}
				}
			}
		}
	}
	return infos, nil
}

// 给peer增加track
func addTracks(tracks []proto.TrackInfo, sub *transport.WebRTCTransport, bAudioSub, bVideoSub bool) {
	for _, track := range tracks {
		if bAudioSub && track.Type == "audio" {
			_, err := sub.AddTrack(uint32(track.Ssrc), uint8(track.Payload), "go-sfu-audio", track.ID)
			if err != nil {
				log.Errorf("addTracks audio err = %v", err)
			}
		}
		if bVideoSub && track.Type == "video" {
			_, err := sub.AddTrack(uint32(track.Ssrc), uint8(track.Payload), "go-sfu-video", track.ID)
			if err != nil {
				log.Errorf("addTracks video err = %v", err)
			}
		}
	}
}
