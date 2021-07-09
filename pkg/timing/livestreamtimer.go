package timing

import (
	"signal/pkg/log"
	"time"
)

type LiveStreamTimer struct {
	*StreamState
	startTime int64
	stopTime  int64
	stop      bool
	mode      string
}

func NewLiveStreamTimer(rid, uid, appid, resolution string) *LiveStreamTimer {
	return &LiveStreamTimer{
		StreamState: &StreamState{
			RID:        rid,
			UID:        uid,
			AppID:      appid,
			Seconds:    0,
			Resolution: resolution,
		},
		mode: "video",
	}
}

func (s *LiveStreamTimer) Start() {
	if s.stop == false {
		s.startTime = time.Now().Unix()
		log.Infof("RID = %s UID = %s live stream timer start", s.RID, s.UID)
	}
}

func (s *LiveStreamTimer) Stop() {
	if s.stop == false {
		s.stopTime = time.Now().Unix()
		s.stop = true
		s.Seconds = s.stopTime - s.startTime
		log.Infof("RID = %s UID = %s live stream timer stop and ran %d seconds, count:%d", s.RID, s.UID, s.Seconds)
	}
}

func (s *LiveStreamTimer) GetTotalSeconds() int64 {
	return s.Seconds
}

func (s *LiveStreamTimer) GetMode() string {
	return s.mode
}

func (s *LiveStreamTimer) IsStopped() bool {
	return s.stop
}
