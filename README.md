dp-deployer
===========

Event handler for Digital Publishing CI

Configuration
-------------

| Environment variable         | Default                | Description
| ---------------------------- | ---------------------- | ---------------------------------------------
| CONSUMER_QUEUE               |                        | The name of the SQS queue to consume from
| CONSUMER_QUEUE_URL           |                        | The url of the SQS queue to consume from
| DEPLOYMENT_ROOT              |                        | The path to download deployment bundles
| NOMAD_CA_CERT                |                        | The path to the CA cert file
| NOMAD_ENDPOINT               | http://localhost:4646  | The endpoint of the Nomad API
| NOMAD_TLS_SKIP_VERIFY        | false                  | When using TLS to nomad, skip checking certs (bool)
| NOMAD_TOKEN                  |                        | The ACL token used to authorise HTTP requests
| PRIVATE_KEY                  |                        | Private key for decrypting secrets
| PRODUCER_QUEUE               |                        | The name of the SQS queue to produce to
| VERIFICATION_KEY             |                        | Public key for verifying SQS messages
| AWS_DEFAULT_REGION           |                        | The AWS region the SQS queues reside in
| VAULT_ADDR                   | https://127.0.0.1:8200 | Vault endpoint URL
| HEALTHCHECK_INTERVAL         | 10s                    | The time between calling healthcheck endpoints for check subsystems
| HEALTHCHECK_CRITICAL_TIMEOUT | 60s                    | The time taken for the health changes from warning state to critical due to subsystem check failures
| HEALTHCHECK_PORT             | :24200                 | The port for the healthcheck

The application also expects your AWS credentials to be configured.

### Healthcheck

 The `/health` endpoint returns the current status of the service. Dependent services are health checked on an interval defined by the `HEALTHCHECK_INTERVAL` environment variable.

On a development machine a request to the health check endpoint can be made by:

`curl localhost:24200/health`

### Licence

Copyright © 2019, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.
