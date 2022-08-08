module github.com/ONSdigital/dp-deployer

go 1.18

// This is to avoid vulnerability in v0.8.7 coming from github.com/hashicorp/nomad
replace github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.20

replace github.com/dgrijalva/jwt-go v3.2.0+incompatible => github.com/golang-jwt/jwt v3.2.2+incompatible

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.6.6
	github.com/go-ldap/ldap/v3 => github.com/go-ldap/ldap/v3 v3.4.3
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/hashicorp/consul => github.com/hashicorp/consul v1.12.2
	github.com/hashicorp/consul/api => github.com/hashicorp/consul/api v1.13.0
	github.com/hashicorp/consul/sdk => github.com/hashicorp/consul/sdk v0.9.0
	github.com/hashicorp/vault/api => github.com/hashicorp/vault/api v1.7.2
	github.com/spf13/viper => github.com/spf13/viper v1.8.1
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.10
	golang/github.com/opencontainers/runc => golang/github.com/opencontainers/runc v1.1.3
	golang/github.com/pkg/sftp => golang/github.com/pkg/sftp v1.13.5
	golang/github.com/valyala/fasthttp => golang/github.com/valyala/fasthttp v1.37.0
// k8s.io/api => k8s.io/api v0.0.0-20190325185214-7544f9db76f6
// k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.1
// k8s.io/apimachinery => k8s.io/apimachinery v0.22.1
// k8s.io/apiserver => k8s.io/apiserver v0.22.1
// k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.1
// k8s.io/client-go => k8s.io/client-go v1.5.2
// k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.1
// k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.1
// k8s.io/code-generator => k8s.io/code-generator v0.22.1
// k8s.io/component-base => k8s.io/component-base v0.22.1
// k8s.io/component-helpers => k8s.io/component-helpers v0.22.1
// k8s.io/controller-manager => k8s.io/controller-manager v0.22.1
// k8s.io/cri-api => k8s.io/cri-api v0.22.1
// k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.1
// k8s.io/gengo => k8s.io/gengo v0.0.0-20210813121822-485abfe95c7c
// k8s.io/klog/v2 => k8s.io/klog/v2 v2.20.0
// k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.1
// k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.1
// k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210817084001-7fbd8d59e5b8
// k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.1
// k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.1
// k8s.io/kubectl => k8s.io/kubectl v0.22.1
// k8s.io/kubelet => k8s.io/kubelet v0.22.1
// k8s.io/kubernetes => k8s.io/kubernetes v1.22.6
// k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.1
// k8s.io/metrics => k8s.io/metrics v0.22.1
// k8s.io/mount-utils => k8s.io/mount-utils v0.22.1
// k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.22.1
// k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.1
// k8s.io/system-validators => k8s.io/system-validators v1.5.0
// k8s.io/utils => k8s.io/utils v0.0.0-20210820185131-d34e5cb4466e
)

require (
	github.com/ONSdigital/dp-healthcheck v1.3.0
	github.com/ONSdigital/dp-net v1.1.0
	github.com/ONSdigital/dp-nomad v0.3.0
	github.com/ONSdigital/dp-s3 v1.6.0
	github.com/ONSdigital/dp-vault v1.1.1
	github.com/ONSdigital/go-ns v0.0.0-20210831102424-ebdecc20fe9e
	github.com/ONSdigital/goamz v0.0.0-20211118152127-9b03aca7c244
	github.com/ONSdigital/log.go/v2 v2.0.9
	github.com/aws/aws-sdk-go v1.44.22
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/nomad v1.3.1
	github.com/hashicorp/nomad/api v0.0.0-20220407202126-2eba643965c4
	github.com/jarcoal/httpmock v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/slimsag/untargz v0.0.0-20160915234413-d9b5a75313e0
	github.com/smartystreets/goconvey v1.7.2
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/sys v0.0.0-20220517195934-5e4e11fc645e
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
)

require (
	github.com/ONSdigital/dp-api-clients-go v1.34.3 // indirect
	github.com/ONSdigital/s3crypto v0.0.0-20180725145621-f8943119a487 // indirect
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/frankban/quicktest v1.14.0 // indirect
	github.com/go-test/deep v1.0.3 // indirect
	github.com/goamz/goamz v0.0.0-20180131231218-8b901b531db8 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181017120253-0766667cb4d1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/consul/api v1.12.0 // indirect
	github.com/hashicorp/cronexpr v1.1.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.2.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.4.3 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.6 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/go-version v1.4.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-3 // indirect
	github.com/hashicorp/raft v1.3.6 // indirect
	github.com/hashicorp/serf v0.9.7 // indirect
	github.com/hashicorp/vault/api v1.4.1 // indirect
	github.com/hashicorp/vault/sdk v0.5.1 // indirect
	github.com/hashicorp/yamux v0.0.0-20211028200310-0bc27b27de87 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20210113012101-fb4e108d2519 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/justinas/alice v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/miekg/dns v1.1.41 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/vaughan0/go-ini v0.0.0-20130923145212-a98ad7ee00ec // indirect
	go.uber.org/atomic v1.9.0 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220224211638-0e9765cccd65 // indirect
	google.golang.org/genproto v0.0.0-20220314164441-57ef72a4c106 // indirect
	google.golang.org/grpc v1.45.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
