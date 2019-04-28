package namespacefederation

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"text/template"

	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var instance = federationv1alpha1.NamespaceFederation{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "ciao",
	},
}

var templateFile = fmt.Sprintf("%s/src/github.com/raffaelespazzoli/federation-lifecycle-operator/templates/federation-controller/federation-controller.yaml", os.Getenv("GOPATH"))

func TestFullConfig(t *testing.T) {
	text, err := ioutil.ReadFile(templateFile)
	if err != nil {
		t.Errorf("Error reading template file: %v", err)
		t.Fail()
	}
	template, err := template.New("template").Parse(string(text))

	objs, err := util.ProcessTemplateArray(&instance, template)
	if err != nil {
		t.Errorf("Error processing the template: %v", err)
		t.Fail()
	}
	t.Logf("array length %d", len(*objs))
	t.Logf("resulting manifest: %+v", *objs)
}
