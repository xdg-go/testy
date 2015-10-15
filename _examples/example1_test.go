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
