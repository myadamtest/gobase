package shaco

import (
	"fmt"
	"strings"
	"time"

	etcd3 "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/myadamtest/gobase/logkit"
	"golang.org/x/net/context"
)

var (
	client       *etcd3.Client
	serviceKey   string
	serviceValue string
)

var stopSignal = make(chan bool, 1)

func register(target, serviceName, host string, port, weight int, interval time.Duration, ttl int) error {
	logkit.Debugf("[I][shaco] register:target:%s,serviceName:%s,host:%s,port:%d", target, serviceName, host, port)
	serviceValue = fmt.Sprintf("{\"Address\":\"%s\",\"Port\":%d,\"Weight\":%d}", host, port, weight)
	serviceKey = fmt.Sprintf("/rpc/%s/%s", serviceName, fmt.Sprintf("%s:%d", host, port))

	var err error
	l := strings.Split(target, "@")
	var endpoints string
	username := "rpc"
	password := "3VM4VAGV3IhF6Z2H"

	if len(l) == 2 {
		endpoints = l[1]
		up := strings.Split(l[0], ":")
		if len(up) == 2 {
			username = up[0]
			password = up[1]
		} else {
			username = l[0]
			password = ""
		}
	} else {
		endpoints = target
	}
	client, err = etcd3.New(etcd3.Config{
		DialTimeout: 2 * time.Second,
		Endpoints:   strings.Split(endpoints, ","),
		Username:    username,
		Password:    password,
	})
	if err != nil {
		return fmt.Errorf("grpclb: create etcd3 client failed: %v", err)
	}

	go func() {
		ticker := time.NewTicker(interval)
		for {
			ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
			resp, err := client.Grant(ctx, int64(ttl))
			if err != nil {
				logkit.Debugf("[E][shaco-server] register: service '%s' grant failed: %s", serviceName, err)
			} else {
				ctx, _ = context.WithTimeout(context.Background(), 2*time.Second)
				_, err = client.Get(ctx, serviceKey)
				if err != nil {
					if err == rpctypes.ErrKeyNotFound {
						ctx, _ = context.WithTimeout(context.Background(), 2*time.Second)
						if _, err = client.Put(ctx, serviceKey, serviceValue, etcd3.WithLease(resp.ID)); err != nil {
							logkit.Debugf("[E][shaco-server] grpc register: set service '%s' with ttl to target failed: %s", serviceName, err)
						}
					} else {
						logkit.Debugf("[E][shaco-server] grpc register: service '%s' connect to etcd3 failed: %s", serviceName, err)
					}
				} else {
					ctx, _ = context.WithTimeout(context.Background(), 2*time.Second)
					if _, err = client.Put(ctx, serviceKey, serviceValue, etcd3.WithLease(resp.ID)); err != nil {
						logkit.Debugf("[E][shaco-server] grpc register: refresh service '%s' with ttl to etcd3 failed: %s", serviceName, err)
					}
				}
			}
			select {
			case <-stopSignal:
				return
			case <-ticker.C:
			}
		}
	}()

	return nil
}

func unRegister() error {
	close(stopSignal)
	if client != nil {
		ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		if _, err := client.Delete(ctx, serviceKey); err != nil {
			logkit.Debugf("[E][shaco] grpclb: deregister '%s' failed: %s", serviceKey, err.Error())
			return err
		} else {
			logkit.Debugf("[I][shaco] grpclb: deregister '%s' ok.", serviceKey)
		}
	}
	return nil
}
