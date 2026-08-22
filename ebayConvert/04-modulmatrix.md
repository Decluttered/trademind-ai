# Modulmatrix: TradeMind zu eBay-first SaaS

## Codes

- **KEEP:** unverändert weiterverwenden.
- **EXTEND:** gleiche Verantwortung, additive MindBay-/Velno-Funktion.
- **LEAVE:** Legacy weiter lauffähig, nicht aktiv ausbauen oder im Endnutzer-Menü zeigen.
- **NEW:** klar abgegrenzte neue Verantwortung; keine Kopie eines bestehenden Moduls.

## Backend

| Modul | Entscheidung | Konkrete Rolle |
| --- | --- | --- |
| auth, admin, adminuser, securitymod | KEEP | Benutzer, Rollen, Tenant, Secret-Schutz. |
| idempotency, operationlog, taskcenter, taskreaper, worker, alerting, observabilitymod | KEEP/EXTEND | Betriebsfundament; MindBay Events/Metriken hinzufügen. |
| catalog, collect, collectrule, collectruleai | EXTEND | Amazon-Erfassung, Snapshot-Freshness, Research-Eingang. |
| listingstudio, listingcalc, productcheck | EXTEND | eBay-Draft, integer-Cents, GPSR, Aspekte/Readiness. |
| publication | EXTEND | Einziger Kalender-/eBay-Publish-Owner; Batch-Slots, Policies/Locations, Reconcile. |
| monitoring | EXTEND | Source/Offer-Vergleich, immutable Regeln, Decision, Expected-/Actual-Ledger. |
| pricing | LEAVE/EXTEND gezielt | Nicht zweite Repricing-Domain schaffen; Legacy-Float nicht migrieren. Neue Regelberechnung gehört entweder klar in Monitoring oder als Adapter um `listingcalc`. |
| inventory, inventorysync, inventoryread | EXTEND | eBay-Quantity-Sync auf Publication/SKU-Mappings; Stale/Out-of-stock-Policy. |
| order, ordersync, orderexception | EXTEND | eBay Fulfillment ingest, Cursor, Match-Ausnahmen, operator-gesteuertes Fulfillment. |
| shop, platformcredential, settings, webhook | EXTEND | eBay OAuth, Policy/Location Cache, Automation-Modus, Notifications. |
| exportmod, files, storagepublic | EXTEND | Gewinnexporte, Rechnungsarchive und Asset-Auslieferung. |
| research | NEW | Runs, Kandidaten, Ranking, Dedupe, Import in Catalog. |
| profit | NEW | Order Financials/Fee Lines und Reporting über tatsächliche Verkäufe; Monitoring-Ledger integriert, nicht kopiert. |
| invoice | NEW | Monatsarchiv als asynchroner, idempotenter Storage-Job. |
| agent | NEW | Tool-Policy und Service-Gateway mit Audit/Freigabe. |
| customerchat, customersync, douyinpreflight, douyinruntime | LEAVE | Nicht Teil des eBay-first MVP; nicht früh löschen. |
| productpublish | LEAVE | Für Legacy behalten; nicht für eBay umnutzen. |

## Provider und Collector

| Pfad | Entscheidung | Umsetzung |
| --- | --- | --- |
| `providers/platform/ebay` | EXTEND | Bestehenden Go-Client mit Account/Taxonomy/Inventory/Offer/Publish/Reconcile testen und ergänzen: Policies, Locations, Order/Fulfillment, Finance, Notifications. |
| `providers/platform/amazon` | LEAVE | Es ist Amazon-Verkaufskanaladapter, kein Source Collector. |
| `providers/platform/{douyinshop,lazada,shopee,tiktok}` | LEAVE | Erst in Produktnavigation/aktiver Konfiguration deaktivieren; erst später entfernen. |
| AI/Image/Storage Provider | KEEP/EXTEND | Reuse; Google Drive ausschließlich als neuer Storage-Adapter hinter dem bestehenden Port. |
| `collector/sourceAmazon` | EXTEND | Mehr Varianten, Verfügbarkeit, Lieferhinweise, Fehlerklassen, API-first wo Credentials vorhanden. |
| `collector/shared`, `normalizer`, `browser`, `security`, `tasks` | KEEP | Keine eBay-Abhängigkeit hinzufügen. |
| andere Collector-Sources | LEAVE | Nicht zu Defaults machen; nicht in der ersten Welle löschen. |

## Admin

| Bereich | Entscheidung | Nutzeroberfläche |
| --- | --- | --- |
| Auth/Layout/Services/shared UI | KEEP | Bestehende technische Basis und Design-Tokens. |
| MindBay Discovery/Collections/Products/ListingStudio/Planner/Monitoring/Profit | EXTEND | Daraus die sichtbare Kernnavigation bilden. |
| Dashboard | EXTEND | echte KPIs: Ready, Scheduled, Published, blocked, source stale, expected vs actual profit. |
| Shops/Settings | EXTEND | eBay Account, Sandbox/Live sichtbar, Policies, Location, Automation-Guards. |
| Orders/Inventory | EXTEND | eBay-Sales, Source-Verfügbarkeit, Sync-Ausnahmen. |
| Files/Exports | EXTEND | Rechnungsarchive und Downloads. |
| TaskCenter/Workers/Logs/Ops | KEEP, intern | Unter `/admin` oder klarer Advanced-Sektion; keine Endnutzer-Primärnavigation. |
| Customer/Douyin-spezifische Seiten | LEAVE/ausblenden | Nicht im eBay-first SaaS-Menü. |

Jede neue/angepasste Admin-Seite braucht reale Loading-, Empty-, Error-, Readonly- und Submit-Zustände, fünf Viewports und einen Write Guard in E2E. Das sichtbare Menü wird fokussiert, aber API-/Berechtigungssemantik wird nicht bei einem UI-Umbau verändert.
