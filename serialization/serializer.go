package serial

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
)

var (
	ErrSerialization = errors.New("failed to serialize payload")
)

func RegisterType(t any) {
	gob.Register(t)
}

func Serialize(in any) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	encoder := gob.NewEncoder(buffer)

	err := encoder.Encode(in)
	if err != nil {
		return nil, ErrSerialization
	}

	return io.ReadAll(buffer)
}

func Deserialize(in []byte, t any) error {
	buffer := bytes.NewBuffer(in)
	decoder := gob.NewDecoder(buffer)

	if err := decoder.Decode(t); err != nil {
		return ErrSerialization
	}
	return nil
}
