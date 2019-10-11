package mysql

import (
	"fmt"
	"github.com/myadamtest/gobase/logkit"

	"github.com/myadamtest/gobase/resolver"
)

type ShardClient struct {
	shardFunc        ShardFunc
	clients          []*Client
	dispatchWatchers []*dispatchWatcher
}

func NewShardClient(dbname string, configs []*ShardConfig, shardFunc ShardFunc) (*ShardClient, error) {
	if shardFunc == nil {
		return nil, fmt.Errorf("shardfunc must not be nil")
	}
	maxEnd := 0
	for _, config := range configs {
		if config.ShardStart < 0 || config.ShardEnd < 0 || config.ShardEnd < config.ShardStart {
			return nil, fmt.Errorf("invaind shard config:%s", config)
		}
		if config.ShardEnd > maxEnd {
			maxEnd = config.ShardEnd
		}
	}
	cli := &ShardClient{
		shardFunc: shardFunc,
		clients:   make([]*Client, maxEnd, maxEnd),
	}
	for _, config := range configs {
		updates, watcher := resolver.ResolveTarget(config.Target)
		if len(updates) == 0 {
			return nil, fmt.Errorf("resolve target failed:%s", config.Target)
		}
		var dispatchWatcher *dispatchWatcher
		if watcher != nil {
			dispatchWatcher = newDispatchWatcher(watcher)
			cli.dispatchWatchers = append(cli.dispatchWatchers, dispatchWatcher)
		}
		for i := config.ShardStart; i <= config.ShardEnd; i++ {
			dbname := fmt.Sprintf("%s_%d", dbname, i)
			var proxyWatcher resolver.Watcher
			if dispatchWatcher != nil {
				proxyWatcher = dispatchWatcher.proxy(i)
			}
			c, err := newClient(dbname, updates, proxyWatcher, config.Opts)
			if err != nil {
				return nil, fmt.Errorf("init client err:%s,config:%s", err, config)
			}
			cli.clients[i] = c
		}
		if dispatchWatcher != nil {
			go dispatchWatcher.watch()
		}
	}
	return cli, nil
}

type ShardConfig struct {
	Target     string
	ShardStart int
	ShardEnd   int
	Opts       []ConnOption
}

func (cli *ShardClient) Close() {
	for _, c := range cli.clients {
		if c != nil {
			c.Close()
		}
	}
	for _, watcher := range cli.dispatchWatchers {
		watcher.Close()
	}
}

func (cli *ShardClient) Insert(key string, sqlstr string, args ...interface{}) (int64, error) {
	c, err := cli.getClient(key)
	if err != nil {
		logkit.Error(err.Error())
		return 0, err
	}
	return c.Insert(sqlstr, args)
}

func (cli *ShardClient) Update(key string, sqlstr string, args ...interface{}) (int64, error) {
	c, err := cli.getClient(key)
	if err != nil {
		logkit.Error(err.Error())
		return 0, err
	}
	return c.Update(sqlstr, args)
}

func (cli *ShardClient) Delete(key string, sqlstr string, args ...interface{}) (int64, error) {
	c, err := cli.getClient(key)
	if err != nil {
		logkit.Error(err.Error())
		return 0, err
	}
	return c.Delete(sqlstr, args)
}

func (cli *ShardClient) FetchRow(key string, sqlstr string, args ...interface{}) (map[string]string, error) {
	c, err := cli.getClient(key)
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}
	return c.FetchRow(sqlstr, args)
}

func (cli *ShardClient) FetchRowForMaster(key string, sqlstr string, args ...interface{}) (map[string]string, error) {
	c, err := cli.getClient(key)
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}
	return c.FetchRowForMaster(sqlstr, args)
}

func (cli *ShardClient) FetchRows(key string, sqlstr string, args ...interface{}) ([]map[string]string, error) {
	c, err := cli.getClient(key)
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}
	return c.FetchRows(sqlstr, args)
}

func (cli *ShardClient) FetchRowsForMaster(key string, sqlstr string, args ...interface{}) ([]map[string]string, error) {
	c, err := cli.getClient(key)
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}
	return c.FetchRowsForMaster(sqlstr, args)
}

func (cli *ShardClient) getClient(key string) (*Client, error) {
	index := cli.shardFunc(key)
	if index < 0 || index >= len(cli.clients) {
		return nil, fmt.Errorf("invaind shard index:%d", index)
	}
	c := cli.clients[index]
	if c == nil {
		return nil, fmt.Errorf("client of key:%d,index:%d is not exist", key, index)
	}
	return c, nil
}
