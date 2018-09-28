package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/ahstn/transitioner/docker"
	"github.com/docker/docker/client"
	"github.com/spf13/viper"
)

const tab = "        "

// Config is the definition of what containers should be tested.
type Config struct {
	Network     string             `yaml:"network"`
	KillTimeout time.Duration      `yaml:"kill_timeout"`
	Cleanup     bool               `yaml:"cleanup"`
	Services    []docker.Container `yaml:"services"`
	Test        Test               `yaml:"test"`
}

// Test is the user defined event that tests their application(s)
type Test struct {
	Cmd string `yaml:"cmd"`
	Dir string `yaml:"dir"`
}

// SetNetwork is syntastic sugar for setting all the Containers' network
func (c *Config) SetNetwork(id, name string) {
	for _, s := range c.Services {
		if s.NetworkID == "" {
			s.NetworkID = id
			s.NetworkName = name
		}
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
			for _, s := range c.Services {
				docker.StopAndRemoveContainer(ctx, cli, s)
			}
		}
		close(done)
	}()

	networkID, err := docker.Network(ctx, cli, c.Network)
	if err != nil {
		panic(err)
	}
	c.SetNetwork(networkID, c.Network)

	for _, s := range c.Services {
		_, err = docker.CreateContainer(ctx, cli, &s)
		if err != nil {
			panic(err)
		}

		err = docker.RunContainer(ctx, cli, s)
		if err != nil {
			panic(err)
		}

		go docker.WatchContainer(ctx, cli, s)
	}

	if c.Test.Dir != "" {
		os.Chdir(c.Test.Dir)
	}

	cmd := strings.Split(c.Test.Cmd, " ")[0]
	args := strings.Split(c.Test.Cmd, " ")[1:]
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))

	if c.Cleanup {
		for _, s := range c.Services {
			docker.StopAndRemoveContainer(ctx, cli, s)
		}
	}
}
