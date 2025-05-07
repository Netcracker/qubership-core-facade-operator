package log

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"testing"
	"time"
)

func TestZapEncoder(t *testing.T) {
	encoder := NewZapEncoder()
	clone := encoder.Clone()

	assert.Equal(t, encoder.(ZapEncoder).context, clone.(ZapEncoder).context)

	encoder.AddInt32("intvalue", int32(1))

	fields := []zapcore.Field{
		{
			Key:    "stringval",
			Type:   zapcore.StringType,
			String: "text here",
		},
		{
			Key:     "boolval",
			Type:    zapcore.BoolType,
			Integer: 1,
		},
		{
			Key:       "marshaler_obj",
			Type:      zapcore.ObjectMarshalerType,
			Interface: MarshalableObject{"AAA"},
		},
		{
			Key:       "marshaler_arr",
			Type:      zapcore.ArrayMarshalerType,
			Interface: MarshalableArray{"BBB"},
		},
		{
			Key:       "bin",
			Type:      zapcore.BinaryType,
			Interface: []byte{1, 2, 3},
		},
		{
			Key:       "byte_str",
			Type:      zapcore.ByteStringType,
			Interface: []byte{3, 2, 1},
		},
		{
			Key:       "complex128",
			Type:      zapcore.Complex128Type,
			Interface: complex128(5),
		},
		{
			Key:       "complex64",
			Type:      zapcore.Complex64Type,
			Interface: complex64(56),
		},
		{
			Key:     "duration",
			Type:    zapcore.DurationType,
			Integer: 5,
		},
		{
			Key:     "float64",
			Type:    zapcore.Float64Type,
			Integer: 45,
		},
		{
			Key:     "float32",
			Type:    zapcore.Float32Type,
			Integer: 54,
		},
		{
			Key:     "int64",
			Type:    zapcore.Int64Type,
			Integer: 15,
		},

		{
			Key:     "int32",
			Type:    zapcore.Int32Type,
			Integer: 16,
		},

		{
			Key:     "int16",
			Type:    zapcore.Int16Type,
			Integer: 17,
		},
		{
			Key:     "int16",
			Type:    zapcore.Int16Type,
			Integer: 18,
		},
		{
			Key:     "int8",
			Type:    zapcore.Int8Type,
			Integer: 19,
		},
		{
			Key:     "int8",
			Type:    zapcore.Int8Type,
			Integer: 19,
		},
		{
			Key:       "time",
			Type:      zapcore.TimeType,
			Integer:   20,
			Interface: time.UTC,
		},
		{
			Key:       "time_full",
			Type:      zapcore.TimeFullType,
			Interface: time.Date(2023, 8, 10, 5, 5, 6, 7, time.UTC),
		},
		{
			Key:     "uint64",
			Type:    zapcore.Uint64Type,
			Integer: 21,
		},
		{
			Key:     "uint32",
			Type:    zapcore.Uint32Type,
			Integer: 22,
		},
		{
			Key:     "uint16",
			Type:    zapcore.Uint16Type,
			Integer: 23,
		},
		{
			Key:     "uint8",
			Type:    zapcore.Uint8Type,
			Integer: 24,
		},
		{
			Key:     "uintptr",
			Type:    zapcore.UintptrType,
			Integer: 25,
		},
		{
			Key:     "reflect",
			Type:    zapcore.ReflectType,
			Integer: 26,
		},
		{
			Key:  "namespace_key",
			Type: zapcore.NamespaceType,
		},
		{
			Key:       "stringer",
			Type:      zapcore.StringerType,
			Interface: StringerObject{"message 1"},
		},
		{
			Key:       "error",
			Type:      zapcore.ErrorType,
			Interface: fmt.Errorf("message 2"),
		},
	}

	data, _ := encoder.EncodeEntry(zapcore.Entry{
		Level:      zapcore.DebugLevel,
		Time:       time.Date(2023, 2, 3, 4, 5, 6, 7, time.UTC),
		LoggerName: "logger-name",
		Message:    "message-here",
	}, fields)

	res := data.String()
	assert.Contains(t, res, "[2023-02-03T04:05:06.000] [DEBUG] [request_id=-] [tenant_id=-] [thread=-] [class=logger-name] message-here")
	assert.Contains(t, res, "{stringval=text here}")
	assert.Contains(t, res, "{intvalue=1}")
	assert.Contains(t, res, "{boolval=true}")
	assert.Contains(t, res, "{marshaler_obj={text:AAA}}")
	assert.Contains(t, res, "{marshaler_arr={text:BBB}}")
	assert.Contains(t, res, "{complex64=(56+0i)}")
	assert.Contains(t, res, "{complex128=(5+0i)}")
	assert.Contains(t, res, "{byte_str=[3 2 1]}")
	assert.Contains(t, res, "{bin=[1 2 3]}")
	assert.Contains(t, res, "{int32=16}")
	assert.Contains(t, res, "{int64=15}")
	assert.Contains(t, res, "{float64=2.2e-322}")
	assert.Contains(t, res, "{float32=7.6e-44}")
	assert.Contains(t, res, "{duration=5ns}")
	assert.Contains(t, res, "{intvalue=1}")
	assert.Contains(t, res, "{int8=19}")
	assert.Contains(t, res, "{int16=18}")
	assert.Contains(t, res, "{uint32=22}")
	assert.Contains(t, res, "{uint8=24}")
	assert.Contains(t, res, "{uint16=23}")
	assert.Contains(t, res, "{uintptr=25}")
	assert.Contains(t, res, "{time=1970-01-01 00:00:00.00000002 +0000 UTC}")
	assert.Contains(t, res, "{time_full=2023-08-10 05:05:06.000000007 +0000 UTC}")
	assert.Contains(t, res, "{namespace=namespace_key}")
	assert.Contains(t, res, "{duration=5ns}")
	assert.Contains(t, res, "{reflect=<nil>}")
	assert.Contains(t, res, "{error=message 2}")
	assert.Contains(t, res, "{stringer=message 1}")
}

type MarshalableObject struct {
	text string
}

func (receiver MarshalableObject) MarshalLogObject(o zapcore.ObjectEncoder) error {
	return nil
}

type StringerObject struct {
	text string
}

func (receiver StringerObject) String() string {
	return receiver.text
}

type MarshalableArray struct {
	text string
}

func (receiver MarshalableArray) MarshalLogArray(o zapcore.ArrayEncoder) error {
	return nil
}
