package main

import (
	"fmt"
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	rancherApp "github.com/rancher/rancher-compose-executor/app"
	"github.com/rancher/rancher-compose-executor/executor"
	"github.com/rancher/rancher-compose-executor/version"
	"github.com/urfave/cli"
)

func beforeApp(c *cli.Context) error {
	if c.GlobalBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	return nil
}

func main() {
	if path.Base(os.Args[0]) == "rancher-compose-executor" {
		executor.Main()
	} else {
		cliMain()
	}
}

func cliMain() {
	factory := &rancherApp.RancherProjectFactory{}

	app := cli.NewApp()
	app.Name = "rancher-compose"
	app.Usage = "Docker-compose to Rancher"
	app.Version = version.VERSION
	app.Author = "Rancher Labs, Inc."
	app.Email = ""
	app.Before = beforeApp
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "verbose,debug",
		},
		cli.StringSliceFlag{
			Name:   "file,f",
			Usage:  "Specify one or more alternate compose files (default: docker-compose.yml)",
			Value:  &cli.StringSlice{},
			EnvVar: "COMPOSE_FILE",
		},
		cli.StringFlag{
			Name:   "project-name,p",
			Usage:  "Specify an alternate project name (default: directory name)",
			EnvVar: "COMPOSE_PROJECT_NAME",
		},
		cli.StringFlag{
			Name: "url",
			Usage: fmt.Sprintf(
				"Specify the Rancher API endpoint URL",
			),
			EnvVar: "RANCHER_URL",
		},
		cli.StringFlag{
			Name: "access-key",
			Usage: fmt.Sprintf(
				"Specify Rancher API access key",
			),
			EnvVar: "RANCHER_ACCESS_KEY",
		},
		cli.StringFlag{
			Name: "secret-key",
			Usage: fmt.Sprintf(
				"Specify Rancher API secret key",
			),
			EnvVar: "RANCHER_SECRET_KEY",
		},
		cli.StringFlag{
			Name:  "rancher-file,r",
			Usage: "Specify an alternate Rancher compose file (default: rancher-compose.yml)",
		},
		cli.StringFlag{
			Name:  "env-file,e",
			Usage: "Specify a file from which to read environment variables",
		},
		cli.StringFlag{
			Name:  "bindings-file,b",
			Usage: "Specify a file from which to read bindings",
		},
	}
	app.Commands = []cli.Command{
		rancherApp.CreateCommand(factory),
		rancherApp.UpCommand(factory),
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
