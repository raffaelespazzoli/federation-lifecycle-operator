package namespacefederation

import (
	"context"
	"fmt"
	"strings"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	federationv2v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	extensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	federationv1alpha1 "github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis/federation/v1alpha1"
)

const (
	defaultFederationGroup   = "types.federation.k8s.io"
	defaultFederationVersion = "v1alpha1"
)

type SimpleTyepCRDPair struct {
	SimpleType metav1.TypeMeta
	CRD        *extensionv1beta1.CustomResourceDefinition
}

func (r *ReconcileNamespaceFederation) createOrUpdateFederatedTypes(instance *federationv1alpha1.NamespaceFederation, c chan<- reconcileresult) {
	c <- reconcileresult{
		Result: reconcile.Result{},
		err:    r._createOrUpdateFederatedTypes(instance),
	}
}

func (r *ReconcileNamespaceFederation) _createOrUpdateFederatedTypes(instance *federationv1alpha1.NamespaceFederation) error {
	addTypes, deleteTypes, err := r.getAddAndDeleteTypes(instance)

	if err != nil {
		log.Error(err, "Error calculating add and delete types for instance", "instance", *instance)
		return err
	}

	log.Info("types to be added: ", "types", addTypes)
	log.Info("types to be removed: ", "types", deleteTypes)

	//for delete types we delete only the federated type and not the CRD, because the CRD may be in use by some other namespaces
	err = r.deleteFederatedTypes(instance, deleteTypes)
	if err != nil {
		log.Error(err, "Error deleting federated types for instance", "instance", *instance)
		return err
	}

	//for add clusters we generate the crd and the federated types and we create them
	pairs, err := r.generateCRDS(addTypes)
	if err != nil {
		log.Error(err, "Error generating crd for addTypes", "addTyoes", addTypes)
		return err
	}
	for i, pair := range pairs {
		err = r.CreateOrUpdateResource(nil, "", pair.CRD)
		if err != nil {
			log.Error(err, "unable to create/update object", "object", &pair.CRD)
			return err
		}
		pairs[i].CRD.APIVersion = "apiextensions.k8s.io/v1beta1"
		pairs[i].CRD.Kind = "CustomResourceDefinition"
	}

	return r.CreateOrUpdateTemplatedResources(instance, instance.GetNamespace(), pairs, federatedTypesTemplate)

}

func (r *ReconcileNamespaceFederation) deleteFederatedTypes(instance *federationv1alpha1.NamespaceFederation, deleteTypes []federationv2v1alpha1.FederatedTypeConfig) error {
	for _, federatedType := range deleteTypes {
		err := r.DeleteResource(&federatedType)
		if err != nil {
			log.Error(err, "Unable to delete federated type", "type", federatedType)
		}
	}
	return nil
}

func (r *ReconcileNamespaceFederation) getAddAndDeleteTypes(instance *federationv1alpha1.NamespaceFederation) ([]metav1.TypeMeta, []federationv2v1alpha1.FederatedTypeConfig, error) {
	federatedTypeList := &federationv2v1alpha1.FederatedTypeConfigList{}
	err := r.GetClient().List(context.TODO(), &client.ListOptions{Namespace: instance.GetNamespace()}, federatedTypeList)
	if err != nil {
		log.Error(err, "Error listing federated types in namespace", "namespace", instance.GetNamespace())
		return nil, nil, err
	}
	// namespaces federatedTypeConfig has to always be there, so lets remove it from the list.
	federatedTypes := []federationv2v1alpha1.FederatedTypeConfig{}
	for _, federatedType := range federatedTypeList.Items {
		if federatedType.Name != "namespaces" {
			federatedTypes = append(federatedTypes, federatedType)
		}
	}
	federatedTypeList.Items = federatedTypes
	// let's calculate the add federatedType
	addTypes := []metav1.TypeMeta{}
	for _, simpleType := range instance.Spec.FederatedTypes {
		if !containsSimpleType(federatedTypeList, simpleType) {
			addTypes = append(addTypes, simpleType)
		}
	}

	//let's calculate the delete federatedType
	deleteTypes := []federationv2v1alpha1.FederatedTypeConfig{}
	for _, federatedType := range federatedTypeList.Items {
		if !containsFederatedType(instance.Spec.FederatedTypes, &federatedType) {
			deleteTypes = append(deleteTypes, federatedType)
		}
	}

	return addTypes, deleteTypes, nil
}

func containsSimpleType(federatedTypeList *federationv2v1alpha1.FederatedTypeConfigList, simpleType metav1.TypeMeta) bool {
	for _, federatedType := range federatedTypeList.Items {
		if simpleType == getAncestorType(&federatedType) {
			return true
		}
	}
	return false
}
func containsFederatedType(simpleTypes []metav1.TypeMeta, federatedType *federationv2v1alpha1.FederatedTypeConfig) bool {
	ancestorType := getAncestorType(federatedType)
	for _, simpleType := range simpleTypes {
		if simpleType == ancestorType {
			return true
		}
	}
	return false
}

func getAncestorType(federatedType *federationv2v1alpha1.FederatedTypeConfig) metav1.TypeMeta {
	if federatedType.Spec.Target.Group != "" {
		return metav1.TypeMeta{
			APIVersion: federatedType.Spec.Target.Group + "/" + federatedType.Spec.Target.Version,
			Kind:       federatedType.Spec.Target.Kind,
		}
	} else {
		return metav1.TypeMeta{
			APIVersion: federatedType.Spec.Target.Version,
			Kind:       federatedType.Spec.Target.Kind,
		}
	}
}

func (r *ReconcileNamespaceFederation) generateCRDS(types []metav1.TypeMeta) ([]SimpleTyepCRDPair, error) {

	simpleTypeCrdPairs := []SimpleTyepCRDPair{}

	if types != nil || len(types) != 0 {
		for _, t := range types {

			// groupKind := t.GroupVersionKind().GroupKind().String()
			// nameParts := strings.SplitN(groupKind, ".", 2)
			// targetPluralName := federationv2v1alpha1.PluralName(nameParts[0])

			// if len(nameParts) > 1 {
			// 	groupKind = fmt.Sprintf("%v.%v", targetPluralName, nameParts[1])
			// } else {
			// 	groupKind = targetPluralName
			// }

			gvk := t.GroupVersionKind()
			gvk.Kind = federationv2v1alpha1.PluralName(gvk.Kind)
			apiResource, err := LookupAPIResource(r.config, gvk.GroupKind().String(), "")
			if err != nil {
				fmt.Printf("Error! %v", err)
			}

			typeConfig := typeConfigForTarget(*apiResource)

			accessor, err := newSchemaAccessor(r.config, *apiResource)
			if err != nil {
				return nil, errors.Wrap(err, "Error initializing validation schema accessor")
			}
			shortNames := []string{}
			for _, shortName := range apiResource.ShortNames {
				shortNames = append(shortNames, fmt.Sprintf("f%s", shortName))
			}

			crd := federatedTypeCRD(typeConfig, accessor, shortNames)
			simpleTypeCrdPairs = append(simpleTypeCrdPairs, SimpleTyepCRDPair{CRD: crd, SimpleType: t})

		}
	}

	return simpleTypeCrdPairs, nil
}

func LookupAPIResource(config *rest.Config, key, targetVersion string) (*metav1.APIResource, error) {
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating discovery client")
	}

	resourceLists, err := client.ServerPreferredResources()
	if err != nil {
		return nil, errors.Wrap(err, "Error listing api resources")
	}

	// TODO(marun) Consider using a caching scheme ala kubectl
	lowerKey := strings.ToLower(key)
	var targetResource *metav1.APIResource
	var matchedResources []string
	var matchResource = func(resource metav1.APIResource, gv schema.GroupVersion) {
		if targetResource == nil {
			targetResource = resource.DeepCopy()
			targetResource.Group = gv.Group
			targetResource.Version = gv.Version
		}

		matchedResources = append(matchedResources, groupQualifiedName(resource.Name, gv.Group))
	}

	for _, resourceList := range resourceLists {
		// The list holds the GroupVersion for its list of APIResources
		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing GroupVersion")
		}
		if len(targetVersion) > 0 && gv.Version != targetVersion {
			continue
		}
		for _, resource := range resourceList.APIResources {
			if lowerKey == resource.Name ||
				lowerKey == resource.SingularName ||
				lowerKey == strings.ToLower(resource.Kind) ||
				lowerKey == fmt.Sprintf("%s.%s", resource.Name, gv.Group) {

				matchResource(resource, gv)
				continue
			}
			for _, shortName := range resource.ShortNames {
				if lowerKey == strings.ToLower(shortName) {
					matchResource(resource, gv)
					break
				}
			}
		}

	}
	if len(matchedResources) > 1 {
		return nil, errors.Errorf("Multiple resources are matched by %q: %s. A group-qualified plural name must be provided.", key, strings.Join(matchedResources, ", "))
	}

	if targetResource != nil {
		return targetResource, nil
	}
	return nil, errors.Errorf("Unable to find api resource named %q.", key)
}

func typeConfigForTarget(apiResource metav1.APIResource) typeconfig.Interface {
	kind := apiResource.Kind
	pluralName := apiResource.Name
	typeConfig := &federationv2v1alpha1.FederatedTypeConfig{
		// Explicitly including TypeMeta will ensure it will be
		// serialized properly to yaml.
		TypeMeta: metav1.TypeMeta{
			Kind:       "FederatedTypeConfig",
			APIVersion: "core.federation.k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: typeconfig.GroupQualifiedName(apiResource),
		},
		Spec: federationv2v1alpha1.FederatedTypeConfigSpec{
			Target: federationv2v1alpha1.APIResource{
				Version: apiResource.Version,
				Kind:    kind,
			},
			Namespaced:         apiResource.Namespaced,
			PropagationEnabled: true,
			FederatedType: federationv2v1alpha1.APIResource{
				Group:      defaultFederationGroup,
				Version:    defaultFederationVersion,
				Kind:       fmt.Sprintf("Federated%s", kind),
				PluralName: fmt.Sprintf("federated%s", pluralName),
			},
		},
	}

	// Set defaults that would normally be set by the api
	federationv2v1alpha1.SetFederatedTypeConfigDefaults(typeConfig)
	return typeConfig
}

func groupQualifiedName(name, group string) string {
	apiResource := metav1.APIResource{
		Name:  name,
		Group: group,
	}

	return typeconfig.GroupQualifiedName(apiResource)
}

func federatedTypeCRD(typeConfig typeconfig.Interface, accessor schemaAccessor, shortNames []string) *apiextv1b1.CustomResourceDefinition {
	var templateSchema map[string]apiextv1b1.JSONSchemaProps
	// Define the template field for everything but namespaces.
	// A FederatedNamespace uses the containing namespace as the
	// template.
	if typeConfig.GetTarget().Kind != "Namespace" {
		templateSchema = accessor.templateSchema()
	}

	schema := federatedTypeValidationSchema(templateSchema)

	return CrdForAPIResource(typeConfig.GetFederatedType(), schema, shortNames)
}

func CrdForAPIResource(apiResource metav1.APIResource, validation *apiextv1b1.CustomResourceValidation, shortNames []string) *apiextv1b1.CustomResourceDefinition {
	scope := apiextv1b1.ClusterScoped
	if apiResource.Namespaced {
		scope = apiextv1b1.NamespaceScoped
	}
	return &apiextv1b1.CustomResourceDefinition{
		// Explicitly including TypeMeta will ensure it will be
		// serialized properly to yaml.
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: typeconfig.GroupQualifiedName(apiResource),
		},
		Spec: apiextv1b1.CustomResourceDefinitionSpec{
			Group:   apiResource.Group,
			Version: apiResource.Version,
			Scope:   scope,
			Names: apiextv1b1.CustomResourceDefinitionNames{
				Plural:     apiResource.Name,
				Kind:       apiResource.Kind,
				ShortNames: shortNames,
			},
			Validation: validation,
		},
	}
}
