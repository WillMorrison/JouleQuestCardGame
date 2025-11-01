package eventlog

import (
	"strings"
	"testing"
)

func Test_JsonLogger_Event_WithKey(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.Event().WithKey("test_key", "test_value").Log()

	got := buf.String()
	want := `{"test_key":"test_value"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_JsonLogger_Event_WithKey_Multiple(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.Event().WithKey("key1", "value1").WithKey("key2", "value2").Log()

	got := buf.String()
	want := `{"key1":"value1","key2":"value2"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type TestLoggable struct {
	Key   string
	Value string
}

func (tl TestLoggable) LogKey() string {
	return tl.Key
}
func (tl TestLoggable) String() string {
	return tl.Value
}

func Test_JsonLogger_Event_With(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.Event().With(TestLoggable{"example_key", "example_value"}).Log()

	got := buf.String()
	want := `{"example_key":"example_value"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_JsonLogger_Event_With_Multiple(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.Event().With(TestLoggable{"key1", "value1"}, TestLoggable{"key2", "value2"}).Log()

	got := buf.String()
	want := `{"key1":"value1","key2":"value2"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_JsonLogger_Event_With_And_WithKey(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.Event().
		With(TestLoggable{"key1", "value1"}).
		WithKey("key2", "value2").
		Log()

	got := buf.String()
	want := `{"key1":"value1","key2":"value2"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_JsonLogger_Event_OverwriteKey(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.Event().
		WithKey("key", "initial_value").
		WithKey("key", "overwritten_value").
		Log()

	got := buf.String()
	want := `{"key":"overwritten_value"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_JsonLogger_Event_WithKey_StructValue(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.Event().
		WithKey("struct_key", struct {
			Field1 string `json:"field1"`
			Field2 int    `json:"field2"`
		}{
			Field1: "value1",
			Field2: 42,
		}).
		Log()

	got := buf.String()
	want := `{"struct_key":{"field1":"value1","field2":42}}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_JsonLogger_SetKey(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.SetKey("persistent_key", "persistent_value")
	logger.Event().WithKey("event_key", "event_value").Log()

	got := buf.String()
	want := `{"event_key":"event_value","persistent_key":"persistent_value"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_JsonLogger_Set(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.Set(TestLoggable{"persistent_key", "persistent_value"})
	logger.Event().WithKey("event_key", "event_value").Log()

	got := buf.String()
	want := `{"event_key":"event_value","persistent_key":"persistent_value"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func Test_JsonLogger_Sub(t *testing.T) {
	buf := strings.Builder{}
	logger := NewJsonLogger(&buf)
	logger.SetKey("parent_key", "parent_value")
	subLogger := logger.Sub().SetKey("child_key", "child_value")

	logger.Event().WithKey("event_key", "event_value_1").Log()
	subLogger.Event().WithKey("event_key", "event_value_2").Log()

	got := buf.String()
	want := `{"event_key":"event_value_1","parent_key":"parent_value"}
{"child_key":"child_value","event_key":"event_value_2","parent_key":"parent_value"}
`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
