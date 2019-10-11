package redis

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/myadamtest/gobase/logkit"
	"github.com/myadamtest/gobase/resolver"
)

var (
	errNoUseablePool    = errors.New("redis-go: No useable pool!")
	errPoolNotAvailable = errors.New("redis_base: pool not available")
)

type redisPool struct {
	*redis.Pool
	id string
	fr failRetry
}

func (this *redisPool) getConn() (redis.Conn, error) {
	if !IsAvailable() {
		return nil, errPoolNotAvailable
	}

	conn := this.Pool.Get()

	if conn.Err() == nil {
		MarkSuccess()
		return conn, nil
	}

	MarkFail()
	logkit.Errorf("[redis]redis addr:%s has error:%s", this.id, conn.Err())
	return nil, conn.Err()
}

type redisPools struct {
	pools []*redisPool
	mux   sync.RWMutex
	index int32
}

func (s *redisPools) Get() (redis.Conn, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	lenPool := int32(len(s.pools))
	if lenPool == 0 {
		return nil, errNoUseablePool
	}
	index := atomic.AddInt32(&s.index, 1)
	pos := index % lenPool
	if pos != index {
		atomic.CompareAndSwapInt32(&s.index, index, pos)
	}

	for i := pos; i < pos+lenPool; i++ {
		tempPos := i % lenPool
		conn, err := s.pools[tempPos].getConn()
		if err == errPoolNotAvailable {
			continue
		}
		return conn, err
	}

	return nil, errNoUseablePool
}

func (s *redisPools) Del(update *resolver.Update) {
	s.mux.Lock()
	defer s.mux.Unlock()
	for k, pool := range s.pools {
		if pool.id == update.Id {
			pool.Close()
			s.pools = append(s.pools[:k], s.pools[k+1:]...)
			return
		}
	}
}

func (s *redisPools) Put(update *resolver.Update, opts *ConnOptions) {
	s.mux.Lock()
	defer s.mux.Unlock()
	p := newRedisPool(update, opts)
	for i, pool := range s.pools {
		if pool.id == update.Id {
			s.pools[i] = p
			pool.Close()
			return
		}
	}
	s.pools = append(s.pools, p)
}

func (s *redisPools) Close() {
	for _, pool := range s.pools {
		pool.Close()
	}
}

func newRedisPool(update *resolver.Update, opts *ConnOptions) *redisPool {
	logkit.Debugf("[redis]connect pool %s", update)
	urlOpts := parseOptions(update)
	connectTimeout := opts.ConnectTimeout
	if urlOpts.ConnectTimeout > 0 {
		connectTimeout = urlOpts.ConnectTimeout
	}
	readTimeout := opts.ReadTimeout
	if urlOpts.ReadTimeout > 0 {
		readTimeout = urlOpts.ReadTimeout
	}
	writeTimeout := opts.WriteTimeout
	if urlOpts.WriteTimeout > 0 {
		writeTimeout = urlOpts.WriteTimeout
	}
	idleTimeout := opts.IdleTimeout
	if urlOpts.IdleTimeout > 0 {
		idleTimeout = urlOpts.IdleTimeout
	}
	maxIdle := opts.MaxIdle
	if urlOpts.MaxIdle > 0 {
		maxIdle = urlOpts.MaxIdle
	}
	maxActive := opts.MaxActive
	if urlOpts.MaxActive > 0 {
		maxActive = urlOpts.MaxActive
	}
	waitConn := opts.WaitConn == 1
	if urlOpts.WaitConn != 0 {
		waitConn = urlOpts.WaitConn == 1
	}

	pool := &redisPool{
		Pool: &redis.Pool{
			MaxIdle:     maxIdle,
			MaxActive:   maxActive,
			IdleTimeout: idleTimeout,
			Wait:        waitConn,
			Dial: func() (redis.Conn, error) {
				c, err := redis.DialTimeout("tcp", update.Addr, connectTimeout, readTimeout, writeTimeout)
				if err != nil {
					logkit.Errorf("[redis]dial error %s", err)
					return nil, err
				}
				pwd := update.Password
				if pwd != "" {
					if _, err := c.Do("AUTH", pwd); err != nil {
						c.Close()
						logkit.Errorf("[redis]auth error %s", err)
						return nil, err
					}
				}
				return c, err
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				if err != nil {
					logkit.Errorf("[redis]TestOnBorrow %s", err)
					return err
				}
				return err
			},
		},
		id: update.Id,
		fr: newMaxFailTimeout(time.Second, 3),
	}
	return pool
}

func (p *redisPool) Close() {
	p.Pool.Close()
}
