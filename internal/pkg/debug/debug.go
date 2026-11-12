// Package debug provides debug utilities for the application
package debug

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	runtime "runtime/debug"

	"kinstack/internal/pkg/console"

	"github.com/DataDog/gostackparse"
)

type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

type StackFrameContext struct {
	Frame      StackFrame
	CodeLines  []CodeLine
	IsInternal bool
}

type CodeLine struct {
	Number    int
	Content   string
	IsCurrent bool
}

func StackTrace() ([]StackFrame, error) {
	stack := runtime.Stack()
	goroutines, err := gostackparse.Parse(bytes.NewReader(stack))
	if err != nil || len(err) > 0 {
		return nil, err[0]
	}

	frames := make([]StackFrame, 0)

	for _, goroutine := range goroutines {
		for _, stack := range goroutine.Stack {
			frames = append(frames, StackFrame{
				Function: stack.Func,
				Line:     stack.Line,
				File:     stack.File,
			})
		}
	}

	return frames, nil
}

func Print(vs ...any) {
	if len(vs) == 0 {
		return
	}
	for _, v := range vs {
		if printValue(v) {
			break
		}
	}
	os.Exit(1)
}

// printValue writes one value and reports whether it ended the dump, which a
// nil value does so the output is never half of two different arguments.
func printValue(v any) (stop bool) {
	if v == nil {
		console.Line("Type: <nil>\nValue: nil")
		return true
	}
	value := reflect.ValueOf(v)
	valueType := reflect.TypeOf(v)
	console.Line("Type: %v", valueType)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			console.Line("Value: nil pointer")
			return true
		}
		console.Line("Pointer Address: %p", v)
		value = value.Elem()
		valueType = value.Type()
	}
	if encoded, err := json.MarshalIndent(v, "", "  "); err == nil {
		console.Line("Value:\n%s", string(encoded))
		return false
	}
	console.Line("Value: %+v", v)
	if value.Kind() == reflect.Struct {
		printStructFields(value, valueType)
	}
	return false
}

func printStructFields(value reflect.Value, valueType reflect.Type) {
	console.Line("Fields:")
	for index := 0; index < value.NumField(); index++ {
		field := valueType.Field(index)
		console.Line("  %s (%v) = %v", field.Name, field.Type, value.Field(index).Interface())
	}
}
