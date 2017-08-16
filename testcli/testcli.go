package testcli

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v3"
	_ "github.com/rancher/rancher-compose-executor/resources"
	"github.com/rancher/rancher-compose-executor/version"
	"github.com/urfave/cli"
)

const (
	rancherURLEnv       = "RANCHER_URL"
	rancherAccessKeyEnv = "RANCHER_ACCESS_KEY"
	rancherSecretKeyEnv = "RANCHER_SECRET_KEY"
)

var (
	rancherClient *client.RancherClient
)

func beforeApp(c *cli.Context) error {
	if c.GlobalBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	url, err := client.NormalizeUrl(os.Getenv(rancherURLEnv))
	if err != nil {
		return err
	}
	rancherClient, err = client.NewRancherClient(&client.ClientOpts{
		Url:       url,
		AccessKey: os.Getenv(rancherAccessKeyEnv),
		SecretKey: os.Getenv(rancherSecretKeyEnv),
	})
	return err
}

func Main() {
	app := cli.NewApp()
	app.Name = "rancher-compose"
	app.Version = version.VERSION
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
			Name:  "rancher-file,r",
			Usage: "Specify an alternate Rancher compose file (default: rancher-compose.yml)",
		},
		cli.StringFlag{
			Name:  "env-file,e",
			Usage: "Specify a file from which to read environment variables",
		},
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name: "create",
			Action: func(c *cli.Context) error {
				return create(c)
			},
		},
		cli.Command{
			Name: "up",
			Action: func(c *cli.Context) error {
				return up(c)
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
