// Status constants mirror net/http so NestGo controllers can use the same
// status vocabulary regardless of the configured HTTP engine.
package http

import stdhttp "net/http"

const (
	// 1xx responses.
	StatusContinue                            = stdhttp.StatusContinue
	StatusSwitchingProtocols                  = stdhttp.StatusSwitchingProtocols
	StatusProcessing                          = stdhttp.StatusProcessing
	StatusEarlyHints                          = stdhttp.StatusEarlyHints

	// 2xx responses.
	StatusOK                                  = stdhttp.StatusOK
	StatusCreated                             = stdhttp.StatusCreated
	StatusAccepted                            = stdhttp.StatusAccepted
	StatusNonAuthoritativeInfo                = stdhttp.StatusNonAuthoritativeInfo
	StatusNoContent                           = stdhttp.StatusNoContent
	StatusResetContent                        = stdhttp.StatusResetContent
	StatusPartialContent                      = stdhttp.StatusPartialContent
	StatusMultiStatus                         = stdhttp.StatusMultiStatus
	StatusAlreadyReported                     = stdhttp.StatusAlreadyReported
	StatusIMUsed                              = stdhttp.StatusIMUsed

	// 3xx responses.
	StatusMultipleChoices                     = stdhttp.StatusMultipleChoices
	StatusMovedPermanently                    = stdhttp.StatusMovedPermanently
	StatusFound                               = stdhttp.StatusFound
	StatusSeeOther                            = stdhttp.StatusSeeOther
	StatusNotModified                         = stdhttp.StatusNotModified
	StatusUseProxy                            = stdhttp.StatusUseProxy
	StatusTemporaryRedirect                   = stdhttp.StatusTemporaryRedirect
	StatusPermanentRedirect                   = stdhttp.StatusPermanentRedirect

	// 4xx responses.
	StatusBadRequest                          = stdhttp.StatusBadRequest
	StatusUnauthorized                        = stdhttp.StatusUnauthorized
	StatusPaymentRequired                     = stdhttp.StatusPaymentRequired
	StatusForbidden                           = stdhttp.StatusForbidden
	StatusNotFound                            = stdhttp.StatusNotFound
	StatusMethodNotAllowed                    = stdhttp.StatusMethodNotAllowed
	StatusNotAcceptable                       = stdhttp.StatusNotAcceptable
	StatusProxyAuthRequired                   = stdhttp.StatusProxyAuthRequired
	StatusRequestTimeout                      = stdhttp.StatusRequestTimeout
	StatusConflict                            = stdhttp.StatusConflict
	StatusGone                                = stdhttp.StatusGone
	StatusLengthRequired                      = stdhttp.StatusLengthRequired
	StatusPreconditionFailed                  = stdhttp.StatusPreconditionFailed
	StatusRequestEntityTooLarge               = stdhttp.StatusRequestEntityTooLarge
	StatusRequestURITooLong                   = stdhttp.StatusRequestURITooLong
	StatusUnsupportedMediaType                = stdhttp.StatusUnsupportedMediaType
	StatusRequestedRangeNotSatisfiable        = stdhttp.StatusRequestedRangeNotSatisfiable
	StatusExpectationFailed                   = stdhttp.StatusExpectationFailed
	StatusTeapot                              = stdhttp.StatusTeapot
	StatusMisdirectedRequest                  = stdhttp.StatusMisdirectedRequest
	StatusUnprocessableEntity                 = stdhttp.StatusUnprocessableEntity
	StatusLocked                              = stdhttp.StatusLocked
	StatusFailedDependency                    = stdhttp.StatusFailedDependency
	StatusTooEarly                            = stdhttp.StatusTooEarly
	StatusUpgradeRequired                     = stdhttp.StatusUpgradeRequired
	StatusPreconditionRequired                = stdhttp.StatusPreconditionRequired
	StatusTooManyRequests                     = stdhttp.StatusTooManyRequests
	StatusRequestHeaderFieldsTooLarge         = stdhttp.StatusRequestHeaderFieldsTooLarge
	StatusUnavailableForLegalReasons          = stdhttp.StatusUnavailableForLegalReasons

	// 5xx responses.
	StatusInternalServerError                 = stdhttp.StatusInternalServerError
	StatusNotImplemented                      = stdhttp.StatusNotImplemented
	StatusBadGateway                          = stdhttp.StatusBadGateway
	StatusServiceUnavailable                  = stdhttp.StatusServiceUnavailable
	StatusGatewayTimeout                      = stdhttp.StatusGatewayTimeout
	StatusHTTPVersionNotSupported             = stdhttp.StatusHTTPVersionNotSupported
	StatusVariantAlsoNegotiates               = stdhttp.StatusVariantAlsoNegotiates
	StatusInsufficientStorage                 = stdhttp.StatusInsufficientStorage
	StatusLoopDetected                        = stdhttp.StatusLoopDetected
	StatusNotExtended                         = stdhttp.StatusNotExtended
	StatusNetworkAuthenticationRequired       = stdhttp.StatusNetworkAuthenticationRequired

	// NestJS-style names for codes whose Go standard-library names differ.
	StatusAmbiguous                           = StatusMultipleChoices
	StatusPayloadTooLarge                     = StatusRequestEntityTooLarge
	StatusURITooLong                          = StatusRequestURITooLong
	StatusIAmATeapot                          = StatusTeapot
	StatusMisdirected                         = StatusMisdirectedRequest
)
