package namespacefederation

import (
	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileNamespaceFederation) createOrUpdateFederationControlPlane(instance *federationv1alpha1.NamespaceFederation, c chan<- reconcileresult) {

	c <- reconcileresult{
		Result: reconcile.Result{},
		err:    r.CreateOrUpdateTemplatedResources(instance, "", instance, federationControllerTemplate),
	}

}
