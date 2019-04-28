package multiplenamespacefederation

import (
	"context"
	"errors"
	"strings"
	"time"

	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type RemoteGlobalLoadBalacerMerge struct {
	Instance federationv1alpha1.MultipleNamespaceFederation
	Secret   corev1.Secret
}

func (r *ReconcileMultipleNamespaceFederation) manageGlobalLoadBalancer(instance *federationv1alpha1.MultipleNamespaceFederation) (reconcile.Result, error) {
	switch instance.Spec.GlobalLoadBalancer.GlobalLoadBalancerType {
	case "self-hosted":
		{
			return r.manageSelfHostedGlobalLoadBalancer(instance)
		}
	case "cloud-provider":
		{
			return r.manageCloudProviderGlobalLoadBalancer(instance)
		}
	default:
		{
			return reconcile.Result{}, errors.New("global load balancer type can be Self-Hosted or Cloud-Provider")
		}
	}

}

func (r *ReconcileMultipleNamespaceFederation) manageCloudProviderGlobalLoadBalancer(instance *federationv1alpha1.MultipleNamespaceFederation) (reconcile.Result, error) {
	err := r.CreateOrUpdateTemplatedResources(instance, "", instance, cloudProviderGlobalLoadBalancerTemplate)

	return reconcile.Result{}, err
}

func (r *ReconcileMultipleNamespaceFederation) createAndGetSecretForExternalDNSServiceAccount(instance *federationv1alpha1.MultipleNamespaceFederation) (corev1.Secret, error) {
	tokenSecret := corev1.Secret{}
	serviceAccount := &corev1.ServiceAccount{}
	err := r.GetClient().Get(context.TODO(), types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      "external-dns",
	}, serviceAccount)
	if apierrors.IsNotFound(err) {
		err := r.CreateOrUpdateTemplatedResources(instance, "", instance, selfHostedGlobalLoadBalancerServiceAccountTemplate)
		if err != nil {
			log.Error(err, "unable to process template for local service account", "namespace", instance.GetNamespace())
			return tokenSecret, err
		}
	}
	if err != nil {
		log.Error(err, "unable to retrieve extenral-dns service account", "namespace", instance.GetNamespace())
		return tokenSecret, err
	}

	var secretName string
	for _, secret := range serviceAccount.Secrets {
		if strings.Contains(secret.Name, "token") {
			secretName = secret.Name
			break
		}
	}
	if secretName == "" {
		err := errors.New("unable to find remote token secret")
		log.Error(err, "unable to find remote token secret", "service account", serviceAccount)
		return tokenSecret, err
	}

	err = r.GetClient().Get(context.TODO(), types.NamespacedName{
		Namespace: serviceAccount.GetNamespace(),
		Name:      secretName,
	}, &tokenSecret)
	if err != nil {
		log.Error(err, "unable to retrieve remote token secret", "token secret", secretName)
		return tokenSecret, err
	}
	return tokenSecret, nil

}

func (r *ReconcileMultipleNamespaceFederation) manageSelfHostedGlobalLoadBalancer(instance *federationv1alpha1.MultipleNamespaceFederation) (reconcile.Result, error) {

	secret, err := r.createAndGetSecretForExternalDNSServiceAccount(instance)
	if apierrors.IsNotFound(err) {
		log.Error(err, "either external-dns service account or secret were not found, will wait for one second")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second,
		}, nil
	}
	if err != nil {
		log.Error(err, "unable to retrieve secret for extenral-dns service account", "namespace", instance.GetNamespace())
		return reconcile.Result{}, err
	}

	remoteGlobalLoadBalancerMerge := RemoteGlobalLoadBalacerMerge{
		Secret:   secret,
		Instance: *instance,
	}

	for _, cluster := range instance.Spec.NamespaceFederationSpec.Clusters {
		log.Info("managing cluster", "cluster", cluster)
		remoteSecret := corev1.Secret{}
		err := r.GetClient().Get(context.TODO(), types.NamespacedName{
			Namespace: cluster.AdminSecretRef.Namespace,
			Name:      cluster.AdminSecretRef.Name,
		}, &remoteSecret)
		if apierrors.IsNotFound(err) {
			log.Error(err, "the secret for the remote cluster could not be found, maybe the cluster is not provisioned yet, waiting for 1 minute")
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: time.Minute,
			}, nil
		}
		if err != nil {
			log.Error(err, "unable to retrieve admin secret", "namespace", cluster.AdminSecretRef.Namespace, "name", cluster.AdminSecretRef.Name, "cluster", cluster)
			return reconcile.Result{}, err
		}
		remoteClusterClient, err := r.GetClientFromKubeconfigSecret(&remoteSecret)
		if err != nil {
			log.Error(err, "unable to create client to remote cluster", "cluster", cluster, "remote secret", remoteSecret)
			return reconcile.Result{}, err
		}
		err = remoteClusterClient.CreateOrUpdateTemplatedResources(nil, "", remoteGlobalLoadBalancerMerge, selfHostedGlobalLoadBalancerTemplate)
	}
	return reconcile.Result{}, err
}
