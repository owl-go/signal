package monitor

import (
	"time"
)

type ProcessingTimeGauge struct {
	name        string
	startTime   time.Time
	processTime time.Duration
	stop        bool
}

func NewProcessingTimeGauge(name string) *ProcessingTimeGauge {
	return &ProcessingTimeGauge{
		name: name,
	}
}

func (ptg *ProcessingTimeGauge) Start() {
	if ptg.stop == false {
		ptg.startTime = time.Now()
	}
}

func (ptg *ProcessingTimeGauge) Stop() {
	if ptg.stop == false {
		ptg.stop = true
		ptg.processTime = time.Now().Sub(ptg.startTime) / time.Millisecond
	}
}

func (ptg *ProcessingTimeGauge) GetName() string {
	return ptg.name
}

func (ptg *ProcessingTimeGauge) GetDuration() float64 {
	return float64(ptg.processTime)
}
