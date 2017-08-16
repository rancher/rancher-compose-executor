package testcli

import (
	"context"
	"io/ioutil"
	"path"
	"strings"

	"github.com/docker/docker/runconfig/opts"
	"github.com/rancher/rancher-compose-executor/project"
	"github.com/rancher/rancher-compose-executor/project/options"
	_ "github.com/rancher/rancher-compose-executor/resources"
	"github.com/urfave/cli"
)

const (
	composeFilename        = "compose.yml"
	dockerComposeFilename  = "docker-compose.yml"
	rancherComposeFilename = "rancher-compose.yml"
)

func create(c *cli.Context) error {
	p, err := getProject(c)
	if err != nil {
		return err
	}
	return p.Create(context.Background(), options.Options{})
}

func up(c *cli.Context) error {
	p, err := getProject(c)
	if err != nil {
		return err
	}
	return p.Up(context.Background(), options.Options{})
}

func getProject(c *cli.Context) (*project.Project, error) {
	files := map[string]interface{}{}

	composeBytes, err := ioutil.ReadFile(composeFilename)
	if err == nil {
		files[composeFilename] = composeBytes
	}

	filenames := c.GlobalStringSlice("file")
	var dockerComposeBytes []byte
	for _, f := range filenames {
		dockerComposeBytes, err = ioutil.ReadFile(f)
		if err == nil {
			files[dockerComposeFilename] = dockerComposeBytes
		}
	}

	if len(dockerComposeBytes) == 0 {
		dockerComposeBytes, err = ioutil.ReadFile(dockerComposeFilename)
		if err == nil {
			files[dockerComposeFilename] = dockerComposeBytes
		}
	}

	var relPath string
	if len(filenames) > 0 {
		relPath = path.Dir(filenames[0])
	}

	rancherFile := c.String("rancher-file")
	if rancherFile == "" {
		rancherFile = rancherComposeFilename
	}
	rancherComposeBytes, err := ioutil.ReadFile(path.Join(relPath, rancherFile))
	if err == nil {
		files[rancherComposeFilename] = rancherComposeBytes
	}

	envFile := c.String("env-file")
	var variables map[string]string
	if envFile != "" {
		variables, err = getVariables(envFile)
		if err != nil {
			return nil, err
		}
	}

	projectName := c.GlobalString("project-name")

	p := project.NewProject(projectName, rancherClient)
	return p, p.Load(files, variables)
}

func getVariables(filename string) (map[string]string, error) {
	variables := map[string]string{}
	values, err := opts.ParseEnvFile(filename)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		variables[parts[0]] = parts[1]
	}
	return variables, nil
}
