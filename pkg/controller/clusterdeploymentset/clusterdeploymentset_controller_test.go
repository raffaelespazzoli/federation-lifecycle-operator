package clusterdeploymentset

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/apparentlymart/go-cidr/cidr"
	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var instance = federationv1alpha1.NamespaceFederation{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "ciao",
	},
}

var templateFile = fmt.Sprintf("%s/src/github.com/raffaelespazzoli/federation-lifecycle-operator/templates/federation-controller/federation-controller.yaml", os.Getenv("GOPATH"))

func TestNetwork(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("10.128.0.0/14")
	if err != nil {
		t.Errorf("parsing cidr 10.128.0.0/14")
		t.Fail()
	}
	t.Logf("ipnet %s", ipnet)
	addrcount := cidr.AddressCount(ipnet)
	t.Logf("address count %d", addrcount)
	ipnet, _ = cidr.NextSubnet(ipnet, 14)
	t.Logf("new ipnet %s", ipnet)
	t.Logf("new ipnet %s %s", ipnet.IP, ipnet.Mask)
	size, lenght := ipnet.Mask.Size()
	t.Logf("new size lenght %d %d ", size, lenght)

}
