package sfu

import (
	"encoding/json"
	"fmt"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc"
	"mgkj/pkg/rtc/transport"
	"mgkj/pkg/util"
	"strings"

	sdptransform "github.com/notedit/sdp"
	"github.com/pion/webrtc/v2"
)

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCMsgs() {
	rpcMsgs, err := amqp.ConsumeRPC()
	if err != nil {
		log.Errorf(err.Error())
		return
	}

	go func() {
		defer util.Recover("sfu.handleRPCMsgs")
		for rpcm := range rpcMsgs {
			var msg map[string]interface{}
			err := json.Unmarshal(rpcm.Body, &msg)
			if err != nil {
				log.Errorf("sfu handleRPCMsgs Unmarshal err = %s", err.Error())
			}

			from := rpcm.ReplyTo
			corrID := rpcm.CorrelationId
			log.Infof("sfu.handleRPCMsgs msg=%v", msg)

			resp := util.Val(msg, "method")
			if resp != "" {
				switch resp {
				case proto.BizToSfuPublish:
					publish(msg, from, corrID)
				case proto.BizToSfuUnPublish:
					unpublish(msg, from, corrID)
				case proto.BizToSfuSubscribe:
					subscribe(msg, from, corrID)
				case proto.BizToSfuUnSubscribe:
					unsubscribe(msg, from, corrID)
				case proto.BizToSfuTrickleICE:
					trickle(msg, from, corrID)
				default:
					log.Warnf("sfu.handleRPCMsgResp invalid protocol corrID=%s, from=%s, resp=%s msg=%v", corrID, from, resp, msg)
				}
			}
		}
	}()
}

/*
	"method", proto.BizToSfuPublish, "rid", rid, "uid", uid, "minfo", minfo, "jsep", jsep
*/
// publish 处理发布流
func publish(msg map[string]interface{}, from, corrID string) {
	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		amqp.RPCCall(from, util.Map("method", proto.SfuToBizPublish, "errorCode", 401), corrID)
		return
	}
	sdp := util.Val(jsep, "sdp")
	rid := util.Val(msg, "rid")
	uid := util.Val(msg, "uid")
	mid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	rtcOptions := make(map[string]interface{})
	rtcOptions["transport-cc"] = "false"
	rtcOptions["publish"] = "true"

	options := msg["minfo"]
	if options != nil {
		options, ok := msg["minfo"].(map[string]interface{})
		if ok {
			rtcOptions["codec"] = options["codec"]
			rtcOptions["audio"] = options["audio"]
			rtcOptions["video"] = options["video"]
			rtcOptions["screen"] = options["screen"]
			rtcOptions["bandwidth"] = options["bandwidth"]
		}
	}
	pub := transport.NewWebRTCTransport(mid, rtcOptions)
	if pub == nil {
		amqp.RPCCall(from, util.Map("method", proto.SfuToBizPublish, "errorCode", 402), corrID)
		return
	}

	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetOrNewRouter(key)
	answer, err := pub.Answer(offer, true)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		amqp.RPCCall(from, util.Map("method", proto.SfuToBizPublish, "errorCode", 403), corrID)
		return
	}

	router.AddPub(uid, pub)

	sdpObj, err := sdptransform.Parse(offer.SDP)
	if err != nil {
		log.Errorf("err=%v sdpObj=%v", err, sdpObj)
		amqp.RPCCall(from, util.Map("method", proto.SfuToBizPublish, "errorCode", 404), corrID)
		return
	}

	tracks := make(map[string][]proto.TrackInfo)
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
			tracks[stream.GetID()+" "+id] = infos
		}
	}
	log.Infof("publish tracks %v, answer = %v", tracks, answer)
	amqp.RPCCall(from, util.Map("method", proto.SfuToBizPublish, "errorCode", 0, "jsep", answer, "mid", mid, "tracks", tracks), corrID)
}

/*
	"method", proto.BizToSfuUnPublish, "rid", rid, "uid", uid, "mid", mid
*/
// unpublish 处理取消发布流
func unpublish(msg map[string]interface{}, from, corrID string) {
	rid := util.Val(msg, "rid")
	uid := util.Val(msg, "uid")
	mid := util.Val(msg, "mid")

	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetOrNewRouter(key)
	if router != nil {
		router.Close()
		rtc.DelRouter(mid)
	}
}

/*
	"method", proto.BizToSfuSubscribe, "rid", rid, "uid", uid, "mid", mid, "tracks", rsp["tracks"], "jsep", jsep
*/
// subscribe 处理订阅流
func subscribe(msg map[string]interface{}, from, corrID string) {
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	uid := proto.GetUIDFromMID(mid)
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetOrNewRouter(key)
	if router == nil {
		amqp.RPCCall(from, util.Map("method", proto.SfuToBizSubscribe, "errorCode", 411), corrID)
		return
	}

	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		amqp.RPCCall(from, util.Map("method", proto.SfuToBizSubscribe, "errorCode", 412), corrID)
		return
	}

	sdp := util.Val(jsep, "sdp")

	rtcOptions := make(map[string]interface{})
	rtcOptions["transport-cc"] = "false"
	rtcOptions["subscribe"] = "true"

	subID := fmt.Sprintf("%s#%s", uid, util.RandStr(6))

	tracksMap := msg["tracks"].(map[string]interface{})
	log.Infof("subscribe tracks=%v", tracksMap)
	ssrcPT := make(map[uint32]uint8)
	rtcOptions["ssrcpt"] = ssrcPT
	sub := transport.NewWebRTCTransport(subID, rtcOptions)
	if sub == nil {
		amqp.RPCCall(from, util.Map("method", proto.SfuToBizSubscribe, "errorCode", 413), corrID)
		return
	}

	tracks := make(map[string]proto.TrackInfo)
	for msid, track := range tracksMap {
		for _, item := range track.([]interface{}) {
			info := item.(map[string]interface{})
			trackInfo := proto.TrackInfo{
				ID:      info["id"].(string),
				Type:    info["type"].(string),
				Ssrc:    int(info["ssrc"].(float64)),
				Payload: int(info["pt"].(float64)),
				Codec:   info["codec"].(string),
				Fmtp:    info["fmtp"].(string),
			}
			ssrcPT[uint32(trackInfo.Ssrc)] = uint8(trackInfo.Payload)
			tracks[msid] = trackInfo
		}
	}

	for msid, track := range tracks {
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
		amqp.RPCCall(from, util.Map("method", proto.SfuToBizSubscribe, "errorCode", 414), corrID)
		return
	}
	router.AddSub(subID, sub)

	log.Infof("subscribe mid %s, answer = %v", subID, answer)
	amqp.RPCCall(from, util.Map("method", proto.SfuToBizSubscribe, "errorCode", 0, "jsep", answer, "mid", subID), corrID)
}

/*
	"method", proto.BizToSfuUnSubscribe, "rid", rid, "uid", uid, "mid", mid
*/
// unsubscribe 处理取消订阅流
func unsubscribe(msg map[string]interface{}, from, corrID string) {
	mid := util.Val(msg, "mid")
	rtc.MapRouter(func(id string, r *rtc.Router) {
		subs := r.GetSubs()
		for sid := range subs {
			if sid == mid {
				r.DelSub(mid)
				return
			}
		}
	})
}

/*
	"method", proto.BizToSfuTrickleICE, "rid", rid, "mid", mid, "sid", sid, "ice", ice, "ispub", ispub
*/
// trickle 处理ice数据
func trickle(msg map[string]interface{}, from, corrID string) {
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	sid := util.Val(msg, "sid")
	uid := proto.GetUIDFromMID(mid)
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetOrNewRouter(key)
	cand := msg["ice"].(string)

	if msg["ispub"].(bool) {
		t := router.GetPub()
		if t != nil {
			t.(*transport.WebRTCTransport).AddCandidate(cand)
		}
	} else {
		t := router.GetSub(sid)
		if t != nil {
			t.(*transport.WebRTCTransport).AddCandidate(cand)
		}
	}
}
