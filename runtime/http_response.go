package runtime

import (
	"net/http"

	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/httpresponse"
	"github.com/kevenmiano/nestgo/metadata"
)

func defaultHTTPStatus(route metadata.Route) int {
	if route.Status != 0 { return route.Status }
	if route.Method == http.MethodPost { return http.StatusCreated }
	return http.StatusOK
}

func routeResponseHeaders(ctx httpadapter.Context, route metadata.Route) {
	for _, header := range route.Headers { ctx.SetHeader(header.Name, header.Value) }
}

func finishHTTPResponse(ctx httpadapter.Context, route metadata.Route, result any, writer *httpresponse.Writer, explicit, passthrough bool) error {
	if writer != nil && writer.Written() { return nil }
	if explicit && !passthrough {
		return httpresponse.WriteError(ctx, httpresponse.InternalServerError("explicit @Res() handler did not write a response"))
	}
	if route.RedirectURL != "" {
		switch value := result.(type) {
		case httpresponse.Response:
			if value.RedirectURL != "" { return httpresponse.Write(ctx, value, route.RedirectStatus) }
		case *httpresponse.Response:
			if value != nil && value.RedirectURL != "" { return httpresponse.Write(ctx, value, route.RedirectStatus) }
		}
		return httpresponse.Write(ctx, httpresponse.Redirect(route.RedirectURL, route.RedirectStatus), route.RedirectStatus)
	}
	status := defaultHTTPStatus(route)
	if writer != nil && passthrough && writer.StatusCode() != 0 { status = writer.StatusCode() }
	if result == nil && writer != nil && passthrough {
		return httpresponse.Write(ctx, httpresponse.Response{StatusCode:status,ContentType:"text/plain; charset=utf-8",Body:""},status)
	}
	return httpresponse.Write(ctx, result, status)
}
