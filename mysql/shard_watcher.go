package mysql

import (
	"errors"

	"github.com/myadamtest/gobase/logkit"
	"github.com/myadamtest/gobase/resolver"
)

type dispatchWatcher struct {
	watcher resolver.Watcher
	proxies map[int]*proxyWatcher
}

func newDispatchWatcher(watcher resolver.Watcher) *dispatchWatcher {
	d := &dispatchWatcher{
		watcher: watcher,
		proxies: make(map[int]*proxyWatcher),
	}
	return d
}

func (d *dispatchWatcher) Close() {
	d.watcher.Close()
	for _, proxy := range d.proxies {
		proxy.Close()
	}
}

func (d *dispatchWatcher) proxy(i int) resolver.Watcher {
	proxy := &proxyWatcher{
		Index: i,
	}
	d.proxies[i] = proxy
	return proxy
}

func (d *dispatchWatcher) watch() {
	for {
		update, err := d.watcher.Next()
		if err != nil {
			logkit.Errorf("[mysql]dispatcheWatcher watch next err:%s", err)
			return
		}
		for _, proxy := range d.proxies {
			if proxy.closed {
				close(proxy.ch)
			} else {
				proxy.ch <- update
			}
		}
	}
}

type proxyWatcher struct {
	Index  int
	closed bool
	ch     chan *resolver.Update
}

func (w *proxyWatcher) Next() (*resolver.Update, error) {
	if w.closed {
		return nil, errors.New("proxywatcher has been closed")
	}
	e, ok := <-w.ch
	if !ok {
		return nil, errors.New("proxywatcher channel has been closed")
	}
	return e, nil
}

func (w *proxyWatcher) Close() error {
	w.closed = true
	return nil
}
