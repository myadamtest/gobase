package redis

import (
	"net"
	"strconv"

	"github.com/garyburd/redigo/redis"
	"github.com/myadamtest/gobase/logkit"
)

var ErrNil = redis.ErrNil

// zset 的 member 和 score 对，字段名根据业务需要再修改
type ScorePair struct {
	Member string `json:"member"`
	Score  int    `json:"score"`
}

type Geo struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type GeoPair struct {
	Geo
	Member string `json:"member"`
}

type GeoPairDistHash struct {
	GeoPair
	Dist float64
	Hash int
}

type LuaScript struct {
	*redis.Script
	write bool
}

type Pipe struct {
	redis.Conn
}

type Message struct {
	redis.Message
}

type SubscribeT struct {
	conn redis.PubSubConn
	C    chan []byte
}

func strSliToInterfSli(keys ...string) []interface{} {
	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}

	return args
}

func strSliToInterfSliTwo(key string, members ...string) []interface{} {
	args := make([]interface{}, len(members)+1)
	args[0] = key
	for i, member := range members {
		args[i+1] = member
	}

	return args
}

func (cli *Client) doInt(read bool, methodname string, masknil bool, args ...interface{}) (int, error) {
	var reply interface{}
	var err error
	if read {
		reply, err = cli.doRead(methodname, args...)
	} else {
		reply, err = cli.doWrite(methodname, args...)
	}

	value, err := redis.Int(reply, err)
	if masknil && err == redis.ErrNil {
		return -1, nil
	}

	return value, err
}

func (cli *Client) doFloat(read bool, methodname string, masknil bool, args ...interface{}) (float64, error) {
	var reply interface{}
	var err error
	if read {
		reply, err = cli.doRead(methodname, args...)
	} else {
		reply, err = cli.doWrite(methodname, args...)
	}

	value, err := redis.Float64(reply, err)
	if masknil && err == redis.ErrNil {
		return -1, nil
	}

	return value, err
}

func (cli *Client) doBool(read bool, methodname string, masknil bool, args ...interface{}) (bool, error) {
	var reply interface{}
	var err error
	if read {
		reply, err = cli.doRead(methodname, args...)
	} else {
		reply, err = cli.doWrite(methodname, args...)
	}

	value, err := redis.Bool(reply, err)
	if masknil && err == redis.ErrNil {
		return false, nil
	}

	return value, err
}

func (cli *Client) doString(read bool, methodname string, masknil bool, args ...interface{}) (string, error) {
	var reply interface{}
	var err error
	if read {
		reply, err = cli.doRead(methodname, args...)
	} else {
		reply, err = cli.doWrite(methodname, args...)
	}

	value, err := redis.String(reply, err)
	if masknil && err == redis.ErrNil {
		return "", nil
	}

	return value, err
}

func (cli *Client) doStringSlice(read bool, methodname string, masknil bool, args ...interface{}) ([]string, error) {
	var reply interface{}
	var err error
	if read {
		reply, err = cli.doRead(methodname, args...)
	} else {
		reply, err = cli.doWrite(methodname, args...)
	}

	value, err := redis.Strings(reply, err)
	if masknil && err == redis.ErrNil {
		return []string{}, nil
	}

	return value, err
}

//Key 相关
func (cli *Client) Del(keys ...string) (int, error) {
	args := strSliToInterfSli(keys...)
	return cli.doInt(false, "DEL", false, args...)
}

func (cli *Client) Expire(key string, ttl int) (int, error) {
	return cli.doInt(false, "EXPIRE", false, key, ttl)
}

func (cli *Client) Exists(key string) (bool, error) {
	return cli.doBool(true, "EXISTS", false, key)
}

func (cli *Client) RandomKey() (string, error) {
	//ErrNil时，表示没有key
	return cli.doString(true, "RANDOMKEY", true)
}

func (cli *Client) Rename(key, newKey string) (string, error) {
	return cli.doString(false, "RENAME", false, key, newKey)
}

func (cli *Client) TTL(key string) (int, error) {
	return cli.doInt(false, "TTL", false, key)
}

//if pattern=="", no regex, if count=0, decide by server
func (cli *Client) Scan(cursor int, pattern string, count int) (int, []string, error) {
	params := make([]interface{}, 0, 6)
	params = append(params, cursor)
	if pattern != "" {
		params = append(params, "MATCH", pattern)
	}
	if count > 0 {
		params = append(params, "COUNT", count)
	}

	reply, err := cli.doRead("SCAN", params...)
	if err != nil {
		return 0, nil, err
	}

	replySlice, _ := reply.([]interface{})
	newCursor, _ := redis.Int(replySlice[0], nil)
	result, _ := redis.Strings(replySlice[1], nil)

	return newCursor, result, nil
}

//string 相关
func (cli *Client) Append(key, val string) (int, error) {
	return cli.doInt(false, "APPEND", false, key, val)
}

func (cli *Client) Get(key string) (string, error) {
	return cli.doString(true, "GET", true, key)
}

func (cli *Client) GetRaw(key string) (string, error) {
	return cli.doString(true, "GET", false, key)
}

func (cli *Client) GetSet(key, val string) (string, error) {
	return cli.doString(false, "GETSET", true, key, val)
}

func (cli *Client) IncrBy(key string, step int) (int, error) {
	return cli.doInt(false, "INCRBY", false, key, step)
}

func (cli *Client) MGet(keys ...string) ([]string, error) {
	args := strSliToInterfSli(keys...)

	return cli.doStringSlice(true, "MGET", false, args...)
}

func (cli *Client) MSet(keyvals ...string) (string, error) {
	args := strSliToInterfSli(keyvals...)
	return cli.doString(false, "MSET", false, args...)
}

func (cli *Client) Set(key, val string, ttl int) (string, error) {
	return cli.SetEPX(key, val, ttl, 0, 0)
}

//ex or px: if ex and px not zeor, only px will work; control by server
//nxxx = 0: for nothing
//nxxx = 1: for NX
//nxxx = 2: for XX
func (cli *Client) SetEPX(key, val string, ex int, px int, nxxx int) (string, error) {
	args := make([]interface{}, 0)
	args = append(args, key, val)
	if ex > 0 {
		args = append(args, "EX", ex)
	}
	if px > 0 {
		args = append(args, "PX", px)
	}
	if nxxx == 1 {
		args = append(args, "NX")
	} else if nxxx == 2 {
		args = append(args, "XX")
	}

	return cli.doString(false, "SET", false, args...)
}

func (cli *Client) SetNx(key, val string) (int, error) {
	return cli.doInt(false, "SETNX", false, key, val)
}

func (cli *Client) StrLen(key string) (int, error) {
	return cli.doInt(true, "STRLEN", false, key)
}

//bit 相关
func (cli *Client) BitCount(key string) (int, error) {
	return cli.doInt(true, "BITCOUNT", false, key)
}

func (cli *Client) BitCountWithPos(key string, start, end int) (int, error) {
	return cli.doInt(true, "BITCOUNT", false, key, start, end)
}

func (cli *Client) BitOp(operation, destkey, key string, key2 ...string) (int, error) {
	params := []string{operation, destkey, key}
	params = append(params, key2...)
	args := strSliToInterfSli(params...)
	return cli.doInt(false, "BITOP", false, args...)
}

func (cli *Client) BitPos(key string, bit int) (int, error) {
	return cli.doInt(true, "BITPOS", false, key, bit)

}
func (cli *Client) BitPosWithPos(key string, bit, start, end int) (int, error) {
	return cli.doInt(true, "BITPOS", false, key, bit, start, end)
}

func (cli *Client) GetBit(key string, offset int) (int, error) {
	return cli.doInt(true, "GETBIT", false, key, offset)
}

func (cli *Client) SetBit(key string, offset, value int) (int, error) {
	return cli.doInt(false, "SETBIT", false, key, offset, value)
}

//hash 相关
func (cli *Client) HDel(key string, fields ...string) (int, error) {
	args := strSliToInterfSliTwo(key, fields...)
	return cli.doInt(false, "HDEL", false, args...)
}

func (cli *Client) HExists(key, field string) (bool, error) {
	return cli.doBool(true, "HEXISTS", false, key, field)
}

func (cli *Client) HGet(key, field string) (string, error) {
	return cli.doString(true, "HGET", true, key, field)
}

func (cli *Client) HGetAll(key string) (map[string]string, error) {
	values, err := cli.doStringSlice(true, "HGETALL", false, key)
	ret := make(map[string]string)
	for i := 0; i < len(values); i += 2 {
		key := values[i]
		value := values[i+1]

		ret[key] = value
	}

	return ret, err
}

func (cli *Client) HIncr(key, field string) (int, error) {
	return cli.doInt(false, "HINCRBY", false, key, field, 1)
}

func (cli *Client) HIncrBy(key, field string, increment int) (int, error) {
	return cli.doInt(false, "HINCRBY", false, key, field, increment)
}

func (cli *Client) HKeys(key string) ([]string, error) {
	return cli.doStringSlice(true, "HKEYS", false, key)
}

func (cli *Client) HLen(key string) (int, error) {
	return cli.doInt(true, "HLEN", false, key)
}

func (cli *Client) HMGet(key string, fields ...string) ([]string, error) {
	args := strSliToInterfSliTwo(key, fields...)
	return cli.doStringSlice(true, "HMGET", false, args...)
}

func (cli *Client) HMSet(key string, fieldvals ...string) (string, error) {
	args := strSliToInterfSliTwo(key, fieldvals...)
	return cli.doString(false, "HMSET", false, args...)
}

func (cli *Client) HSet(key, field, val string) (int, error) {
	return cli.doInt(false, "HSET", false, key, field, val)
}

func (cli *Client) HVals(key string) ([]string, error) {
	return cli.doStringSlice(true, "HVALS", false, key)
}

//if pattern=="", no regex, if count=0, decide by server
func (cli *Client) HScan(key string, cursor int, pattern string, count int) (int, map[string]string, error) {
	params := make([]interface{}, 0, 6)
	params = append(params, key, cursor)
	if pattern != "" {
		params = append(params, "MATCH", pattern)
	}
	if count > 0 {
		params = append(params, "COUNT", count)
	}

	reply, err := cli.doRead("HSCAN", params...)
	if err != nil {
		return 0, nil, err
	}

	replySlice, _ := reply.([]interface{})
	newCursor, _ := redis.Int(replySlice[0], nil)
	result, _ := redis.Strings(replySlice[1], nil)
	ret := make(map[string]string)
	for i := 0; i < len(result); i += 2 {
		key := result[i]
		value := result[i+1]

		ret[key] = value
	}

	return newCursor, ret, nil
}

//list 相关
func (cli *Client) LIndex(key string, index int) (string, error) {
	//ErrNil时，表示对应index不存在
	return cli.doString(true, "LINDEX", true, key, index)
}

func (cli *Client) LInsert(key, op, pivot, val string) (int, error) {
	return cli.doInt(false, "LINSERT", false, key, op, pivot, val)
}

func (cli *Client) LLen(key string) (int, error) {
	return cli.doInt(true, "LLEN", false, key)
}

func (cli *Client) LPop(key string) (string, error) {
	//ErrNil时，表示对应list不存在
	return cli.doString(false, "LPOP", true, key)
}

func (cli *Client) LPush(key string, vals ...string) (int, error) {
	args := strSliToInterfSliTwo(key, vals...)
	return cli.doInt(false, "LPUSH", false, args...)
}

func (cli *Client) LRange(key string, start, stop int) ([]string, error) {
	return cli.doStringSlice(true, "LRANGE", false, key, start, stop)
}

func (cli *Client) LRem(key string, count int, val string) (int, error) {
	return cli.doInt(false, "LREM", false, key, count, val)
}

func (cli *Client) LSet(key string, index int, val string) (string, error) {
	return cli.doString(false, "LSET", false, key, index, val)
}

func (cli *Client) LTrim(key string, start, stop int) (string, error) {
	return cli.doString(false, "LTRIM", false, key, start, stop)
}

func (cli *Client) RPop(key string) (string, error) {
	//ErrNil时，表示对应list不存在
	return cli.doString(false, "RPOP", true, key)
}

func (cli *Client) RPush(key string, vals ...string) (int, error) {
	args := strSliToInterfSliTwo(key, vals...)
	return cli.doInt(false, "RPUSH", false, args...)
}

//set 结构操作
func (cli *Client) SAdd(key string, members ...string) (int, error) {
	args := strSliToInterfSliTwo(key, members...)
	return cli.doInt(false, "SADD", false, args...)
}

func (cli *Client) SCard(key string) (int, error) {
	return cli.doInt(true, "SCARD", false, key)
}

func (cli *Client) SDiff(keys ...string) ([]string, error) {
	args := strSliToInterfSli(keys...)
	return cli.doStringSlice(true, "SDIFF", false, args...)
}

func (cli *Client) SInter(keys ...string) ([]string, error) {
	args := strSliToInterfSli(keys...)
	return cli.doStringSlice(true, "SINTER", false, args...)
}

func (cli *Client) SIsMember(key, member string) (bool, error) {
	return cli.doBool(true, "SISMEMBER", false, key, member)
}

func (cli *Client) SMembers(key string) ([]string, error) {
	return cli.doStringSlice(true, "SMEMBERS", false, key)
}

func (cli *Client) SPop(key string) (string, error) {
	//ErrNil时，表示对应set不存在
	return cli.doString(false, "SPOP", true, key)
}

func (cli *Client) SRandMember(key string) (string, error) {
	//ErrNil时，表示对应set不存在
	return cli.doString(true, "SRANDMEMBER", true, key)
}

func (cli *Client) SRem(key string, members ...string) (int, error) {
	args := strSliToInterfSliTwo(key, members...)
	return cli.doInt(false, "SREM", false, args...)
}

func (cli *Client) SUnion(keys ...string) ([]string, error) {
	args := strSliToInterfSli(keys...)
	return cli.doStringSlice(true, "SUNION", false, args...)
}

//if pattern=="", no regex, if count=0, decide by server
func (cli *Client) SScan(key string, cursor int, pattern string, count int) (int, []string, error) {
	params := make([]interface{}, 0, 6)
	params = append(params, key, cursor)
	if pattern != "" {
		params = append(params, "MATCH", pattern)
	}
	if count > 0 {
		params = append(params, "COUNT", count)
	}

	reply, err := cli.doRead("SSCAN", params...)
	if err != nil {
		return 0, nil, err
	}

	replySlice, _ := reply.([]interface{})
	newCursor, _ := redis.Int(replySlice[0], nil)
	result, _ := redis.Strings(replySlice[1], nil)

	return newCursor, result, nil
}

//zset 相关
func parseScorePair(strAry []string) []*ScorePair {
	ret := make([]*ScorePair, len(strAry)/2)

	for i := 0; i < len(strAry); i = i + 2 {
		member := strAry[i]
		score, err := strconv.ParseInt(strAry[i+1], 10, 64)

		if err == nil {
			ret[i/2] = &ScorePair{member, int(score)}
		}
	}

	return ret
}

func (cli *Client) ZAdd(key string, score int, member string) (int, error) {
	return cli.doInt(false, "ZADD", false, key, score, member)
}

func (cli *Client) ZMAdd(key string, members ...*ScorePair) (int, error) {
	args := make([]interface{}, len(members)*2+1)
	args[0] = key
	for i, member := range members {
		args[2*i+1] = member.Score
		args[2*i+2] = member.Member
	}
	reply, err := cli.doWrite("ZADD", args...)

	return redis.Int(reply, err)
}

func (cli *Client) ZCard(key string) (int, error) {
	return cli.doInt(true, "ZCARD", false, key)
}

//min, max 可以是+inf, -inf
func (cli *Client) ZCount(key string, min, max string) (int, error) {
	return cli.doInt(true, "ZCOUNT", false, key, min, max)
}

func (cli *Client) ZIncrBy(key string, incrNum int, member string) (int, error) {
	return cli.doInt(false, "ZINCRBY", false, key, incrNum, member)
}

func (cli *Client) ZRange(key string, start, stop int) ([]string, error) {
	return cli.doStringSlice(true, "ZRANGE", false, key, start, stop)
}

func (cli *Client) ZRangeWithScores(key string, start, stop int) ([]*ScorePair, error) {
	reply, err := cli.doRead("ZRANGE", key, start, stop, "WITHSCORES")
	values, err := redis.Strings(reply, err)

	return parseScorePair(values), err
}

func (cli *Client) ZRank(key, member string) (int, error) {
	//ErrNil时，表示member不存在
	return cli.doInt(true, "ZRANK", true, key, member)
}

func (cli *Client) ZRem(key string, members ...string) (int, error) {
	args := strSliToInterfSliTwo(key, members...)
	return cli.doInt(false, "ZREM", false, args...)
}

func (cli *Client) ZRemRangeByRank(key string, start, stop int) (int, error) {
	return cli.doInt(false, "ZREMRANGEBYRANK", false, key, start, stop)
}

func (cli *Client) ZRevRange(key string, start, stop int) ([]string, error) {
	return cli.doStringSlice(true, "ZREVRANGE", false, key, start, stop)
}

func (cli *Client) ZRevRangeWithScores(key string, start, stop int) ([]*ScorePair, error) {
	reply, err := cli.doRead("ZREVRANGE", key, start, stop, "WITHSCORES")
	values, err := redis.Strings(reply, err)

	return parseScorePair(values), err
}

func (cli *Client) ZRevRank(key, member string) (int, error) {
	//ErrNil时，表示member不存在
	return cli.doInt(true, "ZREVRANK", true, key, member)
}

func (cli *Client) ZScore(key, member string) (int, error) {
	return cli.doInt(true, "ZSCORE", false, key, member)
}

func (cli *Client) LoadScript(keyCount int, src string, write bool) (*LuaScript, error) {
	script := redis.NewScript(keyCount, src)
	var conn redis.Conn
	var err error
	if write {
		conn, err = cli.masterPools.Get()
	} else {
		conn, err = cli.slavePools.Get()
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	err = script.Load(conn)
	if err != nil {
		return nil, err
	}

	return &LuaScript{script, write}, nil
}

func (cli *Client) Eval(script *LuaScript, keysAndArgs ...interface{}) (interface{}, error) {
	var conn redis.Conn
	var err error
	if script.write {
		conn, err = cli.masterPools.Get()
	} else {
		conn, err = cli.slavePools.Get()
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	reply, err := script.Do(conn, keysAndArgs...)

	return reply, err
}

//pipeline 相关
func (cli *Client) PipeLine(readonly bool) (*Pipe, error) {
	var conn redis.Conn
	var err error
	if readonly {
		conn, err = cli.slavePools.Get()
	} else {
		conn, err = cli.masterPools.Get()
	}
	if err != nil {
		return nil, err
	}

	return &Pipe{conn}, nil
}

func (cli *Client) PipeSend(pipe *Pipe, cmd string, args ...interface{}) error {
	return pipe.Send(cmd, args...)
}

func (cli *Client) PipeExec(pipe *Pipe) (interface{}, error) {
	return pipe.Do("")
}

func (cli *Client) PipeClose(pipe *Pipe) error {
	return pipe.Close()
}

//server 相关
func (cli *Client) BgSave() (string, error) {
	return cli.doString(false, "BGSAVE", false)
}

//pubsub 相关  readtimeout need set to 0
func (cli *Client) Publish(channel, message string) (int, error) {
	return cli.doInt(false, "PUBLISH", false, channel, message)
}

func (cli *Client) Subscribe(channel string) (*SubscribeT, error) {
	conn, err := cli.masterPools.Get()
	if err != nil {
		return nil, err
	}

	psc := redis.PubSubConn{conn}
	err = psc.Subscribe(channel)
	if err != nil {
		conn.Close()
		return nil, err
	}

	st := &SubscribeT{psc, make(chan []byte, 10)}

	go func() {
		defer func() {
			close(st.C)
		}()
		for {
			switch v := psc.Receive().(type) {
			case redis.Message:
				st.C <- v.Data
			case redis.Subscription:
				logkit.Debugf("[redis]subscribe: %s: %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
				netError, ok := v.(net.Error)
				if ok && netError.Timeout() {
					logkit.Errorf("[redis]subscribe: %s", v)
				} else if v.Error() != "redigo: connection closed" {
					logkit.Errorf("[redis]subscribe: %s", v)
				}
				return
			}
		}
	}()

	return st, nil
}

func (cli *Client) SubscribeClose(st *SubscribeT) error {
	return st.conn.Close()
}

//geo 相关
func (cli *Client) GeoAdd(key string, members ...*GeoPair) (int, error) {
	args := make([]interface{}, len(members)*3+1)
	args[0] = key
	for i, member := range members {
		args[3*i+1] = member.Longitude
		args[3*i+2] = member.Latitude
		args[3*i+3] = member.Member
	}
	return cli.doInt(false, "GEOADD", false, args...)
}

//unit: [m|km|mi|ft], if empty then default:m, m:米,km:千米,mi:英里,ft:英尺
func (cli *Client) GeoDist(key, member1, member2, unit string) (float64, error) {
	if unit == "" {
		unit = "m"
	}

	return cli.doFloat(true, "GEODIST", false, key, member1, member2, unit)
}

func (cli *Client) GeoHash(key string, member ...string) ([]string, error) {
	args := make([]interface{}, 0)
	args = append(args, key)
	for _, m := range member {
		args = append(args, m)
	}
	return cli.doStringSlice(true, "GEOHASH", false, args...)
}

func parseGeo(strAry []string) []*Geo {
	ret := make([]*Geo, len(strAry)/2)

	for i := 0; i < len(strAry); i = i + 2 {
		ret[i/2] = &Geo{}
		ret[i/2].Longitude, _ = strconv.ParseFloat(strAry[i], 64)
		ret[i/2].Latitude, _ = strconv.ParseFloat(strAry[i+1], 64)
	}

	return ret
}

func (cli *Client) GeoPos(key string, member ...string) ([]*Geo, error) {
	args := make([]interface{}, 0)
	args = append(args, key)
	for _, m := range member {
		args = append(args, m)
	}
	reply, err := cli.doRead("GEOPOS", args...)
	if err != nil {
		return nil, err
	}
	replySlice := reply.([]interface{})
	values := make([]string, 0)
	for _, r := range replySlice {
		if r == nil {
			values = append(values, "0", "0")
		} else {
			rstr := r.([]interface{})
			rstr_1 := (rstr[0]).([]byte)
			rstr_2 := (rstr[1]).([]byte)
			values = append(values, string(rstr_1), string(rstr_2))
		}
	}

	return parseGeo(values), err
}

func parseGeoPairDistHash(reply interface{}, coord, dist, hash bool) []*GeoPairDistHash {
	replySlice, _ := reply.([]interface{})
	result := make([]*GeoPairDistHash, 0)
	for _, rs := range replySlice {
		gpdh := GeoPairDistHash{}
		if !coord && !dist && !hash {
			member, _ := rs.([]byte)
			gpdh.Member = string(member)
		} else {
			rsSlice, _ := rs.([]interface{})
			pos := 0
			member, _ := rsSlice[pos].([]byte)
			gpdh.Member = string(member)
			pos++
			if dist {
				sdist, _ := rsSlice[pos].([]byte)
				gpdh.Dist, _ = strconv.ParseFloat(string(sdist), 64)
				pos++
			}
			if hash {
				shash, _ := rsSlice[pos].(int64)
				gpdh.Hash = int(shash)
				pos++
			}
			if coord {
				scoord, _ := rsSlice[pos].([]interface{})
				longi, _ := scoord[0].([]byte)
				lati, _ := scoord[1].([]byte)
				gpdh.Longitude, _ = strconv.ParseFloat(string(longi), 64)
				gpdh.Latitude, _ = strconv.ParseFloat(string(lati), 64)
				pos++
			}
		}
		result = append(result, &gpdh)
	}

	return result
}

//unit: [m|km|mi|ft], if empty then default:m, m:米,km:千米,mi:英里,ft:英尺
//count: 0 means no limit
//st: 0-> no sort, 1-> asc, 2-> desc
func (cli *Client) GeoRadius(key string, longitude, latitude, radius float64, unit string, coord, dist, hash bool, count int, st int) ([]*GeoPairDistHash, error) {
	args := make([]interface{}, 0)
	args = append(args, key, longitude, latitude, radius, unit)
	if coord {
		args = append(args, "withcoord")
	}
	if dist {
		args = append(args, "withdist")
	}
	if hash {
		args = append(args, "withhash")
	}
	if count > 0 {
		args = append(args, "count", count)
	}
	if st == 1 {
		args = append(args, "asc")
	} else if st == 2 {
		args = append(args, "desc")
	}
	reply, err := cli.doWrite("GEORADIUS", args...)
	if err != nil {
		return nil, err
	}
	return parseGeoPairDistHash(reply, coord, dist, hash), nil
}

func (cli *Client) GeoRadiusByMember(key, member string, radius float64, unit string, coord, dist, hash bool, count int, st int) ([]*GeoPairDistHash, error) {
	args := make([]interface{}, 0)
	args = append(args, key, member, radius, unit)
	if coord {
		args = append(args, "withcoord")
	}
	if dist {
		args = append(args, "withdist")
	}
	if hash {
		args = append(args, "withhash")
	}
	if count > 0 {
		args = append(args, "count", count)
	}
	if st == 1 {
		args = append(args, "asc")
	} else if st == 2 {
		args = append(args, "desc")
	}
	reply, err := cli.doWrite("GEORADIUSBYMEMBER", args...)
	if err != nil {
		return nil, err
	}
	return parseGeoPairDistHash(reply, coord, dist, hash), nil
}
