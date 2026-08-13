# Disaster Recovery Plan

The reusable disaster-recovery foundation consists of:

- backup and verification records
- isolated restore safety gates and validation records
- application release rollback records
- backup, restore, release and WAL/PITR runbooks
- backup/restore and release dashboards backed by runtime metrics

The standalone drill recorder was retired because it only stored self-declared records and did not execute or verify a recovery workflow. Existing `dr_drills` rows are preserved for retention review, are no longer exposed by an API, and are not managed by `AutoMigrate`.

Production disaster-recovery validation remains an externally controlled operation. A real production restore, PITR validation or traffic switch requires an approved runbook, verified backups, an isolated target where applicable, explicit authorization and human sign-off.

