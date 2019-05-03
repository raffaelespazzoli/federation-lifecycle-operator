package namespacefederation

import (
	"context"
	"errors"
	"strings"
	"time"

	federationv2v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type secretNotFoundError struct {
	secretName    string
	namespace     string
	originalError error
}

func (e secretNotFoundError) Error() string {
	return "could not find secret: " + e.secretName + " in namespace: " + e.namespace + ". Original error: " + e.originalError.Error()
}

const remoteServiceAccountName string = "federation-controllee"

func (r *ReconcileNamespaceFederation) createOrUpdateFederatedClusters(instance *federationv1alpha1.NamespaceFederation, c chan<- reconcileresult) {
	err := r._createOrUpdateFederatedClusters(instance)
	if _, ok := err.(secretNotFoundError); ok {
		log.Info("secret for remote cluster not found, maybe it;s not provisioned yet, waiting for 1 minute: " + err.Error())
		c <- reconcileresult{
			reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Minute,
			},
			nil,
		}
		return
	}
	if err != nil {
		log.Error(err, "unable to reconcile federated cluster", "instance", instance)
		c <- reconcileresult{
			reconcile.Result{},
			err,
		}
	}
}

func (r *ReconcileNamespaceFederation) _createOrUpdateFederatedClusters(instance *federationv1alpha1.NamespaceFederation) error {
	addClusters, deleteClusters, err := r.getAddAndDeleteCluster(instance)
	if err != nil {
		log.Error(err, "Error calculating add and delete clusters for instance", "instance", *instance)
		return err
	}

	log.Info("clusters to be deleted: ", "delClusters", deleteClusters)
	// first we take care of deleting the deleteclusetr

	for _, cluster := range deleteClusters {
		err = r.manageDeleteCluster(cluster, instance)
		log.Error(err, "Unable to successfully delete cluster", "cluster", cluster)
		return err
	}

	//then we add new clusters.
	log.Info("clusters to be added: ", "addClusters", addClusters)
	for _, cluster := range addClusters {
		// retrieve the admin secret
		log.Info("managing cluster: ", "cluster", cluster)
		adminSecret := corev1.Secret{}
		err := r.GetClient().Get(context.TODO(), types.NamespacedName{
			Namespace: cluster.AdminSecretRef.Namespace,
			Name:      cluster.AdminSecretRef.Name,
		}, &adminSecret)
		if apierrors.IsNotFound(err) {
			return secretNotFoundError{
				secretName:    cluster.AdminSecretRef.Name,
				namespace:     cluster.AdminSecretRef.Namespace,
				originalError: err,
			}
		}
		if err != nil {
			log.Error(err, "unable to retrieve admin secret", "namespace", cluster.AdminSecretRef.Namespace, "name", cluster.AdminSecretRef.Name, "cluster", cluster)
			return err
		}
		err = r.manageAddCluster(cluster.Name, instance, &adminSecret)
		if err != nil {
			log.Error(err, "Unable to successfully add cluster", "cluster", cluster)
			return err
		}
	}

	return nil

}

//adding a cluster consist of creating the namespace in the target cluster and populating it with the service account and then creating the federatedcluster and the secret in the same namespace instance
func (r *ReconcileNamespaceFederation) manageAddCluster(cluster string, instance *federationv1alpha1.NamespaceFederation, adminSecret *corev1.Secret) error {
	// create new namespace in remote cluster
	remoteClusterClient, err := r.GetClientFromKubeconfigSecret(adminSecret)
	if err != nil {
		log.Error(err, "Error creating remote client for cluster", "cluster", cluster)
		return err
	}
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: instance.GetNamespace(),
		},
	}

	err = remoteClusterClient.CreateIfNotExists(nil, "", &namespace)
	if err != nil {
		log.Error(err, "Error creating remote namespace for cluster", "cluster", cluster, "namespace", namespace.GetName())
		return err
	}

	// apply template in remote cluster
	remoteClusterClient.CreateOrUpdateTemplatedResources(nil, "", instance, remoteFederatedClusterTemplate)
	if err != nil {
		log.Error(err, "unable to apply template in remote cluster", "cluster", cluster)
		return err
	}

	//apply template in local cluster
	log.Info("rertrieve secrets", "remoteclusterclient", *remoteClusterClient)
	remoteSecret, err := getSecretForRemoteServiceAccount(remoteClusterClient, cluster, instance)
	if err != nil {
		log.Error(err, "unable to retrieve remote secret for cluster", "cluster", cluster, "namespace", instance.GetNamespace())
		return err
	}
	federatedClusterMerge := federatedClusterMerge{
		Namespace:    instance.GetNamespace(),
		Cluster:      cluster,
		CaCRT:        string(remoteSecret.Data["ca.crt"]),
		ServiceCaCRT: string(remoteSecret.Data["service-ca.crt"]),
		Token:        string(remoteSecret.Data["token"]),
		SecretName:   cluster + "-remote",
	}

	return r.CreateOrUpdateTemplatedResources(instance, "", federatedClusterMerge, federatedClusterTemplate)

}

func getSecretForRemoteServiceAccount(remoteClusterClient *util.ReconcilerBase, cluster string, instance *federationv1alpha1.NamespaceFederation) (*corev1.Secret, error) {
	remoteServiceAccount := &corev1.ServiceAccount{}
	err := remoteClusterClient.GetClient().Get(context.TODO(), types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      remoteServiceAccountName,
	}, remoteServiceAccount)
	if err != nil {
		log.Error(err, "unable to retrieve remote service account", "namespace", instance.GetNamespace(), "ServiceAccount", remoteServiceAccountName, "cluster", cluster)
		return nil, err
	}

	var remoteSecretName string
	for _, secret := range remoteServiceAccount.Secrets {
		if strings.Contains(secret.Name, "token") {
			remoteSecretName = secret.Name
			break
		}
	}
	if remoteSecretName == "" {
		err = errors.New("unable to find remote token secret")
		log.Error(err, "unable to find remote token secret", "service account", remoteServiceAccountName, "cluster", cluster)
		return nil, err
	}
	remoteTokenSecret := &corev1.Secret{}
	err = remoteClusterClient.GetClient().Get(context.TODO(), types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      remoteSecretName,
	}, remoteTokenSecret)
	if err != nil {
		log.Error(err, "unable to retrieve remote token secret", "token secret", remoteSecretName, "cluster", cluster)
		return nil, err
	}
	return remoteTokenSecret, nil
}

type federatedClusterMerge struct {
	Namespace    string
	Cluster      string
	CaCRT        string
	ServiceCaCRT string
	Token        string
	SecretName   string
}

// deleting a cluste consist of deleting the federated namespace on the target cluster and deleting the federatedcluster and secret object in the same namespace as the instance
func (r *ReconcileNamespaceFederation) manageDeleteCluster(cluster string, instance *federationv1alpha1.NamespaceFederation) error {

	// retrieve the federated cluster to know what the associated secret is
	federatedCluster := &federationv2v1alpha1.FederatedCluster{}
	err := r.GetClient().Get(context.TODO(), types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      cluster,
	}, federatedCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If we don't find the cluster we assume it has already been correclty deleted
			return nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Error redetrieving federatedcluster", "name", cluster, "namespace", instance.GetNamespace())
		return err
	}

	//delete the local federatedcluster and secret
	federatedClusterMerge := federatedClusterMerge{
		Namespace:    instance.GetNamespace(),
		Cluster:      cluster,
		CaCRT:        "N/A",
		ServiceCaCRT: "N/A",
		Token:        "N/A",
		SecretName:   federatedCluster.Spec.SecretRef.Name,
	}

	objs, err := util.ProcessTemplateArray(federatedClusterMerge, federatedClusterTemplate)
	if err != nil {
		log.Error(err, "error creating manifest from template")
		return err
	}
	for _, obj := range *objs {
		err = r.DeleteResource(&obj)
		if err != nil {
			log.Error(err, "unable to delete object", "object", &obj)
			return err
		}
	}
	return nil
}

func (r *ReconcileNamespaceFederation) getAddAndDeleteCluster(instance *federationv1alpha1.NamespaceFederation) ([]federationv1alpha1.Cluster, []string, error) {
	//we assume that if a federated cluster object exists it means that it has been correclty federated

	federatedClusterList := &federationv2v1alpha1.FederatedClusterList{}
	err := r.GetClient().List(context.TODO(), &client.ListOptions{Namespace: instance.GetNamespace()}, federatedClusterList)
	if err != nil {
		log.Error(err, "Error listing federatedclusters in namespace", "namespace", instance.GetNamespace())
		return nil, nil, err
	}
	// let's calculate the add clusters
	addClusters := []federationv1alpha1.Cluster{}
	for _, cluster := range instance.Spec.Clusters {
		if !containsCluster(federatedClusterList, cluster) {
			addClusters = append(addClusters, cluster)
		}
	}

	//let's calculate the delete clusters
	deleteClusters := []string{}
	for _, federatedCluster := range federatedClusterList.Items {
		if !containsFederatedCluster(instance.Spec.Clusters, &federatedCluster) {
			deleteClusters = append(deleteClusters, federatedCluster.Spec.ClusterRef.Name)
		}
	}

	return addClusters, deleteClusters, nil

}

func containsCluster(federatedClusterList *federationv2v1alpha1.FederatedClusterList, cluster federationv1alpha1.Cluster) bool {
	for _, federatedCluster := range federatedClusterList.Items {
		if cluster.Name == federatedCluster.Spec.ClusterRef.Name {
			return true
		}
	}
	return false
}

func containsFederatedCluster(clusters []federationv1alpha1.Cluster, federatedCluster *federationv2v1alpha1.FederatedCluster) bool {
	for _, cluster := range clusters {
		if cluster.Name == federatedCluster.Spec.ClusterRef.Name {
			return true
		}
	}
	return false
}
