package timing

import (
	"mgkj/pkg/log"
	"sync"
	"time"
)

type StreamState struct {
	RID        string `json:"rid"`
	UID        string `json:"uid"`
	AppID      string `json:"appid"`
	Resolution string `json:"resolution"`
	Seconds    int64  `json:"seconds"`
}

type StreamInfo struct {
	MID        string `json:"mid"`
	SID        string `json:"sid"`
	MediaType  string `json:"mediatype"`
	Resolution string `json:"resolution"`
}

type StreamTimer struct {
	*StreamState
	startTime       int64
	stopTime        int64
	stop            bool
	isDisconnected  bool //for peer disconnect cleaning
	lastResolustion string
	streams         []*StreamInfo
	streamslock     sync.Mutex
	mode            string
	lastmode        string
}

func NewStreamInfo(mid, sid, mediatype, resolution string) *StreamInfo {
	return &StreamInfo{
		MID:        mid,
		SID:        sid,
		MediaType:  mediatype,
		Resolution: resolution,
	}
}

func (si *StreamInfo) UpdateMediaType(mediatype string) {
	si.MediaType = mediatype
}

func (si *StreamInfo) UpdateResolution(resolution string) {
	si.Resolution = resolution
}

func NewStreamTimer(rid, uid, appid string) *StreamTimer {
	return &StreamTimer{
		StreamState: &StreamState{
			RID:     rid,
			UID:     uid,
			AppID:   appid,
			Seconds: 0,
		},
		mode: "audio",
	}
}

func (s *StreamTimer) GetStreamsCount() int {
	return len(s.streams)
}

func (s *StreamTimer) GetLastResolution() string {
	return s.lastResolustion
}

func (s *StreamTimer) GetCurrentResolution() string {
	return s.Resolution
}

func (s *StreamTimer) GetLastMode() string {
	return s.lastmode
}

func (s *StreamTimer) GetCurrentMode() string {
	return s.mode
}

func (s *StreamTimer) AddStream(stream *StreamInfo) bool {
	s.streamslock.Lock()
	defer s.streamslock.Unlock()

	s.streams = append(s.streams, stream)

	if stream.MediaType == "video" {
		s.lastmode = s.mode
		s.mode = "video"
		if s.lastmode == s.mode {
			return false
		} else {
			//reset resolution when audio change to video
			s.Resolution = ""
			s.lastResolustion = ""
			return true
		}
	}
	log.Infof("AddStream count:%d == %v mode:%s,lastmode:%s", len(s.streams), stream, s.mode, s.lastmode)
	return false
}

func (s *StreamTimer) RemoveStreamBySID(sid string) (*StreamInfo, bool) {
	s.streamslock.Lock()
	defer s.streamslock.Unlock()

	var vlen int
	var removed *StreamInfo
	var isModeChanged bool

	for idx, stream := range s.streams {
		if stream.MediaType == "video" {
			vlen++
		}
		if stream.SID == sid {
			vlen--
			removed = s.streams[idx]
			s.streams = append(s.streams[:idx], s.streams[idx+1:]...)
			log.Infof("RemoveStreamBySID delete sid:%s", stream.SID)
		}
	}

	if vlen == 0 {
		s.lastmode = s.mode
		s.mode = "audio"
		if s.lastmode == s.mode {
			isModeChanged = false
		} else {
			isModeChanged = true
		}
	}

	/*if s.mode == "audio" && s.lastmode == "audio" {
		//reset resolution
		s.Resolution = ""
		s.lastResolustion = ""
	}*/

	log.Infof("RemoveStreamBySID uid:%s,count:%d,lastmode:%s,mode:%s,isModeChanged:%v", s.UID, len(s.streams), s.lastmode, s.mode, isModeChanged)
	return removed, isModeChanged
}

func (s *StreamTimer) RemoveStreamByMID(mid string) ([]*StreamInfo, bool) {
	s.streamslock.Lock()
	defer s.streamslock.Unlock()

	var vlen int
	var isModeChanged bool
	var removedstreams []*StreamInfo
	for idx, stream := range s.streams {
		if stream.MediaType == "video" {
			vlen++
		}
		if stream.MID == mid {
			vlen--
			removed := s.streams[idx]
			s.streams = append(s.streams[:idx], s.streams[idx+1:]...)
			removedstreams = append(removedstreams, removed)
			log.Infof("RemoveStreamByMID delete mid:%s", stream.MID)
		}
	}

	if vlen == 0 {
		s.lastmode = s.mode
		s.mode = "audio"
		if s.lastmode == s.mode {
			isModeChanged = false
		} else {
			isModeChanged = true
		}
	}

	/*if s.mode == "audio" && s.lastmode == "audio" {
		//reset resolution
		s.Resolution = ""
		s.lastResolustion = ""
	}*/

	log.Infof("RemoveStreamByMID uid:%s,count:%d,lastmode:%s,mode:%s,isModeChanged:%v", s.UID, len(s.streams), s.lastmode, s.mode, isModeChanged)
	return removedstreams, isModeChanged
}

//when add or remove stream,must call this func to update total resolution
func (s *StreamTimer) UpdateResolution() bool {
	sLen := len(s.streams)
	if sLen >= 1 {
		log.Infof("111:%s", s.Resolution)
		s.lastResolustion = s.Resolution
		log.Infof("222:%s", s.lastResolustion)
		var totalPixels uint64
		for _, stream := range s.streams {
			mediaType := stream.MediaType
			//only calc resolution when mediaType is video
			if mediaType == "video" {
				if sLen == 1 {
					log.Infof("333:%s", stream.Resolution)
					s.Resolution = stream.Resolution
					totalPixels = getPixelsByResolution(s.Resolution)
					log.Infof("444:%s", s.Resolution)
				} else if sLen > 1 {
					totalPixels += getPixelsByResolution(stream.Resolution)
					log.Infof("555:%s", totalPixels)
				}
			}
		}
		s.Resolution = CalcPixelsToResolution(totalPixels)
		log.Infof("666:%s", s.lastResolustion, s.Resolution)
		if s.lastResolustion == s.Resolution {
			return false
		} else {
			return true
		}
	} else {
		//there is no subscribed stream any more,reset resolution
		log.Infof("777:%s--%s", s.lastResolustion, s.Resolution)
		s.lastResolustion = s.Resolution
		s.Resolution = ""
		log.Infof("888:%s--%s", s.lastResolustion, s.Resolution)
		return true
	}
}

func (s *StreamTimer) Start() {
	if s.stop == false {
		s.startTime = time.Now().Unix()
		log.Infof("RID = %s UID = %s stream timer start, count:%d", s.RID, s.UID, len(s.streams))
	}
}

func (s *StreamTimer) Stop() {
	if s.stop == false {
		s.stopTime = time.Now().Unix()
		s.stop = true
		s.Seconds = s.stopTime - s.startTime
		log.Infof("RID = %s UID = %s stream timer stop and ran %d seconds, count:%d", s.RID, s.UID, s.Seconds, len(s.streams))
	}
}

func (s *StreamTimer) Renew() {
	s.stop = false
	s.startTime = time.Now().Unix()
	s.Seconds = 0
	log.Infof("RID = %s UID = %s stream timer renew,count:%d", s.RID, s.UID, len(s.streams))
}

func (s *StreamTimer) GetTotalSeconds() int64 {
	return s.Seconds
}

func (s *StreamTimer) IsStopped() bool {
	return s.stop
}
