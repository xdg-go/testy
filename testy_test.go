// Copyright 2015 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testy_test

import (
	"github.com/xdg/testy"
	"regexp"
	"testing"
)

func TestFacade(t *testing.T) {
	mock := &testing.T{}
	test := testy.New(mock)

	if test.FailCount() != 0 {
		t.Errorf("FailCount didn't start at zero")
	}

	// inject a failure
	test.Fail()

	if !test.Failed() {
		t.Errorf("Failure not recorded in facade")
	}

	if !mock.Failed() {
		t.Errorf("Failure not recorded in test object")
	}
}

func TestHelpers(t *testing.T) {
	mock := &testing.T{}
	test := testy.New(mock)

	// not failures
	test.True(true)
	test.False(false)

	// failures on lines 44 and 45; check later
	test.True(false)
	test.False(true)

	fc := test.FailCount()
	if fc != 2 {
		t.Errorf("Incorrect FailCount. Got %d, but expected %d", fc, 2)
	}

	output := test.Output()
	if ok, _ := regexp.MatchString("testy_test.go:44: Expression was not true", output[0]); !ok {
		t.Errorf("True() had wrong error message: '%s'", output[0])
	}
	if ok, _ := regexp.MatchString("testy_test.go:45: Expression was not false", output[1]); !ok {
		t.Errorf("False() had wrong error message: '%s'", output[1])
	}
}
