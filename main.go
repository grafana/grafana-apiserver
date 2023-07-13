package main

import (
	"os"

	"github.com/grafana/grafana-apiserver/pkg/cmd/server"
	"github.com/grafana/grafana-apiserver/pkg/cmd/server/options"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
)

func main() {
	stopCh := genericapiserver.SetupSignalHandler()
	options := options.NewGrafanaAPIServerOptions(os.Stdout, os.Stderr)
	cmd := server.NewServerCommand(options, stopCh)
	code := cli.Run(cmd)
	os.Exit(code)
}
