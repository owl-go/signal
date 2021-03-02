package sfu

import (
	"fmt"
	"mgkj/pkg/log"
	"mgkj/pkg/proto"
	"mgkj/pkg/rtc"
	"mgkj/pkg/rtc/transport"
	"mgkj/pkg/util"
	"strings"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	sdptransform "github.com/notedit/sdp"
	"github.com/pion/webrtc/v2"
)

// handleRPCMsgs 处理其他模块发送过来的消息
func handleRPCRequest(rpcID string) {

	log.Infof("handleRequest: rpcID => [%v]", rpcID)

	protoo.OnRequest(rpcID, func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {

		go func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {

			defer util.Recover("sfu.handleRPCRequest")

			log.Infof("sfu.handleRPCRequest recv request=%v", request)
			method := request["method"].(string)
			data := request["data"].(map[string]interface{})
			log.Infof("method => %s, data => %v", method, data)

			var result map[string]interface{}
			err := util.NewNpError(400, fmt.Sprintf("Unkown method [%s]", method))

			if method != "" {
				switch method {
				case proto.BizToSfuPublish:
					result, err = publish(data)
				case proto.BizToSfuUnPublish:
					result, err = unpublish(data)
				case proto.BizToSfuSubscribe:
					result, err = subscribe(data)
				case proto.BizToSfuUnSubscribe:
					result, err = unsubscribe(data)
				case proto.BizToSfuTrickleICE:
					result, err = trickle(data)
				default:
					log.Warnf("sfu.handleRPCRequest invalid protocol method=%s data=%v", method, data)
				}
			}
			if err != nil {
				reject(err.Code, err.Reason)
			} else {
				accept(result)
			}
		}(request, accept, reject)
	})
}

/*
	"method", proto.BizToSfuPublish, "rid", rid, "uid", uid, "minfo", minfo, "jsep", jsep
*/
// publish 处理发布流
func publish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		return util.Map("errorCode", 401), nil
	}
	sdp := util.Val(jsep, "sdp")
	rid := util.Val(msg, "rid")
	uid := util.Val(msg, "uid")
	mid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	rtcOptions := make(map[string]interface{})
	rtcOptions["publish"] = true
	options := msg["minfo"]
	if options != nil {
		options, ok := msg["minfo"].(map[string]interface{})
		if ok {
			rtcOptions["codec"] = options["codec"]
			rtcOptions["audio"] = options["audio"]
			rtcOptions["video"] = options["video"]
			rtcOptions["screen"] = options["screen"]
		}
	}
	pub := transport.NewWebRTCTransport(mid, rtcOptions)
	if pub == nil {
		return util.Map("errorCode", 402), nil
	}

	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetOrNewRouter(key)
	answer, err := pub.Answer(offer, true)
	if err != nil {
		log.Errorf("err=%v answer=%v", err, answer)
		return util.Map("errorCode", 403), nil
	}

	router.AddPub(uid, pub)

	sdpObj, err := sdptransform.Parse(offer.SDP)
	if err != nil {
		log.Errorf("err=%v sdpObj=%v", err, sdpObj)
		return util.Map("errorCode", 404), nil
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
	resp := util.Map("errorCode", 0, "jsep", answer, "mid", mid, "tracks", tracks)
	return resp, nil
}

/*
	"method", proto.BizToSfuUnPublish, "rid", rid, "uid", uid, "mid", mid
*/
// unpublish 处理取消发布流
func unpublish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(msg, "rid")
	uid := util.Val(msg, "uid")
	mid := util.Val(msg, "mid")

	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetOrNewRouter(key)
	if router != nil {
		router.Close()
		rtc.DelRouter(mid)
	}
	return util.Map(), nil
}

/*
	"method", proto.BizToSfuSubscribe, "rid", rid, "uid", uid, "mid", mid, "tracks", rsp["tracks"], "jsep", jsep
*/
// subscribe 处理订阅流
func subscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	log.Infof("sfu.subscribe msg => %v", msg)

	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	uid := proto.GetUIDFromMID(mid)
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetOrNewRouter(key)
	if router == nil {
		return util.Map("errorCode", 411), nil
	}

	jsep := msg["jsep"].(map[string]interface{})
	if jsep == nil {
		return util.Map("errorCode", 412), nil
	}

	sdp := util.Val(jsep, "sdp")

	rtcOptions := make(map[string]interface{})
	rtcOptions["publish"] = false

	subID := fmt.Sprintf("%s#%s", uid, util.RandStr(6))

	tracksMap := msg["tracks"].(map[string]interface{})
	log.Infof("subscribe tracks=%v", tracksMap)

	sub := transport.NewWebRTCTransport(subID, rtcOptions)
	if sub == nil {
		return util.Map("errorCode", 413), nil
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
		return util.Map("errorCode", 414), nil
	}
	router.AddSub(subID, sub)

	log.Infof("subscribe mid %s, answer = %v", subID, answer)
	resp := util.Map("errorCode", 0, "jsep", answer, "mid", subID)
	return resp, nil
}

/*
	"method", proto.BizToSfuUnSubscribe, "rid", rid, "uid", uid, "mid", mid
*/
// unsubscribe 处理取消订阅流
func unsubscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
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
	return util.Map(), nil
}

/*
	"method", proto.BizToSfuTrickleICE, "rid", rid, "mid", mid, "sid", sid, "ice", ice, "ispub", ispub
*/
// trickle 处理ice数据
func trickle(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
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
	return util.Map(), nil
}
