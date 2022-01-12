//+go:build arm64

package internal

import (
	"encoding/json"

	"github.com/OneOfOne/otk"
)

func init() {
	log.Printf("WARNING: running with stdlib json on arm64")
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
