package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/macroblock/walkjson"
)

func main() {
	s := `
{
	"one": "aaa",
	"two": "bbb",
	"stock": {
		"type": "object",
		"properties": {
			"warehouse": {
				"type": "number"
			},
			"retail": {
				"type": "number"
			}
		}
	},
	"test": {},
	"list": [1,2,3,4],
	"emtyList": []
}`

	p := walkjson.New()
	p.Reset(bytes.NewReader([]byte(s)))

	err := p.Walk(func(typ int, path []string, key string, val interface{}) bool {
		fmt.Println(">>", strings.Join(path, "/"))
		fmt.Println(key, ":", val)
		return true
	})

	if err != nil {
		fmt.Println("---\n", err)
	}
}
