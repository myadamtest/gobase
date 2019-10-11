package mysql

import "hash/crc32"

type ShardFunc func(key string) int

func ShardFuncCrc32(start, end int) ShardFunc {
	return func(key string) int {
		i := int(crc32.ChecksumIEEE([]byte(key)))
		return i%(end-start+1) + start
	}
}
