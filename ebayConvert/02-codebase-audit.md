# Verifizierter Codebase-Audit

## Methode und Begrenzung

Der Audit basiert auf dem lokalen Branch `ebayamazon`, den vorhandenen Modulen, Provider- und Collector-Verzeichnissen, MindBay-Migrationsdateien, Routen und den bestehenden MindBay-Dokumenten. Es wurden keine externen Konten, APIs, Datenbanken oder Worker gestartet. Aussagen über eine reale eBay-Sandbox-Veröffentlichung bleiben deshalb bewusst **unverifiziert**.

## Bereits vorhandene tragfähige Bausteine

### MindBay-Pipeline

Die Repository-Dokumentation und Codepfade bestätigen bereits diese Pipeline:

```
sourceAmazon -> collect -> catalog immutable snapshots
  -> listingstudio -> publication calendar/job -> Temporal
  -> eBay provider -> marketplace listing -> monitoring
```

- `collector/src/providers/sourceAmazon/`: Amazon-Parser mit ASIN-/EUR-Normalisierung.
- `backend/internal/modules/catalog/`: Collections, Source Products, unveränderliche Snapshots und Opportunity Scores.
- `backend/internal/modules/listingstudio/`: Listing Draft, append-only Content-Versionen, Bildassets und GPSR.
- `backend/internal/modules/publication/`: Kalender-Slots, Preview, idempotentes Apply, Publication Jobs, Listings, Snapshots und Zustandsereignisse.
- `backend/internal/workflow/temporal/` und `temporal/`: langlebige eBay-Publication-Orchestrierung; Queue `mindbay-publication`.
- `backend/internal/providers/platform/ebay/`: Go-Adapter mit OAuth, Taxonomy, Inventory-/Offer-Publish und Reconcile.
- `backend/internal/modules/monitoring/`: Monitor Runs, versionierte Preisregeln, Price Decisions und Profit Ledger.

### Generische Plattform- und Betriebsbasis

Vorhanden sind ferner Auth, Tenant-Scopes, Shops und verschlüsselte Token, Settings, idempotente HTTP-Commands, Operations-Audit, Task Center, Alerts, Redis-Worker/Leases, Task-Reaper, Storage- und AI-Provider. Diese Systeme haben höheren Wiederverwendungswert als bestehende Admin-Seiten.

## Entscheidende reale Architekturgrenzen

1. Der Collector liefert normalisierte Daten; er schreibt niemals die Domain-Datenbank.
2. Der eBay-Provider ist der einzige Plattformadapter. Admin, Extension, Agent und Temporal-Workflow rufen die eBay-Sell-API nicht direkt auf.
3. `modules/publication` ist der alleinige Owner für eBay Publish. Der ältere Redis-Pfad `modules/productpublish` bleibt für eBay außen vor.
4. Der Kalender ist ein Planungs-/Reservierungsmodell. `Apply` erzeugt Slots und Jobs idempotent; Publish wird anschließend über Temporal/den internen Servicepfad ausgeführt.
5. Geld im MindBay-Pfad verwendet `int64`-Cents plus ISO-Währung. Historische `float64`-Felder werden nicht still umgedeutet.
6. `workspace_id` des MindBay-Produkts ist derselbe ID-Raum wie das bestehende `tenant_id`; es darf kein dritter Mandantenschlüssel entstehen.

## Vorhandene Datenbankobjekte

MindBay-Phase-1-Migrationen sichern immutable Source-/Content-Historien. Phase 2 migriert `calendar_slot`, `publication_job`, `marketplace_listing`, `listing_snapshot` und `publication_transition_event`. Phase 3 ergänzt Monitoring- und Finanzobjekte: `monitoring.MonitorRun`, `PriceRule`, `PriceDecision`, `ProfitLedgerEntry`.

Die Tabelle `calendar_slot` besitzt einen partiellen Unique-Index gegen gleichzeitig aktive Slots für dieselbe Workspace-/Zeit-/Slot-Typ-Kombination. Publication Jobs haben ebenfalls Workspace-gebundene Idempotenz. Das sind bereits genau die Schutzvorkehrungen, die ein späterer Velno-Kalender benutzen muss.

## Admin-Ist-Zustand

Es gibt eine `MindBayGroupLayout`-Struktur und Seiten für Discovery, Collections, Products, Listing Studio, Planner, Monitoring und Profit. Zusätzlich existieren MindBay-Service- und E2E-Spezifikationen. Die UX ist somit nicht bei null; sie braucht dennoch eine produktorientierte Informationsarchitektur, klare DRY_RUN-/LIVE-Kommunikation und neue Endnutzer-Flows.

## Offene Produktlücken

| Lücke | Warum sie nicht als vorhanden gilt | Ziel |
| --- | --- | --- |
| Verlässliches End-to-End-Sandbox-Release | Dokumentiert als ausstehende manuelle Abnahme mit externen Zugangsdaten. | Erst kontrollierter Sandbox-Run, dann produktive Freigabe. |
| Source-Monitoring als dauerhafte Automation | Snapshots existieren; Frequenz, Freshness, Stale-Policy und Review-UX sind Produktentscheidungen. | Explizite Monitor-Policy mit Queue/Temporal-Trigger. |
| eBay-Verkauf → tatsächlich realisierter Gewinn | Expected/Monitoring-Ledger existiert; robuste Fee- und Sale-Backfill-Logik ist noch auszubauen. | Order-/Fee-Ledger mit Quellenbeleg. |
| Amazon Retail Fulfillment | Architektur nennt den separaten Adapter, aber keinen fertigen Auto-Checkout-Flow. | Zuerst operator-gesteuert, dann nur bei bestätigter Berechtigung automatisieren. |
| Research über einzelne Capture-URLs hinaus | Catalog/Scoring ist da, aber Discovery-Strategie und Kandidaten-Review sind Produktarbeit. | Research Run / Candidate Inbox als additive Domäne. |
| Google-Drive-Rechnungsarchiv und Agent Gateway | Nicht als aktuelle Kernmodule bestätigt. | Neue, streng begrenzte Module. |

## Audit-Fazit

TradeMind ist nicht nur ein allgemeines Starterkit. Es enthält bereits die richtige Amazon.de-to-eBay.de-Richtung. Die Implementierung muss deshalb primär **konsolidieren, vervollständigen und härten**. Ein paralleles `ebay_listings`, ein zweiter Scheduler oder ein zweiter eBay-Client wären Architekturfehler.
