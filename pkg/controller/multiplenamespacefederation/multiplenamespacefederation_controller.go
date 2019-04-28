package multiplenamespacefederation

import (
	"context"

	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_multiplenamespacefederation")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new MultipleNamespaceFederation Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMultipleNamespaceFederation{
		ReconcilerBase: util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("multiplenamespacefederation-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource MultipleNamespaceFederation
	err = c.Watch(&source.Kind{Type: &federationv1alpha1.MultipleNamespaceFederation{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

var _ reconcile.Reconciler = &ReconcileMultipleNamespaceFederation{}

// ReconcileMultipleNamespaceFederation reconciles a MultipleNamespaceFederation object
type ReconcileMultipleNamespaceFederation struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	util.ReconcilerBase
}

// Reconcile reads that state of the cluster for a MultipleNamespaceFederation object and makes changes based on the state read
// and what is in the MultipleNamespaceFederation.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMultipleNamespaceFederation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling MultipleNamespaceFederation")

	// Fetch the MultipleNamespaceFederation instance
	instance := &federationv1alpha1.MultipleNamespaceFederation{}
	err := r.GetClient().Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// get all the namespace for which this CR apply
	namespaces := &corev1.NamespaceList{}
	selector, err := metav1.LabelSelectorAsSelector(instance.Spec.NamespaceSelector)
	if err != nil {
		log.Error(err, "unable to create label selector from namespace selector", "selector", instance.Spec.NamespaceSelector)
		return reconcile.Result{}, err
	}
	err = r.GetClient().List(context.TODO(), &client.ListOptions{
		LabelSelector: selector,
	}, namespaces)
	if err != nil {
		log.Error(err, "unable to retrieve list of selected namespaces", "selector", selector)
		return reconcile.Result{}, err
	}
	log.Info("selected namespaces", "namespaces", namespaces.Items)
	for _, namespace := range namespaces.Items {
		//err = r.CreateOrUpdateResource(instance, GetNamespaceFederation(instance, &namespace))
		err = r.CreateOrUpdateResource(nil, "", GetNamespaceFederation(instance, &namespace))
		if err != nil {
			log.Error(err, "unable to create namespacefederation", "multiplenamespacefederation", instance, "namespace", namespace, "namespacefederation", GetNamespaceFederation(instance, &namespace))
		}
	}

	if instance.Spec.GlobalLoadBalancer.GlobalLoadBalancerType != "" {
		return r.manageGlobalLoadBalancer(instance)
	}

	return reconcile.Result{}, nil
}

func GetNamespaceFederation(instance *federationv1alpha1.MultipleNamespaceFederation, namespace *corev1.Namespace) *federationv1alpha1.NamespaceFederation {
	return &federationv1alpha1.NamespaceFederation{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NamespaceFederation",
			APIVersion: "federation.raffa.systems/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetName(),
			Namespace: namespace.GetName(),
		},
		Spec: *instance.Spec.NamespaceFederationSpec.DeepCopy(),
	}
}
