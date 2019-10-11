package mysql

import (
	"fmt"

	"github.com/myadamtest/gobase/logkit"
	"github.com/myadamtest/gobase/resolver"
)

type Client struct {
	opts    *ConnOptions
	writeDB *mysqlDB
	readDB  *mysqlDB
	dbname  string
	watcher resolver.Watcher
}

func NewClient(dbname, target string, opts ...ConnOption) (*Client, error) {
	updates, watcher := resolver.ResolveTarget(target)
	if len(updates) == 0 {
		return nil, fmt.Errorf("resolve target failed:%s", target)
	}
	return newClient(dbname, updates, watcher, opts)
}

func newClient(dbname string, updates []*resolver.Update, watcher resolver.Watcher, opts []ConnOption) (*Client, error) {
	if len(updates) == 0 {
		return nil, fmt.Errorf("endpoint is empty ")
	}
	cli := new(Client)
	cli.opts = &defaultOpts
	cli.dbname = dbname
	for _, opt := range opts {
		opt(cli.opts)
	}
	for _, update := range updates {
		if update.Master {
			if cli.writeDB == nil {
				cli.writeDB = &mysqlDB{
					dbname: dbname,
				}
			}
			err := cli.writeDB.Put(update, cli.opts)
			if err != nil {
				logkit.Error(err.Error())
				return nil, fmt.Errorf("new db with endpoint:%s,err:%s", update, err)
			}
		} else {
			if cli.readDB == nil {
				cli.readDB = &mysqlDB{
					dbname: dbname,
				}
			}
			err := cli.readDB.Put(update, cli.opts)
			if err != nil {
				logkit.Error(err.Error())
				return nil, fmt.Errorf("new db with endpoint:%s,err:%s", update, err)
			}
		}
	}

	if watcher != nil {
		cli.watcher = watcher
		go func() {
			for {
				if err := cli.watch(); err != nil {
					logkit.Errorf("[mysql]watch next err:%s", err)
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
	logkit.Debugf("[mysql]watch update:%s", update)
	if update == nil || update.Id == "" {
		return nil
	}
	switch update.Op {
	case resolver.OP_Delete:
		if update.Master {
			if cli.writeDB != nil {
				cli.writeDB.Del(update)
			}
		} else {
			if cli.readDB != nil {
				cli.readDB.Del(update)
			}
		}
	case resolver.OP_Put:
		if update.Master {
			if cli.writeDB == nil {
				cli.writeDB = &mysqlDB{
					dbname: cli.dbname,
				}
			}
			err := cli.writeDB.Put(update, cli.opts)
			if err != nil {
				logkit.Errorf("[mysql]put endpoint err:%s", err)
			}
		} else {
			if cli.readDB == nil {
				cli.readDB = &mysqlDB{
					dbname: cli.dbname,
				}
			}
			err := cli.readDB.Put(update, cli.opts)
			if err != nil {
				logkit.Errorf("[mysql]put endpoint err:%s", err)
			}
		}
	}
	return nil
}

func (cli *Client) Close() {
	if cli.watcher != nil {
		err := cli.watcher.Close()
		if err != nil {
			logkit.Errorf("[mysql]close watcher error:%s", err)
		}
	}
	if cli.writeDB != nil {
		cli.writeDB.Close()
	}
	if cli.readDB != nil {
		cli.readDB.Close()
	}
}
