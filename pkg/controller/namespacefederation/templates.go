package namespacefederation

import (
	"io/ioutil"
	"strings"
	"text/template"

	federationv2v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	extensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var federatedClusterTemplate *template.Template
var remoteFederatedClusterTemplate *template.Template
var federationControllerTemplate *template.Template
var federatedTypesTemplate *template.Template

func InitializeFederatedClusterTemplates(federatedClusterTemplateFileName string, remoteFederatedClusterTemplateFileName string) error {
	text, err := ioutil.ReadFile(federatedClusterTemplateFileName)
	if err != nil {
		log.Error(err, "Error reading rolebinding template file", "filename", federatedClusterTemplateFileName)
		return err
	}

	federatedClusterTemplate = template.New("FederatedCluster").Funcs(template.FuncMap{
		"parseNewLines": func(value string) string {
			return strings.Replace(value, "\n", "\n\n", -1)
		},
	})

	federatedClusterTemplate, err = federatedClusterTemplate.Parse(string(text))
	if err != nil {
		log.Error(err, "Error parsing template", "template", text)
		return err
	}

	text, err = ioutil.ReadFile(remoteFederatedClusterTemplateFileName)
	if err != nil {
		log.Error(err, "Error reading rolebinding template file", "filename", federatedClusterTemplateFileName)
		return err
	}

	remoteFederatedClusterTemplate, err = template.New("RemoteFederatedCluster").Parse(string(text))
	if err != nil {
		log.Error(err, "Error parsing template", "template", text)
		return err
	}

	return nil
}

// InitializeFederationControlPlaneTemplates initializes the temolates needed by this controller, it must be called at controller boot time
func InitializeFederationControlPlaneTemplates(federationControllerTemplateFileName string) error {

	text, err := ioutil.ReadFile(federationControllerTemplateFileName)
	if err != nil {
		log.Error(err, "Error reading statefulset template file", "filename", federationControllerTemplateFileName)
		return err
	}

	federationControllerTemplate, err = template.New("Job").Parse(string(text))
	if err != nil {
		log.Error(err, "Error parsing template", "template", text)
		return err
	}

	return nil
}

func InitializeFederatedTypesTemplates(federatedTypesTemplateFileName string) error {
	text, err := ioutil.ReadFile(federatedTypesTemplateFileName)
	if err != nil {
		log.Error(err, "Error reading rolebinding template file", "filename", federatedTypesTemplateFileName)
		return err
	}

	federatedTypesTemplate = template.New("FederatedTypes").Funcs(template.FuncMap{
		"getLongName": func(simpleType metav1.TypeMeta) string {
			if simpleType.GroupVersionKind().Group != "" {
				return federationv2v1alpha1.PluralName(simpleType.Kind) + "." + simpleType.GroupVersionKind().Group
			} else {
				return federationv2v1alpha1.PluralName(simpleType.Kind)
			}
		},
		"getShortName": func(simpleType metav1.TypeMeta) string {
			return federationv2v1alpha1.PluralName(simpleType.Kind)
		},
		"namespaced": func(crd extensionv1beta1.CustomResourceDefinition) bool {
			return crd.Spec.Scope == "Namespaced"
		},
	})

	federatedTypesTemplate, err = federatedTypesTemplate.Parse(string(text))
	if err != nil {
		log.Error(err, "Error parsing template", "template", text)
		return err
	}

	return nil
}
