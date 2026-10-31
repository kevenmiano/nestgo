package runtime

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestDefaultHTTPEngine(t *testing.T) {
    app, err := NewApplication(ApplicationOptions{})
    if err != nil { t.Fatal(err) }
    if app.Engine() != EngineFastHTTP { t.Fatalf("unexpected default engine: %s", app.Engine()) }
}

func TestNetHTTPEngine(t *testing.T) {
    app, err := NewApplication(ApplicationOptions{Engine: EngineNetHTTP})
    if err != nil { t.Fatal(err) }
    recorder := httptest.NewRecorder()
    app.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missing", nil))
    if recorder.Code != http.StatusNotFound { t.Fatalf("status = %d", recorder.Code) }
}

func TestUnsupportedHTTPEngine(t *testing.T) {
    if _, err := NewApplication(ApplicationOptions{Engine: "invalid"}); err == nil {
        t.Fatal("expected unsupported engine error")
    }
}
