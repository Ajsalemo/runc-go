package main

import (
	"context"
	"fmt"

	runc "github.com/containerd/go-runc"
)

type Conf struct {
	Enabled *bool
}

func listContainers(r runc.Runc, ctx context.Context) error {
	list, err := r.List(ctx)
	if err != nil {
		return fmt.Errorf("error listing containers: %w", err)
	}
	// Check if there are 0 containers. If so, return earlier
	if len(list) == 0 {
		fmt.Println("No containers found")
		return nil
	}
	// Loop through the returned results. Type of list is []runc.Container
	for _, container := range list {
		fmt.Println("Container ID:", container.ID)
		fmt.Println("Container Pid:", container.Pid)
		fmt.Println("Container Bundle:", container.Bundle)
		fmt.Println("Container rootfs:", container.Rootfs)
	}
	return nil
}

func main() {
	// Runc configurations
	isRootless := false

	ctx := context.Background()
	r := runc.Runc{
		Rootless: &isRootless,
	}
	// List all containers actively managed by runc
	listContainers(r, ctx)
}
