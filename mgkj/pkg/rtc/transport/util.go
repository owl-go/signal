package transport

import (
	"strings"
)

// KvOK check flag and value is "true"
func KvOK(m map[string]interface{}, k, v string) bool {
	str := ""
	val, ok := m[k]
	if ok {
		str, ok = val.(string)
		if ok {
			if strings.EqualFold(str, v) {
				return true
			}
		}
	}
	return false
}

// GetUpperString get upper string by key
func GetUpperString(m map[string]interface{}, k string) string {
	val, ok := m[k]
	if ok {
		str, ok := val.(string)
		if ok {
			return strings.ToUpper(str)
		}
	}
	return ""
}
