package main

import (
	"context"
	"flag"
	"time"

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
// Start the container
// runc.CreateOptions starts the container in `Detached` mode so that the invocation doesn't block
// This is the equivalen of 'runc run -d <container_id> <bundle_path> mycontainer'
// Runc 'run' combines 'create' and 'start' functionality
func runContainer(r runc.Runc, ctx context.Context) error {
	i, err := r.Run(ctx, "mycontainer", "./", &runc.CreateOpts{Detach: true})
	if err != nil {
		zap.L().Error("Error starting container", zap.Error(err))
		return err
	}
	zap.L().Info("Container started successfully with PID:", zap.Int("pid", i))
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
		// Run rootless, otherwise we'll probably hit errors
		Rootless: &isRootless,
	}
	// List all containers actively managed by runc
	if *listRuncContainers {
		listContainers(r, ctx)
	}
	// Start the container
	// runc.CreateOptions starts the container in `Detached` mode so that the invocation doesn't block
	// This is the equivalen of 'runc run -d <container_id> <bundle_path> mycontainer'
	// Runc 'run' combines 'create' and 'start' functionality
	err := runContainer(r, ctx)
	if err != nil {
		return 
	}
}
