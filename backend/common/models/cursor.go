package models

import (
	"encoding/base64"
	"encoding/json"
)

const (
	CursorDirectionPrev CursorDirection = "p"
	CursorDirectionNext CursorDirection = "n"
)

type Cursor struct {
	Prev *DirectionalCursor
	Next *DirectionalCursor
}

type CursorDirection string

type DirectionalCursor struct {
	Direction CursorDirection `json:"d"`
	Marker    string          `json:"m"`
}

func (m *DirectionalCursor) Decode(str string) error {
	if str == "" {
		return nil
	}
	buf, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, m)
}

func (m *DirectionalCursor) Encode() (string, error) {
	buf, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}
