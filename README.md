awdry
=====

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

Deploying awdry
---------------

Currently the deployer needs to manually be scheduled as we have a bootstrapping issue.

**NB:** Before continuing ensure you [vault](https://www.vaultproject.io) and [nomad](https://www.nomadproject.io) installed locally with the same versions specified in the `dp-setup` repo.

1. If awdry's secrets file (`../secrets/<environment name>/awdry.json.asc`) does not contain the nomad acl token you need to [get the nomad acl token generated for awdry using ansible](https://github.com/ONSdigital/dp-setup/blob/develop/ansible/README.md#setup-nomad)

2. [Write awdry secrets to vault](#writing-awdry-secrets-to-vault)

3. [Deploy awdry tasks to nomad](#scheduling-awdry-tasks-to-nomad)

Writing awdry secrets to vault
------------------------------

1. In one terminal, change to the `ansible` directory in the [dp-setup repo](https://github.com/ONSdigital/dp-setup) and tunnel to a nomad server instance on port `8200`:
   ```
   ssh -F ssh.cfg -L 8200:localhost:8200 <your user>@<nomad server ip>
   ```

2. In a second terminal, decrypt and write awdry secrets to vault:
   ```
   cd ../secrets/<environment name>
   gpg -d awdry.json.asc > awdry.json
   VAULT_ADDR=http://localhost:8200 VAULT_TOKEN=<root vault token> vault write secret/awdry @awdry.json
   ```

Scheduling awdry on nomad
-------------------------

1. In one terminal, change to the `ansible` directory in the [dp-setup repo](https://github.com/ONSdigital/dp-setup) and tunnel to a nomad server instance on port `4646`:
   ```
   ssh -F ssh.cfg -L 4646:localhost:4646 <your user>@<nomad server ip>
   ```

2. In a second terminal, plan the tasks (from this directory):
   ```
   NOMAD_TOKEN=<nomad acl token> nomad plan -address=https://localhost:4646 -tls-skip-verify awdry.nomad
   ```

3. Schedule the tasks:
   ```
   NOMAD_TOKEN=<nomad acl token> nomad run -address=https://localhost:4646 -tls-skip-verify awdry.nomad
   ```

4. Verify that awdry successfully started:
   ```
   NOMAD_TOKEN=<nomad acl token> nomad status -address=https://localhost:4646 -tls-skip-verify awdry
   ```
