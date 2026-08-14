# P6 Database Rollback Boundary

Application rollback and database recovery are separate.

Application rollback:

- new app version returns to previous app version
- allowed only if old app remains schema-compatible

Database recovery:

- used for corruption or irreversible migration failure
- requires a recovery point verified by the cloud database or operations platform
- requires a recent isolated restore drill recorded by that external platform
- requires an isolated target for validation and explicit human approval for production
- remains a high-risk external operation; TradeMind does not execute database recovery

