package runtime

import (
stdhttp "net/http"

	"github.com/kevenmiano/nestgo/http"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/metadata"
)

func defaultHTTPStatus(route metadata.Route) int {
	if route.Status != 0 {
		return route.Status
	}
	if route.Method == stdhttp.MethodPost {
		return stdhttp.StatusCreated
	}
	return stdhttp.StatusOK
}

func routeResponseHeaders(ctx httpadapter.Context, route metadata.Route) {
	for _, header := range route.Headers {
		ctx.SetHeader(header.Name, header.Value)
	}
}

func finishHTTPResponse(ctx httpadapter.Context, route metadata.Route, result any, writer *http.Writer, explicit, passthrough bool) error {
	if writer != nil && writer.Written() {
		return nil
	}
	if explicit && !passthrough {
		return http.WriteError(ctx, http.InternalServerError("explicit @Res() handler did not write a response"))
	}
	if route.RedirectURL != "" {
		switch value := result.(type) {
		case http.Response:
			if value.RedirectURL != "" {
				return http.Write(ctx, value, route.RedirectStatus)
			}
		case *http.Response:
			if value != nil && value.RedirectURL != "" {
				return http.Write(ctx, value, route.RedirectStatus)
			}
		}
		return http.Write(ctx, http.Redirect(route.RedirectURL, route.RedirectStatus), route.RedirectStatus)
	}
	status := defaultHTTPStatus(route)
	if writer != nil && passthrough && writer.StatusCode() != 0 {
		status = writer.StatusCode()
	}
	if result == nil && writer != nil && passthrough {
		return http.Write(ctx, http.Response{StatusCode: status, ContentType: "text/plain; charset=utf-8", Body: ""}, status)
	}
	return http.Write(ctx, result, status)
}
