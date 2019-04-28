package namespacefederation

import (
	"context"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_namespacefederation")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new NamespaceFederation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNamespaceFederation{
		ReconcilerBase: util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme()),
		config:         mgr.GetConfig(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("namespacefederation-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NamespaceFederation
	err = c.Watch(&source.Kind{Type: &federationv1alpha1.NamespaceFederation{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner NamespaceFederation
	// err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &federationv1alpha1.NamespaceFederation{},
	// })
	// if err != nil {
	// 	return err
	// }

	return nil
}

var _ reconcile.Reconciler = &ReconcileNamespaceFederation{}

// ReconcileNamespaceFederation reconciles a NamespaceFederation object
type ReconcileNamespaceFederation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	util.ReconcilerBase
	config *rest.Config
}

type reconcileresult struct {
	reconcile.Result
	err error
}

func (r *ReconcileNamespaceFederation) GetConfig() *rest.Config {
	return r.config
}

// Reconcile reads that state of the cluster for a NamespaceFederation object and makes changes based on the state read
// and what is in the NamespaceFederation.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNamespaceFederation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NamespaceFederation")

	// Fetch the NamespaceFederation instance
	instance := &federationv1alpha1.NamespaceFederation{}
	err := r.GetClient().Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	c := make(chan reconcileresult)

	defer close(c)

	go r.createOrUpdateFederationControlPlane(instance, c)

	go r.createOrUpdateFederatedTypes(instance, c)

	go r.createDomains(instance, c)

	go r.createOrUpdateFederatedClusters(instance, c)

	result := reconcile.Result{}
	var err1 *multierror.Error
	for i := 0; i < 4; i++ {
		res := <-c
		result.Requeue = result.Requeue || res.Requeue
		result.RequeueAfter = getMinRequeueAfter(result.RequeueAfter, res.RequeueAfter)
		if res.err != nil {
			err1 = multierror.Append(err1, err)
		}
	}

	return result, err1.ErrorOrNil()
}

func getMinRequeueAfter(a time.Duration, b time.Duration) time.Duration {
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}
