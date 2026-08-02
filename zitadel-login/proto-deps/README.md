# Vendored dependency protos

`proto/` (ZITADEL's own protos, copied verbatim from the upstream tag) imports four
third-party definitions. Upstream resolves them from the Buf Schema Registry via
`proto/buf.lock`. We vendor them instead so proto codegen is hermetic: no BSR
account, no network, and a build that cannot break because a remote registry
changed its access policy.

Extracted from the exact BSR commits pinned by upstream's `proto/buf.lock` at
ZITADEL `v4.15.2`:

| Module | Commit | Files |
|---|---|---|
| `buf.build/googleapis/googleapis` | `75b4300737fb4efca0831636be94e517` | `google/api/annotations.proto`, `google/api/http.proto`, `google/api/field_behavior.proto` |
| `buf.build/grpc-ecosystem/grpc-gateway` | `a1ecdc58eccd49aa8bea2a7a9022dc27` | `protoc-gen-openapiv2/options/annotations.proto`, `protoc-gen-openapiv2/options/openapiv2.proto` |
| `buf.build/envoyproxy/protoc-gen-validate` | `6607b10f00ed4a3d98f906807131c44a` | `validate/validate.proto` |

Only the transitive closure of what `proto/` actually imports is vendored;
`google/protobuf/*` are well-known types built into the Buf CLI. Licenses
(Apache-2.0 / BSD-3-Clause) are in the file headers, unmodified.

## Re-syncing

When bumping the vendored ZITADEL tag, diff the new `proto/buf.lock` against the
commits above. If any changed, re-extract the corresponding files from a machine
with BSR access (`buf export buf.build/<owner>/<repo>:<commit> -o <dir>`) and
update this table. If the new `proto/` imports something not listed here, add it
together with its own transitive imports — the build fails loudly otherwise.
