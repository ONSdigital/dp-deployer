module github.com/ONSdigital/dp-deployer

go 1.16

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/hashicorp/consul => github.com/hashicorp/consul v1.8.15
	github.com/spf13/viper => github.com/spf13/viper v1.8.1
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.10
	k8s.io/api => k8s.io/api v0.0.0-20190325185214-7544f9db76f6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.1
	k8s.io/apiserver => k8s.io/apiserver v0.22.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.1
	k8s.io/client-go => k8s.io/client-go v1.5.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.1
	k8s.io/code-generator => k8s.io/code-generator v0.22.1
	k8s.io/component-base => k8s.io/component-base v0.22.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.1
	k8s.io/cri-api => k8s.io/cri-api v0.22.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.1
	k8s.io/gengo => k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.20.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.1
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210817084001-7fbd8d59e5b8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.1
	k8s.io/kubectl => k8s.io/kubectl v0.22.1
	k8s.io/kubelet => k8s.io/kubelet v0.22.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.22.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.1
	k8s.io/metrics => k8s.io/metrics v0.22.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.22.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.1
	k8s.io/system-validators => k8s.io/system-validators v1.5.0
	k8s.io/utils => k8s.io/utils v0.0.0-20210820185131-d34e5cb4466e
)

require (
	github.com/ONSdigital/dp-healthcheck v1.1.0
	github.com/ONSdigital/dp-net v1.1.0
	github.com/ONSdigital/dp-nomad v0.3.0
	github.com/ONSdigital/dp-s3 v1.6.0
	github.com/ONSdigital/dp-ssqs v0.0.0-20170720062323-643bf97d9e14
	github.com/ONSdigital/dp-vault v1.1.1
	github.com/ONSdigital/go-ns v0.0.0-20210831102424-ebdecc20fe9e
	github.com/ONSdigital/log.go/v2 v2.0.9
	github.com/aws/aws-sdk-go v1.38.49
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/goamz/goamz v0.0.0-20180131231218-8b901b531db8
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/nomad v1.1.4
	github.com/hashicorp/nomad/api v0.0.0-20210902134234-9ba1a2fba7d6
	github.com/jarcoal/httpmock v1.0.5
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/slimsag/untargz v0.0.0-20160915234413-d9b5a75313e0
	github.com/smartystreets/goconvey v1.6.4
	github.com/vaughan0/go-ini v0.0.0-20130923145212-a98ad7ee00ec // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	k8s.io/kubernetes v1.22.2 // indirect
)
