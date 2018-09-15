package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/fatih/color"
)

// Container is the definion of a Docker container and related configuration
type Container struct {
	ID          string
	Cmd         string   `yaml:"cmd"`
	Ports       []string `yaml:"ports"`
	Env         []string `yaml:"env"`
	Image       string   `yaml:"image"`
	Hostname    string   `yaml:"hostname"`
	NetworkID   string   `yaml:"network_id"`
	NetworkName string   `yaml:"network_name"`
	LogMedium   io.Writer
}

// SetID is syntastic sugar for setting the Container's ID
func (c *Container) SetID(id string) {
	c.ID = id
}

type emptyStruct struct{}

// CreateContainer initialises the container `c` using `cli`.
// Returns the created container's ID.
func CreateContainer(ctx context.Context, cli *client.Client, c *Container) (string, error) {
	ports := make(map[nat.Port]struct{})
	portBindings := make(map[nat.Port][]nat.PortBinding)

	if len(c.Ports) > 0 {
		for _, v := range c.Ports {
			port := strings.Split(v, ":")
			hostPort := port[0]
			containerPort := port[1]

			portBinding := nat.PortBinding{HostPort: hostPort}
			natPort, err := nat.NewPort("tcp", containerPort)
			if err != nil {
				fmt.Println("Error creating port for container:", err)
			}
			ports[natPort] = emptyStruct{}
			portBindings[natPort] = []nat.PortBinding{portBinding}
		}
	}

	container, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image:        c.Image,
			Cmd:          strings.Split(c.Cmd, " "),
			Env:          c.Env,
			ExposedPorts: ports,
			Hostname:     c.Hostname,
			Domainname:   c.Hostname,
		},
		&container.HostConfig{
			DNS: []string{
				"8.8.8.8",
				"8.8.4.4",
				"2001:4860:4860::8888",
				"2001:4860:4860::8844",
			},
			DNSSearch: []string{
				"8.8.8.8",
				"8.8.4.4",
				"2001:4860:4860::8888",
				"2001:4860:4860::8844",
			},
			PublishAllPorts: false,
			NetworkMode:     container.NetworkMode(c.NetworkName),
			PortBindings:    portBindings,
		},
		nil,
		c.Hostname,
	)

	if err != nil {
		return "", err
	}

	c.SetID(container.ID)
	c.ID = container.ID
	return container.ID, nil
}

// RunContainer starts the container `c` using the `cli`
func RunContainer(ctx context.Context, cli *client.Client, c Container) error {
	err := cli.ContainerStart(ctx, c.ID, types.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("Unable to start %s - %s", c.Hostname, err)
	}
	return nil
}

// StopContainer stops the container `c` using the `cli`
func StopContainer(ctx context.Context, cli *client.Client, c Container) error {
	err := cli.ContainerStop(ctx, c.ID, nil)
	if err != nil {
		return fmt.Errorf("Unable to stop %s - %s", c.Hostname, err)
	}
	return nil
}

// StopAndRemoveContainer stops and removes the container `c` using the `cli`
func StopAndRemoveContainer(ctx context.Context, cli *client.Client, c Container) error {
	err := cli.ContainerStop(ctx, c.ID, nil)
	if err != nil {
		return fmt.Errorf("Unable to stop %s - %s", c.Hostname, err)
	}

	err = cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{})
	if err != nil {
		return fmt.Errorf("Unable to remove %s - %s", c.Hostname, err)
	}
	return nil
}

// WatchContainer prints the logs from container `c`
func WatchContainer(ctx context.Context, cli *client.Client, c Container) error {
	logs, err := cli.ContainerLogs(
		ctx,
		c.ID,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Follow:     true,
			Details:    true,
			Tail:       "all",
		},
	)
	if err != nil {
		return err
	}
	defer logs.Close()
	out := RandomOutputColor()

	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		fmt.Fprintln(os.Stdout, PadNameColor(out, c.Hostname), "|", scanner.Text())
	}

	return nil
}

// PadName fills the container name string with spaces to align all names in the
// log output
func PadName(s string) (out string) {
	p := 10 - len(s)
	for i := 0; i < p; i++ {
		out += " "
	}
	return out + s
}

// PadNameColor does the same as PadName but adds color to the string
func PadNameColor(out func(...interface{}) string, s string) string {
	return out(PadName(s))
}

// RandomOutputColor uses fatih/color to return a function that will be used
// to output a string that is bold and colored
func RandomOutputColor() func(...interface{}) string {
	switch rand.Intn(5) {
	case 1:
		return color.New(color.FgGreen, color.Bold).SprintFunc()
	case 2:
		return color.New(color.FgBlue, color.Bold).SprintFunc()
	case 3:
		return color.New(color.FgMagenta, color.Bold).SprintFunc()
	case 4:
		return color.New(color.FgCyan, color.Bold).SprintFunc()
	default:
		return color.New(color.FgRed, color.Bold).SprintFunc()
	}
}
