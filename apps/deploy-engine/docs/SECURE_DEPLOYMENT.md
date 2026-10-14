# Secure Deployment Reference

**Last updated: 2026-10-12**

This document is for Bluelink maintainers to provide guidance for securely deploying the engine in a production environment.

Every configuration key below exists today.
What the engine does not currently protect is stated in
[Current limitations](#current-limitations) rather than left implied.

---

## 1. Deployment topology

The engine does not terminate TLS on its own (`cmd/main.go` serves plain HTTP), so how it is
reached differs by topology. There are three, and they want different things.

### Binding the engine

| Key                | Default       | Notes                                                                                                |
| ------------------ | ------------- | ---------------------------------------------------------------------------------------------------- |
| `loopback_only`    | `true`        | Secure default. Binds to loopback so only same-host traffic reaches the engine.                      |
| `port`             | `8325`        | TCP port when not using a unix socket.                                                               |
| `use_unix_socket`  | `false`       | Serve the API on a unix domain socket instead of TCP.                                                |
| `unix_socket_path` | per-user path | `$HOME/.bluelink/engine/api.sock`, or `%LOCALAPPDATA%\NewStack\Bluelink\engine\api.sock` on Windows. |

### Served deployment, terminator on a separate host

The common case is a managed load balancer or gateway terminating TLS such as Amazon's ALB/NLB, Google's Cloud Load
Balancing, or Azure's Application Gateway. Behind each of these services, the engine runs on its own host or container.

A unix socket is not available here as the hop crosses the network. Set `loopback_only: false`,
which exposes the engine to its subnet, and secure the hop with network controls:

- Security group or firewall rules admitting only the terminator.
- A private subnet with no public route to the engine.
- Service-mesh mTLS on the terminator-to-engine hop where the platform provides it.

This is an accepted pattern for **SOC 2 CC6.7** and **ISO 27001 A.8.20**. The audit artifact
is the gateway's TLS configuration including the certificate, minimum protocol version, cipher policy in addition to the network rules restricting who can reach the engine. The engine is not responsible for either.

### Served deployment, reverse proxy on the same host

A single-VM or single-container deployment with nginx or Caddy alongside the engine. Keep
`loopback_only: true`, which binds the engine to `127.0.0.1`, and have the proxy reach it
over loopback or a unix socket.

**Where that proxy is public-facing, this is the weaker of the two served topologies.** The
loopback hop is fine, however, the problem is that the engine host becomes the internet-facing host.
The unauthenticated plugin service also listens on loopback, which an HTTP proxy cannot
reach but any local process can, so it is the one asset that running the proxy and the engine
as different users does not protect (see [Plugin transport](#5-plugin-transport)). State and
credentials are also on that host, though both are mitigable by encrypting the state with
`state.encryption.enabled` or the volume holding it, and inject credentials from a secret
manager rather than writing them to a config file.

Prefer separating the terminator from the engine host. Where they must be co-located, proxy
only the engine's own paths to the engine's own address, and never let a request choose its
upstream.

### Local use and CI/CD

A developer running the engine locally, or an engine running inside a CI job alongside the
CLI. There is no terminator and no network hop as the client is on the same host.

**Use a unix socket.** Set `use_unix_socket: true`. The engine creates the containing
directory restricted to the owner, which is what restricts who can reach the socket, and sets
the socket itself to `0600` on unix, where `net.Listen` would otherwise apply the umask.
Startup refuses to remove a path that exists and is not a socket, and refuses to start if
another engine is already listening.

**Keep the socket in a per-user directory.** The defaults are
`$HOME/.bluelink/engine/api.sock` and
`%LOCALAPPDATA%\NewStack\Bluelink\engine\api.sock`. If you override the path, keep it
under a directory your user owns rather than a world-writable one such as `/tmp`, where
another local user could squat the path before the engine starts, this is mostly relevant when using a shared
CI runner. A unix socket path is also limited to roughly 104 bytes, and an over-long one
fails with a bare `bind: invalid argument`.

Do not expose the engine directly to the internet. It has no TLS, and its authentication
mechanisms assume a confidential channel.

Unix sockets work on Windows too, and are still the right choice there. The socket inherits
the ACL of its containing directory, and `bind` and `connect` are checked against it, so the
default path under `%LOCALAPPDATA%` restricts the socket to the current user.

## 2. Authentication

The engine **fails at startup when no authentication mechanism is configured**, so
there is no accidental unauthenticated deployment. Three mechanisms are supported in any combination:

| Mechanism           | Keys                                                                                                                                                                                |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| OAuth2/OIDC JWT     | `auth.oauth2_oidc_jwt_issuer`, `auth.oauth2_oidc_jwt_issuer_secure` (default `true`), `auth.oauth2_oidc_jwt_audience`, `auth.oauth2_oidc_jwt_signature_algorithm` (default `HS256`) |
| Bluelink SigV1 HMAC | `auth.bluelink_signature_v1_key_pairs` (map of key ID → secret)                                                                                                                     |
| API keys            | `auth.bluelink_api_keys` (list)                                                                                                                                                     |

Notes for compliance deployments:

- Keep `auth.oauth2_oidc_jwt_issuer_secure` at `true`. It is what requires the issuer to be
  reachable over HTTPS.
- API keys and SigV1 secrets are **plaintext in configuration and environment** today. Treat
  the engine's config file and environment as credential material, restrict file permissions,
  keep it out of version control, and prefer environment injection from a secret manager over
  a file on disk.
- **Do not let the terminating gateway log auth request headers.** All three mechanisms expect the credentials to be sent in the request headers, so access logs that capture them are credential material. What a captured
  header gives an attacker differs for each, an API key is reusable until it is rotated, a JWT until
  it expires, and a SigV1 signature only replays the one request it was computed for, within
  the ±300s window. Keep identity-provider token lifetimes short for the same reason where possible.

## 3. Encryption at rest

The engine persists blueprint instance state, resource and link state, drift records,
changesets, validation results, reconciliation results and event partitions. Resolved secrets
are among them whether it be a connection string in a resource spec, a token in a changeset, a credential
in a persisted event.

For `postgres` and `objectstore`, at-rest protection comes from the storage layer beneath, and
what that means depends on the provider. For `memfile` there is no storage layer beneath, so
the engine can encrypt the state itself before writing it.

> `state.storage_engine` defaults to **`memfile`**, and `state.encryption.enabled` defaults to
> **`false`**. A deployment that sets neither writes plaintext state to the host's disk. See
> [`memfile`](#memfile) below.

### `postgres`

| Provider                                 | Encryption at rest                                                                                                                                                                   |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Cloud SQL, Azure Database for PostgreSQL | On by default.                                                                                                                                                                       |
| Amazon RDS                               | **Opt-in at instance creation, and not changeable in place afterwards.** Converting an unencrypted instance requires a snapshot restore. Verify `storage_encrypted` on the instance. |
| Self-managed                             | Your responsibility: LUKS/dm-crypt on the data volume, plus encrypted backups (`pgbackrest` repo encryption).                                                                        |

Audit artifact: the provider's database configuration showing encryption enabled, plus the
KMS/CMEK key reference where one is used.

### `objectstore`

| Store                           | Encryption at rest                                                                                        |
| ------------------------------- | --------------------------------------------------------------------------------------------------------- |
| S3                              | On by default for new objects since January 2023 (SSE-S3). Upgrade to SSE-KMS for a customer-managed key. |
| Azure Blob Storage              | Always on and cannot be disabled.                                                                         |
| Google Cloud Storage            | Always on. Upgrade to CMEK for a customer-managed key.                                                    |
| Self-hosted (MinIO and similar) | Supported but **opt-in**, you will need to configure SSE explicitly.                                      |

Audit artifact: the bucket's default-encryption configuration.

Relevant key: `state.objectstore.prefix` (default `bluelink-state/`) - the key prefix all
state objects are written under, useful for scoping a bucket policy.

### `memfile`

Supported for **single-instance production deployments**. State is plaintext JSON under
`state.memfile_state_dir`, chunked according to `state.memfile_max_guide_file_size` and
`state.memfile_max_event_partition_size`.

There is no storage layer beneath, so there are two places to protect it: the engine, and the
host.

**At the engine.** Setting `state.encryption.enabled` makes the engine encrypt every object it
writes to the state directory. Each object gets its own data key, sealed with AES-256-GCM and
bound to the key it is stored under, and that data key is wrapped by a key-encryption key the
`state.encryption.provider` selects:

| Provider        | Key comes from                                                                              | Configuration                                                                                 |
| --------------- | ------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `passphrase`    | Argon2id over a passphrase, with a salt generated once and persisted in the state directory | `state.encryption.passphrase` or `state.encryption.passphrase_file`                           |
| `localkey`      | A raw 32-byte key in a file, which must not be readable by other users                      | `state.encryption.local_key_file`                                                             |
| `awskms`        | AWS KMS, the key never leaving KMS                                                          | `state.encryption.awskms.key_id`, needing `kms:GenerateDataKey` and `kms:Decrypt`             |
| `gcpkms`        | Google Cloud KMS, the key never leaving KMS                                                 | `state.encryption.gcpkms.key_name`, needing `useToEncrypt` and `useToDecrypt`                 |
| `azurekeyvault` | Azure Key Vault or Managed HSM, the key never leaving the vault                             | `state.encryption.azurekeyvault.vault_url` and `.key_name`, needing `wrapKey` and `unwrapKey` |

The KMS providers are the stronger choice where one is available as the key is held by a service
with its own access control and audit trail rather than by the engine's host, and losing the
host does not lose the key. They cost a wrap call per object written.

What to know before enabling it:

- **Losing the key loses the state.** Keep it wherever the deployment keeps its other
  long-lived secrets, and back it up accordingly.
- **Enabling it over an existing plaintext state directory fails at startup** rather than
  reading it. Encrypt what is already there with `bluelink-deploy-engine reencrypt-state`, with
  the engine stopped.
- **Storage keys are not encrypted.** File names carry categories and entity ids, which the
  index and the loader need, so they stay readable.
- **For `azurekeyvault`, pin `key_version`.** A wrapped key does not name the version that
  produced it, so rotating an unpinned key leaves state sealed under the previous version
  unreadable.
- Encryption protects what is at rest, not what is in memory or what the API returns. Secrets
  are decrypted to be used.

**Changing the key** is two steps, because state already written is sealed with the old one:

1. Configure the new key under `state.encryption`, and list the old one under
   `state.encryption.previous_keys`. The engine then writes with the new key and reads either,
   so this step costs no more than a restart.
2. Stop the engine and run `bluelink-deploy-engine reencrypt-state`, which copies the state
   directory and then re-seals every object under the new key. `--dry-run` reports what it
   would change without writing. Once nothing is left under the old key, drop it from
   `previous_keys` and retire it.

Until step 2 has run, the old key is still the only thing protecting most of the state, so a
key that has been exposed is not contained by step 1 alone. The same command decrypts a state
directory with `--decrypt`, so enabling encryption is not a one-way door.

A pass is not atomic: objects are rewritten in place, so a failure part-way through leaves a
directory holding a mix of keys, readable only by a configuration listing both. The command
copies the state directory first for that reason, and an interrupted pass can be run again
since objects already under the new key are recognised and left alone.

**At the host**, whether or not engine encryption is on:

- Put the state directory on an **encrypted volume** (LUKS/dm-crypt, EBS encryption,
  FileVault). The engine cannot verify this and does not claim to.
- The engine creates the state directory `0700` and writes state files `0600`, and tightens
  the directory if an earlier version left it open, so the state is private to the user the
  engine runs as. Files written by an earlier version keep their modes, which the directory
  makes moot: reaching them means traversing it.
- Encrypt backups separately with tools such as `restic` or `age`. A backup of an encrypted
  state directory is only readable with the key, which is one more reason the key has to
  outlive the host.

Be aware that container filesystems, bind mounts and CI runners frequently have no disk
encryption at all, which is where engine-level encryption earns its keep.

## 4. Database connection security

| Key                       | Default     | Recommendation                                                                       |
| ------------------------- | ----------- | ------------------------------------------------------------------------------------ |
| `state.postgres_ssl_mode` | `"disable"` | Set to `require` at minimum. Use `verify-full` for deployment subject to compliance. |
| `state.postgres_host`     | `localhost` |                                                                                      |
| `state.postgres_port`     | `5432`      |                                                                                      |

The default is `disable`. Managed providers generally enforce TLS server-side regardless, so
a managed deployment is usually encrypted in transit despite this setting but the setting
should not be relied on to be harmless, and `verify-full` is the only mode that also
authenticates the server. Note that `verify-full` needs a root certificate, for which there
is no configuration key yet, so it is only usable where the certificate is already trusted
at the OS level.

The engine warns at startup when the mode is `disable` or `prefer` and the host is not the
local machine. `prefer` is included because it falls back to an unencrypted connection,
silently, when the server will not negotiate TLS. A local host is not warned about: an
unencrypted connection to a database on the same machine is how local development and CI
normally run.

## 5. Plugin transport

Provider and transformer plugins run **on the same host** as the engine and communicate over
gRPC. Networked plugins are not supported.

Today that channel is TCP on loopback with `insecure.NewCredentials()` and no registration
authentication, which means **any local process under any user can connect to the plugin
service port, register a plugin, and observe a channel carrying provider credentials.** This
is a least-privilege concern (CC6.1 / A.5.15) rather than a data-in-transit one.

Treat the engine host as a trust boundary: do not run untrusted local workloads alongside
the engine, and restrict who can obtain a shell on it. The engine warns about this transport
at startup, since it is a property of the deployment rather than something an operator
chooses.

Relevant keys: `plugins_v1.plugin_path`, `plugins_v1.log_file_root_dir`.

## 6. Data retention

Four retention windows, all defaulting to **7 days**:

| Key                                                   | Artifact               |
| ----------------------------------------------------- | ---------------------- |
| `maintenance.blueprint_validation_retention_period`   | Validation results     |
| `maintenance.changeset_retention_period`              | Changesets             |
| `maintenance.events_retention_period`                 | Event partitions       |
| `maintenance.reconciliation_results_retention_period` | Reconciliation results |

**Changesets and event partitions hold resolved secret values in plaintext for their whole
retention window,** and this cannot be fixed by redaction: deployment and rollback read the
persisted changeset, and stream resume serves persisted events. Their retention windows are
therefore confidentiality controls, not just housekeeping. Shortening them reduces exposure;
lengthening them for audit convenience increases it.

## Current limitations

The engine does not provide these today. None should be described as a control of a
deployment:

- **No application-layer encryption for `postgres` or `objectstore`.** At-rest protection for
  those is whatever the database or bucket provides. Only `memfile` can encrypt its own state.
- **A re-encryption pass needs the engine stopped**, and cannot be made atomic. There is no
  online rotation and no schedule: rotating a key is an operator-run command.
- **No redaction of secret values.** Resolved secrets appear in API responses, event streams
  and logs wherever they appear in a resource spec, changeset or drift record. The
  sensitivity flags in a blueprint and in provider schemas are recorded but not acted on.
- **No caller attribution or audit log.** Credentials are authenticated and then discarded,
  so "who deployed what, when" cannot be answered from the engine.
- **No authentication on the plugin channel**, and it runs over loopback TCP rather than a
  unix socket.
- **No SigV1 replay protection.** A captured signed request can be replayed within a fixed
  ±300s window, which the engine exposes no setting to narrow.
- **API keys are stored in plaintext**, in engine configuration and in CLI credential files.
- **No root certificate configuration** for `verify-full` database connections.
- **No state erasure endpoint**, so a right-to-erasure request has to be satisfied by
  deleting records directly in the state backend.

A failing provider's error text is also persisted verbatim in resource state. Provider
authors are responsible for keeping secret values out of error messages; nothing in the
engine removes them.

## Checklist for a compliance-oriented deployment

- [ ] TLS-terminating gateway or load balancer in front; engine not directly reachable.
- [ ] If the terminator is on a separate host: `loopback_only: false`, with security group or
      firewall rules admitting only the terminator, and no public route to the engine.
- [ ] If the terminator is on the same host: `loopback_only: true`, and a unix socket for the
      proxy hop if the proxy supports one.
- [ ] At least one authentication mechanism configured (startup enforces this).
- [ ] `auth.oauth2_oidc_jwt_issuer_secure: true` if using OIDC.
- [ ] Engine configuration and environment treated as credential material.
- [ ] `state.storage_engine` set explicitly rather than defaulted.
- [ ] Encryption at rest verified for the chosen backend, for example, `storage_encrypted` on RDS, bucket
      default encryption, or for `memfile` an encrypted volume and, preferably,
      `state.encryption.enabled`.
- [ ] If `memfile` state encryption is enabled: the key held outside the engine host, backed
      up, and its loss understood to mean the loss of the state.
- [ ] A key change understood as configure-then-re-encrypt, with the old key dropped from
      `state.encryption.previous_keys` only once a pass has moved the state onto the new one.
- [ ] `state.postgres_ssl_mode` set to `require` or `verify-full`.
- [ ] Retention windows set deliberately, knowing changesets and events hold plaintext
      secrets.
- [ ] No untrusted local workloads on the engine host.
- [ ] Backups of `memfile` state directories encrypted independently, or covered by the
      engine's own encryption with the key stored apart from them.
