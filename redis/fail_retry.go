package redis

import (
	"container/ring"
	"sync"
	"time"
)

type failRetry interface {
	IsAvailable() bool
	MarkFail()
	MarkSuccess()
}

type maxFailTimeout struct {
	failTimeout time.Duration
	maxFails    int
	mux         *sync.RWMutex

	failRecord       *ring.Ring
	lastMarkFailTime time.Time
}

func newMaxFailTimeout(ft time.Duration, mf int) *maxFailTimeout {
	t := time.Now().Add(-1 * ft)

	mft := &maxFailTimeout{
		failTimeout:      ft,
		maxFails:         mf,
		mux:              new(sync.RWMutex),
		failRecord:       ring.New(mf),
		lastMarkFailTime: t,
	}

	for i := 0; i < mf; i++ {
		mft.failRecord.Value = t
		mft.failRecord = mft.failRecord.Next()
	}

	return mft
}

func (this *maxFailTimeout) IsAvailable() bool {
	this.mux.RLock()
	defer this.mux.RUnlock()

	return time.Now().Sub(this.lastMarkFailTime) > this.failTimeout
}

func (this *maxFailTimeout) MarkSuccess() {
	this.mux.Lock()
	defer this.mux.Unlock()

	t := time.Now().Add(-1 * this.failTimeout)

	if t.Before(this.lastMarkFailTime) {
		return
	}

	var record *ring.Ring
	record = this.failRecord.Prev()

	for i := 0; i < this.maxFails; i++ {
		nt := record.Value.(time.Time)
		if t.After(nt) {
			break
		} else {
			record.Value = t
			record = record.Prev()
		}
	}
}

func (this *maxFailTimeout) MarkFail() {
	this.mux.Lock()
	defer this.mux.Unlock()

	t := time.Now()
	if t.Sub(this.lastMarkFailTime) < this.failTimeout {
		return
	}

	firstt := this.failRecord.Next().Value.(time.Time)
	if t.Before(firstt.Add(this.failTimeout)) {
		this.lastMarkFailTime = t
	}

	this.failRecord.Value = t
	this.failRecord = this.failRecord.Next()
}
