package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	federationv2v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	multiclusterdns "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	hivev1aplha1 "github.com/openshift/hive/pkg/apis/hive/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/apis"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/multiplenamespacefederation"
	"github.com/raffaelespazzoli/federation-lifecycle-operator/pkg/controller/namespacefederation"
	"github.com/spf13/pflag"
	crdinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()

	initializeTemplates()

	// Become the leader before proceeding
	err = leader.Become(ctx, "federation-lifecycle-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := federationv2v1alpha1.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := multiclusterdns.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := hivev1aplha1.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := clusterregistry.SchemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	crdinstall.Install(mgr.GetScheme())

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create Service object to expose the metrics port.
	_, err = metrics.ExposeMetricsPort(ctx, metricsPort)
	if err != nil {
		log.Info(err.Error())
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func initializeTemplates() {
	// initialize the templates
	federationControllerTemplateFileName, found := os.LookupEnv("FEDERATION_CONTROLLER_TEMPLATE")
	if !found {
		log.Info("Error: FEDERATION_CONTROLLER_TEMPLATE must be set")
		os.Exit(1)
	}

	err := namespacefederation.InitializeFederationControlPlaneTemplates(federationControllerTemplateFileName)

	if err != nil {
		log.Error(err, "Unable to initialize federation control plane template")
		os.Exit(1)
	}

	remoteFederatedClusterTemplateFileName, found := os.LookupEnv("REMOTE_FEDERATED_CLUSTER_TEMPLATE")
	if !found {
		log.Info("Error: REMOTE_FEDERATED_CLUSTER_TEMPLATE must be set")
		os.Exit(1)
	}
	federatedClusterTemplateFileName, found := os.LookupEnv("FEDERATED_CLUSTER_TEMPLATE")
	if !found {
		log.Info("Error: FEDERATED_CLUSTER_TEMPLATE must be set")
		os.Exit(1)
	}

	err = namespacefederation.InitializeFederatedClusterTemplates(federatedClusterTemplateFileName, remoteFederatedClusterTemplateFileName)
	if err != nil {
		log.Error(err, "Unable to initialize federated cluster templates")
		os.Exit(1)
	}

	federatedTypesTemplateFileName, found := os.LookupEnv("FEDERATED_TYPES_TEMPLATE")
	if !found {
		log.Info("Error: FEDERATED_TYPES_TEMPLATE must be set")
		os.Exit(1)
	}

	err = namespacefederation.InitializeFederatedTypesTemplates(federatedTypesTemplateFileName)
	if err != nil {
		log.Error(err, "Unable to initialize federated types template")
		os.Exit(1)
	}

	cloudProviderGlobalLodaBalancerTemplateFileName, found := os.LookupEnv("CLOUD_PROVIDER_GLOBALLOADBALANCER_TEMPLATE")
	if !found {
		log.Info("Error: CLOUD_PROVIDER_GLOBALLOADBALANCER_TEMPLATE must be set")
		os.Exit(1)
	}

	err = multiplenamespacefederation.InitializeCloudProviderGlobalLoadBalancerTemplate(cloudProviderGlobalLodaBalancerTemplateFileName)
	if err != nil {
		log.Error(err, "Unable to initialize cloud provider global load balancer template")
		os.Exit(1)
	}

	selfHostedGlobalLodaBalancerTemplateFileName, found := os.LookupEnv("SELF_HOSTED_GLOBALLOADBALANCER_TEMPLATE")
	if !found {
		log.Info("Error: SELF_HOSTED_GLOBALLOADBALANCER_TEMPLATE must be set")
		os.Exit(1)
	}

	err = multiplenamespacefederation.InitializeRemoteGlobaLoadBalancerTemplate(selfHostedGlobalLodaBalancerTemplateFileName)
	if err != nil {
		log.Error(err, "Unable to initialize self-hosted global load balancer template")
		os.Exit(1)
	}

	serviceAccountGlobalLodaBalancerTemplateFileName, found := os.LookupEnv("SERVICE_ACCOUNT_GLOBALLOADBALANCER_TEMPLATE")
	if !found {
		log.Info("Error: SERVICE_ACCOUNT_GLOBALLOADBALANCER_TEMPLATE must be set")
		os.Exit(1)
	}

	err = multiplenamespacefederation.InitializeLocalLoadBalancerServiceAccountTemplate(serviceAccountGlobalLodaBalancerTemplateFileName)
	if err != nil {
		log.Error(err, "Unable to initialize local service account global load balancer template")
		os.Exit(1)
	}

	log.Info("Templates initialized correctly")

}
