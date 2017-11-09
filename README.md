awdry
=====

Event handler for Digital Publishing CI

### Configuration

| Environment variable | Default               | Description
| -------------------- | --------------------- | ---------------------------------------------
| CONSUMER_QUEUE       |                       | The name of the SQS queue to consume from
| CONSUMER_QUEUE_URL   |                       | The url of the SQS queue to consume from
| DEPLOYMENT_ROOT      |                       | The path to download deployment bundles
| NOMAD_ENDPOINT       | http://localhost:4646 | The endpoint of the Nomad API
| PRIVATE_KEY_PATH     |                       | The path on the filesystem to the private key
| PRODUCER_QUEUE       |                       | The name of the SQS queue to produce to
| AWS_DEFAULT_REGION   |                       | The AWS region the SQS queues reside in

The application also expects your AWS credentials to be configured.

### Deploying

Currently the deployer needs to manually be scheduled as we have a bootstrapping issue

* Tunnel to an instance running Nomad within the target environment from the `ansible` directory in `dp-setup`
  * `ssh -F ssh.cfg -L 4646:localhost:4646 -l <your user> <nomad client or server ip>`
* Plan the tasks
  * `nomad plan -address=https://localhost:4646 -tls-skip-verify awdry.nomad`
* Schedule the tasks
  * `nomad run -address=https://localhost:4646 -tls-skip-verify awdry.nomad`
