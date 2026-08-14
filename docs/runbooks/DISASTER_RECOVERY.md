# Disaster Recovery Runbook

Meaning: coordinated recovery workflow for major outage or data-loss event.

Impact: service restoration requires controlled operator actions.

Safety checks: identify environment, freeze writes if needed, preserve evidence, verify the cloud database or operations platform recovery point and never expose secrets.

Triage steps: determine whether application rollback, backup restore or PITR is required.

Recovery steps: prefer application rollback when schema is compatible; database restore or PITR must be executed through the cloud database or operations platform against an isolated target first and requires high-risk approval before production use.

Forbidden actions: TradeMind must not execute database recovery; no production restore without explicit approval; no unverified recovery point; no public backup links.

Rollback boundary: application rollback and database recovery are separate decisions.

Escalate when: RPO/RTO target cannot be met or tenant isolation is uncertain.

Verify: external restore evidence and RPO/RTO results are recorded; health checks, tenant isolation, RBAC, audit chain, object inventory and customer-facing smoke checks pass.
