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

type T struct {
	test      *testing.T
	context   *accumulator
	caseName  string
	label     string
	callDepth int
}

var nameStripper *regexp.Regexp = regexp.MustCompile(`^.*\.`)

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

func NewCase(t *testing.T, name string) *T {
	return &T{test: t, caseName: name, callDepth: 1, context: &accumulator{}}
}

func (t T) Label(s ...interface{}) *T {
	t.label = strings.TrimSpace(fmt.Sprintln(s...)) + ": "
	return &t
}

func (t T) Uplevel(depth int) *T {
	t.callDepth += depth
	return &t
}

func (t *T) Done() string {
	return t.summary() + strings.Join(t.context.outputCopy(), "\n")
}

func (t T) FailCount() int {
	return t.context.getFailCount()
}

func (t T) Output() []string {
	return t.context.outputCopy()
}

// Helper functions

func (t *T) True(cond bool) {
	if !cond {
		t.context.incFailCount()
		t.context.log(t.decorate("Expression was not true"))
		t.test.Fail()
	}
}

func (t *T) False(cond bool) {
	if cond {
		t.context.incFailCount()
		t.context.log(t.decorate("Expression was not false"))
		t.test.Fail()
	}
}

// Facade functions.  Function definitions and implementations adapted from
// testing.go in the Go core library

func (t *T) Fail() {
	t.context.incFailCount()
	t.test.Fail()
}

func (t *T) FailNow() {
	t.context.incFailCount()
	t.test.FailNow()
}

func (t *T) Failed() bool {
	return t.test.Failed()
}

func (t *T) Log(args ...interface{}) {
	t.context.log(t.decorate(fmt.Sprintln(args...)))
}

func (t *T) Logf(format string, args ...interface{}) {
	t.context.log(t.decorate(fmt.Sprintf(format, args...)))
}

func (t *T) Error(args ...interface{}) {
	t.context.incFailCount()
	t.context.log(t.decorate(fmt.Sprintln(args...)))
	t.test.Fail()
}

func (t *T) Errorf(format string, args ...interface{}) {
	t.context.incFailCount()
	t.context.log(t.decorate(fmt.Sprintf(format, args...)))
	t.test.Fail()
}

func (t *T) Fatal(args ...interface{}) {
	t.context.incFailCount()
	t.context.log(t.decorate(fmt.Sprintln(args...)))
	t.test.FailNow()
}

func (t *T) Fatalf(format string, args ...interface{}) {
	t.context.incFailCount()
	t.context.log(t.decorate(fmt.Sprintf(format, args...)))
	t.test.FailNow()
}

func (t *T) Skip(args ...interface{}) {
	t.context.log(t.decorate(fmt.Sprintln(args...)))
	t.test.SkipNow()
}

func (t *T) Skipf(format string, args ...interface{}) {
	t.context.log(t.decorate(fmt.Sprintf(format, args...)))
	t.test.SkipNow()
}

func (t *T) SkipNow() {
	t.test.SkipNow()
}

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
			// Second and subsequent lines are indented an extra tab.
			buf.WriteString("\n\t\t")
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