package mysql

import (
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/myadamtest/gobase/logkit"
	"github.com/myadamtest/gobase/resolver"
)

var (
	ErrNoUseableDB = errors.New("no usealbe mysql")
)

type mysqlEndpoint struct {
	*sql.DB
	id     string
	active bool
	closed bool
}

type mysqlDB struct {
	dbname    string
	endpoints []*mysqlEndpoint
	mux       sync.RWMutex
	index     int32
}

func (db *mysqlDB) Get() (*mysqlEndpoint, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()
	len := int32(len(db.endpoints))
	if len == 0 {
		logkit.Error("len(db.endpoints) == 0")
		return nil, ErrNoUseableDB
	}
	index := atomic.AddInt32(&db.index, 1)
	pos := index % len
	if pos != index {
		atomic.CompareAndSwapInt32(&db.index, index, pos)
	}
	for i := pos; i < pos+len; i++ {
		j := i % len
		if e := db.endpoints[j]; e.active {
			if err := e.Ping(); err != nil {
				e.active = false
				logkit.Error(err.Error())
				return nil, err
			}
			return e, nil
		}
	}
	return nil, ErrNoUseableDB
}

func (db *mysqlDB) Del(update *resolver.Update) {
	db.mux.Lock()
	defer db.mux.Unlock()
	for k, e := range db.endpoints {
		if e.id == update.Id {
			e.Close()
			db.endpoints = append(db.endpoints[:k], db.endpoints[k+1:]...)
			return
		}
	}
}

func (db *mysqlDB) Put(update *resolver.Update, opts *ConnOptions) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	endpoint, err := newMysqlEndpoint(db.dbname, update, opts)
	if err != nil {
		logkit.Error(err.Error())
		return err
	}
	for i, e := range db.endpoints {
		if e.id == update.Id {
			db.endpoints[i] = endpoint
			e.Close()
			return nil
		}
	}
	db.endpoints = append(db.endpoints, endpoint)
	return nil
}

func (db *mysqlDB) Close() {
	for _, e := range db.endpoints {
		e.Close()
	}
}

func (e *mysqlEndpoint) Close() {
	e.closed = true
	e.DB.Close()
}

func newMysqlEndpoint(dbname string, update *resolver.Update, opts *ConnOptions) (*mysqlEndpoint, error) {
	logkit.Debugf("[mysql]connect db %s", update)
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
	config := driver.Config{}
	config.DBName = dbname
	if config.DBName == "" {
		return nil, errors.New("dbname is not defiend")
	}
	config.Net = "tcp"
	config.Addr = update.Addr
	config.User = update.User
	config.Passwd = update.Password
	config.Timeout = connectTimeout
	config.WriteTimeout = writeTimeout
	config.ReadTimeout = readTimeout
	dsn := config.FormatDSN()
	logkit.Debugf("[mysql]connect dsn:%s", dsn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logkit.Error(err.Error())
		return nil, err
	}
	db.SetMaxOpenConns(maxActive)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(idleTimeout)
	endpoint := &mysqlEndpoint{
		DB:     db,
		id:     update.Id,
		active: true,
	}

	go endpoint.checkActive()
	return endpoint, nil
}

func (e *mysqlEndpoint) checkActive() {
	for _ = range time.Tick(time.Second * 2) {
		if e.closed {
			return
		}
		if !e.active {
			if err := e.Ping(); err == nil {
				e.active = true
			}
		}
	}
}
