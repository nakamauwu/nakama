package db

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONValue struct {
	Dst any
}

// Value implements sql driver Valuer interface.
func (jv JSONValue) Value() (driver.Value, error) {
	if jv.Dst == nil {
		return nil, nil
	}

	var buff bytes.Buffer
	enc := json.NewEncoder(&buff)
	enc.SetEscapeHTML(false)
	err := enc.Encode(jv.Dst)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), err
}

// Scan implements sql driver scanner interface.
func (jv *JSONValue) Scan(value any) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unexpected json, got %T", value)
	}

	return json.Unmarshal(b, &jv.Dst)
}
