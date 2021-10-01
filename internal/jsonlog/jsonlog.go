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
	bs, _ := json.Marshal(m)
	fmt.Println(string(bs))
}
