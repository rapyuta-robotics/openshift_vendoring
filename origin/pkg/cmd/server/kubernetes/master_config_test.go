package kubernetes

import (
	"net"
	"reflect"
	"testing"
	"time"

	apiserveroptions "github.com/openshift/kubernetes/cmd/kube-apiserver/app/options"
	cmapp "github.com/openshift/kubernetes/cmd/kube-controller-manager/app/options"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	apiv1 "github.com/openshift/kubernetes/pkg/api/v1"
	"github.com/openshift/kubernetes/pkg/apimachinery/registered"
	"github.com/openshift/kubernetes/pkg/apis/componentconfig"
	extensionsapiv1beta1 "github.com/openshift/kubernetes/pkg/apis/extensions/v1beta1"
	genericapiserveroptions "github.com/openshift/kubernetes/pkg/genericapiserver/options"
	kubeletclient "github.com/openshift/kubernetes/pkg/kubelet/client"
	"github.com/openshift/kubernetes/pkg/storage/storagebackend"
	utilconfig "github.com/openshift/kubernetes/pkg/util/config"
	"github.com/openshift/kubernetes/pkg/util/diff"
	scheduleroptions "github.com/openshift/kubernetes/plugin/cmd/kube-scheduler/app/options"

	configapi "github.com/openshift/origin/pkg/cmd/server/api"
)

func TestAPIServerDefaults(t *testing.T) {
	defaults := apiserveroptions.NewServerRunOptions()

	// This is a snapshot of the default config
	// If the default changes (new fields are added, or default values change), we want to know
	// Once we've reacted to the changes appropriately in BuildKubernetesMasterConfig(), update this expected default to match the new upstream defaults
	expectedDefaults := &apiserveroptions.ServerRunOptions{
		GenericServerRunOptions: &genericapiserveroptions.ServerRunOptions{
			AnonymousAuth:           false,
			BindAddress:             net.ParseIP("0.0.0.0"),
			CertDirectory:           "/var/run/kubernetes",
			InsecureBindAddress:     net.ParseIP("127.0.0.1"),
			InsecurePort:            8080,
			LongRunningRequestRE:    "(/|^)((watch|proxy)(/|$)|(logs?|portforward|exec|attach)/?$)",
			MaxRequestsInFlight:     400,
			SecurePort:              6443,
			EnableProfiling:         true,
			EnableGarbageCollection: true,
			EnableWatchCache:        true,
			MinRequestTimeout:       1800,
			ServiceNodePortRange:    genericapiserveroptions.DefaultServiceNodePortRange,
			RuntimeConfig:           utilconfig.ConfigurationMap{},
			StorageVersions:         registered.AllPreferredGroupVersions(),
			MasterCount:             1,
			DefaultStorageVersions:  registered.AllPreferredGroupVersions(),
			StorageConfig: storagebackend.Config{
				ServerList: nil,
				Prefix:     "/registry",
				DeserializationCacheSize: 0,
			},
			DefaultStorageMediaType:                  "application/json",
			AdmissionControl:                         "AlwaysAdmit",
			AuthorizationMode:                        "AlwaysAllow",
			DeleteCollectionWorkers:                  1,
			MasterServiceNamespace:                   "default",
			AuthorizationWebhookCacheAuthorizedTTL:   5 * time.Minute,
			AuthorizationWebhookCacheUnauthorizedTTL: 30 * time.Second,
		},
		EventTTL: 1 * time.Hour,
		KubeletConfig: kubeletclient.KubeletClientConfig{
			Port: 10250,
			PreferredAddressTypes: []string{
				string(apiv1.NodeHostName),
				string(apiv1.NodeInternalIP),
				string(apiv1.NodeExternalIP),
				string(apiv1.NodeLegacyHostIP),
			},
			EnableHttps: true,
			HTTPTimeout: time.Duration(5) * time.Second,
		},
		WebhookTokenAuthnCacheTTL: 2 * time.Minute,
	}

	if !reflect.DeepEqual(defaults, expectedDefaults) {
		t.Logf("expected defaults, actual defaults: \n%s", diff.ObjectReflectDiff(expectedDefaults, defaults))
		t.Errorf("Got different defaults than expected, adjust in BuildKubernetesMasterConfig and update expectedDefaults")
	}
}

func TestCMServerDefaults(t *testing.T) {
	defaults := cmapp.NewCMServer()

	// This is a snapshot of the default config
	// If the default changes (new fields are added, or default values change), we want to know
	// Once we've reacted to the changes appropriately in BuildKubernetesMasterConfig(), update this expected default to match the new upstream defaults
	expectedDefaults := &cmapp.CMServer{
		KubeControllerManagerConfiguration: componentconfig.KubeControllerManagerConfiguration{
			Port:                              10252, // disabled
			Address:                           "0.0.0.0",
			ConcurrentEndpointSyncs:           5,
			ConcurrentRCSyncs:                 5,
			ConcurrentRSSyncs:                 5,
			ConcurrentDaemonSetSyncs:          2,
			ConcurrentJobSyncs:                5,
			ConcurrentResourceQuotaSyncs:      5,
			ConcurrentDeploymentSyncs:         5,
			ConcurrentNamespaceSyncs:          2,
			ConcurrentSATokenSyncs:            5,
			ConcurrentServiceSyncs:            1,
			ConcurrentGCSyncs:                 20,
			LookupCacheSizeForRC:              4096,
			LookupCacheSizeForRS:              4096,
			LookupCacheSizeForDaemonSet:       1024,
			ConfigureCloudRoutes:              true,
			NodeCIDRMaskSize:                  24,
			ServiceSyncPeriod:                 unversioned.Duration{Duration: 5 * time.Minute},
			ResourceQuotaSyncPeriod:           unversioned.Duration{Duration: 5 * time.Minute},
			NamespaceSyncPeriod:               unversioned.Duration{Duration: 5 * time.Minute},
			PVClaimBinderSyncPeriod:           unversioned.Duration{Duration: 15 * time.Second},
			HorizontalPodAutoscalerSyncPeriod: unversioned.Duration{Duration: 30 * time.Second},
			DeploymentControllerSyncPeriod:    unversioned.Duration{Duration: 30 * time.Second},
			MinResyncPeriod:                   unversioned.Duration{Duration: 12 * time.Hour},
			RegisterRetryCount:                10,
			RouteReconciliationPeriod:         unversioned.Duration{Duration: 10 * time.Second},
			PodEvictionTimeout:                unversioned.Duration{Duration: 5 * time.Minute},
			NodeMonitorGracePeriod:            unversioned.Duration{Duration: 40 * time.Second},
			NodeStartupGracePeriod:            unversioned.Duration{Duration: 60 * time.Second},
			NodeMonitorPeriod:                 unversioned.Duration{Duration: 5 * time.Second},
			ClusterName:                       "kubernetes",
			TerminatedPodGCThreshold:          12500,
			VolumeConfiguration: componentconfig.VolumeConfiguration{
				EnableDynamicProvisioning:  true,
				EnableHostPathProvisioning: false,
				FlexVolumePluginDir:        "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/",
				PersistentVolumeRecyclerConfiguration: componentconfig.PersistentVolumeRecyclerConfiguration{
					MaximumRetry:             3,
					MinimumTimeoutNFS:        300,
					IncrementTimeoutNFS:      30,
					MinimumTimeoutHostPath:   60,
					IncrementTimeoutHostPath: 30,
				},
			},
			ContentType:  "application/vnd.kubernetes.protobuf",
			KubeAPIQPS:   20.0,
			KubeAPIBurst: 30,
			LeaderElection: componentconfig.LeaderElectionConfiguration{
				LeaderElect:   true,
				LeaseDuration: unversioned.Duration{Duration: 15 * time.Second},
				RenewDeadline: unversioned.Duration{Duration: 10 * time.Second},
				RetryPeriod:   unversioned.Duration{Duration: 2 * time.Second},
			},
			ClusterSigningCertFile:            "/etc/kubernetes/ca/ca.pem",
			ClusterSigningKeyFile:             "/etc/kubernetes/ca/ca.key",
			EnableGarbageCollector:            true,
			DisableAttachDetachReconcilerSync: false,
			ReconcilerSyncLoopPeriod:          unversioned.Duration{Duration: 60 * time.Second},
		},
	}

	if !reflect.DeepEqual(defaults, expectedDefaults) {
		t.Logf("expected defaults, actual defaults: \n%s", diff.ObjectReflectDiff(expectedDefaults, defaults))
		t.Errorf("Got different defaults than expected, adjust in BuildKubernetesMasterConfig and update expectedDefaults")
	}
}

func TestSchedulerServerDefaults(t *testing.T) {
	defaults := scheduleroptions.NewSchedulerServer()

	// This is a snapshot of the default config
	// If the default changes (new fields are added, or default values change), we want to know
	// Once we've reacted to the changes appropriately in BuildKubernetesMasterConfig(), update this expected default to match the new upstream defaults
	expectedDefaults := &scheduleroptions.SchedulerServer{
		KubeSchedulerConfiguration: componentconfig.KubeSchedulerConfiguration{
			Port:                           10251, // disabled
			Address:                        "0.0.0.0",
			AlgorithmProvider:              "DefaultProvider",
			ContentType:                    "application/vnd.kubernetes.protobuf",
			KubeAPIQPS:                     50,
			KubeAPIBurst:                   100,
			SchedulerName:                  "default-scheduler",
			HardPodAffinitySymmetricWeight: 1,
			FailureDomains:                 "kubernetes.io/hostname,failure-domain.beta.kubernetes.io/zone,failure-domain.beta.kubernetes.io/region",
			LeaderElection: componentconfig.LeaderElectionConfiguration{
				LeaderElect: true,
				LeaseDuration: unversioned.Duration{
					Duration: 15 * time.Second,
				},
				RenewDeadline: unversioned.Duration{
					Duration: 10 * time.Second,
				},
				RetryPeriod: unversioned.Duration{
					Duration: 2 * time.Second,
				},
			},
		},
	}

	if !reflect.DeepEqual(defaults, expectedDefaults) {
		t.Logf("expected defaults, actual defaults: \n%s", diff.ObjectReflectDiff(expectedDefaults, defaults))
		t.Errorf("Got different defaults than expected, adjust in BuildKubernetesMasterConfig and update expectedDefaults")
	}
}

func TestGetAPIGroupVersionOverrides(t *testing.T) {
	testcases := map[string]struct {
		DisabledVersions         map[string][]string
		ExpectedDisabledVersions []unversioned.GroupVersion
		ExpectedEnabledVersions  []unversioned.GroupVersion
	}{
		"empty": {
			DisabledVersions:         nil,
			ExpectedDisabledVersions: []unversioned.GroupVersion{},
			ExpectedEnabledVersions:  []unversioned.GroupVersion{apiv1.SchemeGroupVersion, extensionsapiv1beta1.SchemeGroupVersion},
		},
		"* -> v1": {
			DisabledVersions:         map[string][]string{"": {"*"}},
			ExpectedDisabledVersions: []unversioned.GroupVersion{apiv1.SchemeGroupVersion},
			ExpectedEnabledVersions:  []unversioned.GroupVersion{extensionsapiv1beta1.SchemeGroupVersion},
		},
		"v1": {
			DisabledVersions:         map[string][]string{"": {"v1"}},
			ExpectedDisabledVersions: []unversioned.GroupVersion{apiv1.SchemeGroupVersion},
			ExpectedEnabledVersions:  []unversioned.GroupVersion{extensionsapiv1beta1.SchemeGroupVersion},
		},
		"* -> v1beta1": {
			DisabledVersions:         map[string][]string{"extensions": {"*"}},
			ExpectedDisabledVersions: []unversioned.GroupVersion{extensionsapiv1beta1.SchemeGroupVersion},
			ExpectedEnabledVersions:  []unversioned.GroupVersion{apiv1.SchemeGroupVersion},
		},
		"extensions/v1beta1": {
			DisabledVersions:         map[string][]string{"extensions": {"v1beta1"}},
			ExpectedDisabledVersions: []unversioned.GroupVersion{extensionsapiv1beta1.SchemeGroupVersion},
			ExpectedEnabledVersions:  []unversioned.GroupVersion{apiv1.SchemeGroupVersion},
		},
	}

	for k, tc := range testcases {
		config := configapi.MasterConfig{KubernetesMasterConfig: &configapi.KubernetesMasterConfig{DisabledAPIGroupVersions: tc.DisabledVersions}}
		overrides := getAPIResourceConfig(config)

		for _, expected := range tc.ExpectedDisabledVersions {
			if overrides.AnyResourcesForVersionEnabled(expected) {
				t.Errorf("%s: Expected %v", k, expected)
			}
		}

		for _, expected := range tc.ExpectedEnabledVersions {
			if !overrides.AllResourcesForVersionEnabled(expected) {
				t.Errorf("%s: Expected %v", k, expected)
			}
		}
	}
}
