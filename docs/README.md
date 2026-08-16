# TradeMind Documentation Center

The project has entered the production maintenance phase. The workspace retains only the documentation needed for current development, deployment, operations, and manual acceptance; historical phase reports, one-off gate reports, and run evidence are no longer kept in the current working tree — query Git history when needed.

## Usage & Deployment

- [Local Development](development.md)
- [Docker Deployment](docker-deployment.md)
- [Environment Variables](env.md)
- [API Contract](api.md)
- [AI Customer Service Reply Design](CUSTOMER_AI_REPLY_SUGGESTION_DESIGN.md)
- [Customer Service Center Design](CUSTOMER_SERVICE_CENTER_DESIGN.md)
- [Provider Extensions](provider.md)
- [System Architecture](architecture.md)
- [MindBay Modul- und Reuse-Matrix](mindbay-module-matrix.md)
- [MindBay Architekturentscheidungen](adr/)

## Production Operations

- [Production Boundaries](PRODUCTION_CAPABILITY_BOUNDARY.md)
- [Pre-production Architecture](PREPRODUCTION_ARCHITECTURE.md)
- [Manual Acceptance Checklist](PRODUCTION_MANUAL_ACCEPTANCE_CHECKLIST.md)
- [Risk Register](PRODUCTION_RISK_REGISTER.md)
- [Observability Architecture](OBSERVABILITY_ARCHITECTURE.md)
- [Database Rollback Boundaries](DATABASE_ROLLBACK_BOUNDARY.md)
- [Disaster Recovery Plan](DISASTER_RECOVERY_PLAN.md)

## Engineering Collaboration

- [AI Workflow](ai-workflow.md)
- [AI Coding Rules](ai-coding-rules.md)
- [Module Reference Index](module-map.md)
- [Task Checklist](task-checklist.md)
- [Branching & PRs](branching.md)
- [Current Maintenance Status](PROGRESS.md)

## Acceptance Conventions

- Automated tests run only through `.github/workflows/`; core tests that workflows depend on must not be removed.
- Final sign-off for features, pages, and business processes is completed manually against the acceptance checklist.
- A local test database is not required; CI uses isolated PostgreSQL/Redis service containers.
- No persistent gates, fixture reports, screenshot reports, or `artifacts/` evidence are created for individual phases, batches, or acceptance runs.
