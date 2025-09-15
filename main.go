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
// Monitor container metrics for cpu, memory and pid (pid shows number of pids in the container, not the actual pids)
func monitorContainerResources(r runc.Runc, ctx context.Context, metricType string, eventInterval time.Duration) error {
	zap.L().Info("Monitoring container resources, this may take a few seconds to start outputting information", zap.String("metric_type", metricType))
	zap.L().Info("Press Ctrl+C to stop monitoring")
	event, err := r.Events(ctx, "mycontainer", eventInterval)
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
	if *runContainerArg {
		err := runContainer(r, ctx)
		if err != nil {
			return
		}
	}
	// Monitor container resources
	if *monitorArg {
		if *metricTypeArg == "" || (*metricTypeArg != "cpu" && *metricTypeArg != "memory" && *metricTypeArg != "pid") {
			zap.L().Error("Invalid metric type. Supported types are 'cpu', 'memory', 'pid'")
			return
		}
		// Check if metricInveralArg is less than 10
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
		monitorContainerResources(r, ctx, string(*metricTypeArg), eventInterval)
		return
	}
}
