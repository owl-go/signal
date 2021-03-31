package transport

import (
	"errors"
	"io"
	"strings"
	"sync"

	"mgkj/pkg/log"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

const (
	maxChanSize = 100
	fmtp        = "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f"
)

var (
	// only support unified plan
	cfg = webrtc.Configuration{
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}

	setting webrtc.SettingEngine

	errChanClosed     = errors.New("channel closed")
	errInvalidTrack   = errors.New("track is nil")
	errInvalidPacket  = errors.New("packet is nil")
	errInvalidPC      = errors.New("pc is nil")
	errInvalidOptions = errors.New("invalid options")
)

// GetUpperString get upper string by key
func GetUpperString(m map[string]interface{}, k string) string {
	val, ok := m[k]
	if ok {
		str, ok := val.(string)
		if ok {
			return strings.ToUpper(str)
		}
	}
	return ""
}

// InitWebRTC 初始化webrtc
func InitWebRTC(iceServers []webrtc.ICEServer, icePortStart, icePortEnd uint16) error {
	var err error
	if icePortStart != 0 || icePortEnd != 0 {
		err = setting.SetEphemeralUDPPortRange(icePortStart, icePortEnd)
	}

	cfg.ICEServers = iceServers
	return err
}

// WebRTCTransport webrtc对象
type WebRTCTransport struct {
	id           string
	mediaEngine  webrtc.MediaEngine
	api          *webrtc.API
	pc           *webrtc.PeerConnection
	outTracks    map[uint32]*webrtc.Track
	outTrackLock sync.RWMutex
	inTracks     map[uint32]*webrtc.Track
	inTrackLock  sync.RWMutex
	writeErrCnt  int

	rtpCh  chan *rtp.Packet
	rtcpCh chan rtcp.Packet
	stop   bool
	alive  bool
	isPub  bool

	stopTrack [2]bool
	nIndex    int
	nCount    int
	bandwidth int
}

func (w *WebRTCTransport) init(options map[string]interface{}, bPub bool) error {
	rtcpfb := make([]webrtc.RTCPFeedback, 0)
	rtcpfb = append(rtcpfb, webrtc.RTCPFeedback{
		Type: webrtc.TypeRTCPFBGoogREMB,
	})
	rtcpfb = append(rtcpfb, webrtc.RTCPFeedback{
		Type: webrtc.TypeRTCPFBCCM,
	})
	rtcpfb = append(rtcpfb, webrtc.RTCPFeedback{
		Type: webrtc.TypeRTCPFBNACK,
	})
	rtcpfb = append(rtcpfb, webrtc.RTCPFeedback{
		Type: "nack pli",
	})

	w.mediaEngine = webrtc.MediaEngine{}
	w.mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	w.mediaEngine.RegisterCodec(webrtc.NewRTPH264CodecExt(127 /*webrtc.DefaultPayloadTypeH264*/, 90000, rtcpfb, fmtp))
	w.api = webrtc.NewAPI(webrtc.WithMediaEngine(w.mediaEngine), webrtc.WithSettingEngine(setting))
	return nil
}

// NewWebRTCTransport create a WebRTCTransport
// options:
//   "video" = webrtc.H264[default] webrtc.VP8  webrtc.VP9
//   "audio" = webrtc.Opus[default] webrtc.PCMA webrtc.PCMU webrtc.G722
//   "transport-cc"  = "true" or "false"[default]
//   "data-channel"  = "true" or "false"[default]
func NewWebRTCTransport(id string, options map[string]interface{}, bPub bool) *WebRTCTransport {
	w := &WebRTCTransport{
		id:        id,
		outTracks: make(map[uint32]*webrtc.Track),
		inTracks:  make(map[uint32]*webrtc.Track),
		rtpCh:     make(chan *rtp.Packet, maxChanSize),
		rtcpCh:    make(chan rtcp.Packet, maxChanSize),
		stopTrack: [2]bool{false, false},
		nIndex:    0,
		nCount:    0,
		bandwidth: 1000,
		stop:      false,
		alive:     true,
	}
	err := w.init(options, bPub)
	if err != nil {
		log.Errorf("NewWebRTCTransport init %v", err)
		return nil
	}

	w.pc, err = w.api.NewPeerConnection(cfg)
	if err != nil {
		log.Errorf("NewWebRTCTransport api.NewPeerConnection %v", err)
		return nil
	}

	_, err = w.pc.AddTransceiver(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		log.Errorf("w.pc.AddTransceiver video %v", err)
		return nil
	}

	_, err = w.pc.AddTransceiver(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		log.Errorf("w.pc.AddTransceiver audio %v", err)
		return nil
	}

	w.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		if connectionState == webrtc.ICEConnectionStateDisconnected {
			log.Errorf("webrtc ice disconnected")
			w.alive = false
		}
	})

	return w
}

// ID return id
func (w *WebRTCTransport) ID() string {
	return w.id
}

// Type return type of transport
func (w *WebRTCTransport) Type() int {
	return TypeWebRTCTransport
}

// Offer return a offer
func (w *WebRTCTransport) Offer() (webrtc.SessionDescription, error) {
	if w.pc == nil {
		return webrtc.SessionDescription{}, errInvalidPC
	}
	offer, err := w.pc.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	err = w.pc.SetLocalDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	return offer, nil
}

// AddTrack add track to pc
func (w *WebRTCTransport) AddTrack(ssrc uint32, pt uint8, streamID string, trackID string) (*webrtc.Track, error) {
	if w.pc == nil {
		return nil, errInvalidPC
	}
	track, err := w.pc.NewTrack(pt, ssrc, trackID, streamID)
	if err != nil {
		return nil, err
	}
	if _, err = w.pc.AddTrack(track); err != nil {
		return nil, err
	}

	w.outTrackLock.Lock()
	w.outTracks[ssrc] = track
	w.outTrackLock.Unlock()
	return track, nil
}

// AddCandidate add candidate to pc
func (w *WebRTCTransport) AddCandidate(candidate string) error {
	if w.pc == nil {
		return errInvalidPC
	}

	err := w.pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: string(candidate)})
	if err != nil {
		return err
	}
	return nil
}

// Answer answer to pub or sub
func (w *WebRTCTransport) Answer(offer webrtc.SessionDescription, bPub bool) (webrtc.SessionDescription, error) {
	w.isPub = bPub
	if w.isPub {
		// 推流，那么只负责接收流
		w.pc.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
			w.inTrackLock.Lock()
			w.inTracks[remoteTrack.SSRC()] = remoteTrack
			w.inTrackLock.Unlock()
			// 启动接收推流RTP包协程
			w.receiveInTrackRTP(remoteTrack)
		})
	} else {
		// 启动接收拉流rtcp包协程
		w.receiveOutTrackRTCP()
	}

	err := w.pc.SetRemoteDescription(offer)
	if err != nil {
		log.Errorf("pc.SetRemoteDescription %v", err)
		return webrtc.SessionDescription{}, err
	}

	answer, err := w.pc.CreateAnswer(nil)
	if err != nil {
		log.Errorf("pc.CreateAnswer answer=%v err=%v", answer, err)
		return webrtc.SessionDescription{}, err
	}

	err = w.pc.SetLocalDescription(answer)
	if err != nil {
		log.Errorf("pc.SetLocalDescription answer=%v err=%v", answer, err)
	}

	return answer, err
}

// receiveInTrackRTP receive all incoming tracks' rtp and sent to one channel
func (w *WebRTCTransport) receiveInTrackRTP(remoteTrack *webrtc.Track) {
	go func() {
		for {
			if w.stop {
				return
			}

			rtp, err := remoteTrack.ReadRTP()
			if err != nil {
				if err == io.EOF {
					if remoteTrack.PayloadType() == 111 {
						w.stopTrack[0] = true
					} else {
						w.stopTrack[1] = true
					}

					if w.stopTrack[0] && w.stopTrack[1] {
						log.Infof("track audio and video both, close rtpCh")
						close(w.rtpCh)
					}
					return
				}
				log.Errorf("ReadRTP err => %s", err.Error())
			}
			w.rtpCh <- rtp
		}
	}()
}

// receiveOutTrackRTCP 接收所有路的rtcp包
func (w *WebRTCTransport) receiveOutTrackRTCP() {
	w.nIndex = 0
	w.nCount = len(w.pc.GetSenders())
	for _, sender := range w.pc.GetSenders() {
		go w.receiveRTCP(sender)
	}
}

// receiveRTCP 接收一路rtcp包
func (w *WebRTCTransport) receiveRTCP(sender *webrtc.RTPSender) {
	for {
		if w.stop {
			return
		}

		pkts, err := sender.ReadRTCP()
		if err != nil {
			if err == io.EOF {
				w.nIndex++
				if w.nIndex == w.nCount {
					close(w.rtcpCh)
				}
				return
			}
			log.Errorf("ReadRTCP err => %v", err)
		}

		for _, pkt := range pkts {
			w.rtcpCh <- pkt
		}
	}
}

// ReadRTP read rtp packet
func (w *WebRTCTransport) ReadRTP() (*rtp.Packet, error) {
	rtp, ok := <-w.rtpCh
	if !ok {
		return nil, errChanClosed
	}
	return rtp, nil
}

// WriteRTP send rtp packet to outgoing tracks
func (w *WebRTCTransport) WriteRTP(pkt *rtp.Packet) error {
	if pkt == nil {
		return errInvalidPacket
	}
	w.outTrackLock.RLock()
	track := w.outTracks[pkt.SSRC]
	w.outTrackLock.RUnlock()

	if track == nil {
		log.Errorf("WebRTCTransport.WriteRTP track==nil pkt.SSRC=%d", pkt.SSRC)
		return errInvalidTrack
	}

	//log.Debugf("WebRTCTransport.WriteRTP pkt=%v", pkt)
	err := track.WriteRTP(pkt)
	if err != nil {
		log.Errorf(err.Error())
		w.writeErrCnt++
		return err
	}
	return nil
}

// WriteRTCP write rtcp packet to pc
func (w *WebRTCTransport) WriteRTCP(pkt rtcp.Packet) error {
	if w.pc == nil {
		return errInvalidPC
	}
	return w.pc.WriteRTCP([]rtcp.Packet{pkt})
}

// GetInTracks return incoming tracks
func (w *WebRTCTransport) GetInTracks() map[uint32]*webrtc.Track {
	w.inTrackLock.RLock()
	defer w.inTrackLock.RUnlock()
	return w.inTracks
}

// GetOutTracks return incoming tracks
func (w *WebRTCTransport) GetOutTracks() map[uint32]*webrtc.Track {
	w.outTrackLock.RLock()
	defer w.outTrackLock.RUnlock()
	return w.outTracks
}

// Close all
func (w *WebRTCTransport) Close() {
	if w.stop {
		return
	}
	w.pc.Close()
	w.stop = true
}

// WriteErrTotal return write error
func (w *WebRTCTransport) WriteErrTotal() int {
	return w.writeErrCnt
}

// WriteErrReset reset write error
func (w *WebRTCTransport) WriteErrReset() {
	w.writeErrCnt = 0
}

// GetRTCPChan return a rtcp channel
func (w *WebRTCTransport) GetRTCPChan() chan rtcp.Packet {
	return w.rtcpCh
}

// GetBandwidth return bandwidth
func (w *WebRTCTransport) GetBandwidth() int {
	return w.bandwidth
}
