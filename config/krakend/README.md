# KrakenD config

`krakend.json` here is a **bootstrap**, not a hand-maintained gateway config. It
holds the stable top-level settings (logging, timeout, cache) with an **empty
`endpoints` array**. The registrator owns the routes:

- **Dev** (registrator `auto` mode): the registrator discovers services, aggregates
  their OpenAPI specs, regenerates this file on the bind mount, and restarts KrakenD.
  So this file will be overwritten locally with the live routes while the stack runs.
- **Swarm/prod** (registrator `artifact` mode): KrakenD boots on this file delivered
  as the `krakend_bootstrap` Swarm config object (see `docker-compose.swarm.yml`),
  then the registrator delivers a generated `krakend-config-<hash>` config object and
  rolls the service.

**Do not commit the regenerated (route-filled) version** — only the empty-endpoints
bootstrap is tracked. The generated routes are derived state, not source of truth
(the source is each service's OpenAPI spec). To stop the dev stack from showing this
file as dirty locally:

```sh
git update-index --skip-worktree config/krakend/krakend.json
# undo with: git update-index --no-skip-worktree config/krakend/krakend.json
```
