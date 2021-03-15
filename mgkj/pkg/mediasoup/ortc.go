package mediasoup

import (
	"strconv"
	"strings"

	mediasoup "github.com/jiyeyuran/mediasoup-go"
	"github.com/jiyeyuran/mediasoup-go/h264"
	"github.com/notedit/sdp/transform"
)

// ExtendedCodec ...
type ExtendedCodec struct {
	MimeType             string                               `json:"mimeType"`
	Kind                 string                               `json:"kind"`
	ClockRate            int                                  `json:"clockRate"`
	LocalPayloadType     int                                  `json:"localPayloadType"`
	LocalRtxPayloadType  int                                  `json:"localRtxPayloadType"`
	RemotePayloadType    int                                  `json:"remotePayloadType"`
	RemoteRtxPayloadType int                                  `json:"remoteRtxPayloadType"`
	LocalParameters      mediasoup.RtpCodecSpecificParameters `json:"localParameters"`
	RemoteParameters     mediasoup.RtpCodecSpecificParameters `json:"remoteParameters"`
	RtcpFeedback         []mediasoup.RtcpFeedback             `json:"rtcpFeedback"`
	Channels             int                                  `json:"channels"`
}

// ExtendedExt ...
type ExtendedExt struct {
	Kind      string `json:"kind"`
	URI       string `json:"uri"`
	SendID    int    `json:"sendId"`
	RecvID    int    `json:"recvId"`
	Encrypt   bool   `json:"encrypt"`
	Direction string `json:"direction"`
}

// ExtendedRtpCapabilities ...
type ExtendedRtpCapabilities struct {
	Codecs           []*ExtendedCodec `json:"codecs"`
	HeaderExtensions []*ExtendedExt   `json:"headerExtensions"`
}

func extractRtpCapabilities(sdpObject transform.SdpStruct) mediasoup.RtpCapabilities {
	codecsMap := make(map[int]*mediasoup.RtpCodecCapability)

	rtpCapabilities := mediasoup.RtpCapabilities{}
	rtpCapabilities.FecMechanisms = make([]string, 0)
	rtpCapabilities.Codecs = make([]*mediasoup.RtpCodecCapability, 0)
	rtpCapabilities.HeaderExtensions = make([]*mediasoup.RtpHeaderExtension, 0)

	gotAudio := false
	gotVideo := false
	for _, m := range sdpObject.Media {
		kind := m.Type
		if kind == "audio" {
			if gotAudio {
				continue
			}
			gotAudio = true
		} else if kind == "video" {
			if gotVideo {
				continue
			}
			gotVideo = true
		} else {
			continue
		}

		for _, rtp := range m.Rtp {
			mimeType := kind
			mimeType = mimeType + "/" + rtp.Codec
			codec := &mediasoup.RtpCodecCapability{
				Kind:                 mediasoup.MediaKind(kind),
				MimeType:             mimeType,
				ClockRate:            rtp.Rate,
				Channels:             rtp.Encoding,
				PreferredPayloadType: byte(rtp.Payload),
				RtcpFeedback:         make([]mediasoup.RtcpFeedback, 0),
			}

			// jsonstr, err := json.Marshal(codec)
			// if err == nil {
			// 	fmt.Println(string(jsonstr))
			// }

			codecsMap[rtp.Payload] = codec
		}

		// Get codec parameters.
		for _, fmtp := range m.Fmtp {

			codec, ok := codecsMap[fmtp.Payload]
			if !ok {
				continue
			}

			parameters := transform.ParseParams(fmtp.Config)
			if parameters != nil {
				codec.Parameters.Usedtx = 0
				codec.Parameters.SpropStereo = 0
				minBate, err1 := strconv.Atoi(parameters["x-google-min-bitrate"])
				if err1 == nil {
					codec.Parameters.XGoogleMinBitrate = uint32(minBate)
				}
				maxBate, err2 := strconv.Atoi(parameters["x-google-max-bitrate"])
				if err2 == nil {
					codec.Parameters.XGoogleMaxBitrate = uint32(maxBate)
				}
				mBate, err3 := strconv.Atoi(parameters["x-google-start-bitrate"])
				if err3 == nil {
					codec.Parameters.XGoogleStartBitrate = uint32(mBate)
				}
				// OPUS
				fec, err4 := strconv.Atoi(parameters["useinbandfec"])
				if err4 == nil {
					codec.Parameters.Useinbandfec = uint8(fec)
				}
				// RTX
				apt, err5 := strconv.Atoi(parameters["apt"])
				if err5 == nil {
					codec.Parameters.Apt = byte(apt)
				}
				// VP9
				codec.Parameters.ProfileId = parameters["profile-id"]
				// H264
				pmode, err6 := strconv.Atoi(parameters["packetization-mode"])
				if err6 == nil {
					codec.Parameters.RtpParameter.PacketizationMode = pmode
				}
				level, err7 := strconv.Atoi(parameters["level-asymmetry-allowed"])
				if err7 == nil {
					codec.Parameters.RtpParameter.LevelAsymmetryAllowed = level
				}
				codec.Parameters.RtpParameter.ProfileLevelId = parameters["profile-level-id"]
			}

			// jsonstr, err := json.Marshal(codec)
			// if err == nil {
			// 	fmt.Println(string(jsonstr))
			// }
		}

		// Get RTCP feedback for each codec.
		for _, fb := range m.RtcpFb {
			codec, ok := codecsMap[fb.Payload]
			if !ok {
				continue
			}

			feedback := mediasoup.RtcpFeedback{
				Type:      fb.Type,
				Parameter: fb.Subtype,
			}
			codec.RtcpFeedback = append(codec.RtcpFeedback, feedback)

			// jsonstr, err := json.Marshal(codec.RtcpFeedback)
			// if err == nil {
			// 	fmt.Println(string(jsonstr))
			// }
		}

		// Get RTP header extensions.
		for _, ext := range m.Ext {
			rtpCapabilities.HeaderExtensions = append(rtpCapabilities.HeaderExtensions, &mediasoup.RtpHeaderExtension{
				Kind:        mediasoup.MediaKind(kind),
				Uri:         ext.Uri,
				PreferredId: ext.Value,
			})
		}
	}

	for _, val := range codecsMap {
		rtpCapabilities.Codecs = append(rtpCapabilities.Codecs, val)
	}
	return rtpCapabilities
}

func extractDtlsParameters(sdpObject transform.SdpStruct) mediasoup.DtlsParameters {
	var pmedia *transform.MediaStruct
	var fingerprint *transform.FingerprintStruct
	var role string

	for _, media := range sdpObject.Media {
		if media.IceUfrag != "" && media.Port != 0 {
			pmedia = media
			break
		}
	}

	if pmedia != nil && pmedia.Fingerprint != nil {
		fingerprint = pmedia.Fingerprint
	} else if sdpObject.Fingerprint != nil {
		fingerprint = sdpObject.Fingerprint
	}

	if pmedia != nil {
		switch pmedia.Setup {
		case "active":
			role = "client"
		case "passive":
			role = "server"
		case "actpass":
			role = "auto"
		}
	}

	return mediasoup.DtlsParameters{
		Role: mediasoup.DtlsRole(role),
		Fingerprints: []mediasoup.DtlsFingerprint{
			{
				Algorithm: fingerprint.Type,
				Value:     fingerprint.Hash,
			},
		},
	}
}

func getCname(offerMediaObject transform.MediaStruct) string {
	if offerMediaObject.Ssrcs == nil {
		return ""
	}

	var ssrcCnameLine *transform.SsrcStruct
	for _, line := range offerMediaObject.Ssrcs {
		if line.Attribute == "cname" {
			ssrcCnameLine = line
			break
		}
	}

	if ssrcCnameLine == nil {
		return ""
	}

	return ssrcCnameLine.Value
}

func validateRtpCapabilities(params *mediasoup.RtpCapabilities) (err error) {
	for _, codec := range params.Codecs {
		if err = validateRtpCodecCapability(codec); err != nil {
			return
		}
	}

	for _, ext := range params.HeaderExtensions {
		if err = validateRtpHeaderExtension(ext); err != nil {
			return
		}
	}

	return
}

func validateRtpCodecCapability(code *mediasoup.RtpCodecCapability) (err error) {
	mimeType := strings.ToLower(code.MimeType)

	//  mimeType is mandatory.
	if !strings.HasPrefix(mimeType, "audio/") && !strings.HasPrefix(mimeType, "video/") {
		return mediasoup.NewTypeError("invalid codec.mimeType")
	}

	code.Kind = mediasoup.MediaKind(strings.Split(mimeType, "/")[0])

	// clockRate is mandatory.
	if code.ClockRate == 0 {
		return mediasoup.NewTypeError("missing codec.clockRate")
	}

	// channels is optional. If unset, set it to 1 (just if audio).
	if code.Kind == mediasoup.MediaKind_Audio && code.Channels == 0 {
		code.Channels = 1
	}

	for _, fb := range code.RtcpFeedback {
		if err = validateRtcpFeedback(fb); err != nil {
			return
		}
	}

	return
}

func validateRtcpFeedback(fb mediasoup.RtcpFeedback) error {
	if len(fb.Type) == 0 {
		return mediasoup.NewTypeError("missing fb.type")
	}
	return nil
}

func validateRtpHeaderExtension(ext *mediasoup.RtpHeaderExtension) (err error) {
	if len(ext.Kind) > 0 && ext.Kind != mediasoup.MediaKind_Audio && ext.Kind != mediasoup.MediaKind_Video {
		return mediasoup.NewTypeError("invalid ext.kind")
	}

	// uri is mandatory.
	if len(ext.Uri) == 0 {
		return mediasoup.NewTypeError("missing ext.uri")
	}

	// preferredId is mandatory.
	if ext.PreferredId == 0 {
		return mediasoup.NewTypeError("missing ext.preferredId")
	}

	// direction is optional. If unset set it to sendrecv.
	if len(ext.Direction) == 0 {
		ext.Direction = mediasoup.Direction_Sendrecv
	}

	return
}

func validateRtpParameters(params *mediasoup.RtpParameters) (err error) {
	for _, codec := range params.Codecs {
		if err = validateRtpCodecParameters(codec); err != nil {
			return
		}
	}

	for _, ext := range params.HeaderExtensions {
		if err = validateRtpHeaderExtensionParameters(ext); err != nil {
			return
		}
	}

	return validateRtcpParameters(&params.Rtcp)
}

func validateRtpCodecParameters(code *mediasoup.RtpCodecParameters) (err error) {
	mimeType := strings.ToLower(code.MimeType)

	//  mimeType is mandatory.
	if !strings.HasPrefix(mimeType, "audio/") && !strings.HasPrefix(mimeType, "video/") {
		return mediasoup.NewTypeError("invalid codec.mimeType")
	}

	// clockRate is mandatory.
	if code.ClockRate == 0 {
		return mediasoup.NewTypeError("missing codec.clockRate")
	}

	kind := mediasoup.MediaKind(strings.Split(mimeType, "/")[0])

	// channels is optional. If unset, set it to 1 (just if audio).
	if kind == mediasoup.MediaKind_Audio && code.Channels == 0 {
		code.Channels = 1
	}

	for _, fb := range code.RtcpFeedback {
		if err = validateRtcpFeedback(fb); err != nil {
			return
		}
	}

	return
}

func validateRtpHeaderExtensionParameters(ext mediasoup.RtpHeaderExtensionParameters) (err error) {
	// uri is mandatory.
	if len(ext.Uri) == 0 {
		return mediasoup.NewTypeError("missing ext.uri")
	}

	// preferredId is mandatory.
	if ext.Id == 0 {
		return mediasoup.NewTypeError("missing ext.id")
	}

	return
}

func validateRtcpParameters(rtcp *mediasoup.RtcpParameters) (err error) {
	// reducedSize is optional. If unset set it to true.
	if rtcp.ReducedSize == nil {
		rtcp.ReducedSize = mediasoup.Bool(true)
	}

	return
}

func isRtxCodec(r *mediasoup.RtpCodecCapability) bool {
	return strings.HasSuffix(strings.ToLower(r.MimeType), "/rtx")
}

func isRtxCodec2(r *mediasoup.RtpCodecParameters) bool {
	return strings.HasSuffix(strings.ToLower(r.MimeType), "/rtx")
}

func matchCodecs(aCodec *mediasoup.RtpCodecCapability, bCodec *mediasoup.RtpCodecCapability, strict bool, modify bool) bool {
	aMimeType := strings.ToLower(aCodec.MimeType)
	bMimeType := strings.ToLower(bCodec.MimeType)

	if aMimeType != bMimeType {
		return false
	}

	if aCodec.ClockRate != bCodec.ClockRate {
		return false
	}

	if strings.HasPrefix(aMimeType, "audio/") &&
		aCodec.Channels > 0 &&
		bCodec.Channels > 0 &&
		aCodec.Channels != bCodec.Channels {
		return false
	}

	switch aMimeType {
	case "video/h264":
		aParameters, bParameters := aCodec.Parameters, bCodec.Parameters

		if aParameters.PacketizationMode != bParameters.PacketizationMode {
			return false
		}

		if strict {
			selectedProfileLevelID, err := h264.GenerateProfileLevelIdForAnswer(
				aParameters.RtpParameter, bParameters.RtpParameter)
			if err != nil {
				return false
			}

			if modify {
				aParameters.ProfileLevelId = selectedProfileLevelID
				aCodec.Parameters = aParameters
			}
		}
	}

	return true
}

func matchCodecs2(aCodec *mediasoup.RtpCodecParameters, bCodec *mediasoup.RtpCodecParameters, strict bool, modify bool) (matched bool) {
	aMimeType := strings.ToLower(aCodec.MimeType)
	bMimeType := strings.ToLower(bCodec.MimeType)

	if aMimeType != bMimeType {
		return
	}

	if aCodec.ClockRate != bCodec.ClockRate {
		return
	}

	if strings.HasPrefix(aMimeType, "audio/") &&
		aCodec.Channels > 0 &&
		bCodec.Channels > 0 &&
		aCodec.Channels != bCodec.Channels {
		return
	}

	switch aMimeType {
	case "video/h264":
		aParameters, bParameters := aCodec.Parameters, bCodec.Parameters

		if aParameters.PacketizationMode != bParameters.PacketizationMode {
			return
		}

		if strict {
			selectedProfileLevelID, err := h264.GenerateProfileLevelIdForAnswer(
				aParameters.RtpParameter, bParameters.RtpParameter)
			if err != nil {
				return
			}

			if modify {
				aParameters.ProfileLevelId = selectedProfileLevelID
				aCodec.Parameters = aParameters
			}
		}
	}

	return true
}

func reduceRtcpFeedback(codecA mediasoup.RtpCodecCapability, codecB mediasoup.RtpCodecCapability) []mediasoup.RtcpFeedback {
	reducedRtcpFeedback := make([]mediasoup.RtcpFeedback, 0)

	for _, aFb := range codecA.RtcpFeedback {
		var rtcpFeedback *mediasoup.RtcpFeedback
		for _, bFb := range codecB.RtcpFeedback {
			if aFb.Type == bFb.Type && aFb.Parameter == bFb.Parameter {
				rtcpFeedback = &bFb
				break
			}
		}

		if rtcpFeedback != nil {
			reducedRtcpFeedback = append(reducedRtcpFeedback, *rtcpFeedback)
		}
	}
	return reducedRtcpFeedback
}

func matchHeaderExtensions(aExt mediasoup.RtpHeaderExtension, bExt mediasoup.RtpHeaderExtension) bool {
	if aExt.Kind != bExt.Kind {
		return false
	}

	return aExt.Uri == bExt.Uri
}

func getExtendedRtpCapabilities(localCaps *mediasoup.RtpCapabilities, remoteCaps *mediasoup.RtpCapabilities) ExtendedRtpCapabilities {
	err := validateRtpCapabilities(localCaps)
	if err != nil {
		panic("validate RtpCapabilities")
	}

	err = validateRtpCapabilities(remoteCaps)
	if err != nil {
		panic("validate RtpCapabilities")
	}

	extendedRtpCapabilities := ExtendedRtpCapabilities{}
	extendedRtpCapabilities.Codecs = make([]*ExtendedCodec, 0)
	extendedRtpCapabilities.HeaderExtensions = make([]*ExtendedExt, 0)

	for _, remoteCodec := range remoteCaps.Codecs {
		if isRtxCodec(remoteCodec) {
			continue
		}

		localCodecs := localCaps.Codecs
		var matchingLocalCodec *mediasoup.RtpCodecCapability
		for _, localCodec := range localCodecs {
			if matchCodecs(localCodec, remoteCodec, true, true) {
				matchingLocalCodec = localCodec
				break
			}
		}

		if matchingLocalCodec == nil {
			continue
		}

		extendedCodec := &ExtendedCodec{
			MimeType:          matchingLocalCodec.MimeType,
			Kind:              string(matchingLocalCodec.Kind),
			ClockRate:         matchingLocalCodec.ClockRate,
			LocalPayloadType:  int(matchingLocalCodec.PreferredPayloadType),
			RemotePayloadType: int(remoteCodec.PreferredPayloadType),
			LocalParameters:   matchingLocalCodec.Parameters,
			RemoteParameters:  remoteCodec.Parameters,
			RtcpFeedback:      reduceRtcpFeedback(*matchingLocalCodec, *remoteCodec),
			Channels:          matchingLocalCodec.Channels,
		}

		extendedRtpCapabilities.Codecs = append(extendedRtpCapabilities.Codecs, extendedCodec)
	}

	// Match RTX codecs.
	extendedCodecs := extendedRtpCapabilities.Codecs
	for _, extendedCodec := range extendedCodecs {
		localCodecs := localCaps.Codecs
		var matchingLocalRtxCodec *mediasoup.RtpCodecCapability
		for _, localCodec := range localCodecs {
			if isRtxCodec(localCodec) && int(localCodec.Parameters.Apt) == extendedCodec.LocalPayloadType {
				matchingLocalRtxCodec = localCodec
				break
			}
		}

		if matchingLocalRtxCodec == nil {
			continue
		}

		remoteCodecs := remoteCaps.Codecs
		var matchingRemoteRtxCodec *mediasoup.RtpCodecCapability
		for _, remoteCodec := range remoteCodecs {
			if isRtxCodec(remoteCodec) && int(remoteCodec.Parameters.Apt) == extendedCodec.RemotePayloadType {
				matchingRemoteRtxCodec = remoteCodec
			}
		}

		extendedCodec.LocalRtxPayloadType = int(matchingLocalRtxCodec.PreferredPayloadType)
		extendedCodec.RemoteRtxPayloadType = int(matchingRemoteRtxCodec.PreferredPayloadType)
	}

	// Match header extensions.
	remoteExts := remoteCaps.HeaderExtensions
	for _, remoteExt := range remoteExts {
		localExts := localCaps.HeaderExtensions
		var matchingLocalExt *mediasoup.RtpHeaderExtension
		for _, localExt := range localExts {
			if matchHeaderExtensions(*localExt, *remoteExt) {
				matchingLocalExt = localExt
				break
			}
		}
		if matchingLocalExt == nil {
			continue
		}

		// TODO: Must do stuff for encrypted extensions.
		extendedExt := &ExtendedExt{
			Kind:    string(remoteExt.Kind),
			URI:     remoteExt.Uri,
			SendID:  matchingLocalExt.PreferredId,
			RecvID:  remoteExt.PreferredId,
			Encrypt: matchingLocalExt.PreferredEncrypt,
		}

		remoteExtDirection := remoteExt.Direction

		if remoteExtDirection == "sendrecv" {
			extendedExt.Direction = "sendrecv"
		} else if remoteExtDirection == "recvonly" {
			extendedExt.Direction = "sendonly"
		} else if remoteExtDirection == "sendonly" {
			extendedExt.Direction = "recvonly"
		} else if remoteExtDirection == "inactive" {
			extendedExt.Direction = "inactive"
		}
		extendedRtpCapabilities.HeaderExtensions = append(extendedRtpCapabilities.HeaderExtensions, extendedExt)
	}

	return extendedRtpCapabilities
}

func getSendingRtpParameters(kind string, extendedRtpCapabilities ExtendedRtpCapabilities) mediasoup.RtpParameters {
	rtpParameters := mediasoup.RtpParameters{}
	rtpParameters.Codecs = make([]*mediasoup.RtpCodecParameters, 0)
	rtpParameters.Encodings = make([]mediasoup.RtpEncodingParameters, 0)
	rtpParameters.HeaderExtensions = make([]mediasoup.RtpHeaderExtensionParameters, 0)

	for _, extendedCodec := range extendedRtpCapabilities.Codecs {
		if kind != extendedCodec.Kind {
			continue
		}

		codec := &mediasoup.RtpCodecParameters{
			MimeType:     extendedCodec.MimeType,
			PayloadType:  byte(extendedCodec.LocalPayloadType),
			ClockRate:    extendedCodec.ClockRate,
			Parameters:   extendedCodec.LocalParameters,
			RtcpFeedback: extendedCodec.RtcpFeedback,
			Channels:     extendedCodec.Channels,
		}
		rtpParameters.Codecs = append(rtpParameters.Codecs, codec)

		// Add RTX codec.
		if extendedCodec.LocalRtxPayloadType != 0 {
			mimeType := extendedCodec.Kind + "/rtx"
			rtxCodec := &mediasoup.RtpCodecParameters{
				MimeType:    mimeType,
				PayloadType: byte(extendedCodec.LocalRtxPayloadType),
				ClockRate:   extendedCodec.ClockRate,
				Parameters: mediasoup.RtpCodecSpecificParameters{
					Apt: byte(extendedCodec.LocalPayloadType),
				},
			}
			rtpParameters.Codecs = append(rtpParameters.Codecs, rtxCodec)
		}

		// NOTE: We assume a single media codec plus an optional RTX codec.
		break
	}

	for _, extendedExtension := range extendedRtpCapabilities.HeaderExtensions {
		if kind != extendedExtension.Kind {
			continue
		}

		direction := extendedExtension.Direction

		// Ignore RTP extensions not valid for sending.
		if direction != "sendrecv" && direction != "sendonly" {
			continue
		}

		ext := mediasoup.RtpHeaderExtensionParameters{
			Uri:     extendedExtension.URI,
			Id:      extendedExtension.SendID,
			Encrypt: extendedExtension.Encrypt,
		}
		rtpParameters.HeaderExtensions = append(rtpParameters.HeaderExtensions, ext)
	}

	return rtpParameters
}

func getSendingRemoteRtpParameters(kind string, extendedRtpCapabilities ExtendedRtpCapabilities) mediasoup.RtpParameters {
	rtpParameters := mediasoup.RtpParameters{}
	rtpParameters.Codecs = make([]*mediasoup.RtpCodecParameters, 0)
	rtpParameters.Encodings = make([]mediasoup.RtpEncodingParameters, 0)
	rtpParameters.HeaderExtensions = make([]mediasoup.RtpHeaderExtensionParameters, 0)

	for _, extendedCodec := range extendedRtpCapabilities.Codecs {
		if kind != extendedCodec.Kind {
			continue
		}

		codec := &mediasoup.RtpCodecParameters{
			MimeType:     extendedCodec.MimeType,
			PayloadType:  byte(extendedCodec.LocalPayloadType),
			ClockRate:    extendedCodec.ClockRate,
			Parameters:   extendedCodec.RemoteParameters,
			RtcpFeedback: extendedCodec.RtcpFeedback,
			Channels:     extendedCodec.Channels,
		}
		rtpParameters.Codecs = append(rtpParameters.Codecs, codec)

		// Add RTX codec.
		if extendedCodec.LocalRtxPayloadType != 0 {
			rtxCodec := &mediasoup.RtpCodecParameters{
				MimeType:    extendedCodec.Kind + "/rtx",
				PayloadType: byte(extendedCodec.LocalRtxPayloadType),
				ClockRate:   extendedCodec.ClockRate,
				Parameters: mediasoup.RtpCodecSpecificParameters{
					Apt: byte(extendedCodec.LocalPayloadType),
				},
			}
			rtpParameters.Codecs = append(rtpParameters.Codecs, rtxCodec)
		}
	}

	for _, extendedExtension := range extendedRtpCapabilities.HeaderExtensions {
		if kind != extendedExtension.Kind || (extendedExtension.Direction != "sendrecv" && extendedExtension.Direction != "sendonly") {
			continue
		}

		ext := mediasoup.RtpHeaderExtensionParameters{
			Uri:     extendedExtension.URI,
			Id:      extendedExtension.SendID,
			Encrypt: extendedExtension.Encrypt,
		}
		rtpParameters.HeaderExtensions = append(rtpParameters.HeaderExtensions, ext)
	}

	var headerExtension *mediasoup.RtpHeaderExtensionParameters
	for _, ext := range rtpParameters.HeaderExtensions {
		if ext.Uri == "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01" {
			headerExtension = &ext
			break
		}
	}
	if headerExtension != nil {
		for _, codec := range rtpParameters.Codecs {
			rtcpFeedback := codec.RtcpFeedback
			codec.RtcpFeedback = []mediasoup.RtcpFeedback{}
			for _, fb := range rtcpFeedback {
				if fb.Type != "goog-remb" {
					codec.RtcpFeedback = append(codec.RtcpFeedback, fb)
				}
			}
		}
		return rtpParameters
	}

	for _, ext := range rtpParameters.HeaderExtensions {
		if ext.Uri == "http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time" {
			headerExtension = &ext
			break
		}
	}
	if headerExtension != nil {
		for _, codec := range rtpParameters.Codecs {
			rtcpFeedback := codec.RtcpFeedback
			codec.RtcpFeedback = []mediasoup.RtcpFeedback{}
			for _, fb := range rtcpFeedback {
				if fb.Type != "transport-cc" {
					codec.RtcpFeedback = append(codec.RtcpFeedback, fb)
				}
			}
		}
		return rtpParameters
	}

	for _, codec := range rtpParameters.Codecs {
		rtcpFeedback := codec.RtcpFeedback
		codec.RtcpFeedback = []mediasoup.RtcpFeedback{}
		for _, fb := range rtcpFeedback {
			if fb.Type != "transport-cc" && fb.Type != "goog-remb" {
				codec.RtcpFeedback = append(codec.RtcpFeedback, fb)
			}
		}
	}
	return rtpParameters
}

func getRecvRtpCapabilities(extendedRtpCapabilities ExtendedRtpCapabilities) mediasoup.RtpCapabilities {
	rtpCapabilities := mediasoup.RtpCapabilities{}
	rtpCapabilities.Codecs = make([]*mediasoup.RtpCodecCapability, 0)
	rtpCapabilities.FecMechanisms = make([]string, 0)
	rtpCapabilities.HeaderExtensions = make([]*mediasoup.RtpHeaderExtension, 0)

	for _, extendedCodec := range extendedRtpCapabilities.Codecs {
		codec := mediasoup.RtpCodecCapability{
			MimeType:             extendedCodec.MimeType,
			Kind:                 mediasoup.MediaKind(extendedCodec.Kind),
			PreferredPayloadType: byte(extendedCodec.RemotePayloadType),
			ClockRate:            extendedCodec.ClockRate,
			Channels:             extendedCodec.Channels,
			Parameters:           extendedCodec.LocalParameters,
			RtcpFeedback:         extendedCodec.RtcpFeedback,
		}
		rtpCapabilities.Codecs = append(rtpCapabilities.Codecs, &codec)
		if extendedCodec.RemoteRtxPayloadType == 0 {
			continue
		}
		rtxCodec := mediasoup.RtpCodecCapability{
			MimeType:             extendedCodec.Kind + "/rtx",
			Kind:                 mediasoup.MediaKind(extendedCodec.Kind),
			PreferredPayloadType: byte(extendedCodec.RemoteRtxPayloadType),
			ClockRate:            extendedCodec.ClockRate,
			Parameters: mediasoup.RtpCodecSpecificParameters{
				Apt: byte(extendedCodec.RemotePayloadType),
			},
		}
		rtpCapabilities.Codecs = append(rtpCapabilities.Codecs, &rtxCodec)
	}
	for _, extendedExtension := range extendedRtpCapabilities.HeaderExtensions {
		if extendedExtension.Direction != "sendrecv" &&
			extendedExtension.Direction != "recvonly" {
			continue
		}
		ext := mediasoup.RtpHeaderExtension{
			Kind:             mediasoup.MediaKind(extendedExtension.Kind),
			Uri:              extendedExtension.URI,
			PreferredId:      extendedExtension.RecvID,
			PreferredEncrypt: extendedExtension.Encrypt,
			Direction:        mediasoup.RtpHeaderExtensionDirection(extendedExtension.Direction),
		}
		rtpCapabilities.HeaderExtensions = append(rtpCapabilities.HeaderExtensions, &ext)
	}
	return rtpCapabilities
}

func canSend(kind string, extendedRtpCapabilities ExtendedRtpCapabilities) bool {
	for _, codec := range extendedRtpCapabilities.Codecs {
		if codec.Kind == kind {
			return true
		}
	}
	return false
}

func reduceCodecs(codecs []*mediasoup.RtpCodecParameters, capCodec *mediasoup.RtpCodecParameters) []*mediasoup.RtpCodecParameters {
	filteredCodecs := make([]*mediasoup.RtpCodecParameters, 0)

	// If no capability codec is given, take the first one (and RTX).
	if capCodec == nil {
		filteredCodecs = append(filteredCodecs, codecs[0])
		if len(codecs) > 1 && isRtxCodec2(codecs[1]) {
			filteredCodecs = append(filteredCodecs, codecs[1])
		}
	} else {
		// Otherwise look for a compatible set of codecs.
		for idx := range codecs {
			if matchCodecs2(codecs[idx], capCodec, false, false) {
				filteredCodecs = append(filteredCodecs, codecs[idx])
				if isRtxCodec2(codecs[idx+1]) {
					filteredCodecs = append(filteredCodecs, codecs[idx+1])
				}
				break
			}
		}

	}
	return filteredCodecs
}

func getRtpEncodings(offerMediaObject transform.MediaStruct) []mediasoup.RtpEncodingParameters {
	ssrcs := make(map[uint32]uint32)
	for _, line := range offerMediaObject.Ssrcs {
		ssrcs[uint32(line.Id)] = uint32(line.Id)
	}

	if len(ssrcs) == 0 {
		panic("no a=ssrc lines found")
	}

	ssrcToRtxSsrc := make(map[uint32]uint32)
	if offerMediaObject.SsrcGroups != nil {
		// First assume RTX is used.
		for _, line := range offerMediaObject.SsrcGroups {
			if line.Semantics != "FID" {
				continue
			}

			fidLine := line.Ssrcs
			v := strings.Split(fidLine, " ")
			ssrc, _ := strconv.Atoi(v[0])
			rtxSsrc, _ := strconv.Atoi(v[1])

			// Remove the RTX SSRC from the List so later we know that they
			// are already handled.
			for _, ssrc := range ssrcs {
				if ssrc == uint32(rtxSsrc) {
					delete(ssrcs, ssrc)
				}
			}
			ssrcToRtxSsrc[uint32(ssrc)] = uint32(rtxSsrc)
		}
	}

	// Fill RTP parameters.
	encodings := []mediasoup.RtpEncodingParameters{}
	for _, ssrc := range ssrcs {
		encoding := mediasoup.RtpEncodingParameters{
			Ssrc: uint32(ssrc),
		}

		rtxSsrc, ok := ssrcToRtxSsrc[ssrc]
		if ok {
			encoding.Rtx = &mediasoup.RtpEncodingRtx{
				Ssrc: rtxSsrc,
			}
		}
		encodings = append(encodings, encoding)
	}

	return encodings
}
