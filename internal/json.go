//go:build !amd64 || go1.18
// +build !amd64 go1.18

package internal

import (
	"encoding/json"
	"log"
	"reflect"

	"go.oneofone.dev/otk"
)

func init() {
	log.Printf("apiserv: running with stdlib json")
}

func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func MarshalIndent(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "\t")
}

func Unmarshal(buf []byte, val interface{}) error {
	return json.Unmarshal(buf, val)
}

func UnmarshalString(buf string, val interface{}) error {
	return json.Unmarshal(otk.UnsafeBytes(buf), val)
}

func Pretouch(vt reflect.Type) error {
	return nil
}
