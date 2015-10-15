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
