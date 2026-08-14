# P10 Production Boundary

Status: **Approved / Runtime remains L0**

| Level | Capability | Current status |
| --- | --- | --- |
| L0 | Fixture / Mock | allowed |
| L1 | Pre-production Real Provider Read-only | approved for future gated promotion; not enabled |
| L2 | One Allowlisted Production Shop Read-only Gray | approved for future gated promotion; not enabled |
| L3 | Controlled Production Write | conditional; not Owner-approved; not enabled |

```text
currentAllowedLevel=L0
productionBoundaryApproved=true
realProviderApprovedForImplementation=true
realReadApprovedForImplementation=true
realInventoryWriteApproved=false
backgroundWorkerApproved=false
automaticRetryApproved=false
realPlatformNetworkEnabled=false
realCredentialsEnabled=false
inventoryMutationEnabled=false
productionReady=false
productionAcceptancePassed=false
```

Promotion is evidence-based and sequential. Every level requires the prior acceptance evidence, verified tenant/shop scope, observability, rollback readiness, active kill switches, and the named approvals. Initial gray is frozen at one tenant, one shop, and at most 100 SKUs. Any expansion requires a Gray Scope Change Request plus Owner and Technical Lead approval.

L3 is outside initial Production Ready. It requires a separate Owner approval after read-only gray and must retain per-operation confirmation, expected revision, idempotency, audit, allowlists, and a write kill switch.
