package example

import (
	"github.com/xdg/testy"
	"testing"
)

func TestExample2(t *testing.T) {
	is := testy.New(t)
	defer func() { t.Logf(is.Done()) }()

	is.True(1+1 == 3) // Line 12
	is.False(2 == 2)  // Line 13
}
