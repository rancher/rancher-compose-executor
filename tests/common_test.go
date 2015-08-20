package tests

import (
	"io/ioutil"
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/client"
)

var apiClient *client.RancherClient
var apiClient2 *client.RancherClient

func TestMain(m *testing.M) {
	var err error

	apiUrl := "http://localhost:8080/v1/projects/1a5/schema"
	accessKey := ""
	secretKey := ""

	//1a5 is admin account, 1a6 will be the service account, therefore trying to create a new account of type starting from 1a7
	//if 1a7 exists, and is not a user account then a new account with the next available id will be created
	id := createIfNoAccount("1a7")
	apiUrl2 := "http://localhost:8080/v1/projects/" + id + "/schema"

	apiClient, err = client.NewRancherClient(&client.ClientOpts{
		Url:       apiUrl,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		log.Fatal("Error while initializing rancher client, err = ", err)
	}
	apiClient2, err = client.NewRancherClient(&client.ClientOpts{
		Url:       apiUrl2,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		log.Fatal("Error while initializing rancher client, err = ", err)
	}
	os.Exit(m.Run())
}

func deleteEnvironment(env *client.Environment, cl *client.RancherClient) {
	cl.Environment.Delete(env)
}

func createIfNoAccount(id string) string {
	apiUrl := "http://localhost:8080/v1"
	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Url: apiUrl,
	})
	if err != nil {
		log.Fatalf("Error while initializing rancher client, err = [%v]", err)
	}
	acc, err := apiClient.Account.ById(id)
	if err != nil || (err == nil && acc != nil && acc.Kind != "project") {
		acc, err = apiClient.Account.Create(&client.Account{
			Kind: "project",
			Name: "user",
		})
		if err != nil {
			log.Fatalf("Error while creating new account, err = [%v]", err)
		}
		waitTransition(apiClient, acc)
	}
	return acc.Id
}

func waitTransition(cl *client.RancherClient, acc *client.Account) {
	newAcc := &client.Account{}
	for {
		cl.Reload(&acc.Resource, newAcc)
		if newAcc.Transitioning == "error" {
			log.Fatalf("Error creating new account, err = [%v]", newAcc.TransitioningMessage)
		}
		if newAcc.Transitioning == "no" {
			return
		}
	}
}

func createEnvironmentWithClient(currClient *client.RancherClient, name, dockerComposePath, rancherComposePath string) (*client.Environment, error) {
	dockerComposeBytes, err := ioutil.ReadFile(dockerComposePath)
	if err != nil {
		return nil, err
	}
	dockerComposeString := string(dockerComposeBytes)
	rancherComposeString := ""
	if rancherComposePath != "" {
		rancherComposeBytes, err := ioutil.ReadFile(rancherComposePath)
		if err != nil {
			return nil, err
		}
		rancherComposeString = string(rancherComposeBytes)
	}
	return currClient.Environment.Create(&client.Environment{
		Name:           name,
		DockerCompose:  dockerComposeString,
		RancherCompose: rancherComposeString,
	})
}

func createEnvironment(name, dockerComposePath, rancherComposePath string) (*client.Environment, error) {
	return createEnvironmentWithClient(apiClient, name, dockerComposePath, rancherComposePath)
}

func createEnvironment2(name, dockerComposePath, rancherComposePath string) (*client.Environment, error) {
	return createEnvironmentWithClient(apiClient2, name, dockerComposePath, rancherComposePath)
}
