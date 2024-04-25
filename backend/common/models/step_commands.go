package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Command string

func (c Command) String() string {
	return string(c)
}

func CommandsToStrings(commands []Command) []string {
	strings := make([]string, len(commands))
	for i, command := range commands {
		strings[i] = string(command)
	}
	return strings
}

type Commands []Command

func (m Commands) Strings() []string {
	commands := make([]string, len(m))
	for i, command := range m {
		commands[i] = string(command)
	}
	return commands
}

func (m *Commands) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("unsupported type: %[1]T (%[1]v)", src)
	}
	err := json.Unmarshal([]byte(str), m)
	if err != nil {
		return fmt.Errorf("error unmarshalling from JSON: %w", err)
	}
	return nil
}

func (m Commands) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	buf, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("error marshalling to JSON: %w", err)
	}
	return string(buf), nil
}
