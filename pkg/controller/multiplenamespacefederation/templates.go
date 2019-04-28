package multiplenamespacefederation

import (
	"encoding/base64"
	"io/ioutil"
	"strings"
	"text/template"
)

var selfHostedGlobalLoadBalancerTemplate *template.Template
var selfHostedGlobalLoadBalancerServiceAccountTemplate *template.Template
var cloudProviderGlobalLoadBalancerTemplate *template.Template

func InitializeRemoteGlobaLoadBalancerTemplate(remoteGlobalLoadBalancerTemplateFileName string) error {

	text, err := ioutil.ReadFile(remoteGlobalLoadBalancerTemplateFileName)
	if err != nil {
		log.Error(err, "Error reading statefulset template file", "filename", remoteGlobalLoadBalancerTemplateFileName)
		return err
	}

	selfHostedGlobalLoadBalancerTemplate = template.New("RemoteGlobalLoadBalancer").Funcs(template.FuncMap{
		"parseNewLines": func(value string) string {
			return strings.Replace(value, "\n", "\n\n", -1)
		},
		"encode64": func(value string) string {
			return base64.StdEncoding.EncodeToString([]byte(value))
		},
	})

	selfHostedGlobalLoadBalancerTemplate, err = selfHostedGlobalLoadBalancerTemplate.Parse(string(text))
	if err != nil {
		log.Error(err, "Error parsing template", "template", text)
		return err
	}

	return nil
}

func InitializeLocalLoadBalancerServiceAccountTemplate(localLoadBalancerServiceAccountTemplateFileName string) error {

	text, err := ioutil.ReadFile(localLoadBalancerServiceAccountTemplateFileName)
	if err != nil {
		log.Error(err, "Error reading statefulset template file", "filename", localLoadBalancerServiceAccountTemplateFileName)
		return err
	}

	selfHostedGlobalLoadBalancerServiceAccountTemplate, err = template.New("RemoteGlobalLoadBalancer").Parse(string(text))
	if err != nil {
		log.Error(err, "Error parsing template", "template", text)
		return err
	}

	return nil
}

func InitializeCloudProviderGlobalLoadBalancerTemplate(cloudProviderGlobalLoadBalancerTemplateFileName string) error {

	text, err := ioutil.ReadFile(cloudProviderGlobalLoadBalancerTemplateFileName)
	if err != nil {
		log.Error(err, "Error reading statefulset template file", "filename", cloudProviderGlobalLoadBalancerTemplateFileName)
		return err
	}

	cloudProviderGlobalLoadBalancerTemplate, err = template.New("CloudProviderGlobalLoadBalancer").Parse(string(text))
	if err != nil {
		log.Error(err, "Error parsing template", "template", text)
		return err
	}

	return nil
}
