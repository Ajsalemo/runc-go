package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// List all containers managed by runc
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
//
// runc.CreateOptions starts the container in `Detached` mode so that the invocation doesn't block
// This is the equivalent of 'runc run -d <container_id> <bundle_path> mycontainer'
//
// runc 'run' combines 'create' and 'start' functionality
func runContainer(r runc.Runc, containerName string, ctx context.Context) error {
	zap.L().Info("Starting container", zap.String("container", containerName))
	zap.L().Info("Bundle path is set to ./")
	i, err := r.Run(ctx, containerName, "./", &runc.CreateOpts{Detach: true})
	if err != nil {
		zap.L().Error("Error starting container", zap.Error(err))
		return err
	}
	zap.L().Info("Container started successfully with PID:", zap.Int("pid", i), zap.String("container", containerName))

	return nil
}

// Monitor container metrics for cpu, memory and pid (pid shows number of pids in the container, not the actual pids)
//
// Example usage: --monitor-container --metric-type "cpu" --metric-interval 3 --container-name "gotcpservertls"
func monitorContainerResources(r runc.Runc, containerName string, ctx context.Context, metricType string, eventInterval time.Duration) error {
	zap.L().Info("Monitoring container resources, this may take a few seconds to start outputting information", zap.String("metric_type", metricType), zap.String("container", containerName))
	zap.L().Info("Press Ctrl+C to stop monitoring")
	event, err := r.Events(ctx, containerName, eventInterval)
	if err != nil {
		zap.L().Error("Error getting events", zap.Error(err))
		return err
	}
	for e := range event {
		if e.Err != nil {
			zap.L().Error("Error in event stream", zap.Error(e.Err))
			continue
		}
		switch metricType {
		// Display metricType of CPU for the container
		case "cpu":
			if e.Stats != nil {
				zap.L().Info("CPU Usage (in nanoseconds):", zap.Uint64("cpu_usage", e.Stats.Cpu.Usage.User))
			}
		// Display metricType of memory for the container
		case "memory":
			if e.Stats != nil {
				zap.L().Info("Memory Usage (in bytes):", zap.Uint64("memory_usage", e.Stats.Memory.Usage.Usage))
			}
		case "pid":
			if e.Stats != nil {
				zap.L().Info("Number of PIDs:", zap.Uint64("pids_current", e.Stats.Pids.Current))
			}
		// If a user provides an unsupported metricType, log an error and return out of this
		default:
			zap.L().Error("Unsupported metric type - supported types are 'cpu', 'memory', 'pid'", zap.String("metric_type", metricType))
			return nil
		}

	}
	return nil
}
// Delete the container
func deleteRuncContainer(r runc.Runc, containerName string, ctx context.Context) error {
	zap.L().Info("Deleting container", zap.String("container", containerName))
	err := r.Delete(ctx, containerName, &runc.DeleteOpts{Force: true})
	if err != nil {
		zap.L().Error("Error deleting container", zap.Error(err))
		return err
	}
	zap.L().Info("Container deleted successfully", zap.String("container", containerName))
	return nil
}

// Helper function to create a bundle. This is a two part process
//
// 1. Create a rootfs directory and populate it with content from an existing image
// 2. Create a config.json
// 3. Export the image contents into rootfs (TODO: implement programmatically)
func createOCIBundle(exportImage string) {
	// If exportImage is empty return immediately
	if (exportImage == "") {
		zap.L().Error("OCI bundle creation was invoked but no image was provided through --export-image flag. Please provide an image to export")
		return
	}
	zap.L().Info("Creating an OCI bundle in the current directory - creating a rootfs directory and config.json")
	zap.L().Info("Checking if config.json already exists. If it doesn't, running `runc spec` to create it now")
	// Check if config.json already exists
	_, err := os.Stat("config.json")
	if err == nil {
		zap.L().Info("config.json already exists, skipping creation")
	}
	// If the file doesn't exist then run `runc spec` to create it. This is created with assumed defaults: https://github.com/opencontainers/runc
	if errors.Is(err, os.ErrNotExist) {
		zap.L().Info("config.json does not exist, creating it now")
		cmd := exec.Command("runc", "spec")
		err := cmd.Run()
		if err != nil {
			zap.L().Error("Error creating config.json through 'runc spec'", zap.Error(err))
			return
		}
		// File has successfully been created
		zap.L().Info("config.json created successfully through 'runc spec'")
	}

	zap.L().Info("Checking if rootfs directory already exists. If it doesn't, creating it now and populating it with contents from the exported image")
	// Check if rootfs already exists
	_, errRootFs := os.Stat("rootfs")
	if errRootFs == nil {
		zap.L().Info("rootfs exists, skipping creation")
		// Check if rootfs is populated with content
		// If it isnt, then export and untar the image into rootfs
		dir, err := os.Open("rootfs")
		if err != nil {
			zap.L().Error("Error opening rootfs directory", zap.Error(err))
			return
		}
		defer dir.Close()	
		// Read the first entry in the directory
		_, err = dir.Readdir(1)
		if err == io.EOF {
			zap.L().Info("rootfs is empty")
		} else {
			// Otherwise rootfs is not empty - to risk of overwriting existing content, we will not populate it
			zap.L().Error("rootfs is not empty, not proceeding with image content exporting")
			return
		}
	}
	// If rootfs doesn't exist then create it - but only do this if it doesn't exist
	if errors.Is(errRootFs, os.ErrNotExist) {
		zap.L().Info("rootfs does not exist, creating it now")
		cmd := exec.Command("mkdir", "rootfs")
		err := cmd.Run()
		if err != nil {
			zap.L().Error("Error creating rootfs through 'mkdir rootfs'", zap.Error(err))
			return
		}
		// directory has successfully been created
		zap.L().Info("rootfs created successfully through 'mkdir rootfs'")
	}
	// By this point, rootfs and config.json should both exist and rootfs should be empty if it did previously exist
	// Start using `$(docker create [some_image]) | tar -C rootfs -xvf -` to populate rootfs
	zap.L().Info(fmt.Sprintf("Populating rootfs with contents from the exported %s image", exportImage))
	// Concat user provided image args into the docker create command
	dockerExportAndCreateCmd := fmt.Sprintf("docker export $(docker create %s) | tar -C rootfs -xvf -", exportImage)
	cmd := exec.Command("sh", "-c", dockerExportAndCreateCmd)
	err = cmd.Run()
	if err != nil {
		zap.L().Error(fmt.Sprintf("Error populating rootfs with contents from the exported %s image", exportImage), zap.Error(err))
		return
	}
	// rootfs has successfully been populated
	zap.L().Info(fmt.Sprintf("rootfs populated successfully with contents from the exported %s image", exportImage))
}

func main() {
	// Command line args to invoke various runc functionality
	listRuncContainers := flag.Bool("list-containers", false, "List containers started by runc")
	runContainerArg := flag.Bool("run-container", false, "Run a container using runc")
	// Boolean to monitor container resources
	monitorArg := flag.Bool("monitor-container", false, "Monitor container resources such as cpu, memory, etc.")
	// Metric type to monitor for the container. Supported types are 'cpu', 'memory'
	metricTypeArg := flag.String("metric-type", "", "Metric type to monitor for the container. Supported types are 'cpu', 'memory', 'pid")
	// Interval in seconds to fetch resource metrics. Default is 10 seconds
	metricInveralArg := flag.Int("metric-interval", 10, "Interval in seconds to fetch resource metrics. Max interval duration is 60 seconds")
	containerName := flag.String("container-name", "", "Name of the container to run/monitor/delete")
	deleteContainer := flag.Bool("delete-container", false, "Delete a container using runc")
	exportImage := flag.String("export-image", "", "Export an image to populate the rootfs directory in the OCI bundle")
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
	if *runContainerArg && *containerName != "" {
		err := runContainer(r, *containerName, ctx)
		if err != nil {
			return
		}
	}
	// Monitor container resources
	if *monitorArg && *containerName != "" {
		if *metricTypeArg == "" || (*metricTypeArg != "cpu" && *metricTypeArg != "memory" && *metricTypeArg != "pid") {
			zap.L().Error("Invalid metric type. Supported types are 'cpu', 'memory', 'pid'")
			return
		}
		// Check if metricInveralArg is less than 1
		if *metricInveralArg < 1 {
			zap.L().Warn("Metric interval is less than 1. Setting it to a default of 10 seconds")
			*metricInveralArg = 10
		}
		// Check if metric internval arg is greater than 60 seconds
		if *metricInveralArg > 60 {
			zap.L().Warn("Metric interval is greater than 60 seconds. Setting it to a default of 10 seconds")
			*metricInveralArg = 10
		}
		// Convert metricInveralArg to string
		eventInterval := time.Duration(*metricInveralArg) * time.Second
		zap.L().Info("Metric interval set to", zap.Duration("interval", eventInterval))
		// Call the function to monitor container resources
		monitorContainerResources(r, *containerName, ctx, string(*metricTypeArg), eventInterval)
		return
	}
	// Delete the container
	if *containerName != "" && *deleteContainer {
		err := deleteRuncContainer(r, *containerName, ctx)
		if err != nil {
			return
		}
	}
	// This function will create an OCI bundle in the current directory
	// It will create a rootfs directory and populate it with contents from the exported image provided by the user
	// It will also create a config.json if it doesn't already exist
	if *exportImage != "" {
		createOCIBundle(*exportImage)
	}
}
