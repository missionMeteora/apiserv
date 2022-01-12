//+go:build amd64

package internal

import (
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/encoder"
)

func Marshal(v interface{}) ([]byte, error) {
	return sonic.Marshal(v)
}

func MarshalIndent(v interface{}) ([]byte, error) {
	return encoder.EncodeIndented(v, "", "\t", 0)
}

func Unmarshal(buf []byte, val interface{}) error {
	return sonic.Unmarshal(buf, val)
}

func UnmarshalString(buf string, val interface{}) error {
	return sonic.UnmarshalString(buf, val)
}

func Pretouch(vt reflect.Type) error {
	return sonic.Pretouch(vt)
}
