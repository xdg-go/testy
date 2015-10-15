# Testy – An extensible testing facade

**If Go's standard testing package annoys you, you might like Testy**

There is a lot to like about Go's [testing](https://golang.org/pkg/testing/)
package.

There are also two extremely annoying things about it:

1. You can't refactor repetitive tests without reporting errors from the
   wrong place in the code.
2. The testing package is locked down tight, preventing trivial solutions
   to the prior problem.

The Go FAQ [suggests](https://golang.org/doc/faq#testing_framework) that
table-driven testing is the way to avoid repetitive test code.  If that
works for you for all situations, fine.

Testy has a different solution.

**Testy implements a facade around the testing package and hijacks its
logging features.**

This means:

* You can report test errors at any level up the call stack.
* You can label all errors in a scope to disambiguate repetitive tests.

The downside is an extra level of log message nesting (which your
editor's quickfix window should ignore, anyway).

It also gives a few convenient helper functions for common cases.

# Examples

## Using a custom helper function

Consider this [simple example](/_examples/example1_test.go):

```go
package example

import (
	"github.com/xdg/testy"
	"testing"
)

func TestExample1(t *testing.T) {
	is := testy.New(t)
	defer func() { t.Logf(is.Done()) }() // Line 10
	is.Error("First failure")            // Line 11
	checkTrue(is, 1+1 == 3)              // Line 12
}

func checkTrue(is *testy.T, cond bool) {
	if !cond {
		is.Uplevel(1).Error("Expression was not true")
	}
}
```

In the `TestExample1` function, the `is` variable wraps the test variable,
`t`.  The `defer` closure schedules test logging output to be delivered to
`t` via `is.Done()` when the test function exits.

When run in Vim, with [vim-go](https://github.com/fatih/vim-go), the
quickfix window looks like this:

```
_examples/example1_test.go|10| TestExample1: 2 tests failed
_examples/example1_test.go|11| First failure
_examples/example1_test.go|12| Expression was not true
```

Note that the `checkTrue` error is reported from the call to `checkTrue` at
line 12, not from inside the `checkTrue` function.  The `Uplevel` method in
`checkTrue` tells Testy to report the error one level up the call stack.

## Using Testy helpers

The `checkTrue` pattern is so common that testing true and false are
built-in to Testy:

```go
package example

import (
	"github.com/xdg/testy"
	"testing"
)

func TestExample2(t *testing.T) {
	is := testy.New(t)
	defer func() { t.Logf(is.Done()) }()

	is.True(1+1 == 3)
	is.False(2 == 2)
}
```

## Using error labels

To prefix error messages with some descriptive text, you can use the
`Label` method like this:

```go
package example

import (
	"github.com/xdg/testy"
	"testing"
)

func TestExample3(t *testing.T) {
	is := testy.New(t)
	defer func() { t.Logf(is.Done()) }()

	for i := 1; i <= 5; i++ {
		is.Label("Checking", i).True(i == 3) // Line 13
	}
}
```

The output with labels looks like this, making it clear which tests failed:


```
_examples/example3_test.go|10| TestExample3: 4 tests failed
_examples/example3_test.go|13| Checking 1: Expression was not true
_examples/example3_test.go|13| Checking 2: Expression was not true
_examples/example3_test.go|13| Checking 4: Expression was not true
_examples/example3_test.go|13| Checking 5: Expression was not true
```

## Combining Uplevel and Label in a new facade

Because `Uplevel` and `Label` just return new facades, you can chain them
at the start of a helper function and modify all subsequent logging:

```go
package example

import (
	"github.com/xdg/testy"
	"testing"
)

func TestExample4(t *testing.T) {
	is := testy.New(t)
	defer func() { t.Logf(is.Done()) }()

	for i := -1; i <= 2; i++ {
		checkEvenPositive(is, i) // Line 13
	}
}

func checkEvenPositive(is *testy.T, n int) {
	// replace 'is' with a labeled, upleveled equivalent
	is = is.Uplevel(1).Label("Testing", n)

	if n < 1 {
		is.Error("was not positive")
	}
	if n%2 != 0 {
		is.Error("was not even")
	}
}
```

This lets you write test helpers that report errors where they are
called (line 13 in this case), but with detailed errors you can
tie back to the original input data:

```
_examples/example4_test.go|10| TestExample4: 4 tests failed
_examples/example4_test.go|13| Testing -1: was not positive
_examples/example4_test.go|13| Testing -1: was not even
_examples/example4_test.go|13| Testing 0: was not positive
_examples/example4_test.go|13| Testing 1: was not even
```

# Copyright and License

Copyright 2015 by David A. Golden. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License"). You may
obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
