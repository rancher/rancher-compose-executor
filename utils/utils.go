package utils

import (
	"fmt"
	"encoding/json"

	"github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ConvertByJSON converts a struct (src) to another one (target) using json marshalling/unmarshalling.
// If the structure are not compatible, this will throw an error as the unmarshalling will fail.
func ConvertByJSON(src, target interface{}) error {
	newBytes, err := json.Marshal(src)
	if err != nil {
		return err
	}

	err = json.Unmarshal(newBytes, target)
	if err != nil {
		logrus.Errorf("Failed to unmarshall: %v\n%s", err, string(newBytes))
	}
	return err
}

// Convert converts a struct (src) to another one (target) using yaml marshalling/unmarshalling.
// If the structure are not compatible, this will throw an error as the unmarshalling will fail.
func Convert(src, target interface{}) error {
	newBytes, err := yaml.Marshal(src)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(newBytes, target)
	if err != nil {
		logrus.Errorf("Failed to unmarshall: %v\n%s", err, string(newBytes))
	}
	return err
}

func CopySlice(s []string) []string {
	if s == nil {
		return nil
	}
	r := make([]string, len(s))
	copy(r, s)
	return r
}

func CopyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	r := map[string]string{}
	for k, v := range m {
		r[k] = v
	}
	return r
}

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
	return len(selected) == 0 || Contains(selected, name)
}
