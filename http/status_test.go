package http

import (
	stdhttp "net/http"
	"testing"
)

func TestHTTPStatusConstants(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"StatusContinue", StatusContinue, 100},
		{"StatusSwitchingProtocols", StatusSwitchingProtocols, 101},
		{"StatusProcessing", StatusProcessing, 102},
		{"StatusEarlyHints", StatusEarlyHints, 103},
		{"StatusOK", StatusOK, 200},
		{"StatusCreated", StatusCreated, 201},
		{"StatusAccepted", StatusAccepted, 202},
		{"StatusNonAuthoritativeInfo", StatusNonAuthoritativeInfo, 203},
		{"StatusNoContent", StatusNoContent, 204},
		{"StatusResetContent", StatusResetContent, 205},
		{"StatusPartialContent", StatusPartialContent, 206},
		{"StatusMultiStatus", StatusMultiStatus, 207},
		{"StatusAlreadyReported", StatusAlreadyReported, 208},
		{"StatusIMUsed", StatusIMUsed, 226},
		{"StatusMultipleChoices", StatusMultipleChoices, 300},
		{"StatusMovedPermanently", StatusMovedPermanently, 301},
		{"StatusFound", StatusFound, 302},
		{"StatusSeeOther", StatusSeeOther, 303},
		{"StatusNotModified", StatusNotModified, 304},
		{"StatusUseProxy", StatusUseProxy, 305},
		{"StatusTemporaryRedirect", StatusTemporaryRedirect, 307},
		{"StatusPermanentRedirect", StatusPermanentRedirect, 308},
		{"StatusBadRequest", StatusBadRequest, 400},
		{"StatusUnauthorized", StatusUnauthorized, 401},
		{"StatusPaymentRequired", StatusPaymentRequired, 402},
		{"StatusForbidden", StatusForbidden, 403},
		{"StatusNotFound", StatusNotFound, 404},
		{"StatusMethodNotAllowed", StatusMethodNotAllowed, 405},
		{"StatusNotAcceptable", StatusNotAcceptable, 406},
		{"StatusProxyAuthRequired", StatusProxyAuthRequired, 407},
		{"StatusRequestTimeout", StatusRequestTimeout, 408},
		{"StatusConflict", StatusConflict, 409},
		{"StatusGone", StatusGone, 410},
		{"StatusLengthRequired", StatusLengthRequired, 411},
		{"StatusPreconditionFailed", StatusPreconditionFailed, 412},
		{"StatusRequestEntityTooLarge", StatusRequestEntityTooLarge, 413},
		{"StatusRequestURITooLong", StatusRequestURITooLong, 414},
		{"StatusUnsupportedMediaType", StatusUnsupportedMediaType, 415},
		{"StatusRequestedRangeNotSatisfiable", StatusRequestedRangeNotSatisfiable, 416},
		{"StatusExpectationFailed", StatusExpectationFailed, 417},
		{"StatusTeapot", StatusTeapot, 418},
		{"StatusMisdirectedRequest", StatusMisdirectedRequest, 421},
		{"StatusUnprocessableEntity", StatusUnprocessableEntity, 422},
		{"StatusLocked", StatusLocked, 423},
		{"StatusFailedDependency", StatusFailedDependency, 424},
		{"StatusTooEarly", StatusTooEarly, 425},
		{"StatusUpgradeRequired", StatusUpgradeRequired, 426},
		{"StatusPreconditionRequired", StatusPreconditionRequired, 428},
		{"StatusTooManyRequests", StatusTooManyRequests, 429},
		{"StatusRequestHeaderFieldsTooLarge", StatusRequestHeaderFieldsTooLarge, 431},
		{"StatusUnavailableForLegalReasons", StatusUnavailableForLegalReasons, 451},
		{"StatusInternalServerError", StatusInternalServerError, 500},
		{"StatusNotImplemented", StatusNotImplemented, 501},
		{"StatusBadGateway", StatusBadGateway, 502},
		{"StatusServiceUnavailable", StatusServiceUnavailable, 503},
		{"StatusGatewayTimeout", StatusGatewayTimeout, 504},
		{"StatusHTTPVersionNotSupported", StatusHTTPVersionNotSupported, 505},
		{"StatusVariantAlsoNegotiates", StatusVariantAlsoNegotiates, 506},
		{"StatusInsufficientStorage", StatusInsufficientStorage, 507},
		{"StatusLoopDetected", StatusLoopDetected, 508},
		{"StatusNotExtended", StatusNotExtended, 510},
		{"StatusNetworkAuthenticationRequired", StatusNetworkAuthenticationRequired, 511},
		{"StatusAmbiguous", StatusAmbiguous, 300},
		{"StatusPayloadTooLarge", StatusPayloadTooLarge, 413},
		{"StatusURITooLong", StatusURITooLong, 414},
		{"StatusIAmATeapot", StatusIAmATeapot, 418},
		{"StatusMisdirected", StatusMisdirected, 421},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("%s=%d, want %d", tc.name, tc.got, tc.want)
			}
		})
	}
	// Check the core HTTP aliases against Go's standard library.
	for _, tc := range []struct{ got, want int }{
		{StatusOK, stdhttp.StatusOK},
		{StatusNotFound, stdhttp.StatusNotFound},
		{StatusTooManyRequests, stdhttp.StatusTooManyRequests},
		{StatusPayloadTooLarge, stdhttp.StatusRequestEntityTooLarge},
		{StatusHTTPVersionNotSupported, stdhttp.StatusHTTPVersionNotSupported},
	} {
		if tc.got != tc.want {
			t.Fatalf("status alias %d != net/http %d", tc.got, tc.want)
		}
	}
}
