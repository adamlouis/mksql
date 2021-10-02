package jsonlog

import (
	"encoding/json"
	"fmt"
)

func Log(kvs ...interface{}) {
	m := map[string]interface{}{}
	for i := 0; i < len(kvs)-1; i += 2 {
		m[fmt.Sprintf("%v", kvs[i])] = kvs[i+1]
	}
	bs, err := json.Marshal(m)
	if err != nil {
		LogE(err)
		return
	}
	fmt.Println(string(bs))
}

func LogE(err error) {
	if err == nil {
		return
	}
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	fmt.Println(string(b))
}
