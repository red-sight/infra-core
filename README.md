# Installation

- Create .env file and fill it with the minimum required variables:

  ```
  APP_NAME=infra
  API_GATEWAY_PORT=8080
  ```

- Build a local KrakenD watch Docker image

- Create a network if not exist:

  ```bash
  docker create network {{APP_NAME}}
  ```

- First start:
  ```
  docker compose up
  ```

# API service registration

The API service should be started as a container in the same docker network and have docker label:

```
infra.enabled=true
```

# Docker labels

- enable\*

  Mandatory

  Available values: 'true'

- name\*

  Service name

  Mandatory

- version

  API service version

  Not mandatory

  Default value: 1

- port

  API server port

  Not mandatory

  Default value: 3000

- openapi_spec_endpoint

  Not mandatory

  Default value: '/openapi'

- health_endpoint

  Not mandatory

  Default value: '/health'
