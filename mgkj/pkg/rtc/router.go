package rtc

import (
	"strings"
	"sync"
	"time"

	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc/plugins"
	"mgkj/pkg/rtc/transport"
	"mgkj/pkg/util"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	sdptransform "github.com/notedit/sdp"
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
	tracks      map[string][]proto.TrackInfo
}

// NewRouter return a new Router
func NewRouter(id string) *Router {
	log.Infof("NewRouter id=%s", id)
	return &Router{
		subs:        make(map[string]transport.Transport),
		liveTime:    time.Now().Add(liveCycle),
		pluginChain: plugins.NewPluginChain(),
		tracks:      make(map[string][]proto.TrackInfo),
	}
}

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
func (r *Router) AddPub(id, sdp string, options map[string]interface{}) (map[string]interface{}, error) {
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	rtcOptions := make(map[string]interface{})
	rtcOptions["publish"] = true
	if options != nil {
		rtcOptions["codec"] = options["codec"]
		rtcOptions["audio"] = options["audio"]
		rtcOptions["video"] = options["video"]
		rtcOptions["screen"] = options["screen"]
	}
	pub := transport.NewWebRTCTransport(id, rtcOptions)
	if pub == nil {
		return util.Map("errorCode", 402), nil
	}

	answer, err := pub.Answer(offer, true)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return util.Map("errorCode", 403), nil
	}

	r.pub = pub
	r.pluginChain.AttachPub(pub)
	r.start()

	sdpObj, err := sdptransform.Parse(offer.SDP)
	if err != nil {
		log.Errorf("err=%v sdpObj=%v", err, sdpObj)
		return util.Map("errorCode", 404), nil
	}

	for _, stream := range sdpObj.GetStreams() {
		for id, track := range stream.GetTracks() {
			pt := int(0)
			codecType := ""
			media := sdpObj.GetMedia(track.GetMedia())
			codecs := media.GetCodecs()

			for payload, codec := range codecs {
				if track.GetMedia() == "audio" {
					codecType = strings.ToUpper(codec.GetCodec())
					if strings.ToUpper(codec.GetCodec()) == strings.ToUpper(webrtc.Opus) {
						pt = payload
						break
					}
				} else if track.GetMedia() == "video" {
					codecType = strings.ToUpper(codec.GetCodec())
					if codecType == webrtc.H264 || codecType == webrtc.VP8 || codecType == webrtc.VP9 {
						pt = payload
						if pt == 96 {
							codecType = webrtc.VP8
							break
						}
						//break
					}
				}
			}
			var infos []proto.TrackInfo
			infos = append(infos, proto.TrackInfo{Ssrc: int(track.GetSSRCS()[0]), Payload: pt, Type: track.GetMedia(), ID: id, Codec: codecType})
			r.tracks[stream.GetID()+" "+id] = infos
		}
	}
	log.Infof("publish tracks %v, answer = %v", r.tracks, answer)
	resp := util.Map("errorCode", 0, "jsep", answer, "mid", id)
	return resp, nil
}

// DelPub del pub
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

// GetPub get pub
func (r *Router) GetPub() transport.Transport {
	// log.Infof("Router.GetPub %v", r.pub)
	return r.pub
}

func MapRouter(fn func(id string, r *Router)) {
	routerLock.RLock()
	defer routerLock.RUnlock()
	for id, r := range routers {
		fn(id, r)
	}
}

// AddSub add a pub to router
func (r *Router) AddSub(id, sdp string, options map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rtcOptions := make(map[string]interface{})
	rtcOptions["publish"] = false
	if options != nil {
		rtcOptions["audio"] = options["audio"]
		rtcOptions["video"] = options["video"]
	}
	sub := transport.NewWebRTCTransport(id, rtcOptions)
	if sub == nil {
		return util.Map("errorCode", 402), nil
	}

	tracks1 := make(map[string]proto.TrackInfo)
	for msid, track := range r.tracks {
		for _, item := range track {
			tracks1[msid] = item
		}
	}

	for msid, track := range tracks1 {
		ssrc := uint32(track.Ssrc)
		pt := uint8(track.Payload)
		// I2AacsRLsZZriGapnvPKiKBcLi8rTrO1jOpq c84ded42-d2b0-4351-88d2-b7d240c33435
		//                streamID                        trackID
		streamID := strings.Split(msid, " ")[0]
		trackID := track.ID
		log.Infof("AddTrack: codec:%s, ssrc:%d, pt:%d, streamID %s, trackID %s", track.Codec, ssrc, pt, streamID, trackID)
		_, err := sub.AddTrack(ssrc, pt, streamID, track.ID)
		if err != nil {
			log.Errorf("err=%v", err)
		}
	}

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := sub.Answer(offer, false)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return util.Map("errorCode", 414), nil
	}

	//fix panic: assignment to entry in nil map
	if r.stop {
		return util.Map("errorCode", 415), nil
	}
	r.subLock.Lock()
	defer r.subLock.Unlock()
	r.subs[id] = sub
	log.Infof("Router.AddSub id=%s t=%p", id, sub)
	go func() {
		for {
			pkt := <-sub.GetRTCPChan()
			if r.stop {
				return
			}
			switch pkt.(type) {
			case *rtcp.PictureLossIndication:
				if r.GetPub() != nil {
					// Request a Key Frame
					log.Infof("Router.AddSub got pli: %+v", pkt)
					r.GetPub().WriteRTCP(pkt)
				}
			case *rtcp.TransportLayerNack:
				// log.Infof("Router.AddSub got nack: %+v", pkt)
				nack := pkt.(*rtcp.TransportLayerNack)
				for _, nackPair := range nack.Nacks {
					if !r.ReSendRTP(id, nack.MediaSSRC, nackPair.PacketID) {
						n := &rtcp.TransportLayerNack{
							//origin ssrc
							SenderSSRC: nack.SenderSSRC,
							MediaSSRC:  nack.MediaSSRC,
							Nacks:      []rtcp.NackPair{rtcp.NackPair{PacketID: nackPair.PacketID}},
						}
						if r.pub != nil {
							r.GetPub().WriteRTCP(n)
						}
					}
				}

			default:
			}
		}
	}()
	log.Infof("subscribe mid %s, answer = %v", id, answer)
	resp := util.Map("errorCode", 0, "jsep", answer, "mid", id)
	return resp, nil
}

// GetSub get a sub by id
func (r *Router) GetSub(id string) transport.Transport {
	r.subLock.Lock()
	defer r.subLock.Unlock()
	// log.Infof("Router.GetSub id=%s sub=%v", id, r.subs[id])
	return r.subs[id]
}

// GetSubs get all subs
func (r *Router) GetSubs() map[string]transport.Transport {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	// log.Infof("Router.GetSubs len=%v", len(r.subs))
	return r.subs
}

// HasNoneSub check if sub == 0
func (r *Router) HasNoneSub() bool {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	isNoSub := len(r.subs) == 0
	log.Infof("Router.HasNoneSub=%v", isNoSub)
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
	r.DelPub()
	r.stop = true
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
