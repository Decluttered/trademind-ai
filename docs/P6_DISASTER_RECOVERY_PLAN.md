# Disaster Recovery Plan

TradeMind does not execute or record database backups and restore validation inside the application. The cloud database and operations platform own:

- automatic backups, encryption, retention and cross-region or cross-zone redundancy
- point-in-time recovery capability and backup failure or age alerts
- isolated restore drills with recorded RPO/RTO results
- access control, approval, audit evidence and escalation for production recovery

Application rollback remains separate from database recovery. Release approval must reference a recovery point that the external platform has verified and a recent isolated restore drill; a self-reported application record is not acceptable evidence.

The P10 pre-production backup, isolated restore and rollback scripts remain deployment-level acceptance tools. They do not expose an Admin page or application API and must not target production. A real production restore, PITR operation or traffic switch requires the approved external runbook, explicit authorization and human sign-off.

Historical application backup, restore, release and drill tables are no longer exposed or managed by `AutoMigrate`. Existing rows remain untouched until a separate retention or archival decision is approved.

