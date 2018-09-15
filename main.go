package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ahstn/transitioner/docker"
	"github.com/docker/docker/client"
	"github.com/fatih/color"
	"github.com/spf13/viper"
	"gopkg.in/resty.v1"
)

const tab = "        "

// Config is the definition of what containers should be tested.
type Config struct {
	Network     string           `yaml:"network"`
	KillTimeout time.Duration    `yaml:"kill_timeout"`
	Cleanup     bool             `yaml:"cleanup"`
	Gateway     docker.Container `yaml:"gateway"`
	Service     docker.Container `yaml:"service"`
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
	v.SetDefault("cleanup", true)

	err = v.ReadInConfig()
	if err != nil {
		panic(err)
	}

	var c Config
	err = v.Unmarshal(&c)
	if err != nil {
		panic(err)
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		fmt.Println("Shutting down...")

		ctx, cancel := context.WithTimeout(ctx, c.KillTimeout*time.Second)
		defer cancel()

		if c.Cleanup {
			docker.StopAndRemoveContainer(ctx, cli, c.Gateway)
			docker.StopAndRemoveContainer(ctx, cli, c.Service)
		}
		close(done)
	}()

	networkID, err := docker.Network(ctx, cli, c.Network)
	if err != nil {
		panic(err)
	}
	c.SetNetwork(networkID, c.Network)

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

	go docker.WatchContainer(ctx, cli, c.Gateway)
	go docker.WatchContainer(ctx, cli, c.Service)
	logTitle := docker.PadNameColor(color.New(color.FgYellow).SprintFunc(), "testing")

	// This is where tests will be ran, for now it's a simple GET request
	// to validate the docker setup
	resp, err := resty.R().Get("http://127.0.0.1:5050/health")
	fmt.Println(logTitle, "| Health Response:", resp)

	time.Sleep(500 * time.Millisecond)
	resp, err = resty.R().Post("http://127.0.0.1:5050/auth")
	fmt.Println(logTitle, "| Auth Response:\n", resp)
	fmt.Println(logTitle, "| Test Case 1: Contains 'msg=success'")
	if strings.Contains(resp.String(), "msg=success") {
		fmt.Println(logTitle, "| Passed! ✔")
	} else {
		fmt.Println(logTitle, "| Failed ✘")
	}

	if c.Cleanup {
		docker.StopAndRemoveContainer(ctx, cli, c.Gateway)
		docker.StopAndRemoveContainer(ctx, cli, c.Service)
	}
}
