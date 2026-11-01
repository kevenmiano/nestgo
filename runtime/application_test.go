package runtime

import (
	fastadapter "github.com/kevenmiano/nestgo/platform/fasthttp"
	"github.com/kevenmiano/nestgo/platform/nethttp"
	"testing"
)

func TestDefaultHTTPAdapter(t *testing.T) {
	app, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := app.HTTPAdapter().(*fastadapter.Adapter); !ok {
		t.Fatalf("default adapter = %T", app.HTTPAdapter())
	}
}

func TestCustomHTTPAdapter(t *testing.T) {
	adapter := nethttp.New()
	app, err := New(WithAdapter(adapter))
	if err != nil {
		t.Fatal(err)
	}
	if app.HTTPAdapter() != adapter {
		t.Fatal("application did not preserve supplied adapter")
	}
}

func TestApplicationReportsBootstrapMetrics(t *testing.T) {
	called := false
	_, err := New(WithBootstrapMetrics(func(m BootstrapMetrics) {
		called = true
		if m.Total < m.RouteRegistration || m.Total < m.LifecycleInitialization {
			t.Fatalf("invalid bootstrap metrics: %+v", m)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("bootstrap callback not invoked")
	}
}
