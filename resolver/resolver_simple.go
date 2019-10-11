package resolver

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type simpleResolver struct {
}

func (r *simpleResolver) Resolve(target string) ([]*Update, Watcher, error) {
	if target == "" {
		return nil, nil, errors.New("target is invaild")
	}
	updates := []*Update{}
	for i, str := range strings.Split(target, ",") {
		if str != "" {
			if strings.Index(str, "://") < 0 {
				str = "simple://" + str
			}
			l, err := url.Parse(str)
			if err != nil || l.Scheme != "simple" {
				return nil, nil, fmt.Errorf("simple resolver[%s]%s", target, err)
			}
			update := &Update{
				Options: make(map[string]string),
			}
			update.Id = l.Host
			update.Op = OP_Put
			update.Addr = l.Host
			if l.User != nil {
				password, ok := l.User.Password()
				if ok {
					update.Password = password
					update.User = l.User.Username()
				} else {
					update.Password = l.User.Username()
				}
			}
			for k, v := range l.Query() {
				update.Options[k] = v[0]
			}

			if i == 0 {
				update.Master = true
			}

			updates = append(updates, update)
		}
	}
	return updates, nil, nil
}
