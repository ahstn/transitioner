package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Container is the definion of a Docker container and related configuration
type Container struct {
	ID          string
	Ports       []string `yaml:"ports"`
	Env         []string `yaml:"env"`
	Image       string   `yaml:"image"`
	Hostname    string   `yaml:"hostname"`
	NetworkID   string   `yaml:"network_id"`
	NetworkName string   `yaml:"network_name"`
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
	fmt.Println("Container:", c)
	fmt.Println("ID: ", c.ID)

	err := cli.ContainerStart(ctx, c.ID, types.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("Unable to start %s - %s", c.Hostname, err)
	}
	return nil
}
