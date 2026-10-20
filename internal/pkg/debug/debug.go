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

// Print takes any value and pretty prints it with indentation
func Print(vs ...any) {
	if len(vs) == 0 {
		return
	}

	for _, v := range vs {
		t := reflect.TypeOf(v)
		val := reflect.ValueOf(v)

		if v == nil {
			console.Line("Type: <nil>\nValue: nil")
			return
		}

		console.Line("Type: %v", t)
		if val.Kind() == reflect.Pointer {
			if val.IsNil() {
				console.Line("Value: nil pointer")
				return
			}
			console.Line("Pointer Address: %p", v)
			val = val.Elem()
			t = val.Type()
		}

		jsonBytes, err := json.MarshalIndent(v, "", "  ")
		if err == nil {
			console.Line("Value:\n%s", string(jsonBytes))
		} else {
			console.Line("Value: %+v", v)
			if val.Kind() == reflect.Struct {
				console.Line("Fields:")
				for i := 0; i < val.NumField(); i++ {
					field := t.Field(i)
					value := val.Field(i)
					console.Line("  %s (%v) = %v", field.Name, field.Type, value.Interface())
				}
			}
		}
	}

	os.Exit(1)
}
