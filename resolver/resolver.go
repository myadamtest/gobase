package resolver

import (
	"fmt"

	"github.com/myadamtest/gobase/logkit"
)

var resolvers = []Resolver{&simpleResolver{}}

type Operation uint8

const (
	OP_Put Operation = iota
	OP_Delete
)

type Update struct {
	Id       string
	Op       Operation
	Addr     string
	User     string
	Password string
	Weight   int
	Master   bool
	Options  map[string]string
	Metadata interface{}
}

func (u *Update) String() string {
	return fmt.Sprintf("id:%s,op:%v,addr:%s,user:%s,weight:%d,master:%t", u.Id, u.Op, u.Addr, u.User, u.Weight, u.Master)
}

type Resolver interface {
	Resolve(target string) ([]*Update, Watcher, error)
}

type Watcher interface {
	Next() (*Update, error)
	Close() error
}

func RegisterResolver(r Resolver) {
	if r != nil {
		resolvers = append(resolvers, r)
	}
}

func ResolveTarget(target string) ([]*Update, Watcher) {
	var updates []*Update
	var watcher Watcher
	for i := len(resolvers) - 1; i >= 0; i-- {
		resolver := resolvers[i]
		if resolver != nil {
			var err error
			updates, watcher, err = resolver.Resolve(target)
			if err != nil {
				logkit.Errorf("[resolver]resolve target err:%s", err)
			} else {
				break
			}
		}
	}
	return updates, watcher
}
