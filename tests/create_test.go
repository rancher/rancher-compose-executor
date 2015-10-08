package tests

import (
	"strings"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/client"
)

func TestCreateOnlyDockerCompose(t *testing.T) {
	dockerComposePath := "assets/only_docker_compose/docker-compose.yml"
	env, err := createEnvironment("onlyDCompose", dockerComposePath, "")
	if err != nil {
		log.Error("Error creating environment, err = ", err)
		t.FailNow()
	}
	defer deleteEnvironment(env, apiClient)
	newEnv := &client.Environment{}
	for {
		err = apiClient.Reload(&(env.Resource), newEnv)
		if err != nil {
			log.Error("Error updating environment, err = ", err)
			t.FailNow()
		}
		if newEnv.Transitioning != "yes" {
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	servicesMap := map[string]interface{}{}
	err = apiClient.GetLink(env.Resource, "services", &servicesMap)
	if err != nil {
		log.Error("Error getting services, err = ", err)
		t.FailNow()
	}
	services := servicesMap["data"].([]interface{})
	if len(services) == 0 {
		log.Errorf("Expected at least one service, but obtained 0")
		t.FailNow()
	}
}

func TestCreateDockerComposeAndRancherCompose(t *testing.T) {
	dockerComposePath := "assets/docker_compose_and_rancher_compose/docker-compose.yml"
	rancherComposePath := "assets/docker_compose_and_rancher_compose/rancher-compose.yml"
	env, err := createEnvironment("DockerRancherCompose", dockerComposePath, rancherComposePath)
	if err != nil {
		log.Error("Error creating environment, err = ", err)
		t.FailNow()
	}
	defer deleteEnvironment(env, apiClient)
	newEnv := &client.Environment{}
	for {
		err = apiClient.Reload(&(env.Resource), newEnv)
		if err != nil {
			log.Error("Error updating environment, err = ", err)
			t.FailNow()
		}
		if newEnv.Transitioning != "yes" {
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	servicesMap := map[string]interface{}{}
	err = apiClient.GetLink(env.Resource, "services", &servicesMap)
	if err != nil {
		log.Error("Error getting services, err = ", err)
		t.FailNow()
	}
	services := servicesMap["data"].([]interface{})
	scale := services[0].(map[string]interface{})["scale"].(float64)
	if scale != 2 {
		log.Errorf("Expected scale = 2, but obtained %f", scale)
		t.FailNow()
	}
}

func TestCreateDockerMultipleServices(t *testing.T) {
	dockerComposePath := "assets/multiple_services/docker-compose.yml"
	rancherComposePath := "assets/multiple_services/rancher-compose.yml"
	env, err := createEnvironment("MultipleServices", dockerComposePath, rancherComposePath)
	if err != nil {
		log.Error("Error creating environment, err = ", err)
		t.FailNow()
	}
	defer deleteEnvironment(env, apiClient)
	newEnv := &client.Environment{}
	for {
		err = apiClient.Reload(&(env.Resource), newEnv)
		if err != nil {
			log.Error("Error updating environment, err = ", err)
			t.FailNow()
		}
		if newEnv.Transitioning != "yes" {
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	servicesMap := map[string]interface{}{}
	err = apiClient.GetLink(env.Resource, "services", &servicesMap)
	if err != nil {
		log.Error("Error getting services, err = ", err)
		t.FailNow()
	}
	services := servicesMap["data"].([]interface{})
	if len(services) != 2 {
		log.Errorf("Expected No. of Services = 2, but obtained %d", len(services))
		t.FailNow()
	}
}

func TestCreateMultipleEnvs(t *testing.T) {
	dockerComposePath := "assets/multiple_environments/docker-compose-1a5.yml"
	rancherComposePath := "assets/multiple_environments/rancher-compose-1a5.yml"
	env1, err := createEnvironment("MultipleEnvironments", dockerComposePath, rancherComposePath)
	if err != nil {
		log.Error("Error creating environment, err = ", err)
		t.FailNow()
	}
	defer deleteEnvironment(env1, apiClient)
	dockerComposePath2 := "assets/multiple_environments/docker-compose-1a6.yml"
	rancherComposePath2 := "assets/multiple_environments/rancher-compose-1a6.yml"
	env2, err := createEnvironment2("MultipleEnvironments", dockerComposePath2, rancherComposePath2)
	if err != nil {
		log.Error("Error creating environment2, err = ", err)
		t.FailNow()
	}
	defer deleteEnvironment(env2, apiClient2)
	newEnv1 := &client.Environment{}
	newEnv2 := &client.Environment{}
	for {
		err = apiClient.Reload(&(env1.Resource), newEnv1)
		if err != nil {
			log.Error("Error updating environment, err = ", err)
			t.FailNow()
		}
		if newEnv1.Transitioning != "yes" {
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	for {
		err = apiClient.Reload(&(env2.Resource), newEnv2)
		if err != nil {
			log.Error("Error updating environment, err = ", err)
			t.FailNow()
		}
		if newEnv2.Transitioning != "yes" {
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	servicesMap := map[string]interface{}{}
	err = apiClient.GetLink(env1.Resource, "services", &servicesMap)
	if err != nil {
		log.Error("Error getting services, err = ", err)
		t.FailNow()
	}
	servicesMap2 := map[string]interface{}{}
	err = apiClient2.GetLink(env2.Resource, "services", &servicesMap2)
	if err != nil {
		log.Error("Error getting services, err = ", err)
		t.FailNow()
	}
	services := servicesMap["data"].([]interface{})
	services2 := servicesMap2["data"].([]interface{})
	acc1 := services2[0].(map[string]interface{})["accountId"].(string)
	acc2 := services[0].(map[string]interface{})["accountId"].(string)
	if acc1 == acc2 {
		log.Errorf("Expected different accountIds for env1(%s) and env2(%s) ", acc1, acc2)
		t.FailNow()
	}
}

func TestCreateWithBuildNoFile(t *testing.T) {
	dockerComposePath := "assets/build_image_from_url/docker-compose.yml"
	rancherComposePath := "assets/build_image_from_url/rancher-compose.yml"
	env, err := createEnvironment("BuildNoFile", dockerComposePath, rancherComposePath)
	if err != nil {
		log.Error("Error creating environment, err = ", err)
		t.FailNow()
	}
	defer deleteEnvironment(env, apiClient)
	newEnv := &client.Environment{}
	for {
		err = apiClient.Reload(&(env.Resource), newEnv)
		if err != nil {
			log.Error("Error updating environment, err = ", err)
			t.FailNow()
		}
		if newEnv.Transitioning != "yes" {
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	servicesMap := map[string]interface{}{}
	err = apiClient.GetLink(env.Resource, "services", &servicesMap)
	if err != nil {
		log.Error("Error getting services, err = ", err)
		t.FailNow()
	}
	services := servicesMap["data"].([]interface{})
	state := services[0].(map[string]interface{})["state"].(string)
	if state != "inactive" && newEnv.State != "error" {
		log.Errorf("Expected service to be created and inactive but got service.state = [%s] and env.state = [%s]", state, newEnv.State)
		t.FailNow()
	}
}

func TestCreateWithBuildFile(t *testing.T) {
	dockerComposePath := "assets/build_image_from_file/docker-compose.yml"
	rancherComposePath := "assets/build_image_from_file/rancher-compose.yml"
	env, err := createEnvironment("Build", dockerComposePath, rancherComposePath)
	if err != nil {
		log.Error("Error creating environment, err = ", err)
		t.FailNow()
	}
	defer deleteEnvironment(env, apiClient)
	newEnv := &client.Environment{}
	for {
		err = apiClient.Reload(&(env.Resource), newEnv)
		if err != nil {
			log.Error("Error updating environment, err = ", err)
			t.FailNow()
		}
		if newEnv.Transitioning != "yes" {
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	if newEnv.State != "error" {
		log.Error("Expected state is error, found state = ", newEnv.State)
		t.Fail()
	}
}

func TestCreateWithSidekicks(t *testing.T) {
	dockerComposePath := "assets/sidekick/docker-compose.yml"
	rancherComposePath := "assets/sidekick/rancher-compose.yml"
	env, err := createEnvironment("Sidekick", dockerComposePath, rancherComposePath)
	if err != nil {
		log.Error("Error creating environment, err = ", err)
		t.FailNow()
	}
	defer deleteEnvironment(env, apiClient)
	newEnv := &client.Environment{}
	for {
		err = apiClient.Reload(&(env.Resource), newEnv)
		if err != nil {
			log.Error("Error updating environment, err = ", err)
			t.FailNow()
		}
		if newEnv.Transitioning == "error" {
			log.Errorf("Error creating sidekick, err = [%v]", newEnv.TransitioningMessage)
			t.FailNow()
		}
		if newEnv.Transitioning != "yes" {
			break
		}
		<-time.After(100 * time.Millisecond)
	}
	servicesMap := map[string]interface{}{}
	err = apiClient.GetLink(newEnv.Resource, "services", &servicesMap)
	if err != nil {
		log.Error("Error getting services, err = ", err)
		t.FailNow()
	}
	services := servicesMap["data"].([]interface{})
	if len(services) != 1 {
		log.Errorf("Expected No. of Services = 1, but obtained %d", len(services))
		t.FailNow()
	}
}

func TestBadDockerCompose(t *testing.T) {
	dockerComposePath := "assets/bad_format_compose/docker-compose.yml"
	env, err := createEnvironment("badFormat"+randString(), dockerComposePath, "")
	if err != nil {
		t.Fatal("Error creating environment, err = ", err)
	}
	defer deleteEnvironment(env, apiClient)
	waitForEnvironment(t, env)

	if env.Transitioning != "error" {
		t.Fatal("Parse worked")
	}

	if strings.Index(env.TransitioningMessage, "cannot unmarshal !!str `blah`") == -1 {
		t.Fatal("Bad error message", env.TransitioningMessage)
	}
}

func TestBadRancherCompose(t *testing.T) {
	dockerComposePath := "assets/bad_format_rancher_compose/docker-compose.yml"
	rancherComposePath := "assets/bad_format_rancher_compose/rancher-compose.yml"
	env, err := createEnvironment("badFormat"+randString(), dockerComposePath, rancherComposePath)
	if err != nil {
		t.Fatal("Error creating environment, err = ", err)
	}
	defer deleteEnvironment(env, apiClient)
	waitForEnvironment(t, env)

	if env.Transitioning != "error" {
		t.Fatal("Parse worked")
	}

	if strings.Index(env.TransitioningMessage, "cannot unmarshal !!str `blah`") == -1 {
		t.Fatal("Bad error message", env.TransitioningMessage)
	}
}

func TestEnv(t *testing.T) {
	dockerComposePath := "assets/env/docker-compose.yml"
	// Issue with setting numbers from variables
	//rancherComposePath := "assets/env/rancher-compose.yml"
	env, err := apiClient.Environment.Create(&client.Environment{
		Name:          "envtest" + randString(),
		DockerCompose: readFileToString(t, dockerComposePath),
		// Issue with setting numbers from variables
		//RancherCompose: readFileToString(t, rancherComposePath),
		Environment: map[string]interface{}{
			"scale": "2",
			"image": "nginx",
		},
	})
	if err != nil {
		t.Fatal("Error creating environment, err = ", err)
	}
	defer deleteEnvironment(env, apiClient)
	waitForEnvironmentSuccess(t, env)

	var services client.ServiceCollection
	if err := apiClient.GetLink(env.Resource, "services", &services); err != nil {
		t.Fatal(err)
	}

	if services.Data[0].LaunchConfig.ImageUuid != "docker:nginx" {
		t.Fatal("Bad image", services.Data[0].LaunchConfig.ImageUuid)
	}

	// Issue with setting numbers from variables
	//if services.Data[0].Scale != 2 {
	//	t.Fatal("Bad scale", services.Data[0].Scale)
	//}
}

func waitForEnvironmentSuccess(t *testing.T, env *client.Environment) {
	waitForEnvironment(t, env)
	if env.Transitioning != "no" {
		t.Fatal("Failed to create environment", env.Name, env.TransitioningMessage)
	}
}

func waitForEnvironment(t *testing.T, env *client.Environment) {
	for {
		err := apiClient.Reload(&(env.Resource), env)
		if err != nil {
			t.Fatal("Error updating environment, err = ", err)
		}
		if env.Transitioning != "yes" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}
