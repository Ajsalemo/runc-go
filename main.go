package main

import (
	"context"
	"flag"

	runc "github.com/containerd/go-runc"
	zap "go.uber.org/zap"
)

type Conf struct {
	Enabled *bool
}

func init() {
	zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
}

func listContainers(r runc.Runc, ctx context.Context) error {
	list, err := r.List(ctx)
	if err != nil {
		zap.L().Error("Error listing containers", zap.Error(err))
		return err
	}
	// Check if there are 0 containers. If so, return earlier
	if len(list) == 0 {
		zap.L().Info("No containers found")
		return nil
	}
	// Loop through the returned results. Type of list is []runc.Container
	for _, container := range list {
		zap.L().Info("Container ID:", zap.String("id", container.ID))
	}
	return nil
}

func main() {
	// Command line args to invoke various runc functionality
	listRuncContainers := flag.Bool("list-containers", false, "List containers started by runc")
	flag.Parse()
	// Runc configurations
	isRootless := false
	ctx := context.Background()
	r := runc.Runc{
		Rootless: &isRootless,
	}
	// List all containers actively managed by runc
	if *listRuncContainers {
		listContainers(r, ctx)
	}
}
