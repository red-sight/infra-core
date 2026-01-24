# Infra

The project is in the development state, do not use this in the production.

## Installation

- Create .env file and fill it with the minimum required variables (local dev example below):

  ```
  API_GATEWAY_PORT=8080
  APP_NAME=infra

  REALM=demo
  CLIENT_ID=traefik-forward-auth
  CLIENT_SECRET=mDdKXKOO8FVOiuJpcZzTJfWxt8MHbuG6
  OIDC_PLUGIN_SECRET=asjhu76215hgjfJHFUFJRUYHhgvjnhjJ

  # Keycloak demo realm config
  DEMO_REALM_SECRET=mDdKXKOO8FVOiuJpcZzTJfWxt8MHbuG6
  DEMO_REALM_TEST_USER_LOGIN=testuser
  DEMO_REALM_TEST_USER_EMAIL=testuser@test.com
  DEMO_REALM_TEST_USER_PASSWORD=password
  DEMO_REALM_CLIENT_ID=fe

  #Keycloak config
  KC_BOOTSTRAP_ADMIN_USERNAME=admin
  KC_BOOTSTRAP_ADMIN_PASSWORD=admin
  KC_SECRET=somerandomsecret
  KC_HEALTH_ENABLED=true
  KC_PROTOCOL=http
  KC_HOSTNAME=keycloak.localhost
  KC_PROXY=edge
  KC_HTTP_ENABLED=true

  FRONTEND_HOST=localhost
  FRONTEND_PROTOCOL=http

  #Postgres
  POSTGRES_USER=postgres
  POSTGRES_PASSWORD=postgres
  POSTGRES_DB=keycloak
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

## API service registration

The API service should be started as a container in the same docker network and have docker label:

```
infra.enabled=true
```

## Docker labels

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
