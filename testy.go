// Copyright 2015 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package testy is an extensible facade around Go's core testing library.
//
// Go's core testing package doesn't let you refactor repetitive tests
// without reporting errors from the wrong place in the code.  Testy
// implements a facade around the testing package and hijacks its logging
// features.  This means:
//
// * You can report test errors at any level up the call stack.
//
// * You can label all errors in a scope to disambiguate repetitive tests.
//
// The downside is an extra level of log message nesting (which your
// editor's quickfix window should ignore, anyway).
//
// It gives a few convenient helper functions for common cases and makes it
// easy to implement your own.
//
// The following example shows how to set up testy and use test helpers.
// In particular, note the use of 'defer' and a closure to schedule testy
// to output the log at the end of the function's execution.
//
//	package example
//
// 	import (
// 		"github.com/xdg/testy"
// 		"testing"
// 	)
//
// 	func TestExample(t *testing.T) {
// 		is := testy.New(t)
// 		defer func() { t.Logf(is.Done()) }()
//
// 		is.True(1+1 == 3)
// 		is.False(2 == 2)
//
// 		is.Equal(1, 2)
// 		is.Equal(1.0, 1)
// 		is.Equal("foo\tbar", "foo\tbaz")
// 		is.Equal(true, false)
//
// 		is.Unequal(42, 42)
//
// 		is.NotNil(is)
// 	}
//
// Each error will be reported at the calling line.  Calls to 'Equal' and
// 'Unequal' will return diagnostic details.  Here is how some of the output
// would look in Vim's quickfix window:
//
//	...
// 	_examples/example_test.go|15| Values were not equal:
// 	|| 			   Got: 1 (int)
// 	|| 			Wanted: 2 (int)
// 	_examples/example_test.go|16| Values were not equal:
// 	|| 			   Got: 1 (float64)
// 	|| 			Wanted: 1 (int)
// 	_examples/example_test.go|17| Values were not equal:
// 	|| 			   Got: "foo\tbar"
// 	|| 			Wanted: "foo\tbaz"
//	...
//
// You can use the 'Uplevel' and 'Label' methods to return new facades, which
// you can use to implement custom helpers in various ways:
//
// 	func TestExample(t *testing.T) {
// 		is := testy.New(t)
// 		defer func() { t.Logf(is.Done()) }()
//
// 		// Check for numbers equal to 3
// 		for i := 1; i <= 5; i++ {
// 			is.Label("Checking", i).True(i == 3) // Line 13
// 		}
//
// 		// Check for positive, even numbers
// 		for i := -1; i <= 2; i++ {
// 			checkEvenPositive(is, i)             // Line 18
// 		}
// 	}
//
// 	func checkEvenPositive(is *testy.T, n int) {
//		// Report one level up with a custom label
// 		is = is.Uplevel(1).Label("Testing", n)
//
// 		if n < 1 {
// 			is.Error("Value was not positive")
// 		}
// 		if n%2 != 0 {
// 			is.Error("Value was not even")
// 		}
// 	}
//
// The example above would return errors to a quickfix window like this:
// 	...
// 	_examples/example_test.go|13| Checking 1: Expression was not true
// 	_examples/example_test.go|13| Checking 2: Expression was not true
// 	_examples/example_test.go|13| Checking 4: Expression was not true
// 	_examples/example_test.go|13| Checking 5: Expression was not true
// 	_examples/example_test.go|18| Testing -1: Value was not positive
// 	_examples/example_test.go|18| Testing -1: Value was not even
// 	_examples/example_test.go|18| Testing 0: Value was not positive
// 	_examples/example_test.go|18| Testing 1: Value was not even
// 	...
package testy

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// T is a facade around the testing.T type passed to Test functions.  It
// intercepts log messages to attribute them to the correct level of the
// call stack.
type T struct {
	test      *testing.T
	context   *accumulator
	caseName  string
	label     string
	callDepth int
}

var nameStripper = regexp.MustCompile(`^.*\.`)

// New wraps a testy.T struct around a testing.T struct. The resulting
// struct can be used in the same way the testing.T struct would be, plus
// has additional methods specific to Testy.  It calls NewCase with
// the calling function's name as the test case name.
func New(t *testing.T) *T {
	var n string
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		n = nameStripper.ReplaceAllLiteralString(runtime.FuncForPC(pc).Name(), "")

	} else {
		n = "Anonymous function"
	}
	return NewCase(t, n)
}

// NewCase wraps a testy.T struct around a testing.T struct. The resulting
// struct can be used in the same way the testing.T struct would be, plus
// has additional methods specific to Testy.  It takes a name argument
// that is used in the summary line during log output.
func NewCase(t *testing.T, name string) *T {
	return &T{test: t, caseName: name, callDepth: 1, context: &accumulator{}}
}

// Label returns a testy.T struct that will prefix a label to all log
// messages.  The label is constructed by concatenating arguments separated
// by a space (like fmt.Sprintln without the trailing space).  A colon
// character and space will be added automatically
func (t T) Label(s ...interface{}) *T {
	t.label = strings.TrimSpace(fmt.Sprintln(s...)) + ": "
	return &t
}

// Uplevel returns a testy.T struct that will report log messages 'depth'
// frames higher up the call stack.
func (t T) Uplevel(depth int) *T {
	t.callDepth += depth
	return &t
}

// Done returns any test log output formatted suitably for passing to a
// testing.T struct Logf method.
func (t *T) Done() string {
	return t.summary() + strings.Join(t.context.outputCopy(), "\n")
}

// FailCount returns the number of Fail, Error, Fatal or test helper
// failures recorded by the testy.T struct.
func (t T) FailCount() int {
	return t.context.getFailCount()
}

// Output returns a copy of the slice of log messages recorded by the
// testy.T struct.
func (t T) Output() []string {
	return t.context.outputCopy()
}

// Helper functions

// True checks if its argument is true; if false, it logs an error.
func (t *T) True(cond bool) {
	if !cond {
		t.context.incFailCount()
		t.context.log(t.decorate("Expression was not true"))
		t.test.Fail()
	}
}

// False checks if its argument is false; if true, it logs an error.
func (t *T) False(cond bool) {
	if cond {
		t.context.incFailCount()
		t.context.log(t.decorate("Expression was not false"))
		t.test.Fail()
	}
}

func checkNil(x interface{}) bool {
	if x == nil {
		return true
	}

	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}

	return false
}

// Nil checks if its argument is nil (literal or nil slice, map, etc.); if
// non-nil, it logs an error.
func (t *T) Nil(got interface{}) {
	if !checkNil(got) {
		t.context.incFailCount()
		t.context.log(t.decorate("Expression was not nil"))
		t.test.Fail()
	}
}

// Nil checks if its argument is nil (literal or nil slice, map, etc.); if
// non-nil, it logs an error.
func (t *T) NotNil(got interface{}) {
	if checkNil(got) {
		t.context.incFailCount()
		t.context.log(t.decorate("Expression was nil"))
		t.test.Fail()
	}
}

// Equal checks if its arguments are equal using reflect.DeepEqual.  It
// is subject to all the usual limitations of that function.  If the values
// are not equal, an error is logged and the 'got' and 'want' values are
// logged on subsequent lines for comparison.
func (t *T) Equal(got, want interface{}) {
	if got == nil || want == nil {
		t.context.incFailCount()
		t.context.log(t.decorate(
			fmt.Sprintf("Can't safely compare nil values for equality:\n%s%s", diag("   Got", got), diag("Wanted", want))))
		t.test.Fail()
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.context.incFailCount()
		t.context.log(t.decorate(
			fmt.Sprintf("Values were not equal:\n%s%s", diag("   Got", got), diag("Wanted", want))))
		t.test.Fail()
	}
}

// Unequal inverts the logic of Equal but is otherwise similar.
func (t *T) Unequal(got, want interface{}) {
	if got == nil || want == nil {
		t.context.incFailCount()
		t.context.log(t.decorate(
			fmt.Sprintf("Can't safely compare nil values for equality:\n%s%s", diag("   Got", got), diag("Got", want))))
		t.test.Fail()
		return
	}
	if reflect.DeepEqual(got, want) {
		t.context.incFailCount()
		t.context.log(t.decorate(fmt.Sprintf("Values were not unequal:\n%s", diag("  Both", got))))
		t.test.Fail()
	}
}

// Facade functions.  Function definitions and implementations adapted from
// testing.go in the Go core library

// Fail marks the test as having failed.
func (t *T) Fail() {
	t.context.incFailCount()
	t.test.Fail()
}

// FailNow marks the test as having failed and stops execution.  It is
// subject to the same restrictions as FailNow from the testing package.
func (t *T) FailNow() {
	t.context.incFailCount()
	t.test.FailNow()
}

// Failed reports whether the test has been marked as having failed.
func (t *T) Failed() bool {
	return t.test.Failed()
}

// Log joins its arguments by spaces like fmt.Sprintln and records the
// result for later delivery by the Done method.
func (t *T) Log(args ...interface{}) {
	t.context.log(t.decorate(fmt.Sprintln(args...)))
}

// Logf joins its arguments like fmt.Sprintf and records the result for later
// delivery by the Done method.
func (t *T) Logf(format string, args ...interface{}) {
	t.context.log(t.decorate(fmt.Sprintf(format, args...)))
}

// Error is equivalent to Log followed by Fail
func (t *T) Error(args ...interface{}) {
	t.context.incFailCount()
	t.context.log(t.decorate(fmt.Sprintln(args...)))
	t.test.Fail()
}

// Errorf is equivalent to Logf followed by Fail
func (t *T) Errorf(format string, args ...interface{}) {
	t.context.incFailCount()
	t.context.log(t.decorate(fmt.Sprintf(format, args...)))
	t.test.Fail()
}

// Fatal is equivalent to Log followed by FailNow
func (t *T) Fatal(args ...interface{}) {
	t.context.incFailCount()
	t.context.log(t.decorate(fmt.Sprintln(args...)))
	t.test.FailNow()
}

// Fatalf is equivalent to Logf followed by FailNow
func (t *T) Fatalf(format string, args ...interface{}) {
	t.context.incFailCount()
	t.context.log(t.decorate(fmt.Sprintf(format, args...)))
	t.test.FailNow()
}

// Skip is equivalent to Log followed by SkipNow
func (t *T) Skip(args ...interface{}) {
	t.context.log(t.decorate(fmt.Sprintln(args...)))
	t.test.SkipNow()
}

// Skipf is equivalent to Logf followed by SkipNow
func (t *T) Skipf(format string, args ...interface{}) {
	t.context.log(t.decorate(fmt.Sprintf(format, args...)))
	t.test.SkipNow()
}

// SkipNow marks the test as skipped and halts execution. It is subject to
// the same restrictions as SkipNow from the testing package.
func (t *T) SkipNow() {
	t.test.SkipNow()
}

// Skipped reports whether the test was skipped.
func (t *T) Skipped() bool {
	return t.test.Skipped()
}

func (t T) summary() string {
	count := t.context.getFailCount()

	if count == 0 {
		return fmt.Sprintf("%s: all tests passed\n", t.caseName)
	}

	if count == 1 {
		return fmt.Sprintf("%s: %d test failed\n", t.caseName, count)
	}

	return fmt.Sprintf("%s: %d tests failed\n", t.caseName, count)
}

// copied from core testing package for formatting similarity
func (t T) decorate(s string) string {
	// decorate + public func depth
	_, file, line, ok := runtime.Caller(1 + t.callDepth)
	if ok {
		// Truncate file name at last file name separator.
		if index := strings.LastIndex(file, "/"); index >= 0 {
			file = file[index+1:]
		} else if index = strings.LastIndex(file, "\\"); index >= 0 {
			file = file[index+1:]
		}
	} else {
		file = "???"
		line = 1
	}
	buf := new(bytes.Buffer)
	// Every line is indented at least one tab.
	buf.WriteByte('\t')
	fmt.Fprintf(buf, "%s:%d: %s", file, line, t.label)
	lines := strings.Split(s, "\n")
	if l := len(lines); l > 1 && lines[l-1] == "" {
		lines = lines[:l-1]
	}
	for i, line := range lines {
		if i > 0 {
			// Unlike package testing, second and subsequent lines are NOT
			// indented an extra tab as package testing will do it for us.
			buf.WriteString("\n\t")
		}
		buf.WriteString(line)
	}
	buf.WriteByte('\n')
	return buf.String()
}

// Accumulator stores test results and guards concurrent access

type accumulator struct {
	mutex     sync.RWMutex
	failCount int
	output    []string // any logging, not just failures
}

func (a *accumulator) getFailCount() int {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.failCount
}

func (a *accumulator) outputCopy() []string {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	out := make([]string, len(a.output))
	for i, v := range a.output {
		out[i] = v
	}
	return out
}

func (a *accumulator) log(s string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.output = append(a.output, strings.TrimSpace(s))
}

func (a *accumulator) incFailCount() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.failCount++
}

// internal comparison support functions

func diag(prefix string, value interface{}) string {
	if value == nil {
		return fmt.Sprintf("%s: nil\n", prefix)
	}
	vType := reflect.TypeOf(value)
	switch vType.Kind() {
	case reflect.String:
		return fmt.Sprintf("%s: %s\n", prefix, strconv.Quote(value.(string)))
	case reflect.Bool:
		return fmt.Sprintf("%s: %v\n", prefix, value)
	default:
		return fmt.Sprintf("%s: %v (%v)\n", prefix, value, vType)
	}
}
