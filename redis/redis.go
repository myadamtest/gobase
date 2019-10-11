package redis

import (
	"errors"
	"fmt"

	"github.com/myadamtest/gobase/logkit"
	"github.com/myadamtest/gobase/resolver"
)

var (
	ErrInvaildPools = errors.New("invaild pools")
)

type Client struct {
	opts           *ConnOptions
	masterPools    *redisPools
	slavePools     *redisPools
	readFromMaster bool
	watcher        resolver.Watcher
}

func NewClient(target string, opts ...ConnOption) (*Client, error) {
	updates, watcher := resolver.ResolveTarget(target)
	if len(updates) == 0 {
		return nil, fmt.Errorf("resolve target faild:%s", target)
	}
	cli := new(Client)
	cli.opts = &defaultOpts

	for _, opt := range opts {
		opt(cli.opts)
	}

	for _, update := range updates {
		if update.Master {
			if cli.masterPools == nil {
				cli.masterPools = &redisPools{}
			}
			cli.masterPools.Put(update, cli.opts)
		} else {
			if cli.slavePools == nil {
				cli.slavePools = &redisPools{}
			}
			cli.slavePools.Put(update, cli.opts)
		}
	}

	if watcher != nil {
		cli.watcher = watcher
		go func() {
			for {
				if err := cli.watch(); err != nil {
					logkit.Errorf("[redis]watch next err:%s", err)
					return
				}
			}
		}()
	}

	return cli, nil
}

func (cli *Client) watch() error {
	update, err := cli.watcher.Next()
	if err != nil {
		return err
	}
	logkit.Infof("[redis]watch update:%s", update)
	if update == nil {
		return nil
	}
	switch update.Op {
	case resolver.OP_Delete:
		if update.Master {
			if cli.masterPools != nil {
				cli.masterPools.Del(update)
			}
		} else {
			if cli.slavePools != nil {
				cli.slavePools.Del(update)
			}
		}
	case resolver.OP_Put:
		if update.Master {
			if cli.masterPools == nil {
				cli.masterPools = &redisPools{}
			}
			cli.masterPools.Put(update, cli.opts)
		} else {
			if cli.slavePools == nil {
				cli.slavePools = &redisPools{}
			}
			cli.slavePools.Put(update, cli.opts)
		}
	}
	return nil
}

func (cli *Client) Close() {
	if cli.watcher != nil {
		err := cli.watcher.Close()
		if err != nil {
			logkit.Errorf("[redis]close redis janna watch error:%s", err)
		}
	}
	if cli.masterPools != nil {
		cli.masterPools.Close()
	}
	if cli.slavePools != nil {
		cli.slavePools.Close()
	}
}

func (cli *Client) doWrite(cmd string, args ...interface{}) (reply interface{}, err error) {
	if cli.masterPools == nil {
		return nil, ErrInvaildPools
	}
	conn, err := cli.masterPools.Get()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return conn.Do(cmd, args...)
}

func (cli *Client) doRead(cmd string, args ...interface{}) (reply interface{}, err error) {
	if cli.readFromMaster || cli.slavePools == nil {
		return cli.doWrite(cmd, args...)
	}
	conn, err := cli.slavePools.Get()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return conn.Do(cmd, args...)
}

func (cli *Client) ReadFromMaster(b bool) {
	cli.readFromMaster = b
}
