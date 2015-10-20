// Copyright 2015 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package testy is an extensible facade around Go's core testing library
package testy

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
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
// by a space (like fmt.Sprintln without the trailing space).
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
