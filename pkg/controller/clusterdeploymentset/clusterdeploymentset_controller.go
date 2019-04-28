package clusterdeploymentset

import (
	"context"

	"errors"
	"net"
	"strconv"

	"github.com/apparentlymart/go-cidr/cidr"
	networkoperatorv1 "github.com/openshift/cluster-network-operator/pkg/apis/networkoperator/v1"
	hivev1aplha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_clusterdeploymentset")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ClusterDeploymentSet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileClusterDeploymentSet{
		ReconcilerBase: util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterdeploymentset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ClusterDeploymentSet
	err = c.Watch(&source.Kind{Type: &federationv1alpha1.ClusterDeploymentSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ClusterDeploymentSet
	err = c.Watch(&source.Kind{Type: &hivev1aplha1.ClusterDeployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &federationv1alpha1.ClusterDeploymentSet{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileClusterDeploymentSet{}

// ReconcileClusterDeploymentSet reconciles a ClusterDeploymentSet object
type ReconcileClusterDeploymentSet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	util.ReconcilerBase
}

// Reconcile reads that state of the cluster for a ClusterDeploymentSet object and makes changes based on the state read
// and what is in the ClusterDeploymentSet.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileClusterDeploymentSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ClusterDeploymentSet")

	// Fetch the ClusterDeploymentSet instance
	instance := &federationv1alpha1.ClusterDeploymentSet{}
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

	//create cluster deployments if they don't already exist
	clusterDeployments, err := r.getClusterDeployments(instance)
	if err != nil {
		log.Error(err, "unable to calculate cluster deployments from instance", "instance", instance)
		return reconcile.Result{}, err
	}
	for _, clusterDeployment := range clusterDeployments {
		log.Info("debug outside", "clusterdeployment", clusterDeployment.Name, "region", clusterDeployment.Spec.AWS.Region)
		err = r.CreateIfNotExists(instance, "", &clusterDeployment)
		if err != nil {
			log.Error(err, "unable to create cluster deployment", "clusterdeployment", clusterDeployment)
			return reconcile.Result{}, err
		}
		if instance.Spec.RegisterClusters {
			err = r.manageClusterRegistration(&clusterDeployment)
			if err != nil {
				log.Error(err, "unable to register cluster", "clusterdeployment", clusterDeployment)
				return reconcile.Result{}, err
			}
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileClusterDeploymentSet) getClusterDeployments(instance *federationv1alpha1.ClusterDeploymentSet) ([]hivev1aplha1.ClusterDeployment, error) {
	clusterDeployments := []hivev1aplha1.ClusterDeployment{}
	for i := 0; i < instance.Spec.Replicas; i++ {
		clusterDeployment := hivev1aplha1.ClusterDeployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "hive.openshift.io/v1alpha1",
				Kind:       "ClusterDeployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        instance.Name + "-" + strconv.Itoa(i),
				Namespace:   instance.GetNamespace(),
				Labels:      instance.Spec.Template.GetLabels(),
				Annotations: instance.Spec.Template.GetAnnotations(),
			},
			Spec: *instance.Spec.Template.Spec.DeepCopy(),
		}
		clusterDeployment.Spec.ClusterName = instance.Name + "-" + strconv.Itoa(i)
		log.Info("data", "i", i, "len", len(instance.Spec.Regions), "modulo", i%len(instance.Spec.Regions), "regions", instance.Spec.Regions)
		clusterDeployment.Spec.AWS.Region = instance.Spec.Regions[i%len(instance.Spec.Regions)]
		log.Info("selected region", "instance.Spec.Regions[i%len(instance.Spec.Regions)]", instance.Spec.Regions[i%len(instance.Spec.Regions)], "clusterDeployment.Spec.AWS.Region", clusterDeployment.Spec.AWS.Region)
		//network cidrs
		if instance.Spec.EnsureNoOverlappingCIDR {
			for j := range instance.Spec.Template.Spec.Networking.ClusterNetworks {
				_, ipnet, err := net.ParseCIDR(instance.Spec.Template.Spec.Networking.ClusterNetworks[j].CIDR)
				if err != nil {
					log.Error(err, "unable to parse cidr", "cidr", instance.Spec.Template.Spec.Networking.ClusterNetworks[j].CIDR)
					return []hivev1aplha1.ClusterDeployment{}, err
				}
				ipnet, ko := getNextNSubnet(ipnet, getIPPrefix(ipnet.Mask), i)
				if ko {
					err = errors.New("network address overflow")
					log.Error(err, "overflow while calculating next cidr", "net", ipnet, "times", i)
					return []hivev1aplha1.ClusterDeployment{}, err
				}
				clusterDeployment.Spec.ClusterNetworks[j] = networkoperatorv1.ClusterNetwork{
					CIDR:             ipnet.String(),
					HostSubnetLength: instance.Spec.Template.Spec.Networking.ClusterNetworks[j].HostSubnetLength,
				}

			}
			//service cidr
			_, ipnet, err := net.ParseCIDR(instance.Spec.Template.Spec.Networking.ServiceCIDR)
			if err != nil {
				log.Error(err, "unable to parse cidr", "cidr", instance.Spec.Template.Spec.Networking.ServiceCIDR)
				return []hivev1aplha1.ClusterDeployment{}, err
			}
			ipnet, ko := getNextNSubnet(ipnet, getIPPrefix(ipnet.Mask), i)
			if ko {
				err = errors.New("network address overflow")
				log.Error(err, "overflow while calculating next cidr", "net", ipnet, "times", i)
				return []hivev1aplha1.ClusterDeployment{}, err
			}
			clusterDeployment.Spec.ServiceCIDR = ipnet.String()
			// machine cidrs
			_, ipnet, err = net.ParseCIDR(instance.Spec.Template.Spec.Networking.MachineCIDR)
			if err != nil {
				log.Error(err, "unable to parse cidr", "cidr", instance.Spec.Template.Spec.Networking.MachineCIDR)
				return []hivev1aplha1.ClusterDeployment{}, err
			}
			ipnet, ko = getNextNSubnet(ipnet, getIPPrefix(ipnet.Mask), i)
			if ko {
				err = errors.New("network address overflow")
				log.Error(err, "overflow while calculating next cidr", "net", ipnet, "times", i)
				return []hivev1aplha1.ClusterDeployment{}, err
			}
			clusterDeployment.Spec.MachineCIDR = ipnet.String()
		}
		log.Info("debug inside", "clusterdeployment", clusterDeployment.Name, "region", clusterDeployment.Spec.AWS.Region)
		clusterDeployments = append(clusterDeployments, clusterDeployment)
	}
	return clusterDeployments, nil
}

func (r *ReconcileClusterDeploymentSet) manageClusterRegistration(instance *hivev1aplha1.ClusterDeployment) error {
	clusterDeployment := &hivev1aplha1.ClusterDeployment{}
	err := r.GetClient().Get(context.TODO(), types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}, clusterDeployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Perhaps it doesn't exits because it has not been created yet, we exit with no error.
			return nil
		}
		// Error reading the object - requeue the request.
		return err
	}
	if clusterDeployment.Status.Installed {
		cluster := &clusterregistry.Cluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "clusterregistry.k8s.io/v1alpha1",
				Kind:       "Cluster",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: clusterDeployment.Namespace,
				Name:      clusterDeployment.Name,
			},
			Spec: clusterregistry.ClusterSpec{
				KubernetesAPIEndpoints: clusterregistry.KubernetesAPIEndpoints{
					ServerEndpoints: []clusterregistry.ServerAddressByClientCIDR{
						clusterregistry.ServerAddressByClientCIDR{
							ServerAddress: clusterDeployment.Status.APIURL,
							ClientCIDR:    "0.0.0.0/0",
						},
					},
				},
			},
		}
		err = r.CreateIfNotExists(clusterDeployment, "", cluster)
		if err != nil {
			log.Error(err, "unable to create cluster registration", "cluster", "cluster")
			return err
		}

	}
	return nil
}

func getNextNSubnet(network *net.IPNet, prefixLen int, n int) (*net.IPNet, bool) {
	ipnet := network
	var overflow bool
	for i := 0; i < n; i++ {
		var ko bool
		ipnet, ko = cidr.NextSubnet(ipnet, prefixLen)
		if ko {
			overflow = true
		}
	}
	return ipnet, overflow
}

func getIPPrefix(mask net.IPMask) int {
	size, _ := mask.Size()
	return size
}
