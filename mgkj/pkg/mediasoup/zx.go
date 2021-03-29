package mediasoup

import (
	"encoding/json"
	"fmt"
	"strings"

	mediasoup "github.com/jiyeyuran/mediasoup-go"
	"github.com/notedit/sdp/transform"
)

// ZX ...
type ZX struct {
	routerRtpCapabilities   mediasoup.RtpCapabilities
	nativeRtpCapabilities   mediasoup.RtpCapabilities
	extendedRtpCapabilities ExtendedRtpCapabilities
	rtpCapabilities         mediasoup.RtpCapabilities
	remoteSdp               *RemoteSdp
	sdpObject               *transform.SdpStruct
	dtlsParameters          mediasoup.DtlsParameters
	RtpParameters           map[string]*mediasoup.RtpParameters
	direction               string
	canKind                 []string
}

// Sdp ...
func (zx *ZX) Sdp() (string, error) {
	if zx.direction == "recv" {
		answer, err := zx.remoteSdp.GetSdp()
		if err != nil {
			return "", err
		}
		sdpObject, err := transform.Parse(answer)
		if err != nil {
			return "", err
		}
		for _, mediaObject := range sdpObject.Media {
			mediaObject.Direction = "sendonly"
			mediaObject.Setup = "passive"
		}
		return transform.Write(sdpObject)
	}
	return zx.remoteSdp.GetSdp()
}

// DtlsParameters ...
func (zx *ZX) DtlsParameters() mediasoup.DtlsParameters {
	return zx.dtlsParameters
}

// RtpCapabilities ...
func (zx *ZX) RtpCapabilities() map[string]*mediasoup.RtpParameters {
	return zx.RtpParameters
}

// CanKind ...
func (zx *ZX) CanKind() []string {
	return zx.canKind
}

// NewZX ...
func NewZX(direction string, offer string, routerRtpCapabilities mediasoup.RtpCapabilities) *ZX {
	zx := new(ZX)
	zx.direction = direction
	zx.RtpParameters = make(map[string]*mediasoup.RtpParameters)
	zx.routerRtpCapabilities = routerRtpCapabilities

	validateRtpCapabilities(&zx.routerRtpCapabilities)
	sdpobj, err := transform.Parse(offer)
	if err != nil {
		return nil
	}
	zx.sdpObject = sdpobj
	jsonStr, err := json.Marshal(sdpobj)
	if err != nil {
		fmt.Println("fuck")
	}
	fmt.Println("mono", string(jsonStr))

	zx.nativeRtpCapabilities = extractRtpCapabilities(*sdpobj)
	validateRtpCapabilities(&zx.nativeRtpCapabilities)
	zx.extendedRtpCapabilities = getExtendedRtpCapabilities(
		&zx.nativeRtpCapabilities,
		&zx.routerRtpCapabilities)
	for _, media := range zx.sdpObject.Media {
		zx.canKind = append(zx.canKind, media.Type)
	}
	zx.dtlsParameters = extractDtlsParameters(*sdpobj)
	if zx.direction == "send" {
		zx.dtlsParameters.Role = "server"
	} else if zx.direction == "recv" {
		for _, codec := range zx.extendedRtpCapabilities.Codecs {
			codec.RemotePayloadType = codec.LocalPayloadType
			codec.RemoteRtxPayloadType = codec.LocalRtxPayloadType
		}
		zx.rtpCapabilities = getRecvRtpCapabilities(zx.extendedRtpCapabilities)
		zx.dtlsParameters.Role = "client"
	}
	return zx
}

// Run ...
func (zx *ZX) Run(iceParameters mediasoup.IceParameters, iceCandidates []mediasoup.IceCandidate,
	dtlsParameters mediasoup.DtlsParameters, sctpParameters mediasoup.SctpParameters) {
	dtlsParams := dtlsParameters
	var fingerprints []mediasoup.DtlsFingerprint
	for _, fingerprint := range dtlsParameters.Fingerprints {
		if fingerprint.Algorithm == "sha-256" {
			fingerprints = append(fingerprints, fingerprint)
		}
	}
	dtlsParams.Fingerprints = fingerprints
	zx.remoteSdp = &RemoteSdp{
		iceParameters:  iceParameters,
		iceCandidates:  iceCandidates,
		dtlsParameters: dtlsParams,
		sctpParameters: sctpParameters,
	}
	zx.remoteSdp.Init()
	codecOptions := CodecOptions{
		OpusDtx:    1,
		OpusStereo: 1,
	}
	if zx.direction == "send" {
		for _, canKind := range zx.canKind {
			fmt.Println("canKind => %s", canKind)
			zx.send(canKind, nil, codecOptions, nil)
		}
		zx.remoteSdp.UpdateDtlsRole("client")
	} else if zx.direction == "recv" {
		zx.remoteSdp.UpdateDtlsRole("server")
	}
}

func (zx *ZX) send(kind string, encodings []mediasoup.RtpEncodingParameters, codecOptions CodecOptions,
	codec *mediasoup.RtpCodecParameters) {
	fmt.Println("kind => %s", kind)
	nativeRtpParams := getSendingRtpParameters(kind, zx.extendedRtpCapabilities)
	nativeRtpParams.Codecs = reduceCodecs(nativeRtpParams.Codecs, codec)
	routerRtpParams := getSendingRemoteRtpParameters(kind, zx.extendedRtpCapabilities)
	routerRtpParams.Codecs = reduceCodecs(routerRtpParams.Codecs, codec)
	mediaSectionIdx := zx.remoteSdp.GetNextMediaSectionIdx()
	hackVp9Svc := false
	//localId := mediaSectionIdx.reuseMid

	offerMediaObject := zx.sdpObject.Media[mediaSectionIdx.Idx]
	jsonStr, err := json.Marshal(zx.sdpObject.Media[mediaSectionIdx.Idx])
	if err != nil {
		fmt.Println("0", err)
	}
	fmt.Println("1", "------", string(jsonStr))

	nativeRtpParams.Rtcp.Cname = getCname(*offerMediaObject)
	if encodings == nil {
		nativeRtpParams.Encodings = getRtpEncodings(*offerMediaObject)
	} else if len(encodings) == 1 {
		newEncodings := getRtpEncodings(*offerMediaObject)
		newEncodings[0] = encodings[0]
		if hackVp9Svc {
			newEncodings = []mediasoup.RtpEncodingParameters{
				newEncodings[0],
			}
		}
	} else {
		nativeRtpParams.Encodings = encodings
	}
	if len(nativeRtpParams.Encodings) > 1 &&
		(strings.ToLower(nativeRtpParams.Codecs[0].MimeType) == "video/vp8" ||
			strings.ToLower(nativeRtpParams.Codecs[0].MimeType) == "video/h264") {
		for _, encoding := range nativeRtpParams.Encodings {
			encoding.ScalabilityMode = "S1T3"
		}
	}
	zx.remoteSdp.Send(offerMediaObject, mediaSectionIdx.ReuseMid, nativeRtpParams,
		routerRtpParams, codecOptions)
	zx.RtpParameters[kind] = &nativeRtpParams
}

// receive ...
func (zx *ZX) receive(trackID string, kind string, rtpParameters mediasoup.RtpParameters) {
	localID := rtpParameters.Mid
	zx.remoteSdp.Receive(localID, kind, rtpParameters, rtpParameters.Rtcp.Cname, trackID)
	localSdpObject := zx.sdpObject
	var answerMediaObject []*transform.MediaStruct
	for _, m := range localSdpObject.Media {
		if m.Mid == localID {
			answerMediaObject = append(answerMediaObject, m)
		}
	}
	//utils := new(Utils)
	//utils.ApplyCodecParameters(rtpParameters, answerMediaObject)
}

// GetRtpParameter ...
func (zx *ZX) GetRtpParameter(kind string) *mediasoup.RtpParameters {
	return zx.RtpParameters[kind]
}

// GetNativeSctpCapabilities ...
func GetNativeSctpCapabilities() mediasoup.SctpCapabilities {
	return mediasoup.SctpCapabilities{
		NumStreams: mediasoup.NumSctpStreams{
			OS:  1024,
			MIS: 1024,
		},
	}
}
