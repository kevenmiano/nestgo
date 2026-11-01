package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/kevenmiano/nestgo/metadata"
)

func TestNewRejectsInvalidFunctionalOptions(t *testing.T) {
	cases := []struct {
		name    string
		options []Option
		want    string
	}{
		{"nil option", []Option{nil}, "nil application option"},
		{"mixed metadata sources", []Option{WithPackages(metadata.Package{}), WithRegistry("shop")}, "cannot be combined"},
		{"unregistered package", []Option{WithRegistry("missing-single-application-fixture")}, "metadata package not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, err := New(tc.options...)
			if app != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("New returned (%v, %v), expected %q", app, err, tc.want)
			}
		})
	}
}

func TestNewDefaultOptionsAndLifecycleOwner(t *testing.T) {
	app, err := New(WithPackages(metadata.Package{}))
	if err != nil {
		t.Fatal(err)
	}
	if app.HTTPAdapter() == nil {
		t.Fatal("missing HTTP engine")
	}
	if app.lifecycle == nil {
		t.Fatal("missing lifecycle owner")
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
