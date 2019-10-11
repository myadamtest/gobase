package redis

import (
	"strconv"
	"time"

	"github.com/myadamtest/gobase/resolver"
)

type ConnOption func(*ConnOptions)

type ConnOptions struct {
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxActive      int
	MaxIdle        int
	WaitConn       int
}

func ConnectTimeout(v time.Duration) ConnOption {
	return func(o *ConnOptions) {
		o.ConnectTimeout = v
	}
}

func ReadTimeout(v time.Duration) ConnOption {
	return func(o *ConnOptions) {
		o.ReadTimeout = v
	}
}

func WriteTimeout(v time.Duration) ConnOption {
	return func(o *ConnOptions) {
		o.WriteTimeout = v
	}
}

func IdleTimeout(v time.Duration) ConnOption {
	return func(o *ConnOptions) {
		o.IdleTimeout = v
	}
}

func MaxActive(v int) ConnOption {
	return func(o *ConnOptions) {
		o.MaxActive = v
	}
}

func MaxIdle(v int) ConnOption {
	return func(o *ConnOptions) {
		o.MaxIdle = v
	}
}

func WaitConn(v bool) ConnOption {
	return func(o *ConnOptions) {
		if v {
			o.WaitConn = 1
		} else {
			o.WaitConn = -1
		}
	}
}

var defaultOpts = ConnOptions{
	ConnectTimeout: 2 * time.Second,
	ReadTimeout:    500 * time.Millisecond,
	WriteTimeout:   500 * time.Millisecond,
	IdleTimeout:    60 * time.Second,
	MaxActive:      50,
	MaxIdle:        20,
	WaitConn:       1,
}

func parseOptions(update *resolver.Update) *ConnOptions {
	options := &ConnOptions{}
	for k, v := range update.Options {
		if k != "" && v != "" {
			switch k {
			case "connect_timeout":
				i, err := strconv.Atoi(v)
				if err == nil && i > 0 {
					options.ConnectTimeout = time.Duration(i) * time.Millisecond
				}
			case "write_timeout":
				i, err := strconv.Atoi(v)
				if err == nil && i > 0 {
					options.WriteTimeout = time.Duration(i) * time.Millisecond
				}
			case "read_timeout":
				i, err := strconv.Atoi(v)
				if err == nil && i > 0 {
					options.ReadTimeout = time.Duration(i) * time.Millisecond
				}
			case "idle_timeout":
				i, err := strconv.Atoi(v)
				if err == nil && i > 0 {
					options.IdleTimeout = time.Duration(i) * time.Millisecond
				}
			case "max_idle":
				i, err := strconv.Atoi(v)
				if err == nil && i > 0 {
					options.MaxIdle = i
				}
			case "max_active":
				i, err := strconv.Atoi(v)
				if err == nil && i > 0 {
					options.MaxActive = i
				}
			case "wait_conn":
				if v == "true" {
					options.WaitConn = 1
				} else if v == "false" {
					options.WaitConn = -1
				}
			}
		}
	}
	return options
}
