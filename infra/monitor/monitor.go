package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
)

type monitor struct {
}

func NewMonitorGauge(name, help string, labels []string) *prometheus.GaugeVec {
	duration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labels,
	)
	prometheus.MustRegister(duration)
	return duration
}

func NewMonitorCounter(name, help string, labels []string) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels,
	)
	prometheus.MustRegister(counter)
	return counter
}
