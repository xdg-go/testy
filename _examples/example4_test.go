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
