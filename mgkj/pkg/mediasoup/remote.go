package mediasoup

import (
	"strconv"
	"strings"

	mediasoup "github.com/jiyeyuran/mediasoup-go"
	"github.com/notedit/sdp/transform"
)

// CodecOptions ...
type CodecOptions struct {
	OpusDtx    uint8 `json:"opusDtx"`
	OpusStereo uint8 `json:"opusStereo"`
}

func getCodecName(codec *mediasoup.RtpCodecParameters) string {
	mimeType := codec.MimeType
	return mimeType[6:]
}

// MediaSection ...
type MediaSection struct {
	mediaObject *transform.MediaStruct
}

// Init ...
func (m *MediaSection) Init(iceParameters mediasoup.IceParameters, iceCandidates []mediasoup.IceCandidate) {
	m.mediaObject = &transform.MediaStruct{}
	m.mediaObject.Bandwidth = make([]*transform.BandwithStruct, 0)
	m.mediaObject.Candidates = make([]*transform.CandidateStruct, 0)
	m.mediaObject.Ext = make([]*transform.ExtStruct, 0)
	m.mediaObject.Fmtp = make([]*transform.FmtpStruct, 0)
	m.mediaObject.Rids = make([]*transform.RidStruct, 0)
	m.mediaObject.RtcpFb = make([]*transform.RtcpFbStruct, 0)
	m.mediaObject.Rtp = make([]*transform.RtpStruct, 0)
	m.mediaObject.SsrcGroups = make([]*transform.SsrcGroupStruct, 0)
	m.mediaObject.Ssrcs = make([]*transform.SsrcStruct, 0)
	m.SetIceParameters(iceParameters)

	for _, candidate := range iceCandidates {
		candidateObject := &transform.CandidateStruct{
			Component:  1,
			Foundation: candidate.Foundation,
			Ip:         candidate.Ip,
			Port:       int(candidate.Port),
			Priority:   int(candidate.Priority),
			Transport:  string(candidate.Protocol),
			Type:       candidate.Type,
		}
		m.mediaObject.Candidates = append(m.mediaObject.Candidates, candidateObject)

		// this->mediaObject["endOfCandidates"] = "end-of-candidates";
		// this->mediaObject["iceOptions"]      = "renomination";
	}
}

// SetIceParameters ...
func (m *MediaSection) SetIceParameters(iceParameters mediasoup.IceParameters) {
	m.mediaObject.IceUfrag = iceParameters.UsernameFragment
	m.mediaObject.IcePwd = iceParameters.Password
}

// AnswerMediaSection ...
type AnswerMediaSection struct {
	m           *MediaSection
	mediaObject *transform.MediaStruct
}

// Init ...
func (m *AnswerMediaSection) Init(iceParameters mediasoup.IceParameters, iceCandidates []mediasoup.IceCandidate,
	dtlsParameters mediasoup.DtlsParameters, sctpParameters mediasoup.SctpParameters,
	offerMediaObject *transform.MediaStruct, offerRtpParameters mediasoup.RtpParameters,
	answerRtpParameters mediasoup.RtpParameters, codecOptions *CodecOptions) {

	m.m = new(MediaSection)
	m.m.Init(iceParameters, iceCandidates)
	m.mediaObject = m.m.mediaObject

	// AnswerMediaSection
	mtype := offerMediaObject.Type
	m.mediaObject.Mid = offerMediaObject.Mid
	m.mediaObject.Type = mtype
	m.mediaObject.Protocal = offerMediaObject.Protocal

	m.mediaObject.Connection = &transform.ConnectionStruct{
		Version: 4,
		Ip:      "127.0.0.1",
	}
	m.mediaObject.Port = 7

	// Set DTLS role.
	dtlsRole := string(dtlsParameters.Role)
	if dtlsRole == "client" {
		m.mediaObject.Setup = "active"
	} else if dtlsRole == "server" {
		m.mediaObject.Setup = "passive"
	} else if dtlsRole == "auto" {
		m.mediaObject.Setup = "actpass"
	}

	if mtype == "audio" || mtype == "video" {
		m.mediaObject.Direction = "recvonly"

		for _, codec := range answerRtpParameters.Codecs {
			// clang-format off
			rtp := &transform.RtpStruct{
				Payload:  int(codec.PayloadType),
				Codec:    getCodecName(codec),
				Rate:     codec.ClockRate,
				Encoding: codec.Channels,
			}
			// clang-format on
			m.mediaObject.Rtp = append(m.mediaObject.Rtp, rtp)

			codecParameters := codec.Parameters

			if codecOptions != nil {
				offerCodecs := offerRtpParameters.Codecs
				var offerCodec *mediasoup.RtpCodecParameters
				for _, c := range offerCodecs {
					if c.PayloadType == codec.PayloadType {
						offerCodec = c
					}
				}
				mimeType := codec.MimeType
				mimeType = strings.ToLower(mimeType)

				if mimeType == "audio/opus" {
					offerCodec.Parameters.SpropStereo = codecOptions.OpusStereo
					codecParameters.SpropStereo = codecOptions.OpusStereo

					// auto opusFecIt = codecOptions->find("opusFec");
					// if (opusFecIt != codecOptions->end())
					// {
					// 	auto opusFec                             = opusFecIt->get<bool>();
					// 	offerCodec["parameters"]["useinbandfec"] = opusFec ? 1 : 0;
					// 	codecParameters["useinbandfec"]          = opusFec ? 1 : 0;
					// }

					offerCodec.Parameters.Usedtx = codecOptions.OpusDtx
					codecParameters.Usedtx = codecOptions.OpusDtx

					// auto opusMaxPlaybackRateIt = codecOptions->find("opusMaxPlaybackRate");
					// if (opusMaxPlaybackRateIt != codecOptions->end())
					// {
					// 	auto opusMaxPlaybackRate           = opusMaxPlaybackRateIt->get<uint32_t>();
					// 	codecParameters["maxplaybackrate"] = opusMaxPlaybackRate;
					// }

					// auto opusPtimeIt = codecOptions->find("opusPtime");
					// if (opusPtimeIt != codecOptions->end())
					// {
					// 	auto opusPtime           = opusPtimeIt->get<uint32_t>();
					// 	codecParameters["ptime"] = opusPtime;
					// }
				} else if mimeType == "video/vp8" || mimeType == "video/vp9" || mimeType == "video/h264" || mimeType == "video/h265" {
					// auto videoGoogleStartBitrateIt = codecOptions->find("videoGoogleStartBitrate");
					// if (videoGoogleStartBitrateIt != codecOptions->end())
					// {
					// 	auto videoGoogleStartBitrate = videoGoogleStartBitrateIt->get<uint32_t>();
					// 	codecParameters["x-google-start-bitrate"] = videoGoogleStartBitrate;
					// }

					// auto videoGoogleMaxBitrateIt = codecOptions->find("videoGoogleMaxBitrate");
					// if (videoGoogleMaxBitrateIt != codecOptions->end())
					// {
					// 	auto videoGoogleMaxBitrate              = videoGoogleMaxBitrateIt->get<uint32_t>();
					// 	codecParameters["x-google-max-bitrate"] = videoGoogleMaxBitrate;
					// }

					// auto videoGoogleMinBitrateIt = codecOptions->find("videoGoogleMinBitrate");
					// if (videoGoogleMinBitrateIt != codecOptions->end())
					// {
					// 	auto videoGoogleMinBitrate              = videoGoogleMinBitrateIt->get<uint32_t>();
					// 	codecParameters["x-google-min-bitrate"] = videoGoogleMinBitrate;
					// }
				}

				fmtp := &transform.FmtpStruct{
					Payload: int(codec.PayloadType),
				}
				if len(codecParameters.ProfileId) != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "profile-id=" + codecParameters.ProfileId
				}
				if codecParameters.Apt != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "apt=" + strconv.Itoa(int(codecParameters.Apt))
				}
				if codecParameters.SpropStereo != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "stereo=" + strconv.Itoa(int(codecParameters.SpropStereo))
				}
				if codecParameters.Useinbandfec != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "useinbandfec=" + strconv.Itoa(int(codecParameters.Useinbandfec))
				}
				if codecParameters.Usedtx != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "usedtx=" + strconv.Itoa(int(codecParameters.Usedtx))
				}
				if codecParameters.Maxplaybackrate != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "maxplaybackrate=" + strconv.FormatUint(uint64(codecParameters.Maxplaybackrate), 10)
				}
				if codecParameters.XGoogleMinBitrate != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "x-google-min-bitrate=" + strconv.FormatUint(uint64(codecParameters.XGoogleMinBitrate), 10)
				}
				if codecParameters.XGoogleMaxBitrate != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "x-google-max-bitrate=" + strconv.FormatUint(uint64(codecParameters.XGoogleMaxBitrate), 10)
				}
				if codecParameters.XGoogleStartBitrate != 0 {
					if len(fmtp.Config) != 0 {
						fmtp.Config += ";"
					}
					fmtp.Config += "x-google-start-bitrate=" + strconv.FormatUint(uint64(codecParameters.XGoogleStartBitrate), 10)
				}

				if len(fmtp.Config) != 0 {
					m.mediaObject.Fmtp = append(m.mediaObject.Fmtp, fmtp)
				}

				for _, fb := range codec.RtcpFeedback {
					m.mediaObject.RtcpFb = append(m.mediaObject.RtcpFb, &transform.RtcpFbStruct{
						Payload: int(codec.PayloadType),
						Type:    fb.Type,
						Subtype: fb.Parameter,
					})
				}

				var payloads string

				for _, codec := range answerRtpParameters.Codecs {
					payloadType := codec.PayloadType
					if len(payloads) != 0 {
						payloads = payloads + " "
					}
					payloads = payloads + strconv.Itoa(int(payloadType))
				}
				m.mediaObject.Payloads = payloads

				// Don't add a header extension if not present in the offer.
				for _, ext := range answerRtpParameters.HeaderExtensions {
					localExts := offerMediaObject.Ext

					var localExt *transform.ExtStruct
					for _, e := range localExts {
						if e.Uri == ext.Uri {
							localExt = e
							break
						}
					}
					if localExt == nil {
						continue
					}

					// clang-format off
					m.mediaObject.Ext = append(m.mediaObject.Ext, &transform.ExtStruct{
						Uri:   ext.Uri,
						Value: ext.Id,
					})
					// clang-format on
				}

				// // Allow both 1 byte and 2 bytes length header extensions.
				// auto extmapAllowMixedIt = offerMediaObject.find("extmapAllowMixed");

				// // clang-format off
				// if (
				// 	extmapAllowMixedIt != offerMediaObject.end() &&
				// 	extmapAllowMixedIt->is_string()
				// )
				// // clang-format on
				// {
				// 	this->mediaObject["extmapAllowMixed"] = "extmap-allow-mixed";
				// }

				// Simulcast.
				simulcast := offerMediaObject.Simulcast
				rids := offerMediaObject.Rids

				if simulcast != nil && rids != nil {
					m.mediaObject.Simulcast = &transform.SimulcastStruct{
						Dir1:  "recv",
						List1: simulcast.List1,
					}

					for _, rid := range rids {
						if rid.Direction != "send" {
							continue
						}

						m.mediaObject.Rids = append(m.mediaObject.Rids, &transform.RidStruct{
							Id:        rid.Id,
							Direction: "recv",
						})
					}
				}

				m.mediaObject.RtcpMux = "rtcp-mux"
				m.mediaObject.RtcpRsize = "rtcp-rsize"
			}
		}
	} else if mtype == "application" {
		m.mediaObject.Payloads = "webrtc-datachannel"
		// m.mediaObject.SctpPort = sctpParameters.Port
		// m.mediaObject.MaxMessageSize = sctpParameters.MaxMessageSize
	}
}

// SetDtlsRole ...
func (m *AnswerMediaSection) SetDtlsRole(role string) {
	if role == "client" {
		m.mediaObject.Setup = "active"
	} else if role == "server" {
		m.mediaObject.Setup = "passive"
	} else if role == "auto" {
		m.mediaObject.Setup = "actpass"
	}
}

// GetObject ...
func (m *AnswerMediaSection) GetObject() *transform.MediaStruct {
	return m.mediaObject
}

// GetMid ...
func (m *AnswerMediaSection) GetMid() string {
	return m.mediaObject.Mid
}

// IsClosed ...
func (m *AnswerMediaSection) IsClosed() bool {
	return m.mediaObject.Port == 0
}

// OfferMediaSection ...
type OfferMediaSection struct {
	m           *MediaSection
	mediaObject *transform.MediaStruct
}

// Init ...
func (m *OfferMediaSection) Init(iceParameters mediasoup.IceParameters, iceCandidates []mediasoup.IceCandidate, dtlsParameters mediasoup.DtlsParameters, sctpParameters *mediasoup.SctpParameters,
	mid string, kind string, offerRtpParameters mediasoup.RtpParameters, streamID string, trackID string) {
	m.m = new(MediaSection)
	m.m.Init(iceParameters, iceCandidates)
	m.mediaObject = m.m.mediaObject

	m.mediaObject.Mid = mid
	m.mediaObject.Type = kind

	if sctpParameters == nil {
		m.mediaObject.Protocal = "UDP/TLS/RTP/SAVPF"
	} else {
		m.mediaObject.Protocal = "UDP/DTLS/SCTP"
	}

	m.mediaObject.Connection = &transform.ConnectionStruct{
		Ip:      "127.0.0.1",
		Version: 4,
	}
	m.mediaObject.Port = 7

	// Set DTLS role.
	m.mediaObject.Setup = "actpass"

	if kind == "audio" || kind == "video" {
		m.mediaObject.Direction = "sendonly"
		for _, codec := range offerRtpParameters.Codecs {
			rtp := &transform.RtpStruct{
				Payload:  int(codec.PayloadType),
				Codec:    getCodecName(codec),
				Rate:     codec.ClockRate,
				Encoding: codec.Channels,
			}
			m.mediaObject.Rtp = append(m.mediaObject.Rtp, rtp)

			codecParameters := codec.Parameters

			fmtp := &transform.FmtpStruct{
				Payload: int(codec.PayloadType),
			}
			if len(codecParameters.ProfileId) != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "profile-id=" + codecParameters.ProfileId
			}
			if codecParameters.Apt != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "apt=" + strconv.Itoa(int(codecParameters.Apt))
			}
			if codecParameters.SpropStereo != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "stereo=" + strconv.Itoa(int(codecParameters.SpropStereo))
			}
			if codecParameters.Useinbandfec != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "useinbandfec=" + strconv.Itoa(int(codecParameters.Useinbandfec))
			}
			if codecParameters.Usedtx != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "usedtx=" + strconv.Itoa(int(codecParameters.Usedtx))
			}
			if codecParameters.Maxplaybackrate != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "maxplaybackrate=" + strconv.FormatUint(uint64(codecParameters.Maxplaybackrate), 10)
			}
			if codecParameters.XGoogleMinBitrate != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "x-google-min-bitrate=" + strconv.FormatUint(uint64(codecParameters.XGoogleMinBitrate), 10)
			}
			if codecParameters.XGoogleMaxBitrate != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "x-google-max-bitrate=" + strconv.FormatUint(uint64(codecParameters.XGoogleMaxBitrate), 10)
			}
			if codecParameters.XGoogleStartBitrate != 0 {
				if len(fmtp.Config) != 0 {
					fmtp.Config += ";"
				}
				fmtp.Config += "x-google-start-bitrate=" + strconv.FormatUint(uint64(codecParameters.XGoogleStartBitrate), 10)
			}

			if len(fmtp.Config) != 0 {
				m.mediaObject.Fmtp = append(m.mediaObject.Fmtp, fmtp)
			}

			for _, fb := range codec.RtcpFeedback {
				m.mediaObject.RtcpFb = append(m.mediaObject.RtcpFb, &transform.RtcpFbStruct{
					Payload: int(codec.PayloadType),
					Type:    fb.Type,
					Subtype: fb.Parameter,
				})
			}

			var payloads string

			for _, codec := range offerRtpParameters.Codecs {
				if len(payloads) != 0 {
					payloads = payloads + " "
				}

				payloads = payloads + strconv.Itoa(int(codec.PayloadType))
			}
			m.mediaObject.Payloads = payloads

			for _, ext := range offerRtpParameters.HeaderExtensions {
				// clang-format off
				m.mediaObject.Ext = append(m.mediaObject.Ext, &transform.ExtStruct{
					Uri:   ext.Uri,
					Value: ext.Id,
				})
			}
			m.mediaObject.RtcpMux = "rtcp-mux"
			m.mediaObject.RtcpRsize = "rtcp-rsize"

			encoding := offerRtpParameters.Encodings[0]
			ssrc := encoding.Ssrc

			var rtxSsrc uint32
			if encoding.Rtx != nil && encoding.Rtx.Ssrc != 0 {
				rtxSsrc = encoding.Rtx.Ssrc
			}

			if len(offerRtpParameters.Rtcp.Cname) != 0 {
				cname := offerRtpParameters.Rtcp.Cname

				msid := streamID
				msid = msid + " " + trackID

				m.mediaObject.Ssrcs = append(m.mediaObject.Ssrcs, &transform.SsrcStruct{
					Id:        uint(ssrc),
					Attribute: "cname",
					Value:     cname,
				})

				m.mediaObject.Ssrcs = append(m.mediaObject.Ssrcs, &transform.SsrcStruct{
					Id:        uint(ssrc),
					Attribute: "msid",
					Value:     msid,
				})

				if rtxSsrc != 0 {
					ssrcs := strconv.Itoa(int(ssrc)) + " " + strconv.Itoa(int(rtxSsrc))

					m.mediaObject.Ssrcs = append(m.mediaObject.Ssrcs, &transform.SsrcStruct{
						Id:        uint(rtxSsrc),
						Attribute: "cname",
						Value:     cname,
					})

					m.mediaObject.Ssrcs = append(m.mediaObject.Ssrcs, &transform.SsrcStruct{
						Id:        uint(rtxSsrc),
						Attribute: "msid",
						Value:     msid,
					})

					// Associate original and retransmission SSRCs.
					m.mediaObject.SsrcGroups = append(m.mediaObject.SsrcGroups, &transform.SsrcGroupStruct{
						Semantics: "FID",
						Ssrcs:     ssrcs,
					})
				}
			}

		}
	} else if kind == "application" {
		m.mediaObject.Payloads = "webrtc-datachannel"
		// m.mediaObject.SctpPort       = sctpParameters.Port
		// m.mediaObject.MaxMessageSize["maxMessageSize"] = sctpParameters["maxMessageSize"];
	}
}

// SetDtlsRole ...
func (m *OfferMediaSection) SetDtlsRole(role string) {
	m.mediaObject.Setup = "actpass"
}

// RemoteSdp ...
type RemoteSdp struct {
	iceParameters  mediasoup.IceParameters
	iceCandidates  []mediasoup.IceCandidate
	dtlsParameters mediasoup.DtlsParameters
	sctpParameters mediasoup.SctpParameters

	sdpObject      transform.SdpStruct
	mediaSections  []*AnswerMediaSection
	mediaSections2 []*OfferMediaSection
	firstMid       string
	MidToIndex     map[string]int
}

// MediaSectionIdx ...
type MediaSectionIdx struct {
	Idx      int
	ReuseMid string
}

// Init ...
func (s *RemoteSdp) Init() {
	s.MidToIndex = make(map[string]int)
	// s.iceCandidates = make([]mediasoup.IceCandidate, 0)
	s.mediaSections = make([]*AnswerMediaSection, 0)
	s.mediaSections2 = make([]*OfferMediaSection, 0)

	s.sdpObject = transform.SdpStruct{
		Version: 0,
		Origin: &transform.OriginStruct{
			Address:        "0.0.0.0",
			IpVer:          4,
			NetType:        "IN",
			SessionId:      "10000",
			SessionVersion: 0,
			Username:       "zhuanxin",
		},
		Name: "-",
		Timing: &transform.TimingStruct{
			Start: 0,
			Stop:  0,
		},
	}

	s.sdpObject.Groups = make([]*transform.GroupStruct, 0)
	s.sdpObject.Media = make([]*transform.MediaStruct, 0)

	if s.iceParameters.IceLite != false {
		s.sdpObject.Icelite = "ice-lite"
	}

	s.sdpObject.MsidSemantic = &transform.MsidSemanticStruct{
		Semantic: "WMS",
		Token:    "*",
	}

	numFingerprints := len(s.dtlsParameters.Fingerprints)
	s.sdpObject.Fingerprint = &transform.FingerprintStruct{
		Type: s.dtlsParameters.Fingerprints[numFingerprints-1].Algorithm,
		Hash: s.dtlsParameters.Fingerprints[numFingerprints-1].Value,
	}

	group := transform.GroupStruct{
		Type: "BUNDLE",
		Mids: "",
	}
	s.sdpObject.Groups = append(s.sdpObject.Groups, &group)
}

// UpdateDtlsRole ...
func (s *RemoteSdp) UpdateDtlsRole(role string) {
	s.dtlsParameters.Role = mediasoup.DtlsRole(role)

	if s.iceParameters.IceLite != false {
		s.sdpObject.Icelite = "ice-lite"
	}

	for idx, mediaSection := range s.mediaSections {
		mediaSection.SetDtlsRole(role)
		s.sdpObject.Media[idx] = mediaSection.GetObject()
	}
}

// AddMediaSection ...
func (s *RemoteSdp) AddMediaSection(newMediaSection *AnswerMediaSection) {
	if len(s.firstMid) == 0 {
		s.firstMid = newMediaSection.GetMid()
	}
	s.mediaSections = append(s.mediaSections, newMediaSection)

	s.MidToIndex[newMediaSection.GetMid()] = len(s.mediaSections) - 1

	s.sdpObject.Media = append(s.sdpObject.Media, newMediaSection.GetObject())
	s.RegenerateBundleMids()
}

// RegenerateBundleMids ...
func (s *RemoteSdp) RegenerateBundleMids() {
	var mids string
	for _, mediaSection := range s.mediaSections {
		if !mediaSection.IsClosed() {
			if len(mids) == 0 {
				mids = mediaSection.GetMid()
			} else {
				mids = mids + " " + mediaSection.GetMid()
			}
		}
	}
	s.sdpObject.Groups[0].Mids = mids
}

// Send ...
func (s *RemoteSdp) Send(offerMediaObject *transform.MediaStruct, reuseMid string, offerRtpParameters mediasoup.RtpParameters, answerRtpParameters mediasoup.RtpParameters, codecOptions CodecOptions) {
	mediaSection := &AnswerMediaSection{}
	mediaSection.Init(s.iceParameters, s.iceCandidates, s.dtlsParameters, s.sctpParameters, offerMediaObject, offerRtpParameters, answerRtpParameters, &codecOptions)
	if len(reuseMid) != 0 {
		// s.ReplaceMediaSection(mediaSection, reuseMid)
	} else {
		s.AddMediaSection(mediaSection)
	}
}

// Receive ...
func (s *RemoteSdp) Receive(mid string, kind string, offerRtpParameters mediasoup.RtpParameters, streamID string, trackID string) {
	mediaSection := &OfferMediaSection{}
	mediaSection.Init(s.iceParameters, s.iceCandidates, s.dtlsParameters, nil, mid, kind, offerRtpParameters, streamID, trackID)

	mediaSections2 := &AnswerMediaSection{}
	mediaSections2.m = mediaSection.m
	mediaSections2.mediaObject = mediaSection.mediaObject
	s.AddMediaSection(mediaSections2)
}

// GetSdp ...
func (s *RemoteSdp) GetSdp() (string, error) {
	version := s.sdpObject.Origin.SessionVersion
	s.sdpObject.Origin.SessionVersion = version + 1
	return transform.Write(&s.sdpObject)
}

// GetNextMediaSectionIdx ...
func (s *RemoteSdp) GetNextMediaSectionIdx() MediaSectionIdx {
	for idx, mediaSection := range s.mediaSections {
		if mediaSection.IsClosed() {
			return MediaSectionIdx{
				Idx:      idx,
				ReuseMid: mediaSection.GetMid(),
			}
		}
	}
	return MediaSectionIdx{
		Idx: len(s.mediaSections),
	}
}
