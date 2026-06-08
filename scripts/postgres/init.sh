#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
  CREATE DATABASE ${INFRA_PG_LOGTO_DB:-logto};
  CREATE DATABASE ${INFRA_PG_CORE_DB:-service_core};
EOSQL
