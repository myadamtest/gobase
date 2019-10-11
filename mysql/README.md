## Base Usage
```
cli, err := mysql.NewClient("DSN")
```

#### DSN
```
[user]:[pwd]@[host]:[port]?dbname=[DBName]&tag=[master|slave],[pwd]@[host]:[port]?dbname[DBName]&tag=[master|slave]
```

- 支持多个地址，以,分割，默认第一个为master，其他的为slave,可以通过`tag=master`指定master节点,`tag=slave`指定slave节点

#### Config

- with code

```
cli, err := mysql.NewClient("DSN",
    mysql.ConnectTimeout(1000*time.Millisecond),
    mysql.ReadTimeout(500*time.Millisecond),
    mysql.WriteTimeout(500*time.Millisecond),
    mysql.IdleTimeout(60*time.Second),
    mysql.MaxActive(50),
    mysql.MaxIdle(3),
```

- with DSN

```
[user]:[pwd]@[host]:[port]?dbname=[DBName]?connect_time=1000&write_timeout=500&read_timeout=500&idle_timeout=500&max_active=50&max_idle=3
```

- 配置项全为可选，可以单独使用, url中的配置优先级高于代码的配置

## ETCD Usage

要增加对etcd的支持，只需要在master中增加对janna_resolver的依赖，以及将DSN修改成etcd格式的地址即可

1. 增加etcd支持的依赖,只需在一个地方增加janna_mysql，建议在main中加载
```
import (
	_ "git.pandatv.com/panda-public/janna-resolver"
)
```

2. 修改DSN格式为etcd地址格式

- passwd mode

```
etcd://10.0.0.1:1234,10.20.0.2:1234/riven/mysql?tag=gw&dbname=[DBName]&master_user=xxx&master_pwd=xxx&slave_user=xxx&slave_pwd=xxx

// riven:业务名，切换成自己的业务
// mysql:mysql模式下不需要修改,后期可增加对mysql的支持
// tag: 对应的janna配置中的tag
// master_user,master_pwd,slave_user,slave_pwd:mysql用户和密码
```

- token mode

```
etcd://token@10.0.0.1:1234,10.20.0.2:1234/riven/mysql?tag=gw&dbname=[DBName]

// token: etcd token
// riven:业务名，切换成自己的业务
// tag: 对应的janna配置中的tag
```
### Shard模式

```
configs := []*ShardConfig{
    &ShardConfig{
        Target:     "[dsn|etcd]",
        ShardStart: 0,
        ShardEnd:   1,
        Opts:       nil, // 可选配置
    },
    &ShardConfig{
        Target:     "[dsn|etcd]",
        ShardStart: 2,
        ShardEnd:   3,
        Opts:       nil, // 可选配置
    },
}
client, err := NewShardClient(configs, ShardFuncCrc32)

```
