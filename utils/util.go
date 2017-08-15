package utils

import (
	"fmt"

	"github.com/docker/libcompose/utils"
)

func NestedMapsToMapInterface(data map[string]interface{}) map[string]interface{} {
	newMapInterface := map[string]interface{}{}
	for k, v := range data {
		newMapInterface[k] = convertObj(v)
	}
	return newMapInterface
}

func convertObj(v interface{}) interface{} {
	switch k := v.(type) {
	case map[interface{}]interface{}:
		return mapWalk(k)
	case map[string]interface{}:
		return NestedMapsToMapInterface(k)
	case []interface{}:
		return listWalk(k)
	default:
		return v
	}
}

func listWalk(val []interface{}) []interface{} {
	for i, v := range val {
		val[i] = convertObj(v)
	}
	return val
}

func mapWalk(val map[interface{}]interface{}) map[string]interface{} {
	newMap := map[string]interface{}{}
	for k, v := range val {
		newMap[fmt.Sprintf("%v", k)] = convertObj(v)
	}
	return newMap
}

func Contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func ToMapInterface(data map[string]string) map[string]interface{} {
	ret := map[string]interface{}{}

	for k, v := range data {
		ret[k] = v
	}

	return ret
}

func ToMapString(data map[string]interface{}) map[string]string {
	ret := map[string]string{}

	for k, v := range data {
		if str, ok := v.(string); ok {
			ret[k] = str
		} else {
			ret[k] = fmt.Sprint(v)
		}
	}

	return ret
}

func ToMapByte(data map[string]interface{}) map[string][]byte {
	ret := map[string][]byte{}

	for k, v := range data {
		if str, ok := v.(string); ok {
			ret[k] = []byte(str)
		} else if b, ok := v.([]byte); ok {
			ret[k] = b
		} else {
			ret[k] = []byte(fmt.Sprint(v))
		}
	}

	return ret
}

func MapUnion(left, right map[string]string) map[string]string {
	ret := map[string]string{}

	for k, v := range left {
		ret[k] = v
	}

	for k, v := range right {
		ret[k] = v
	}

	return ret
}

func MapUnionI(left, right map[string]interface{}) map[string]interface{} {
	ret := map[string]interface{}{}

	for k, v := range left {
		ret[k] = v
	}

	for k, v := range right {
		ret[k] = v
	}

	return ret
}

func IsSelected(selected []string, name string) bool {
	return len(selected) == 0 || utils.Contains(selected, name)
}
