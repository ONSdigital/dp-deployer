Deploying apps
==============

In order to deploy an app into an environment using dp-deployer an accompanying .yml file iis needed. 

Example .yml
------------

```yml
---
name: <app-name>
repo_uri: <url for app>
type: nomad-job

nomad:
  service: 
    - name: healthcheck
      enabled: true
      path: /healthcheck
  groups:
    - class: web
      userns_mode: true
      volumes: "/var/babbage/site:/content:ro"
      distinct_hosts: true
      mount: true
      profiles:
        development:
          count: 2
          resources:
            cpu: 500
            memory: 1024
          command_line_args: []
        production:
          count: 2
          resources:
            cpu: 8000
            memory: 2816
          command_line_args: []
    groups:
    - class: publishing
      userns_mode: true
      volumes: "/var/babbage/site:/content:ro"
      distinct_hosts: true
      mount: false
      profiles:
        development:
          count: 2
          resources:
            cpu: 500
            memory: 1024
          command_line_args: []
        production:
          count: 2
          resources:
            cpu: 8000
            memory: 2816
          command_line_args: []
```

The above is an example of all the variables that should go into the .yml file. The variable should only be included in the .yml if it differs from the default. 

Default Values
--------------

| Variable | Default Value | Descriiption |
|----------|---------------| -------------|
| enabled  | true          | Does the app have a healthcheck?|
| path     | /health       | Path for the healthcheck |
| usersns_mode| false | |
| volumes  |  |
| distinct_hosts| false | |
| mount |  |  |
| cmmand_line_args | ./{app-name} | What are the args used by nomad to run the app? |