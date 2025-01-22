# dp-deployer

Event handler for Digital Publishing CI

## Configuration

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
| AWS_REGION                   | eu-west-1              | The AWS region used
| VAULT_ADDR                   | https://127.0.0.1:8200 | Vault endpoint URL
| HEALTHCHECK_INTERVAL         | 10s                    | The time between calling healthcheck endpoints for check subsystems
| HEALTHCHECK_CRITICAL_TIMEOUT | 60s                    | The time taken for the health changes from warning state to critical due to subsystem check failures
| BIND_ADDR                    | :24300                 | The listen address to bind to
| DEPLOYMENT_TIMEOUT           | 20m                    | The max time to wait for a deployment to complete
| CONSUMER_QUEUE_NEW           |                        | The name of the new SQS queue to consume from
| CONSUMER_QUEUE_URL_NEW       |                        | The url of the new SQS queue to consume from

The application also expects your AWS credentials to be configured.

### Healthcheck

 The `/health` endpoint returns the current status of the service. Dependent services are health checked on an interval defined by the `HEALTHCHECK_INTERVAL` environment variable.

On a development machine a request to the health check endpoint can be made by:

`curl localhost:24300/health`

### How to test the deployer in the environment

There are various ways to test the deployer code. The [dp-operations guide](https://github.com/ONSdigital/dp-operations/blob/main/guides/deploying-the-deployer.md) gives you a brief introduction about the deployer and an overview about how to deploy it.

This section shows you how to test the deployer code changes in the environment and how to rollback to the previous version by just reverting the `dp_deployer_version` in `dp-setup`  and running the `ansible-playbook` command for easy deployment.

1. Update the deployer code and update the tests as per requirement.
2. Run `make test` and `make build` to check if your code is ready for testing
3. Start colima by running the command `colima start`.
4. Prepare ECR authentication by running `make prep-ecr`.
5. Run `make deployment` and this should build an image for your new updated code, push the image to `ECR` and bundle it to s3.
    **Note:** The tar bundle which includes a nomad plan can be seen in s3 which is always under `production/` no matter which environment ansible is targetting. The nomad plan points to the ECR image.
6. Go to `dp-setup` and check you are in the right environment to run ansible. It is recommended you stick with `sandbox` for testing. Amend the `dp_deployer_version` from the output of the `make deployment` command.

    ```bash
    vim +/dp_deployer_version dp-setup/ansible/roles/bootstrap-deployer/defaults/main.yml
    ```

7. After updating the `dp_deployer_version`, run the ansible-playbook command to bootstrap the deployer.

    ```bash
    export ONS_DP_ENV = sandbox
    ansible-playbook --vault-id=$(ONS_DP_ENV)@.$(ONS_DP_ENV).pass -i inventories/$(ONS_DP_ENV) bootstrap-deployer.yml
    ```

8. Check [nomad-ui](https://nomad.dp.aws.onsdigital.uk/ui/jobs/dp-deployer/versions) if the deployer has been deployed successfully.
9. Go to [concourse-ui](https://concourse.dp-ci.aws.onsdigital.uk/) and deploy the `dp-import-reporter` and then trigger `<env>-ship-it` to test the deployer code.
10. If the previous step has been successful, trigger the `secrets` pipeline to confirm that it is working as expected.
11. If it hasn't been successful, rollback to the previous version of the deployer, by reverting the `dp_deployer-version` in `dp-setup` as mentioned in step 6 and then re-apply the `bootstrap-deployer` playbook command as shown in step 7.

### Licence

Copyright © 2025, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.
