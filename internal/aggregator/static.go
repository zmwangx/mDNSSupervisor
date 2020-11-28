package aggregator

import (
	log "github.com/sirupsen/logrus"
)

type StaticAggregator struct {
	interval               uint
	ch                     chan int64
	intervalStartTimestamp int64
	intervalAggregate      uint
}

type ReportingFunc func(timestamp int64, aggregate uint)

func NewStaticAggregator(interval uint) *StaticAggregator {
	return &StaticAggregator{
		interval:               interval,
		ch:                     make(chan int64, 100),
		intervalStartTimestamp: 0,
		intervalAggregate:      0,
	}
}

func (a *StaticAggregator) Send(timestamp int64) {
	a.ch <- timestamp
}

func (a *StaticAggregator) Process(report ReportingFunc) {
	interval := int64(a.interval)
	// The reportit function should be called on exit (including SIGINT/SIGTERM)
	// to flush stats of the final minute. For simplicity that's not the case
	// right now, but it might change in the future.
	reportit := func() {
		if a.intervalStartTimestamp != 0 {
			log.Debugf("%d-%d: %ds aggregate %d",
				a.intervalStartTimestamp, a.intervalStartTimestamp+interval, interval, a.intervalAggregate)
			go report(a.intervalStartTimestamp, a.intervalAggregate)
		}
	}

	for timestamp := range a.ch {
		if timestamp < a.intervalStartTimestamp {
			// Discard as illegal
			log.Warnf("illegal timestamp: %d < %d", timestamp, a.intervalStartTimestamp)
			continue
		}
		intervalStartTimestamp := timestamp - (timestamp%interval+interval)%interval
		if intervalStartTimestamp > a.intervalStartTimestamp {
			reportit()
			a.intervalStartTimestamp = intervalStartTimestamp
			a.intervalAggregate = 0
		}
		a.intervalAggregate++
	}
}
