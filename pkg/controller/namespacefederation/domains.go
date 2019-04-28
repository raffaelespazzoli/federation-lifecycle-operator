package namespacefederation

import (
	"strings"

	multiclusterdnsv1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileNamespaceFederation) createDomains(instance *federationv1alpha1.NamespaceFederation, c chan<- reconcileresult) {

	c <- reconcileresult{
		Result: reconcile.Result{},
		err:    r._createDomains(instance),
	}

}

func (r *ReconcileNamespaceFederation) _createDomains(instance *federationv1alpha1.NamespaceFederation) error {
	for _, domain := range instance.Spec.Domains {
		domainResource := multiclusterdnsv1alpha1.Domain{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "multiclusterdns.federation.k8s.io/v1alpha1",
				Kind:       "Domain",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      strings.Replace(domain, ".", "-", -1),
				Namespace: instance.GetNamespace(),
			},
			Domain: domain,
		}
		err := r.CreateOrUpdateResource(instance, "", &domainResource)
		if err != nil {
			log.Error(err, "unable to create domain", "domain", domainResource)
			return err
		}
	}
	return nil
}
