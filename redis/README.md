## Base Usage
```
cli, err := redis.NewClient("DSN")
```

#### DSN
```
[pwd]@[host]:[port],[pwd]@[host]:[port]
```

- 支持多个地址，以,分割，默认第一个为master，其他的为slave

#### Config

- with code

```
cli, err := redis.NewClient("DSN",
    redis.ConnectTimeout(1000*time.Millisecond),
    redis.ReadTimeout(500*time.Millisecond),
    redis.WriteTimeout(500*time.Millisecond),
    redis.IdleTimeout(60*time.Second),
    redis.MaxActive(50),
    redis.MaxIdle(3),
    redis.WaitConn(true))
```

- with DSN

```
[pwd]@[host]:[port]?connect_time=1000&write_timeout=500&read_timeout=500&idle_timeout=500&max_active=50&max_idle=3&wait_conn=false
```

- 配置项全为可选，可以单独使用, url中的配置优先级高于代码的配置

## ETCD Usage

要增加对etcd的支持，只需要在master中增加对janna_resolver的依赖，以及将DSN修改成etcd格式的地址即可

1. 增加etcd支持的依赖,只需在一个地方增加janna_redis，建议在main中加载
```
import (
	_ "git.pandatv.com/panda-public/janna-resolver/redis"
)
```

2. 修改DSN格式为etcd地址格式

- passwd mode

```
etcd://10.0.0.1:1234,10.20.0.2:1234/riven/redis?tag=gw&master_pwd=xxx&slave_pwd=xxx

// riven:业务名，切换成自己的业务
// redis:redis模式下不需要修改,后期可增加对mysql的支持
// tag: 对应的janna配置中的tag
// master_pwd,slave_pwd:如果redis有密码，需要在此配置密码
```

- token mode

```
etcd://token@10.0.0.1:1234,10.20.0.2:1234/riven/redis?tag=gw

// token: etcd token
// riven:业务名，切换成自己的业务
// redis:redis模式下不需要修改,后期可增加对mysql的支持
// tag: 对应的janna配置中的tag
```
