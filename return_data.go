package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

// returnData represents data with json tags for Marshal and  Unmarshal http response.
type returnData struct {
	Result int    `json:"result"`
	Error  error  `json:"error"`
	Expr   string `json:"expr"`
}

// unmarshalJSON is custom handler for writing error golang type to json struct field.
func (rd *returnData) unmarshalJSON(b []byte) error {
	type Alias returnData

	aux := &struct {
		Error string `json:"error"`
		*Alias
	}{
		Alias: (*Alias)(rd),
	}

	if err := json.Unmarshal(b, &aux); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	rd.Error = errors.New(aux.Error) // nolint

	return nil
}
