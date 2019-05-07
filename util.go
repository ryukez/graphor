package graphor

import (
	"encoding/json"
	"fmt"
)

func eval(x interface{}) string {
	s := ""

	switch v := x.(type) {
	case int:
		s = fmt.Sprintf("%d", v)
	case bool:
		s = fmt.Sprintf("%t", v)
	case string:
		s = fmt.Sprintf("\"%s\"", v)
	}

	return s
}

func isEmpty(x interface{}) bool {
	switch v := x.(type) {
	case int:
		return v == 0
	case bool:
		return v == false
	case string:
		return v == ""
	}

	return false
}

func isValidUid(uids ...string) bool {
	for _, uid := range uids {
		if len(uid) < 2 || uid[0:2] != "0x" {
			return false
		}
	}
	return true
}

func keyExists(hash map[string]interface{}, key string) bool {
	_, ok := hash[key]
	return ok
}

func cast(src interface{}, dist interface{}) {
	j, _ := json.Marshal(src)
	json.Unmarshal(j, dist)
}

func toJSON(obj interface{}) string {
	b, _ := json.Marshal(obj)
	return string(b)
}

func decodeString(x interface{}) string {
	return x.(string)
}

func decodeInt(x interface{}) int {
	return int(x.(float64))
}
