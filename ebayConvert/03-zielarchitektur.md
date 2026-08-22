# Zielarchitektur und Datenmodell

## Fachlicher Datenfluss

```
Amazon.de source
  -> sourceAmazon / offizielle Read API / Operator-Import
  -> catalog source product + immutable snapshot
  -> Research candidate and score
  -> Listing Studio (facts, images, GPSR, category/aspects)
  -> Publication calendar slot + idempotent job
  -> Temporal activity -> eBay provider -> Inventory Item / Offer / publish
  -> marketplace listing + immutable listing snapshot

Source refresh -> snapshot changed -> monitoring decision
  -> approved/LIVE price or quantity update through eBay provider

eBay orders + finance records -> order/financial ledger -> profit UI/export/archive
```

## Ownership und erlaubte Abhängigkeiten

| Eigentümer | Verantwortung | Darf nicht |
| --- | --- | --- |
| `sourceAmazon` | Rohdaten abrufen, validieren, normalisieren | GORM/Domain-DB oder eBay importieren |
| `catalog` | Quellen, Snapshots, Collections, Kandidatenübergabe | eBay API aufrufen |
| `listingstudio` | Faktenbasierter Draft, Content-Version, GPSR/Assets | Publish selbst ausführen |
| `publication` | Slots, Jobzustand, idempotente Veröffentlichung, Reconcile | einen zweiten Redis-Worker verwenden |
| eBay provider | Plattformdaten in interne Modelle übersetzen | Berechtigungen, UI-Zustand oder Business-Freigaben entscheiden |
| `monitoring` | Snapshot-Vergleich, Preisentscheidung, Ledger | externe Daten direkt aus UI übernehmen |
| `agent` | autorisierte Service-Commands orchestrieren | DB oder eBay direkt schreiben |

Der Aufrufpfad bleibt: Router/Handler -> Service -> Repository/Provider -> Modell. Worker und Temporal Activities rufen Services; sie rufen keine Handler und implementieren keine zweite Zustandsmaschine.

## Zielzustände

### Veröffentlichung

`DRAFT -> READY -> SCHEDULED -> HELD -> PUBLISHING -> PUBLISHED`  
Fehlerpfade: `VALIDATION_FAILED`, `RETRYABLE_FAILURE`, `UNKNOWN_RESULT`, `CANCELLED`, `DRY_RUN`.

`UNKNOWN_RESULT` ist kein Fehlertyp, bei dem blind erneut veröffentlicht wird. Vor jedem Retry wird per stabiler SKU/Offer-ID reconciled. LIVE ist nur zulässig, wenn `EBAY_ENV=production` und `AUTOMATION_MODE=LIVE` explizit gesetzt sind.

### Source und Monitoring

Ein Snapshot ist append-only. Eine Entscheidung referenziert immer genau einen Source-Snapshot, Listing-Snapshot und immutable Regelversion. Ein veralteter Snapshot blockiert Update/Auto-Apply statt einen Preis zu raten.

## Additives Datenmodell

Vorhandene MindBay-Tabellen bleiben Owner ihrer Verantwortung. Neue Tabellen sind additiv und tragen `workspace_id` im bestehenden Tenant-ID-Raum.

| Objekt | Entscheidung | Pflichtfelder und Constraints |
| --- | --- | --- |
| Source Snapshot | Bestehenden Catalog-Snapshot erweitern oder genau dort neue Felder ergänzen | source, external ID, URL, cents, currency, availability, captured_at, raw reference; append-only |
| Research Run | NEW | workspace, source, query/filter JSON, status, started/finished, bounded result count |
| Research Candidate | NEW | run, snapshot/source ref, scores, estimate inputs, status; unique source per run |
| Publication External Ref | NEW, falls vorhandenes Listingmodell IDs nicht ausreichend kapselt | listing/job, `OFFER_ID`/`LISTING_ID`/`INVENTORY_GROUP`, external ID, metadata JSON; unique owner/type/value |
| Marketplace Policy / Location | NEW | workspace/shop, marketplace, external ID, normalized payload, fetched_at, expiry/freshness |
| Order Financial | NEW | order, gross revenue, source cost, shipping, fees, refunds, net cents, currency, calculated_at, source version |
| Order Fee Line | NEW | order financial, type, cents, external transaction ID, raw reference; uniqueness by provider transaction |
| Invoice Archive | NEW | workspace, year/month, provider, status, external file ID, checksum, created/completed; unique month/workspace |
| Agent Run/Tool Call | NEW only when Agent ships | actor, tool, idempotency key, input/output redacted, approval state, audit-log link |

Eine neue `ebay_accounts`- oder `ebay_tokens`-Parallelwelt ist ausdrücklich nicht vorgesehen. `shops` und `shop_auth_tokens` bleiben die Account- und OAuth-Basis.

## eBay-Fachgrenzen

Vor dem Publish validiert der bestehende Readiness-Pfad mindestens Seller Policies, Inventory Location, Leaf Category, verpflichtende Item Aspects, Condition, Quantity, cents price/currency, eindeutige SKU, Images und GPSR. Amazon-Browse-Nodes werden nicht automatisch als eBay-Kategorie akzeptiert; fehlende verpflichtende Aspekte blockieren.

Bei Bildern und Beschreibungen gilt Quellenschutz: Amazon-Assets werden als Herkunftsfakten referenziert, nicht kommentarlos nach eBay kopiert. AI darf nur aus gelieferten Fakten formulieren, keine Zertifikate, Material-, Sicherheits- oder Verfügbarkeitsbehauptungen erfinden.

## Agent-Gateway

Die erste Toolliste bleibt read-first: `searchCandidates`, `getListing`, `getOrders`, `getProfitReport`, `previewCalendar`, `previewReprice`. Schreibende Tools wie `applyCalendar`, `approvePublication`, `applyPriceDecision` benötigen dieselbe Berechtigung, Idempotency-Key, Readiness-Prüfung, Audit-Event und bei Live eine explizite Operatorfreigabe wie der Admin.

## Observability und Sicherheitsminimum

- Jede Plattformoperation führt Trace-/Task-/Job-ID, Shop und Workspace, niemals vollständige Tokens oder Payload-Rohdaten in Logs.
- Provider-HTTP hat Timeout, begrenzte Retries, Rate-Limit-Behandlung und Response-Validierung.
- Webhook-Ingestion ACKt schnell, persistiert idempotent und verarbeitet asynchron.
- Alle Tests verwenden Fakes/Mock-Server und isolierte `TEST_DATABASE_URL`/`TEST_REDIS_URL`; keine realen Plattform-Write-Aufrufe.
