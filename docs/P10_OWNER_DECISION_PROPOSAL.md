# P10 Owner Decision Proposal

Status: **Awaiting Owner Approval**

This document contains recommendations only. No decision is approved by this planning pack, and P10 implementation has not started.

```text
ownerDecisionCount=15
ownerApprovalPending=true
recommendationOnly=true
approved=false
```

| ID | Decision | Recommended option | Status |
| --- | --- | --- | --- |
| D01 | Real Douyin Provider in P10 | Integrate behind production boundaries | pending owner approval |
| D02 | First real-provider stage | Strict read-only | pending owner approval |
| D03 | Shop scope | Pre-production, then one allowlisted production shop | pending owner approval |
| D04 | Real inventory write | Defer until read-only gray passes | pending owner approval |
| D05 | Write confirmation | Per-operation human confirmation if write is approved | pending owner approval |
| D06 | Background worker | Disabled during first gray | pending owner approval |
| D07 | Automatic retry | Disabled during first gray | pending owner approval |
| D08 | First gray scope | One tenant, one shop, bounded SKU set | pending owner approval |
| D09 | Pre-production | Independent environment required | pending owner approval |
| D10 | Credential and OAuth storage | Encrypted, rotatable, revocable, audited, backend-only | pending owner approval |
| D11 | PostgreSQL RPO/RTO | Owner-defined production SLA | pending owner approval |
| D12 | P7 performance threshold | Dedicated-host benchmark and owner-approved thresholds | pending owner approval |
| D13 | Repository baseline | Pending owner disposition after impact review | pending owner approval |
| D14 | Gray Go/No-Go | Explicit named approver | pending owner approval |
| D15 | Production Ready | Explicit owner final approval | pending owner approval |

The complete options, rationale, risks, and dependencies are recorded in [`p10-owner-decision-proposal.json`](p10-owner-decision-proposal.json).

## Boundary

Until these decisions are approved, the allowed level remains L0 fixture/mock. Real Provider access, OAuth, real platform network, real inventory reads or writes, Worker, automatic retry, gray release, Tag, Release, and Production Ready remain disabled.
