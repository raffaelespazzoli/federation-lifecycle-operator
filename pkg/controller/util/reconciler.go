package util

import (
	"context"
	"errors"
	"fmt"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type SchemaAwareClient interface {
	GetClient() client.Client
	GetScheme() *runtime.Scheme
	CreateOrUpdateResource(owner metav1.Object, obj metav1.Object) error
	DeleteResource(obj metav1.Object) error
	CreateIfNotExists(obj metav1.Object) error
}

type ReconcilerBase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func NewReconcilerBase(client client.Client, scheme *runtime.Scheme) ReconcilerBase {
	return ReconcilerBase{
		client: client,
		scheme: scheme,
	}
}

func (r *ReconcilerBase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r *ReconcilerBase) GetClient() client.Client {
	return r.client
}

func (r *ReconcilerBase) GetScheme() *runtime.Scheme {
	return r.scheme
}

func (r *ReconcilerBase) CreateOrUpdateResource(owner metav1.Object, namespace string, obj metav1.Object) error {
	runtimeObj, ok := (obj).(runtime.Object)
	if !ok {
		return fmt.Errorf("is not a %T a runtime.Object", obj)
	}

	if owner != nil {
		_ = controllerutil.SetControllerReference(owner, obj, r.GetScheme())
	}
	if namespace != "" {
		obj.SetNamespace(namespace)
	}

	obj2 := unstructured.Unstructured{}
	obj2.SetKind(runtimeObj.GetObjectKind().GroupVersionKind().Kind)
	if runtimeObj.GetObjectKind().GroupVersionKind().Group != "" {
		obj2.SetAPIVersion(runtimeObj.GetObjectKind().GroupVersionKind().Group + "/" + runtimeObj.GetObjectKind().GroupVersionKind().Version)
	} else {
		obj2.SetAPIVersion(runtimeObj.GetObjectKind().GroupVersionKind().Version)
	}

	err := r.GetClient().Get(context.TODO(), types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, &obj2)

	if apierrors.IsNotFound(err) {
		err = r.GetClient().Create(context.TODO(), runtimeObj)
		if err != nil {
			log.Error(err, "unable to create object", "object", runtimeObj)
		}
		return err
	}
	if err == nil {
		obj.SetResourceVersion(obj2.GetResourceVersion())
		err = r.GetClient().Update(context.TODO(), runtimeObj)
		if err != nil {
			log.Error(err, "unable to update object", "object", runtimeObj)
		}
		return err

	}
	log.Error(err, "unable to lookup object", "object", runtimeObj)
	return err
}

// DeleteResource delete an  existing resource. It doesn't fail if the resource does not exists
func (r *ReconcilerBase) DeleteResource(obj metav1.Object) error {
	runtimeObj, ok := (obj).(runtime.Object)
	if !ok {
		return fmt.Errorf("is not a %T a runtime.Object", obj)
	}

	err := r.GetClient().Delete(context.TODO(), runtimeObj, nil)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "unable to delete object ", "object", runtimeObj)
		return err
	}
	return nil
}

func (r *ReconcilerBase) CreateIfNotExists(owner metav1.Object, namespace string, obj metav1.Object) error {
	runtimeObj, ok := (obj).(runtime.Object)
	if !ok {
		return fmt.Errorf("is not a %T a runtime.Object", obj)
	}

	if owner != nil {
		_ = controllerutil.SetControllerReference(owner, obj, r.GetScheme())
	}
	if namespace != "" {
		obj.SetNamespace(namespace)
	}

	err := r.GetClient().Create(context.TODO(), runtimeObj)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		log.Error(err, "unable to create object ", "object", runtimeObj)
		return err
	}
	return nil
}

func (r *ReconcilerBase) GetClientFromKubeconfigSecret(secret *corev1.Secret) (*ReconcilerBase, error) {

	if len(secret.Data) == 0 {
		return nil, fmt.Errorf("Secret contains no values")
	}

	var val []byte
	var restConfig *rest.Config

	for key, value := range secret.Data {
		if key == "kubeconfig" {
			val = value
		}
	}

	if val == nil {
		return nil, errors.New("kubeconfig entry not found")
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(val)
	if err != nil {
		log.Error(err, "unable to create rest config", "kubeconfig", val)
		return nil, err
	}

	mapper, err := apiutil.NewDiscoveryRESTMapper(restConfig)
	if err != nil {
		log.Error(err, "unable to create mapper", "restconfig", restConfig)
		return nil, err
	}

	c, err := client.New(restConfig, client.Options{
		Scheme: scheme.Scheme,
		Mapper: mapper,
	})

	if err != nil {
		log.Error(err, "unable to create new client")
		return nil, err
	}
	remoteClusterClient := NewReconcilerBase(c, scheme.Scheme)

	return &remoteClusterClient, nil

}

func (r *ReconcilerBase) CreateOrUpdateTemplatedResources(owner metav1.Object, namespace string, data interface{}, template *template.Template) error {
	objs, err := ProcessTemplateArray(data, template)
	if err != nil {
		log.Error(err, "error creating manifest from template")
		return err
	}
	for _, obj := range *objs {
		err = r.CreateOrUpdateResource(owner, namespace, &obj)
		if err != nil {
			log.Error(err, "unable to create object", "object", &obj)
			return err
		}
	}
	return nil
}
