package main

import (
	"github.com/teknogeek/ssrf-sheriff/handler"
	"go.uber.org/fx"
)

func main() {
	fx.New(opts()).Run()
}

func opts() fx.Option {
	return fx.Options(
		fx.Provide(
			handler.NewLogger,
			handler.NewConfigProvider,
			handler.NewSSRFSheriffRouter,
			handler.NewServerRouter,
			handler.NewHTTPServer,
		),
		fx.Invoke(handler.StartFilesGenerator, handler.StartServer),
	)
}
