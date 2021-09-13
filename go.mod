module github.com/ONSdigital/dp-deployer

go 1.13

replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2

require (
	github.com/ONSdigital/dp-healthcheck v1.0.4
	github.com/ONSdigital/dp-net v1.0.2
	github.com/ONSdigital/dp-nomad v0.2.0
	github.com/ONSdigital/dp-s3 v1.5.0
	github.com/ONSdigital/dp-ssqs v0.0.0-20170720062323-643bf97d9e14
	github.com/ONSdigital/dp-vault v1.1.1
	github.com/ONSdigital/go-ns v0.0.0-20200205115900-a11716f93bad
	github.com/ONSdigital/log.go v1.0.0
	github.com/aws/aws-sdk-go v1.38.20
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/goamz/goamz v0.0.0-20180131231218-8b901b531db8
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/nomad v1.1.4
	github.com/hashicorp/nomad/api v0.0.0-20210910134105-b2b9013e524c
	github.com/jarcoal/httpmock v1.0.5
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/slimsag/untargz v0.0.0-20160915234413-d9b5a75313e0
	github.com/smartystreets/goconvey v1.6.4
	github.com/vaughan0/go-ini v0.0.0-20130923145212-a98ad7ee00ec // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
)
