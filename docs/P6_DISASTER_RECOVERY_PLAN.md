# Disaster Recovery Plan

The reusable disaster-recovery foundation consists of:

- backup and verification records
- isolated restore safety gates and validation records
- backup, restore and WAL/PITR runbooks
- externally approved application rollback and deployment procedures
- backup and restore dashboards backed by runtime metrics

The standalone release recorder and drill recorder were retired because they only stored self-declared state and did not execute deployment, traffic switching or a verified recovery workflow. Existing `release_*` and `dr_drills` rows are preserved for retention review, are no longer exposed by an API, and are not managed by `AutoMigrate`.

Production disaster-recovery validation remains an externally controlled operation. A real production restore, PITR validation or traffic switch requires an approved runbook, verified backups, an isolated target where applicable, explicit authorization and human sign-off.

