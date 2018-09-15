package main

import (
	"context"
	"fmt"

	"github.com/ahstn/transitioner/docker"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

// Config is the definition of what containers should be tested.
type Config struct {
	Network string           `yaml:"network"`
	Gateway docker.Container `yaml:"gateway"`
	Service docker.Container `yaml:"service"`
}

// SetNetwork is syntastic sugar for setting all the Containers' network
func (c *Config) SetNetwork(id, name string) {
	if c.Gateway.NetworkID == "" {
		c.Gateway.NetworkID = id
		c.Gateway.NetworkName = name
	}
	if c.Service.NetworkID == "" {
		c.Service.NetworkID = id
		c.Service.NetworkName = name
	}
}

func main() {
	fmt.Println("hello")

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("transitioner")
	v.AddConfigPath("$HOME/.config/transitioner")
	v.AddConfigPath(".")

	err = v.ReadInConfig()
	if err != nil {
		panic(err)
	}

	var c Config
	err = v.Unmarshal(&c)
	if err != nil {
		panic(err)
	}
	fmt.Println("NETWORK:", c.Network, v.Get("network"))
	fmt.Println("CONFIG:", c)

	v.Set("Gateway.ID", "123")
	v.Set("gateway.ID", "123")
	fmt.Print(v.Get("gateway.ID"))

	networkID, err := docker.Network(ctx, cli, c.Network)
	if err != nil {
		panic(err)
	}
	c.SetNetwork(networkID, c.Network)

	fmt.Println("Network: ", networkID)

	_, err = docker.CreateContainer(ctx, cli, &c.Gateway)
	if err != nil {
		panic(err)
	}

	_, err = docker.CreateContainer(ctx, cli, &c.Service)
	if err != nil {
		panic(err)
	}

	err = docker.RunContainer(ctx, cli, c.Gateway)
	if err != nil {
		panic(err)
	}

	err = docker.RunContainer(ctx, cli, c.Service)
	if err != nil {
		panic(err)
	}
}
