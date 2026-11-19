# Trust Configuration Transport and Distribution Bootstrap-Key Provisioning

**Status:** Accepted implementation boundary  
**Scope:** Plugin Module trust configuration source and PRIDE distribution composition

## 1. Transport boundary

The Plugin Module exposes `artifactverify.TrustConfigurationSource`. The concrete `HTTPTrustConfigurationSource` is a bounded HTTPS adapter and does not define a Marketplace API endpoint.

The PRIDE distribution MUST provide:
- the exact HTTPS trust-configuration endpoint URL;
- an exact hostname allowlist containing that endpoint host;
- an explicit timeout within the supported 1–60 second range;
- the embedded bootstrap public-key set passed to `NewTrustResolver`.

The transport:
- requires HTTPS, hostname allowlisting, and port 443;
- rejects URL user-info, fragments, IP-literal hosts, wildcard allowlist entries, and redirects;
- disables environment proxies and automatic compression;
- requires TLS 1.2 or later;
- resolves DNS at connection time and rejects non-public destination addresses;
- bounds response headers to 64 KiB and response content to 1 MiB;
- requires HTTP 200 and `application/json`;
- rejects unexpected content encoding and empty responses.

The downloaded bytes remain untrusted. The existing `TrustResolver` MUST authenticate the complete document with an embedded Ed25519 bootstrap public key before accepting the trust sequence or using Marketplace keys. Transport success is not trust.

## 2. Bootstrap-key provisioning

Bootstrap keys are public verification keys, not secrets. Their trust comes from the PRIDE distribution release process, not from the remote trust endpoint.

A distribution composition MUST construct `artifactverify.BootstrapKeys` from reviewed, release-controlled key material and pass it directly to `NewTrustResolver`. The key ID and 32-byte Ed25519 public key MUST be included in the signed release/build artifact. Private keys MUST remain in the Marketplace/distribution signing environment and MUST NOT be present in PRIDE client builds.

The Module intentionally does not ship a fabricated default key, sample production key, environment-variable override, or remote first-use key download. A distribution that has not supplied at least one valid bootstrap public key MUST fail composition; `NewTrustResolver` already rejects an empty or malformed key set.

Bootstrap-key rotation is a distribution update/release operation. A remote trust configuration can rotate Marketplace descriptor keys, but cannot add or replace the bootstrap root. Recovery from a compromised bootstrap root requires a separately reviewed PRIDE distribution update or recovery mechanism.

## 3. Explicit non-goals

This contract does not invent:
- an Official Marketplace trust endpoint or Registry API;
- endpoint authentication headers, OAuth scopes, or service credentials;
- a production bootstrap public-key value;
- Marketplace signing infrastructure or private-key custody;
- application composition-root wiring.

Those values and operational integrations must be supplied by the corresponding PRIDE distribution and Marketplace services. Until then, this adapter is production-grade transport infrastructure, not proof that a live production trust service exists.
