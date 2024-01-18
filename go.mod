module github.com/ONSdigital/dp-deployer

go 1.21

// This, to fix: [CVE-2022-23471] CWE-400: Uncontrolled Resource Consumption ('Resource Exhaustion')
replace github.com/containerd/containerd => github.com/containerd/containerd v1.6.23

// This to fix: [CVE-2023-28642] CWE-59: Improper Link Resolution Before File Access ('Link Following')
replace github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.5

// This, to fix: [CVE-2023-0475] CWE-Other
replace github.com/hashicorp/go-getter => github.com/hashicorp/go-getter v1.7.2

// This, to fix: [CVE-2021-37219] CWE-295: Improper Certificate Validation
replace github.com/hashicorp/consul => github.com/hashicorp/consul v1.16.1

require (
	github.com/ONSdigital/dp-healthcheck v1.6.0
	github.com/ONSdigital/dp-net/v2 v2.11.0
	github.com/ONSdigital/dp-nomad v0.4.0
	github.com/ONSdigital/dp-s3 v1.6.0
	github.com/ONSdigital/dp-vault v1.3.0
	github.com/ONSdigital/go-ns v0.0.0-20210831102424-ebdecc20fe9e
	github.com/ONSdigital/goamz v0.0.0-20211118152127-9b03aca7c244
	github.com/ONSdigital/log.go/v2 v2.4.0
	github.com/aws/aws-sdk-go v1.44.289
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/nomad v0.12.10
	github.com/hashicorp/nomad/api v0.0.0-20210128170732-239be4e591fe
	github.com/jarcoal/httpmock v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/slimsag/untargz v0.0.0-20160915234413-d9b5a75313e0
	github.com/smartystreets/goconvey v1.8.0
	golang.org/x/crypto v0.18.0
)

require (
	github.com/ONSdigital/dp-api-clients-go/v2 v2.252.0 // indirect
	github.com/ONSdigital/s3crypto v0.0.0-20180725145621-f8943119a487 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/fatih/color v1.15.0 // indirect
	github.com/goamz/goamz v0.0.0-20180131231218-8b901b531db8 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/consul/api v1.24.0 // indirect
	github.com/hashicorp/cronexpr v1.1.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.4.10 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.2 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.7 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-3 // indirect
	github.com/hashicorp/raft v1.5.0 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/hashicorp/vault/api v1.9.1 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/justinas/alice v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/miekg/dns v1.1.50 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.2-0.20210821155943-2d9075ca8770 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/shoenig/test v0.6.0 // indirect
	github.com/smartystreets/assertions v1.13.1 // indirect
	github.com/vaughan0/go-ini v0.0.0-20130923145212-a98ad7ee00ec // indirect
	golang.org/x/exp v0.0.0-20230321023759-10a507213a29 // indirect
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.9.1 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230525234030-28d5490b6b19 // indirect
	google.golang.org/grpc v1.57.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
)
