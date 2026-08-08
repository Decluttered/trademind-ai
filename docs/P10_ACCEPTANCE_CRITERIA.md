# P10 Acceptance Criteria

Status: **Finalized / Not Yet Verified**

The Owner-approved P10 acceptance contract contains 28 criteria. Criteria 1-27 are required for initial Production Ready. Criterion 28 is a separately approved conditional write extension and is not required for the first production release.

| Range | Area | Initial Production Ready |
| --- | --- | --- |
| P10-AC-01 | Independent pre-production | required |
| P10-AC-02 to 04 | Backend-only encryption, rotation, revocation, audit and redaction | required |
| P10-AC-05 to 06 | Tenant isolation and 1 tenant / 1 shop / <=100 SKU fail-closed scope | required |
| P10-AC-07 to 10 | Real Provider read-only authentication, normalized reads and bounded failures | required |
| P10-AC-11 | Unauthorized write impossible and `inventoryMutationEnabled=false` | required |
| P10-AC-12 to 14 | Backup, measured RPO <=15m, restore RTO <=60m and rollback | required |
| P10-AC-15 to 17 | Provider, tenant, shop, read and write kill switches | required |
| P10-AC-18 | Security and adversarial verification | required |
| P10-AC-19 to 21 | Dedicated-host matrix, >=3 repeat runs, frozen SLOs and stability | required |
| P10-AC-22 to 23 | Observability and alert drills | required |
| P10-AC-24 to 25 | Read-only gray and Owner + Technical Lead Go approval | required |
| P10-AC-26 to 27 | Release integrity and Owner final acceptance after prerequisite sign-offs | required |
| P10-AC-28 | Controlled real inventory write | conditional; excluded |

The initial release validates that real mutation cannot happen. It does not require or permit a successful real inventory write. Background Worker and automatic business retry are likewise outside the initial approved boundary.

```text
acceptanceCriteriaFinalized=true
acceptanceCriteriaCount=28
allCriteriaApproved=true
allCriteriaPassed=false
realInventoryWriteRequired=false
inventoryMutationEnabled=false
productionAcceptancePassed=false
productionReady=false
```

The exact criterion text and required/conditional flags are authoritative in [`p10-acceptance-criteria.json`](p10-acceptance-criteria.json). The historical draft remains unchanged in [`P10_ACCEPTANCE_CRITERIA_DRAFT.md`](P10_ACCEPTANCE_CRITERIA_DRAFT.md) and [`p10-acceptance-criteria-draft.json`](p10-acceptance-criteria-draft.json).
