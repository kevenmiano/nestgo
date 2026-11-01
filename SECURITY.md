# Security policy

## Supported versions

NestGo is currently experimental. No version is certified for unrestricted
production use until the acceptance matrix in
[docs/RELEASE_ACCEPTANCE.md](docs/RELEASE_ACCEPTANCE.md) is signed off.
Security fixes are prioritized on the main development branch; no long-term
support window is promised for experimental releases.

## Report a vulnerability privately

Use the repository's **Security** tab and select **Report a vulnerability**
to open a private security advisory when that feature is enabled. If private
reporting is unavailable, contact maintainers through a public issue or
discussion **without publishing credentials, exploitable payloads, proof-of-
concept exploits or private environment details**; request a private contact
channel first.

Include affected version/commit, Go version, minimum reproduction steps,
expected versus observed behavior and likely impact when communicating in a
private channel. Do not publish sensitive material without coordination.

## Scope and deployment guidance

Treat framework HTTP endpoints, generated metadata, request bodies and custom
middleware as untrusted input. Deploy behind a configured reverse proxy, set
connection/body/header limits, use least-privilege credentials, and run release
CI and race tests before upgrades. The presence of automated checks is not a
security guarantee. No fixed security response SLA is currently advertised.
