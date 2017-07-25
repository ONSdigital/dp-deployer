awdry
=====

Event handler for Digital Publishing CI

### Configuration

| Environment variable | Default               | Description
| -------------------- | --------------------- | -----------------------------------------
| CONSUMER_QUEUE       |                       | The name of the SQS queue to consume from
| CONSUMER_QUEUE_URL   |                       | The url of the SQS queue to consume from
| DEPLOYMENT_ROOT      |                       | The path to download deployment bundles
| NOMAD_ENDPOINT       | http://localhost:4646 | The endpoint of the Nomad API
| PRODUCER_QUEUE       |                       | The name of the SQS queue to produce to
| AWS_DEFAULT_REGION   |                       | The AWS region the SQS queues reside in

The application also expects your AWS credentials to be configured.

### Deploying

Currently the deployer needs to manually be scheduled as we have a bootstrapping issue

* Tunnel to an instance running Nomad within the target environment from the `ansible` directory in `dp-setup`
  * `ssh -F ssh.cfg -L 4646:localhost:4646 <node-ip>`
* Manully set the `ENV` variable in the `env` stanza to the applicable environment
  * `env { ENV = "bleed" }`
* Plan the tasks
  * `nomad plam awdry.nomad`
* Schedule the tasks
  * `nomad run awdry.nomad`
