# KrakenD config

Routes are **generated**, not hand-maintained — the source of truth is each
service's OpenAPI spec, aggregated by the registrator.

- **`krakend.bootstrap.json`** (committed) — stable top-level settings (logging,
  timeout, cache) with an **empty `endpoints` array**. The config KrakenD boots on
  before routes are delivered.
- **`krakend.json`** (gitignored, generated) — the live config with routes. Never
  committed.

How it's served:

- **Dev** (registrator `auto` mode): KrakenD seeds `krakend.json` from the bootstrap
  on first boot (see the `command` in `docker-compose.override.yml`), then the
  registrator discovers services, regenerates `krakend.json` on the bind mount, and
  restarts KrakenD.
- **Swarm/prod** (registrator `artifact` mode): the bootstrap is delivered as the
  `krakend_bootstrap` Swarm config object (`docker-compose.swarm.yml`); the
  registrator then delivers a generated `krakend-config-<hash>` object and rolls the
  service.
