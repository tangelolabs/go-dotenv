package dotenv

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type empty struct{}

var withOverridePackagePath = reflect.TypeOf(empty{}).PkgPath() + ".WithOverride"

var overrideStack = sync.Map{}

// WithOverride overrides the environment variables with the given ones
// and restores them after the callback is executed.
//
// Any call to the Load, LoadAndParse or similar within the callback will be
// affected by the overridden values.
//
// This function will panic if the number of arguments is not even, or if there is
// an error setting or unsetting the environment variables.
//
// Typical Usage Example:
//
//	dotenv.WithOverride(func() {
//	   functionThatCallsLoadAndParse()
//	}, "FOO", "bar")
func WithOverride(callback func(), kv ...string) {
	if len(kv)%2 != 0 {
		panic("dotenv.WithOverride requires an even number of arguments")
	}

	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)

	tuples := make(map[string]string, len(kv)/2)

	for i := 0; i < len(kv); i += 2 {
		k := kv[i]
		v := kv[i+1]

		tuples[k] = v
	}

	key := fmt.Sprintf("%d:%d:%s", goid(), pc[0], runtime.FuncForPC(pc[0]).Name())

	overrideStack.Store(key, tuples)
	callback()
	overrideStack.Delete(key)
}

func isOverridden() (map[string]string, bool) {
	var pc [maxStackLen]uintptr

	n := runtime.Callers(2, pc[:])
	override := false
	tuples := make(map[string]string)

	for i := 0; i < n; i++ {
		f := runtime.FuncForPC(pc[i])
		if f == nil {
			continue
		}

		if f.Name() == withOverridePackagePath {
			override = true
			overrider := runtime.FuncForPC(pc[i+1])

			if overrider != nil {
				key := fmt.Sprintf("%d:%d:%s", goid(), pc[i+1], overrider.Name())
				found, ok := overrideStack.Load(key)

				if ok {
					pairs, validType := found.(map[string]string)
					if validType {
						tuples = pairs
					}
				}
			}

			break
		}
	}

	return tuples, override
}

func goid() int {
	var buf [64]byte

	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)

	if err != nil {
		return 0
	}

	return id
}
