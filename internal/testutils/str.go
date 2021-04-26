package testutils

import "encoding/json"

func ToJSON(v interface{}) string {
	bytes, _ := json.Marshal(v)
	return string(bytes)
}

func ToPrettyJSON(v interface{}) string {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	return string(bytes)
}
