// package eventlog provides game event logging functionality for JouleQuest.
package eventlog

import (
	"encoding/json"
	"fmt"
	"io"
)

// Loggable represents a value that can be logged as a string and knows its own key. Useful for logging enums.
type Loggable interface {
	fmt.Stringer
	LogKey() string
}

type LogEvent interface {
	// WithKey adds a key-value pair to the event.
	WithKey(key string, value any) LogEvent
	// With adds one or more loggable values to the event.
	With(value ...Loggable) LogEvent
	// Log finalizes and writes the log event.
	Log()
}

// Logger is an interface for logging events.
type Logger interface {
	Event() LogEvent
	Sub() Logger
	Set(value ...Loggable) Logger
	SetKey(key string, value any) Logger
}

// NullLogger is a no-op Logger that does nothing.
type NullLogger struct{}

func (l NullLogger) Event() LogEvent                     { return nil }
func (l NullLogger) Sub() Logger                         { return nil }
func (l NullLogger) Set(value ...Loggable) Logger        { return nil }
func (l NullLogger) SetKey(key string, value any) Logger { return nil }

var _ Logger = NullLogger{}

// WriterLogger is a logger that writes log events using a provided io.Writer.
type jsonLogger struct {
	data   map[string]any
	writer io.Writer
}

var _ Logger = (*jsonLogger)(nil)

func NewJsonLogger(w io.Writer) Logger {
	return &jsonLogger{writer: w}
}

// Event starts a new log event.
func (l *jsonLogger) Event() LogEvent {
	var e = jsonLogEvent{
		encoder: json.NewEncoder(l.writer),
	}
	if l.data != nil {
		e.data = make(map[string]any)
		for k, v := range l.data {
			e.data[k] = v
		}
	}
	return &e
}

// Set sets one or more loggable values that will be included in all subsequent log events.
func (l *jsonLogger) Set(value ...Loggable) Logger {
	if l.data == nil {
		l.data = make(map[string]any)
	}
	for _, v := range value {
		l.data[v.LogKey()] = v.String()
	}
	return l
}

// SetKey sets a key-value pair that will be included in all subsequent log events.
func (l *jsonLogger) SetKey(key string, value any) Logger {
	if l.data == nil {
		l.data = make(map[string]any)
	}
	l.data[key] = value
	return l
}

// Sub creates a new logger that includes the same provided values as this logger.
func (l jsonLogger) Sub() Logger {
	nl := NewJsonLogger(l.writer)
	for k, v := range l.data {
		nl.SetKey(k, v)
	}
	return nl
}

type jsonLogEvent struct {
	data    map[string]any
	encoder *json.Encoder
}

var _ LogEvent = (*jsonLogEvent)(nil)

func (e *jsonLogEvent) WithKey(key string, value any) LogEvent {
	if e.data == nil {
		e.data = make(map[string]any)
	}
	e.data[key] = value
	return e
}

func (e *jsonLogEvent) With(values ...Loggable) LogEvent {
	if e.data == nil {
		e.data = make(map[string]any)
	}
	for _, v := range values {
		e.data[v.LogKey()] = v.String()
	}
	return e
}

func (e *jsonLogEvent) Log() {
	if e.encoder != nil {
		e.encoder.Encode(e.data)
	}
}
