package mediasoup

import (
	"mgkj/pkg/log"
	"sync"

	mediasoup "github.com/jiyeyuran/mediasoup-go"
)

// Publish 推流对象
type Publish struct {
	transport *mediasoup.WebRtcTransport
	audiopub  *mediasoup.Producer
	videopub  *mediasoup.Producer
	audioLive bool
	videoLive bool
	nAudio    int
	nVideo    int
}

// Subscribe 拉流对象
type Subscribe struct {
	transport *mediasoup.WebRtcTransport
	audiosub  *mediasoup.Consumer
	videosub  *mediasoup.Consumer
}

// Router 结构
type Router struct {
	router  *mediasoup.Router
	pub     Publish
	subs    map[string]Subscribe
	subLock sync.RWMutex
}

// NewRouter 新建一个对象
func NewRouter(work *mediasoup.Worker) *Router {
	var options = mediasoup.RouterOptions{}
	options.MediaCodecs = make([]*mediasoup.RtpCodecCapability, 0)
	options.MediaCodecs = append(options.MediaCodecs, NewCodecOpus())
	options.MediaCodecs = append(options.MediaCodecs, NewCodecVP8())
	router, err := work.CreateRouter(options)
	if err != nil {
		log.Errorf("CreateRouter err = %s", err.Error())
		return nil
	}

	routerObj := new(Router)
	routerObj.router = router
	routerObj.subs = make(map[string]Subscribe)
	return routerObj
}

// NewCodecOpus 新建opus的codec
func NewCodecOpus() *mediasoup.RtpCodecCapability {
	opus := new(mediasoup.RtpCodecCapability)
	opus.Kind = "audio"
	opus.MimeType = "audio/opus"
	opus.ClockRate = 48000
	opus.Channels = 2
	opus.PreferredPayloadType = 111
	opus.Parameters.Useinbandfec = 1
	opus.RtcpFeedback = make([]mediasoup.RtcpFeedback, 0)
	opus.RtcpFeedback = append(opus.RtcpFeedback, mediasoup.RtcpFeedback{Type: "transport-cc", Parameter: ""})
	return opus
}

// NewCodecH264 新建H264的codec
func NewCodecH264() *mediasoup.RtpCodecCapability {
	h264 := new(mediasoup.RtpCodecCapability)
	h264.Kind = "video"
	h264.MimeType = "video/H264"
	h264.ClockRate = 90000
	h264.Parameters.RtpParameter.PacketizationMode = 1
	h264.Parameters.RtpParameter.LevelAsymmetryAllowed = 1
	h264.Parameters.RtpParameter.ProfileLevelId = "42e01f"
	h264.RtcpFeedback = make([]mediasoup.RtcpFeedback, 0)
	h264.RtcpFeedback = append(h264.RtcpFeedback, mediasoup.RtcpFeedback{Type: "nack", Parameter: ""})
	h264.RtcpFeedback = append(h264.RtcpFeedback, mediasoup.RtcpFeedback{Type: "nack", Parameter: "pli"})
	h264.RtcpFeedback = append(h264.RtcpFeedback, mediasoup.RtcpFeedback{Type: "ccm", Parameter: "fir"})
	h264.RtcpFeedback = append(h264.RtcpFeedback, mediasoup.RtcpFeedback{Type: "goog-remb", Parameter: ""})
	h264.RtcpFeedback = append(h264.RtcpFeedback, mediasoup.RtcpFeedback{Type: "transport-cc", Parameter: ""})
	return h264
}

// NewCodecVP8 新建VP8的codec
func NewCodecVP8() *mediasoup.RtpCodecCapability {
	vp8 := new(mediasoup.RtpCodecCapability)
	vp8.Kind = "video"
	vp8.MimeType = "video/VP8"
	vp8.ClockRate = 90000
	vp8.PreferredPayloadType = 96
	vp8.RtcpFeedback = make([]mediasoup.RtcpFeedback, 0)
	vp8.RtcpFeedback = append(vp8.RtcpFeedback, mediasoup.RtcpFeedback{Type: "nack", Parameter: ""})
	vp8.RtcpFeedback = append(vp8.RtcpFeedback, mediasoup.RtcpFeedback{Type: "nack", Parameter: "pli"})
	vp8.RtcpFeedback = append(vp8.RtcpFeedback, mediasoup.RtcpFeedback{Type: "ccm", Parameter: "fir"})
	vp8.RtcpFeedback = append(vp8.RtcpFeedback, mediasoup.RtcpFeedback{Type: "goog-remb", Parameter: ""})
	vp8.RtcpFeedback = append(vp8.RtcpFeedback, mediasoup.RtcpFeedback{Type: "transport-cc", Parameter: ""})
	return vp8
}

// Alive 判断存活
func (r *Router) Alive() bool {
	if r.pub.transport != nil {
		dtls := r.pub.transport.DtlsState()
		if dtls == mediasoup.DtlsState_Failed || dtls == mediasoup.DtlsState_Closed {
			return false
		}
	}

	if r.pub.audiopub != nil {
		stats, err := r.pub.audiopub.GetStats()
		if err != nil {
			log.Errorf(err.Error())
		} else {
			for _, stat := range stats {
				if stat.Score > 0 {
					r.pub.audioLive = true
					nAudio = 0
				} else {
					nAudio = nAudio + 1
				}
			}
		}
		if nAudio == 3 {
			nAudio = 0
			r.pub.audioLive = false
		}
	} else {
		r.pub.audioLive = false
	}
	if r.pub.videopub != nil {
		stats, err := r.pub.videopub.GetStats()
		if err != nil {
			log.Errorf(err.Error())
		} else {
			for _, stat := range stats {
				if stat.Score > 0 {
					r.pub.videoLive = true
					nVideo = 0
				} else {
					nVideo = nVideo + 1
				}
			}
		}
		if nVideo == 3 {
			nVideo = 0
			r.pub.videoLive = false
		}
	} else {
		r.pub.videoLive = false
	}
	if !r.pub.audioLive && !r.pub.videoLive {
		return false
	}
	return true
}

// AddPub 增加发布流对象
func (r *Router) AddPub(sdp string, id, ip string, options map[string]interface{}) (string, error) {
	option := mediasoup.WebRtcTransportOptions{}
	option.ListenIps = make([]mediasoup.TransportListenIp, 0)
	mediasoupIP := mediasoup.TransportListenIp{}
	mediasoupIP.Ip = "0.0.0.0"
	mediasoupIP.AnnouncedIp = ip
	option.ListenIps = append(option.ListenIps, mediasoupIP)
	sendTransport, err := r.router.CreateWebRtcTransport(option)
	if err != nil {
		log.Errorf(err.Error())
		return "", err
	}

	zx := NewZX("send", sdp, r.router.RtpCapabilities())
	zx.Run(sendTransport.IceParameters(), sendTransport.IceCandidates(),
		sendTransport.DtlsParameters(), sendTransport.SctpParameters())

	dtls := zx.DtlsParameters()
	err = sendTransport.Connect(mediasoup.TransportConnectOptions{
		DtlsParameters: &dtls,
	})
	if err != nil {
		log.Errorf(err.Error())
		return "", err
	}

	r.pub.transport = sendTransport
	r.pub.nAudio = 0
	r.pub.nVideo = 0
	canKind := zx.CanKind()

	bAudioPub := options["audio"].(bool)
	bVideoPub := options["video"].(bool)
	if canKind["audio"] && bAudioPub {
		kind := "audio"
		rtpParameters := zx.GetRtpParameter(kind)
		producer, err := sendTransport.Produce(mediasoup.ProducerOptions{
			Kind:          mediasoup.MediaKind(kind),
			RtpParameters: *rtpParameters,
		})
		if err != nil {
			log.Errorf(err.Error())
			return "", err
		}
		r.pub.audiopub = producer
		r.pub.audioLive = true
	}
	if canKind["video"] && bVideoPub {
		kind := "video"
		rtpParameters := zx.GetRtpParameter(kind)
		producer, err := sendTransport.Produce(mediasoup.ProducerOptions{
			Kind:          mediasoup.MediaKind(kind),
			RtpParameters: *rtpParameters,
		})
		if err != nil {
			log.Errorf(err.Error())
			return "", err
		}
		r.pub.videopub = producer
		r.pub.videoLive = true
	}
	return zx.Sdp()
}

// DelPub 删除发布流
func (r *Router) DelPub() {
	if r.pub.audiopub != nil {
		r.pub.audiopub.Close()
		r.pub.audiopub = nil
	}
	if r.pub.videopub != nil {
		r.pub.videopub.Close()
		r.pub.videopub = nil
	}
	if r.pub.transport != nil {
		r.pub.transport.Close()
		r.pub.transport = nil
	}
}

// AddSub add a pub to router
func (r *Router) AddSub(sdp string, id, ip string, options map[string]interface{}) (string, error) {
	option := mediasoup.WebRtcTransportOptions{}
	option.ListenIps = make([]mediasoup.TransportListenIp, 0)
	mediasoupIP := mediasoup.TransportListenIp{}
	mediasoupIP.Ip = "0.0.0.0"
	mediasoupIP.AnnouncedIp = ip
	option.ListenIps = append(option.ListenIps, mediasoupIP)
	recvTransport, err := r.router.CreateWebRtcTransport(option)
	if err != nil {
		log.Errorf(err.Error())
		return "", err
	}

	zx := NewZX("recv", sdp, r.router.RtpCapabilities())
	zx.Run(recvTransport.IceParameters(), recvTransport.IceCandidates(),
		recvTransport.DtlsParameters(), recvTransport.SctpParameters())

	sub := Subscribe{}
	sub.transport = recvTransport

	canKind := zx.CanKind()
	bAudioSub := options["audio"].(bool)
	bVideoSub := options["video"].(bool)
	if canKind["audio"] && bAudioSub {
		kind := "audio"
		producer := r.pub.audiopub
		consumer, err := recvTransport.Consume(mediasoup.ConsumerOptions{
			ProducerId:      producer.Id(),
			RtpCapabilities: zx.rtpCapabilities,
			Paused:          true,
		})
		if err != nil {
			log.Errorf(err.Error())
			return "", err
		}
		zx.receive(consumer.Id(), kind, consumer.RtpParameters())
		consumer.Resume()
		sub.audiosub = consumer
	}
	if canKind["video"] && bVideoSub {
		kind := "video"
		producer := r.pub.videopub
		consumer, err := recvTransport.Consume(mediasoup.ConsumerOptions{
			ProducerId:      producer.Id(),
			RtpCapabilities: zx.rtpCapabilities,
			Paused:          true,
		})
		if err != nil {
			log.Errorf(err.Error())
			return "", err
		}
		zx.receive(consumer.Id(), kind, consumer.RtpParameters())
		consumer.Resume()
		sub.videosub = consumer
	}
	r.subs[id] = sub
	dtls := zx.DtlsParameters()
	recvTransport.Connect(mediasoup.TransportConnectOptions{
		DtlsParameters: &dtls,
	})
	return zx.Sdp()
}

// GetSubs get all subs
func (r *Router) GetSubs() map[string]Subscribe {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	return r.subs
}

// HasNoneSub check if sub == 0
func (r *Router) HasNoneSub() bool {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	isNoSub := len(r.subs) == 0
	return isNoSub
}

// DelSub del sub by id
func (r *Router) DelSub(id string) {
	r.subLock.Lock()
	defer r.subLock.Unlock()
	sub := r.subs[id]
	if sub.audiosub != nil {
		sub.audiosub.Close()
		sub.audiosub = nil
	}
	if sub.videosub != nil {
		sub.videosub.Close()
		sub.videosub = nil
	}
	if sub.transport != nil {
		sub.transport.Close()
		sub.transport = nil
	}
	delete(r.subs, id)
}

// DelSubs del all sub
func (r *Router) DelSubs() {
	r.subLock.Lock()
	defer r.subLock.Unlock()
	for _, sub := range r.subs {
		if sub.audiosub != nil {
			sub.audiosub.Close()
			sub.audiosub = nil
		}
		if sub.videosub != nil {
			sub.videosub.Close()
			sub.videosub = nil
		}
		if sub.transport != nil {
			sub.transport.Close()
			sub.transport = nil
		}
	}
	r.subs = nil
}

// Close release all
func (r *Router) Close() {
	r.DelPub()
	r.DelSubs()
	r.router.Close()
}

// MapRouter 找到指定的map对象并调用
func MapRouter(fn func(id string, r *Router)) {
	routerLock.RLock()
	defer routerLock.RUnlock()
	for id, r := range routers {
		fn(id, r)
	}
}
