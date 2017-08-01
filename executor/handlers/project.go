package handlers

import (
	"fmt"

	"net/url"
	"strings"

	"github.com/davecgh/go-spew/spew"
	catalog "github.com/rancher/go-rancher/catalog"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/rancher-compose-executor/project"
	_ "github.com/rancher/rancher-compose-executor/resources"
	"github.com/rancher/rancher-compose-executor/utils"
)

func constructProject(stack *client.Stack, opts client.ClientOpts) (*project.Project, error) {
	if stack.ExternalId == "" && len(stack.Templates) == 0 {
		return nil, nil
	}

	// TODO: don't create each time
	opts.Url = fmt.Sprintf("%s/projects/%s/schemas", opts.Url, stack.AccountId)
	rancherClient, err := client.NewRancherClient(&opts)

	templateVersion, err := loadTemplateVersion(stack, rancherClient)
	if err != nil {
		return nil, err
	}

	answers := buildAnswers(stack, templateVersion)

	p := project.NewProject(stack.Name, rancherClient)
	if templateVersion == nil {
		return p, p.Load(stack.Templates, answers)
	}

	return p, p.LoadFromTemplateVersion(*templateVersion, answers)
}

func buildAnswers(stack *client.Stack, templateVersion *catalog.TemplateVersion) map[string]string {
	result := map[string]string{}
	if templateVersion != nil {
		for _, q := range templateVersion.Questions {
			result[q.Variable] = q.Default
		}
	}

	return utils.MapUnion(result, utils.ToMapString(stack.Answers))
}

func loadTemplateVersion(stack *client.Stack, client *client.RancherClient) (*catalog.TemplateVersion, error) {
	if !strings.HasPrefix(stack.ExternalId, "catalog://") {
		return nil, nil
	}

	parsed, err := url.Parse(client.GetOpts().Url)
	if err != nil {
		return nil, err
	}
	parsed.Path = "/v1-catalog/schemas"

	opts := client.GetOpts()
	catalogClient, err := catalog.NewRancherClient(&catalog.ClientOpts{
		Url:       parsed.String(),
		AccessKey: opts.AccessKey,
		SecretKey: opts.SecretKey,
	})
	spew.Dump(catalogClient.GetOpts())
	if err != nil {
		return nil, err
	}

	catalogClient.SetCustomHeaders(map[string]string{
		"X-API-Project-Id": stack.AccountId,
	})

	return catalogClient.TemplateVersion.ById(strings.TrimPrefix(stack.ExternalId, "catalog://"))
}
