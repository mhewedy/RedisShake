package commands

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"RedisShake/internal/log"
	"RedisShake/internal/utils"
)

// CalcKeys https://redis.io/docs/reference/key-specs/
func CalcKeys(argv []string) (cmaName string, group string, keys []string, keysIndexes []int) {
	argc := len(argv)
	group = "unknown"
	cmaName = strings.ToUpper(argv[0])
	if _, ok := containers[cmaName]; ok {
		if len(argv) > 1 {
			cmaName = fmt.Sprintf("%s-%s", cmaName, strings.ToUpper(argv[1]))
		}
	}
	cmd, ok := redisCommands[cmaName]
	if !ok {
		log.Warnf("unknown command. argv=%v", argv)
		return
	}
	group = cmd.group
	for _, spec := range cmd.keySpec {
		begin := 0
		switch spec.beginSearchType {
		case "index":
			begin = spec.beginSearchIndex
		case "keyword":
			var inx, step int
			if spec.beginSearchStartFrom > 0 {
				inx = spec.beginSearchStartFrom
				step = 1
			} else {
				inx = -spec.beginSearchStartFrom
				step = -1
			}
			for ; ; inx += step {
				if inx == argc {
					log.Panicf("not found keyword. argv=%v", argv)
				}
				if strings.ToUpper(argv[inx]) == spec.beginSearchKeyword {
					begin = inx + 1
					break
				}
			}
		default:
			log.Panicf("wrong type: %s", spec.beginSearchType)
		}
		switch spec.findKeysType {
		case "range":
			var lastKeyInx int
			if spec.findKeysRangeLastKey >= 0 {
				lastKeyInx = begin + spec.findKeysRangeLastKey
			} else {
				lastKeyInx = argc + spec.findKeysRangeLastKey
			}
			limitCount := math.MaxInt32
			if spec.findKeysRangeLimit <= -2 {
				limitCount = (argc - begin) / (-spec.findKeysRangeLimit)
			}
			keyStep := spec.findKeysRangeKeyStep
			for inx := begin; inx <= lastKeyInx && limitCount > 0; inx += keyStep {
				keys = append(keys, argv[inx])
				keysIndexes = append(keysIndexes, inx+1)
				limitCount -= 1
			}
		case "keynum":
			keynumIdx := begin + spec.findKeysKeynumIndex
			if keynumIdx < 0 || keynumIdx > argc {
				log.Panicf("keynumInx wrong. argv=%v, keynumIdx=[%d]", argv, keynumIdx)
			}
			keyCount, err := strconv.Atoi(argv[keynumIdx])
			if err != nil {
				log.Panicf(err.Error())
			}
			firstKey := spec.findKeysKeynumFirstKey
			step := spec.findKeysKeynumKeyStep
			for inx := begin + firstKey; keyCount > 0; inx += step {
				keys = append(keys, argv[inx])
				keysIndexes = append(keysIndexes, inx+1)
				keyCount -= 1
			}
		default:
			log.Panicf("wrong type: %s", spec.findKeysType)
		}
	}
	return
}

func CalcSlots(keys []string) []int {
	slots := make([]int, len(keys))
	for inx, key := range keys {
		slots[inx] = int(keyHash(key))
	}
	return slots
}

func keyHash(key string) uint16 {
	hashtag := ""
findHashTag:
	for i, s := range key {
		if s == '{' {
			for k := i; k < len(key); k++ {
				if key[k] == '}' {
					hashtag = key[i+1 : k]
					break findHashTag
				}
			}
		}
	}
	if len(hashtag) > 0 {
		key = hashtag
	}
	return utils.Crc16(key) & 0x3FFF
}
