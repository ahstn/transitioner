package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// Network returns a network, creating one if it doesn't already exist.
// Returns the Network ID.
func Network(ctx context.Context, cli *client.Client, target string) (string, error) {
	// Network existance check
	networks, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return "", err
	}
	for _, n := range networks {
		if n.Name == target {
			return n.ID, nil
		}
	}

	res, err := cli.NetworkCreate(
		ctx,
		target,
		types.NetworkCreate{
			CheckDuplicate: true,
			Driver:         "bridge",
			EnableIPv6:     false,
			IPAM:           &network.IPAM{Driver: "default"},
			Internal:       false,
			Attachable:     true,
		},
	)
	if err != nil {
		return "", err
	}
	return res.ID, nil
}
