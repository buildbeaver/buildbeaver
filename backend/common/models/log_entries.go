package models

import (
	"encoding/json"
)

var (
	// logEntryFactoriesByKind maps a log entry kind to a factory that produces an empty instance of that kind.
	logEntryFactoriesByKind = map[LogEntryKind]func() derivedLogEntry{}
)

func init() {
	logEntryFactoriesByKind[LogEntryKindBlock] = func() derivedLogEntry { return &LogEntryBlock{} }
	logEntryFactoriesByKind[LogEntryKindLine] = func() derivedLogEntry { return &LogEntryLine{} }
	logEntryFactoriesByKind[LogEntryKindError] = func() derivedLogEntry { return &LogEntryError{} }
	logEntryFactoriesByKind[LogEntryKindEnd] = func() derivedLogEntry { return &LogEntryEnd{} }
}

const (
	LogEntryKindBlock LogEntryKind = "block"
	LogEntryKindLine  LogEntryKind = "line"
	LogEntryKindError LogEntryKind = "error"
	LogEntryKindEnd   LogEntryKind = "log_end"
)

type derivedLogEntry interface {
	setLogEntryBase(base *BaseLogEntry)
}

type LogEntryKind string

func (m LogEntryKind) String() string {
	return string(m)
}

type BaseLogEntry struct {
	Kind LogEntryKind `json:"kind"`
}

type PersistentLogEntry interface {
	GetSeqNo() int
	SetSeqNo(seqNo int)
	GetServerTimestamp() Time
	SetServerTimestamp(t Time)
	GetClientTimestamp() Time
}

type PlainTextLogEntry interface {
	PersistentLogEntry
	GetText() string
	SetText(text string)
	GetParentBlockName() *ResourceName
}

type persistentLogEntry struct {
	SeqNo           int  `json:"seq_no"`
	ServerTimestamp Time `json:"server_timestamp,omitempty"` // omitempty as entries can be shown in BB (using --json) and won't have a server timestamp
	ClientTimestamp Time `json:"client_timestamp"`
}

func (e *persistentLogEntry) GetSeqNo() int {
	return e.SeqNo
}

func (e *persistentLogEntry) SetSeqNo(seqNo int) {
	e.SeqNo = seqNo
}

func (e *persistentLogEntry) GetServerTimestamp() Time {
	return e.ServerTimestamp
}

func (e *persistentLogEntry) SetServerTimestamp(t Time) {
	e.ServerTimestamp = t
}

func (e *persistentLogEntry) GetClientTimestamp() Time {
	return e.ClientTimestamp
}

type plaintextLogEntry struct {
	Text            string        `json:"text"`
	ParentBlockName *ResourceName `json:"parent_block_name"`
}

func (e *plaintextLogEntry) GetText() string {
	return e.Text
}

func (e *plaintextLogEntry) SetText(text string) {
	e.Text = text
}

func (e *plaintextLogEntry) GetParentBlockName() *ResourceName {
	return e.ParentBlockName
}

// LogEntryBlock defines a block of log entries. Subsequent log entries can nominate this block by name to
// appear under the block.
// NOTE: This log entry does not mark 'the start of a block' - it just defines a block that can be referenced by
// an arbitrary set of log entries later in the log stream. If a subsequent log entry does not nominate this block
// in its ParentBlockName field then it will not appear under this block.
type LogEntryBlock struct {
	*BaseLogEntry
	persistentLogEntry
	plaintextLogEntry
	Name ResourceName `json:"name"`
}

func NewLogEntryBlock(seqNo int, clientTimestamp Time, text string, name ResourceName, parentBlockName *ResourceName) *LogEntry {
	base := &BaseLogEntry{Kind: LogEntryKindBlock}
	derived := &LogEntryBlock{
		BaseLogEntry:       base,
		Name:               name,
		persistentLogEntry: persistentLogEntry{SeqNo: seqNo, ClientTimestamp: clientTimestamp},
		plaintextLogEntry:  plaintextLogEntry{ParentBlockName: parentBlockName, Text: text},
	}
	return &LogEntry{BaseLogEntry: base, derived: derived}
}

func (m *LogEntryBlock) setLogEntryBase(base *BaseLogEntry) {
	m.BaseLogEntry = base
}

type LogEntryLine struct {
	*BaseLogEntry
	persistentLogEntry
	plaintextLogEntry
	LineNo int `json:"line_no"`
}

func NewLogEntryLine(seqNo int, clientTimestamp Time, text string, lineNo int, parentBlockName *ResourceName) *LogEntry {
	base := &BaseLogEntry{Kind: LogEntryKindLine}
	derived := &LogEntryLine{
		BaseLogEntry:       base,
		persistentLogEntry: persistentLogEntry{SeqNo: seqNo, ClientTimestamp: clientTimestamp},
		plaintextLogEntry:  plaintextLogEntry{ParentBlockName: parentBlockName, Text: text},
		LineNo:             lineNo}
	return &LogEntry{BaseLogEntry: base, derived: derived}
}

func (m *LogEntryLine) setLogEntryBase(base *BaseLogEntry) {
	m.BaseLogEntry = base
}

type LogEntryError struct {
	*LogEntryLine
}

func NewLogEntryError(seqNo int, clientTimestamp Time, text string, lineNo int, parentBlockName *ResourceName) *LogEntry {
	base := &BaseLogEntry{Kind: LogEntryKindError}
	logEntryLine := &LogEntryLine{
		BaseLogEntry:       base,
		persistentLogEntry: persistentLogEntry{SeqNo: seqNo, ClientTimestamp: clientTimestamp},
		plaintextLogEntry:  plaintextLogEntry{ParentBlockName: parentBlockName, Text: text},
		LineNo:             lineNo}

	derived := &LogEntryError{LogEntryLine: logEntryLine}
	return &LogEntry{BaseLogEntry: base, derived: derived}
}

func (m *LogEntryError) setLogEntryBase(base *BaseLogEntry) {
	m.LogEntryLine.BaseLogEntry = base
}

type LogEntryEnd struct {
	*BaseLogEntry
}

func NewLogEntryEnd() *LogEntry {
	base := &BaseLogEntry{Kind: LogEntryKindEnd}
	derived := &LogEntryEnd{BaseLogEntry: base}
	return &LogEntry{BaseLogEntry: base, derived: derived}
}

func (m *LogEntryEnd) setLogEntryBase(base *BaseLogEntry) {
	m.BaseLogEntry = base
}

// LogEntry represents a standard log entry with an optional derived type that contains
// additional kind-specific fields.
// Call LogEntry.Derived to get at the derived type and additional fields (if any).
// LogEntry can safely be marshaled to and from JSON and the derived type and fields will
// be correctly ferried back and forth.
type LogEntry struct {
	*BaseLogEntry
	derived derivedLogEntry
}

// Derived returns the derived type of the log entry, or the log entry itself if there is no derived type.
// Guaranteed to never return nil. Prefer a type switch on the returned interface over a check on LogEntry.Kind
// when determining what the derived type is.
func (m *LogEntry) Derived() interface{} {
	if m.derived != nil {
		return m.derived
	}
	return m
}

func (m *LogEntry) MarshalJSON() ([]byte, error) {
	var x interface{} = m.BaseLogEntry
	if m.derived != nil {
		x = m.derived
	}
	return json.Marshal(x)
}

func (m *LogEntry) UnmarshalJSON(data []byte) error {
	if m.BaseLogEntry == nil {
		m.BaseLogEntry = &BaseLogEntry{}
	}
	err := json.Unmarshal(data, m.BaseLogEntry)
	if err != nil {
		return err
	}
	factory, ok := logEntryFactoriesByKind[m.Kind]
	if !ok {
		return nil
	}
	m.derived = factory()
	err = json.Unmarshal(data, m.derived)
	if err != nil {
		return err
	}
	m.derived.setLogEntryBase(m.BaseLogEntry)
	return nil
}
