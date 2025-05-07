package log

import (
	"fmt"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"strings"
	"time"
)

// To support logging guide
func NewZapEncoder() zapcore.Encoder {
	return ZapEncoder{make(map[string]interface{})}
}

type ZapEncoder struct {
	context map[string]interface{}
}

func (encoder ZapEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	encoder.context[key] = marshaler
	return nil
}

func (encoder ZapEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	encoder.context[key] = marshaler
	return nil
}

func (encoder ZapEncoder) AddBinary(key string, value []byte) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddByteString(key string, value []byte) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddBool(key string, value bool) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddComplex128(key string, value complex128) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddComplex64(key string, value complex64) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddDuration(key string, value time.Duration) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddFloat64(key string, value float64) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddFloat32(key string, value float32) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddInt(key string, value int) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddInt64(key string, value int64) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddInt32(key string, value int32) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddInt16(key string, value int16) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddInt8(key string, value int8) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddString(key, value string) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddTime(key string, value time.Time) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddUint(key string, value uint) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddUint64(key string, value uint64) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddUint32(key string, value uint32) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddUint16(key string, value uint16) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddUint8(key string, value uint8) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddUintptr(key string, value uintptr) {
	encoder.context[key] = value
}

func (encoder ZapEncoder) AddReflected(key string, value interface{}) error {
	encoder.context[key] = value
	return nil
}

func (encoder ZapEncoder) OpenNamespace(key string) {
	encoder.context["namespace"] = key
}

func (encoder ZapEncoder) Clone() zapcore.Encoder {
	newContext := make(map[string]interface{}, len(encoder.context))
	for k, v := range encoder.context {
		newContext[k] = v
	}
	return ZapEncoder{newContext}
}

func (encoder ZapEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	timeFormat := "2006-01-02T15:04:05.000"

	enc := encoder.Clone()
	for _, field := range fields {
		field.AddTo(enc)
	}

	contextData := []string{}
	for k, v := range enc.(ZapEncoder).context {
		contextData = append(contextData, fmt.Sprintf("{%s=%+v}", k, v))
	}

	line := buffer.NewPool().Get()

	fmt.Fprintf(line, "[%s] [%s] [request_id=-] [tenant_id=-] [thread=-] [class=%s] %s %s",
		entry.Time.Format(timeFormat),
		entry.Level.CapitalString(),
		entry.LoggerName,
		entry.Message,
		strings.Join(contextData, ", "),
	)

	line.WriteByte('\n')
	return line, nil
}
