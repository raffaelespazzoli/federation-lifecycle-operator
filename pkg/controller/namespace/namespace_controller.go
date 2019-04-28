package namespace

import (
	"context"

	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"

	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/multiplenamespacefederation"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_namespace")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Namespace Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNamespace{
		ReconcilerBase: util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("namespace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Namespace
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNamespace{}

// ReconcileNamespace reconciles a Namespace object
type ReconcileNamespace struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	util.ReconcilerBase
}

// Reconcile reads that state of the cluster for a Namespace object and makes changes based on the state read
// and what is in the Namespace.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNamespace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Namespace")

	// Fetch the Namespace instance
	instance := &corev1.Namespace{}
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

	multipleNamespaceFederations, err := r.findApplicableMultipleNamespaceFederation(instance)
	if err != nil {
		log.Error(err, "unable to create the list of applicable multiplenamepsacefederation", "namespace", instance)
		return reconcile.Result{}, err
	}

	for _, multipleNamespaceFederation := range multipleNamespaceFederations {
		//err = r.CreateOrUpdateResource(&multipleNamespaceFederation, multiplenamespacefederation.GetNamespaceFederation(&multipleNamespaceFederation, instance))
		err = r.CreateOrUpdateResource(nil, "", multiplenamespacefederation.GetNamespaceFederation(&multipleNamespaceFederation, instance))
		if err != nil {
			log.Error(err, "unable to create nanemspacefederation", "multiplenamespacefederation", multipleNamespaceFederation, "namespace", instance, "namespacefederation", multiplenamespacefederation.GetNamespaceFederation(&multipleNamespaceFederation, instance))
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileNamespace) findApplicableMultipleNamespaceFederation(instance *corev1.Namespace) ([]federationv1alpha1.MultipleNamespaceFederation, error) {
	multipleNamespaceFederationList := &federationv1alpha1.MultipleNamespaceFederationList{}
	err := r.GetClient().List(context.TODO(), &client.ListOptions{}, multipleNamespaceFederationList)
	if err != nil {
		if errors.IsNotFound(err) {
			return []federationv1alpha1.MultipleNamespaceFederation{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "unable to retrieve the list of multiplenamespacefederation")
		return []federationv1alpha1.MultipleNamespaceFederation{}, err
	}
	applicableMultipleNamespaceFederationList := []federationv1alpha1.MultipleNamespaceFederation{}
	for _, multipleNamespaceFederation := range multipleNamespaceFederationList.Items {
		selector, err := metav1.LabelSelectorAsSelector(multipleNamespaceFederation.Spec.NamespaceSelector)
		if err != nil {
			log.Error(err, "unable to create se;ector from label selector", "labelSelector", multipleNamespaceFederation.Spec.NamespaceSelector)
			return []federationv1alpha1.MultipleNamespaceFederation{}, err
		}
		if selector.Matches(labels.Set(instance.Labels)) {
			applicableMultipleNamespaceFederationList = append(applicableMultipleNamespaceFederationList, multipleNamespaceFederation)
		}
	}

	return applicableMultipleNamespaceFederationList, nil
}
