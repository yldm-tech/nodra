# Security policy

Nodra is a local state and persistence library. Do not include credentials, tokens, personal data, or untrusted secrets in DevTools snapshots, exported debug JSON, issue reports, or pull requests. `AESGCMCodec` protects serialized state only when the application supplies and protects its `KeyProvider`; Nodra does not manage keychains or secret storage.

## Reporting a vulnerability

Please report suspected vulnerabilities privately through the repository maintainer's GitHub security advisory flow. Include the affected release, operating system and Go version, a minimal reproduction, and the impact. Do not open a public issue for an unpatched vulnerability or attach real secrets.

Reports are acknowledged as soon as practical, triaged against the supported matrix in [docs/support.md](docs/support.md), and fixed releases are recorded in `CHANGELOG.md`. If private advisories are unavailable for a fork, contact the repository owners through their verified GitHub profiles and request a private security channel.

The security process does not promise a response time or a bounty. It does promise that reports will not be intentionally exposed before a fix or mitigation is available.
