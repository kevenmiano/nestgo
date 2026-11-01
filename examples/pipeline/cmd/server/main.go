package main

import (
	"context"
	"log"
	"os"

	_ "github.com/kevenmiano/nestgo/examples/pipeline"
	"github.com/kevenmiano/nestgo/httpadapter"
	"github.com/kevenmiano/nestgo/runtime"
)

func main() {
	if os.Getenv("DEMO_API_KEY") == "" {
		log.Fatal("configure DEMO_API_KEY before starting the example server")
	}
	app, err := runtime.New(
		runtime.WithMiddleware(func(ctx httpadapter.Context, next httpadapter.Next) error {
			ctx.SetHeader("X-Example", "pipeline")
			return next()
		}),
		runtime.WithRoutes(runtime.Route{
			Method: "GET",
			Path:   "/health",
			Handler: func(ctx runtime.Context) error {
				ctx.Status(200)
				return ctx.Write([]byte("ok"))
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	log.Fatal(app.Listen(":8080"))
}
