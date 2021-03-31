package timing

import (
	"mgkj/pkg/log"
	"strings"
	"time"
)

type StreamState struct {
	RID        string `json:"rid"`
	UID        string `json:"uid"`
	MID        string `json:"mid"`
	SID        string `json:"sid"`
	AppID      string `json:"appid"`
	MediaType  string `json:"mediatype"`
	Resolution string `json:"resolution"`
	Seconds    int64  `json:"seconds"`
}

type StreamTimer struct {
	*StreamState
	startTime int64
	stopTime  int64
	stop      bool
}

func NewStreamTimer(rid, mid, sid, appid, resolution, mediatype string) *StreamTimer {
	return &StreamTimer{
		StreamState: &StreamState{
			RID:        rid,
			UID:        strings.Split(sid, "#")[0],
			MID:        mid,
			SID:        sid,
			AppID:      appid,
			MediaType:  mediatype,
			Resolution: resolution,
			Seconds:    0,
		},
	}
}

func (s *StreamTimer) UpdateStreamResolution(resolution string) {
	s.Resolution = resolution
}

func (s *StreamTimer) Start() {
	if s.stop == false {
		s.startTime = time.Now().Unix()
		log.Infof("MID => %s SID => %s stream timer start", s.MID, s.SID)
	}
}

func (s *StreamTimer) Stop() {
	if s.stop == false {
		s.stopTime = time.Now().Unix()
		s.stop = true
		s.Seconds = s.stopTime - s.startTime
		log.Infof("MID => %s SID => %s stream timer closed and ran %d seconds", s.MID, s.SID, s.Seconds)
	}
}

func (s *StreamTimer) GetTotalTime() int64 {
	return s.Seconds
}

func (s *StreamTimer) GetState() bool {
	return s.stop
}
