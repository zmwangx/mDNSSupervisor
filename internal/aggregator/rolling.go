package aggregator

import (
	log "github.com/sirupsen/logrus"
)

type RollingAggregator struct {
	interval         uint
	ch               chan int64
	secbuf           []uint
	secidx           uint
	currentTimestamp int64
	currentSecCount  uint
	rollingAggregate uint
}

type ActionFunc func(timestamp int64, rollingAverage float64)

func NewRollingAggregator(interval uint) *RollingAggregator {
	return &RollingAggregator{
		interval:         interval,
		ch:               make(chan int64, 100),
		secbuf:           make([]uint, interval),
		secidx:           0,
		currentTimestamp: 0,
		currentSecCount:  0,
		rollingAggregate: 0,
	}
}

func (a *RollingAggregator) Send(timestamp int64) {
	a.ch <- timestamp
}

func (a *RollingAggregator) Process(threshold uint, action ActionFunc) {
	for timestamp := range a.ch {
		if timestamp < a.currentTimestamp {
			// Discard as illegal
			log.Warnf("illegal timestamp: %d < %d", timestamp, a.currentTimestamp)
			continue
		}
		if timestamp > a.currentTimestamp+int64(a.interval) {
			// The last interval was completely quiet.
			a.reset(timestamp)
		}
		sameSecond := timestamp == a.currentTimestamp
		for timestamp > a.currentTimestamp {
			a.advanceSecond()
		}
		if !sameSecond {
			// Check rolling average against threshold.
			if a.rollingAggregate > threshold*a.interval {
				rollingAverage := float64(a.rollingAggregate) / float64(a.interval)
				log.Infof("%d: %ds rolling average %.1f, above threshold %d",
					timestamp, a.interval, rollingAverage, threshold)
				go action(timestamp, rollingAverage)
				a.reset(timestamp)
			}
		}
		a.currentSecCount++
	}
}

func (a *RollingAggregator) reset(newTimestamp int64) {
	log.Debugf("%d: rolling buffer reset (last timestamp %d)", newTimestamp, a.currentTimestamp)
	for i := uint(0); i < a.interval; i++ {
		a.secbuf[i] = 0
	}
	a.secidx = 0
	a.currentTimestamp = newTimestamp
	a.currentSecCount = 0
	a.rollingAggregate = 0
}

func (a *RollingAggregator) advanceSecond() {
	a.rollingAggregate -= a.secbuf[a.secidx]
	a.secbuf[a.secidx] = a.currentSecCount
	a.rollingAggregate += a.currentSecCount
	log.Debugf("%d: %d hits this second, %ds rolling aggregate %d",
		a.currentTimestamp, a.currentSecCount, a.interval, a.rollingAggregate)
	a.currentSecCount = 0
	a.secidx = (a.secidx + 1) % a.interval
	a.currentTimestamp++
}
