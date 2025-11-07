# runc-go
A program that uses [github.com/containerd/go-runc](github.com/containerd/go-runc) to interact with runc installed on a machine for container management, programmatically.

## Usage
This is essentially just a wrapper around [github.com/containerd/go-runc](github.com/containerd/go-runc) (and therefor runc)

- `--list-containers`: List runc containers
- `--run-container`: Run a container. Must pass also `--container-name "yourcontainername` to it. The OCI bundle must be relative to `main.go`.
- `--monitor-arg`: Monitor metrics for a container. Must pass the following:
    - `--container-name`: Name of the container to monitor
    - `--metric-type`: Metric type to monitor. Can be `cpu`, `memory` or `pid`
- `--delete-container`: Delete a runc container. Must also pass `--container-name "containertodelete"`
- `--export-image`: Creates an OCI bundle a directory relative to the program root. This option can only be used if Docker is installed on the machine. The image must be a public image. This also generates a basic `config.json` from `runc spec`

Output of `-h` / `--help`:

```
-container-name string
        Name of the container to run/monitor/delete
  -delete-container
        Delete a container using runc
  -export-image string
        Export an image to populate the rootfs directory in the OCI bundle
  -list-containers
        List containers started by runc
  -metric-interval int
        Interval in seconds to fetch resource metrics. Max interval duration is 60 seconds (default 10)
  -metric-type string
        Metric type to monitor for the container. Supported types are 'cpu', 'memory', 'pid
  -monitor-container
        Monitor container resources such as cpu, memory, etc.
  -run-container
        Run a container using runc
```