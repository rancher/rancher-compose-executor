package tests

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/client"
)

var apiClient *client.RancherClient
var apiClient2 *client.RancherClient

func TestMain(m *testing.M) {
	var err error

	adminUrl := "http://localhost:8080/v1"
	apiUrl := adminUrl + "/projects/1a5/schemas"
	accessKey := ""
	secretKey := ""

	adminClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       adminUrl,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; ; i++ {
		handlers, err := adminClient.ExternalHandler.List(&client.ListOpts{
			Filters: map[string]interface{}{
				"name":  "rancher-compose-executor",
				"state": "active",
			},
		})
		if err != nil {
			log.Fatal(err)
		}
		if len(handlers.Data) > 0 {
			break
		}
		if i > 3 {
			log.Fatal("Handler is not available")
		}
		time.Sleep(1 * time.Second)
	}

	id := createIfNoAccount(adminUrl, "rancher-compose-executor-tests")
	apiUrl2 := adminUrl + "/projects/" + id + "/schemas"

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

func createIfNoAccount(apiUrl, name string) string {
	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Url: apiUrl,
	})
	if err != nil {
		log.Fatalf("Error while initializing rancher client, err = [%v]", err)
	}
	accs, err := apiClient.Account.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name": name,
		},
	})
	if err != nil {
		log.Fatalf("Failed to list accounts")
	}

	if len(accs.Data) == 0 {
		acc, err := apiClient.Account.Create(&client.Account{
			Kind: "project",
			Name: name,
		})
		if err != nil {
			log.Fatalf("Error while creating new account, err = [%v]", err)
		}
		waitTransition(apiClient, acc)
		return acc.Id
	}

	return accs.Data[0].Id
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

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString() string {
	b := make([]rune, 7)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func readFileToString(t *testing.T, path string) string {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal("Failed to read", path, err)
	}

	return string(bytes)
}
