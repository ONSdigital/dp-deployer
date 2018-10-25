dp-deployer
===========

Event handler for Digital Publishing CI

Configuration
-------------

| Environment variable | Default               | Description
| -------------------- | --------------------- | ---------------------------------------------
| CONSUMER_QUEUE       |                       | The name of the SQS queue to consume from
| CONSUMER_QUEUE_URL   |                       | The url of the SQS queue to consume from
| DEPLOYMENT_ROOT      |                       | The path to download deployment bundles
| NOMAD_ENDPOINT       | http://localhost:4646 | The endpoint of the Nomad API
| NOMAD_TOKEN          |                       | The ACL token used to authorise HTTP requests
| PRIVATE_KEY_PATH     |                       | The path on the filesystem to the private key
| PRODUCER_QUEUE       |                       | The name of the SQS queue to produce to
| AWS_DEFAULT_REGION   |                       | The AWS region the SQS queues reside in

The application also expects your AWS credentials to be configured.

### Licence

Copyright © 2017, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.
