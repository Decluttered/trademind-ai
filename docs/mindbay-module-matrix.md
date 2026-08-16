# MindBay Modul- und Reuse-Matrix

Stand: 2026-08-15  
Geltungsbereich: Amazon.de als Quelle, eBay.de als Zielmarkt  
Verbindliche Codebasis: Repository-Root `trademind-ai/`

Diese Matrix beschreibt den in Phase 0 verifizierten Ausgangspunkt und den nach Phase 2 erreichten additiven Stand. Phase 2 ist implementiert und lokal getestet; ein echter eBay-Sandbox-Publish bleibt bis zur manuellen Abnahme mit externen Sandbox-Zugangsdaten ausstehend.

## Entscheidungscodes

| Code | Bedeutung |
| --- | --- |
| **Reuse** | Vorhandene Verantwortung und Schnittstelle unverändert weiterverwenden. |
| **Extend** | Bestehende Verantwortung beibehalten und additiv für MindBay ergänzen. |
| **Leave** | Bestehender Legacy-Pfad bleibt lauffähig, wird für das MindBay-MVP aber nicht ausgebaut. |
| **Do-not-touch** | Kein Produktcode; weder ändern noch in die MindBay-Domäne importieren. |

## Schnellnavigation für Phase 1 und 2

| Gesuchte Verantwortung | Verbindlicher Einstieg |
| --- | --- |
| eBay Publish | Nativer Go-Adapter `backend/internal/providers/platform/ebay/`; alleiniger Domain-Owner ist `backend/internal/modules/publication/`, gestartet durch `backend/internal/workflow/temporal/`. Der Redis-`productpublish`-Pfad ist für eBay deaktiviert. |
| Amazon Snapshot | `collector/src/providers/sourceAmazon/` hinter `CollectorProvider`; Persistenz und unveränderliche Snapshot-Domäne in `backend/internal/modules/catalog/`. |
| Listing Studio | `backend/internal/modules/listingstudio/` für Draft-Zustände, Content-Versionen, GPSR und Bilder; Admin unter `admin/src/pages/MindBay/`. |
| Preis in Cents | `backend/internal/modules/listingcalc/` verwendet `int64`-Cents plus ISO-Währung; vorhandene `float64`-Verträge bleiben unverändert. |
| Temporal | Produktiver TypeScript-Worker unter `temporal/`, Go-Starter unter `backend/internal/workflow/temporal/`; `../temporal-worker/` bleibt ausschließlich Lern-Scaffold. |
| Scope / Berechtigung | `workspace_id` im Blueprint entspricht dem bestehenden `tenant_id`; Service-Layer nutzt weiterhin `backend/internal/pkg/adminperm/` und `backend/internal/pkg/tenantquery/`. |

## Backend-Module

Jedes derzeit vorhandene Paket unter `backend/internal/modules/*` ist erfasst.

| Pfad | Verantwortung im Ist-Zustand | MindBay-Relevanz | Entscheidung |
| --- | --- | --- | --- |
| `modules/admin` | Admin-Bootstrap, Store-Berechtigungen | Bestehende Identität und Mandantenbasis | **Reuse** |
| `modules/adminuser` | Admin-Benutzerverwaltung und Rollen | Operator-/Reviewer-Zugriff | **Reuse** |
| `modules/aioperationbatch` | Übergreifende AI-Batches | Spätere Listing-Batches möglich | **Reuse** |
| `modules/aiopsworkbench` | AI-Operations-Arbeitsliste | Mögliche Review-Inbox-Basis | **Extend** |
| `modules/aiproductimage` | AI-Bildoperationen und Apply-Idempotenz | Bildpipeline-Baustein | **Extend** |
| `modules/aiproducttext` | AI-Produkttexte, Qualität und Apply | Listing-Content-Versionen darauf aufbauen | **Extend** |
| `modules/aiprompt` | Versionierbare Prompts | Listing- und Command-Prompts | **Reuse** |
| `modules/aitask` | AI-Task-Lebenszyklus und Sanitizing | Nachvollziehbare AI-Ausführung | **Reuse** |
| `modules/alerting` | Systemalarme und Regeln | MindBay-Run-/Provider-Alarme | **Extend** |
| `modules/auth` | Login, JWT und Sessions | Bestehende Admin-Authentifizierung | **Reuse** |
| `modules/collect` | Collector-Aufträge, Import, Queue, Retry und Lease | Go-Eigentümer der Amazon-Aufnahme | **Extend** |
| `modules/collectbrowserprofile` | Browserprofil-Metadaten und API | Workspace-bezogene Amazon/eBay-Sessions | **Extend** |
| `modules/collectrule` | Regelbasierte Quell-Erfassung | Amazon Discovery-Regeln | **Extend** |
| `modules/collectruleai` | AI-Unterstützung für Collector-Regeln | Optionale Discovery-Regelhilfe | **Extend** |
| `modules/configstatus` | Konfigurations- und Capability-Status | MindBay-Betriebsmodus und Integrationsstatus | **Extend** |
| `modules/customerchat` | Kundenkonversationen und review-gated Antworten | Support bleibt außerhalb der MVP-Automation | **Leave** |
| `modules/catalog` | Amazon.de Source Products, immutable Snapshots, Collections und Opportunity Scores | Phase-1-Katalogdomäne | **Extend** |
| `modules/extensiontoken` | Kurzlebige Capture-Grants mit eigener JWT-Audience und Widerruf | Sichere Companion-Extension-Grenze | **Extend** |
| `modules/listingcalc` | Deterministische Preis-/Margin-Berechnung in Integer-Cents | MindBay-Preisbasis ohne Legacy-Floats | **Extend** |
| `modules/listingstudio` | Listing-Drafts, append-only Content, GPSR und Image Assets | Phase-1-Review- und Readiness-Domäne | **Extend** |
| `modules/publication` | Kalender-Slots, idempotente Publication Jobs, eBay-Publish, Reconcile und Listing-Snapshots | Phase-2-Publish-Domäne und alleiniger eBay-Write-Owner | **Extend** |
| `modules/customersync` | Kundennachrichten-Synchronisierung und Worker | Kein MindBay-MVP-Schwerpunkt | **Leave** |
| `modules/demoseed` | Kontrollierte Demo-Daten | Keine Phase-0-Erweiterung | **Leave** |
| `modules/douyinpreflight` | Douyin-Produktionsprüfung | Legacy-Zielmarkt | **Leave** |
| `modules/douyinruntime` | Douyin-Laufzeitstatus und Release Gate | Legacy-Zielmarkt | **Leave** |
| `modules/exportmod` | Exportaufträge | Spätere Reports können wiederverwenden | **Reuse** |
| `modules/files` | Dateiablage, Zugriff und Scan | Bild-/Dokumentreferenzen | **Reuse** |
| `modules/idempotency` | Idempotente Ausführung und Scope-Keys | Publish und Fulfillment zwingend darüber absichern | **Reuse** |
| `modules/imagetask` | Bild-Tasks, Provider, Queue und Qualitätslogik | MindBay-Image-Pipeline erweitern | **Extend** |
| `modules/inventory` | Lokaler Bestand und Worker | eBay-Angebotsbestand additiv anbinden | **Extend** |
| `modules/inventoryread` | Geschützte Plattform-Bestandslesung | Muster für fail-closed Provider-Reads | **Reuse** |
| `modules/inventorysync` | Plattform-Bestandssync, Kalibrierung und Repository | Später eBay-Inventar additiv anbinden | **Extend** |
| `modules/observabilitymod` | Betriebsübersicht und SLO | MindBay-Metriken ergänzen | **Extend** |
| `modules/operationdashboard` | Operations-Dashboard | MindBay-Kennzahlen und Review-Inbox | **Extend** |
| `modules/operationlog` | Verkettetes Operations-/Audit-Log | Audit-Basis für mutierende Commands | **Reuse** |
| `modules/operationtask` | Review-, Freigabe- und Outbox-Ausführung | Muster für optionale Bestätigung, kein eBay-Bypass | **Extend** |
| `modules/order` | Lokale Orders, Status und Plattform-Upsert | eBay-Sale-Modell additiv erweitern | **Extend** |
| `modules/orderexception` | Order-Ausnahmen und Operator-Commands | Fulfillment-Exceptions | **Extend** |
| `modules/ordersync` | Plattform-Ordersync, Cursor, Queue und Worker | eBay Fulfillment/Polling ergänzen | **Extend** |
| `modules/performance` | Performance-Datenmodell | Keine MindBay-Domänenverantwortung | **Leave** |
| `modules/platformcredential` | Verschlüsselte Plattform-Credentials | eBay-Credential-Referenzen und OAuth | **Extend** |
| `modules/pricing` | Preisregeln und Kalkulation, historisch teilweise `float64` | Neue MindBay-Beträge ausschließlich in Integer-Cents | **Extend** |
| `modules/product` | Produktentwurf, Bilder, SKU und AI-Apply | Basis für Source Product und Listing Draft | **Extend** |
| `modules/productcheck` | Publish- und Qualitätsprüfungen | GPSR, Specifics und eBay-Readiness ergänzen | **Extend** |
| `modules/productioncontrol` | Allowlist, Gray/Kill-Switch und Sicherheitsstufen | Globaler MindBay Kill Switch und Live-Gates | **Extend** |
| `modules/productpublish` | Drafts, Publish-Tasks, Queue, Lease und Idempotenz | Legacy-/Douyin-Pfad bleibt bestehen; eBay ist hier bewusst nicht ausführbar | **Leave** |
| `modules/securitymod` | Secret-Rotation und Re-Encryption | eBay-/Browser-Credential-Schutz | **Reuse** |
| `modules/settings` | Typisierte, verschlüsselte Settings-Gruppen | Automation Mode, Limits und Provider-Konfiguration | **Extend** |
| `modules/shop` | Shops, OAuth-Bridges und Plattformkonfiguration | eBay-Shop/OAuth additiv ergänzen | **Extend** |
| `modules/skucandidate` | SKU-Kandidaten und Scoring | eBay SKU bleibt stabile interne Referenz | **Reuse** |
| `modules/storagepublic` | Kontrollierter öffentlicher Storage-Zugriff | Listing-Bildauslieferung hinter Storage-Port | **Reuse** |
| `modules/taskcenter` | Vereinheitlichte Task-/Alert-Sicht | Temporal- und MindBay-Run-Sicht ergänzen | **Extend** |
| `modules/taskreaper` | Rückgewinnung verwaister DB-Leases | Legacy-Redis-Jobs bleiben bestehen | **Reuse** |
| `modules/webhook` | Persistente, verifizierte Webhook-Ingestion | eBay Platform Notifications ergänzen | **Extend** |
| `modules/worker` | Worker-Registry, Heartbeats und Health | Bestehende Redis-Worker unverändert weiterbetreiben | **Reuse** |

## Backend-Provider

### Platform Provider

| Pfad / Registry-Key | Ist-Zustand | MindBay-Entscheidung | Code |
| --- | --- | --- | --- |
| `providers/platform/` | Gemeinsame Registry und Capability-Verträge | eBay implementiert diese Ports; keine Direktaufrufe aus Modulen | **Extend** |
| `providers/platform/amazon/` | Amazon SP-API als **Verkaufskanal**, derzeit OrderSync beta | Nicht als Amazon.de-Quell-Collector verwenden | **Leave** |
| `providers/platform/douyinshop/` | Douyin-Shop-Integration | Legacy unverändert lassen | **Leave** |
| `providers/platform/lazada/` | Lazada-Integration | Legacy unverändert lassen | **Leave** |
| `providers/platform/shopee/` | Shopee-Integration | Legacy unverändert lassen | **Leave** |
| `providers/platform/tiktok/` | TikTok-Shop-Integration | Legacy unverändert lassen | **Leave** |
| Registry `manual` / `mock` | Eingebaute manuelle bzw. Test-Provider | Für isolierte Tests und sichere lokale Flows | **Reuse** |
| Registry `ebay` | Nativer Go-Client für OAuth, Taxonomy, Inventory Item, Offer, Publish und SKU-Reconcile; Providerstatus beta | Nur über Kalender/Temporal verwenden, nicht über generischen Redis-Batch-Publish | **Extend** |
| `modules/monitoring/` | Versionierte Preisregeln, immutable Listing-Snapshots, PriceDecision Plan/Apply und Profit-Ledger | Eigener MindBay-Domain-Service; nutzt eBay nur über den Provider und bestehende Marketplace-Listing-Identität | **Extend** |
| Registry `aliexpress`, `shopify`, `woocommerce`, `temu`, `shein`, `custom` | Geplante Platzhalter | Nicht für MindBay ausbauen | **Leave** |

### AI, Storage und Image

| Pfad | Verantwortung | MindBay-Entscheidung | Code |
| --- | --- | --- | --- |
| `providers/ai/` | Bestehendes AI-Gateway mit mehreren Adaptern | Für Listing Studio und Command Agent wiederverwenden | **Reuse** |
| `providers/storage/` | Local/S3/R2/MinIO/COS/OSS hinter Storage-Verträgen | Originale und abgeleitete Assets darüber speichern | **Reuse** |
| `providers/image/` | Bestehende Bild-Provider einschließlich `removebg` | `rembg` später nur hinter diesem Port evaluieren | **Extend** |
| `providers/email/`, `providers/ocr/` | E-Mail- und OCR-Adapter | Kein Phase-0-Ausbau; bei Bedarf wiederverwenden | **Reuse** |

## Collector-Provider

Der Vertrag bleibt `collector/src/providers/collector-provider.ts`. Collector-Resultate sind normalisierte JSON-Daten mit Rohdatenbezug; Domain-Persistenz erfolgt ausschließlich im Go-Service.

| Pfad | Ist-Zustand | MindBay-Entscheidung | Code |
| --- | --- | --- | --- |
| `providers/collector-provider.ts` | `CollectorProvider` mit `canHandle` und `collect` | Unverändert als Quell-Port verwenden | **Reuse** |
| `providers/registry.ts` | Stabile Registrierung aller Quellen | `sourceAmazon` später additiv registrieren | **Extend** |
| `providers/shared/` | Gemeinsame Page-Guards | Nur wirklich quellenübergreifende Logik aufnehmen | **Reuse** |
| `providers/source1688/` | 1688-Quelle und Login-/Parser-Logik | Legacy, nicht zum Amazon-Default machen | **Leave** |
| `providers/sourceAliExpress/` | AliExpress-Quelle | Legacy | **Leave** |
| `providers/sourceCustom/` | Regelbasierte Custom-Quelle und Normalisierung | Wiederverwendbare Normalisierungsbausteine prüfen | **Reuse** |
| `providers/sourcePinduoduo/` | Pinduoduo-Quelle | Legacy | **Leave** |
| `providers/sourceTaobaoTmall/` | Taobao/Tmall-Quelle | Legacy | **Leave** |
| `providers/stub/` | SHEIN/Temu-Platzhalter | Nicht für MindBay ausbauen | **Leave** |
| `providers/sourceAmazon/` | Amazon.de-Produktseitenparser mit ASIN-/EUR-Normalisierung und Fehlerklassen | Phase-1-Quelladapter; keine Domain-DB-Schreibrechte | **Extend** |

## Admin-Oberflächen

| Pfad | MindBay-Nutzung | Entscheidung |
| --- | --- | --- |
| `admin/src/pages/Product/` | Catalog, Collections und Listing Studio additiv entwickeln | **Extend** |
| `admin/src/pages/Collect/` | Amazon Discovery und Capture additiv entwickeln | **Extend** |
| `admin/src/pages/Orders/` | eBay Sales und Fulfillment Cases additiv entwickeln | **Extend** |
| `admin/src/pages/Shops/` | eBay OAuth und verschlüsselte Shop-Verknüpfung | **Extend** |
| `admin/src/pages/Settings/` | Automation Mode, Limits und Provider-Konfiguration | **Extend** |
| `admin/src/pages/Dashboard/` | MindBay Operations-, Profit- und Review-Sicht | **Extend** |
| `admin/src/pages/MindBay/{Monitoring,Profit}` | Repricing-Inbox, Regelversionen und getrennte Expected-/Actual-Ledger-Sicht | **Extend** |
| Übrige `admin/src/pages/*` | Bestehende TradeMind-Funktionen | Unverändert wiederverwenden oder als Legacy belassen; kein zweites Designsystem | **Reuse** |

## Scope-, API- und Datenkonventionen

- `workspace_id` ist der Begriff des MindBay-Blueprints; im bestehenden Service-Layer bezeichnet er denselben Mandanten wie `tenant_id`.
- Der aktuelle Tenant-Typ ist `int64`. Neue MindBay-Tabellen dürfen die physische Spalte `workspace_id` verwenden, müssen aber exakt denselben ID-Raum und dieselben Scope-Guards aus `pkg/adminperm` und `pkg/tenantquery` nutzen.
- Es gibt keine Migration bestehender Tabellen und keinen dritten Scope-Key in Phase 0. DTOs dürfen den Produktbegriff `workspaceId` nur an einer ausdrücklich definierten MindBay-Grenze verwenden; intern wird einmalig auf den vorhandenen Tenant-Scope abgebildet.
- Bestehende Backend-Routen bleiben unter `/api/v1` kompatibel. Spätere MindBay-Routen werden additiv als klar abgegrenzte `/v1/...`-Gruppe oder kompatible additive Untergruppe registriert; Phase 0 ändert keine Route.
- Neue MindBay-Geldwerte sind `int64`-Cents plus ISO-4217-Währung. Historische `float64`-Felder werden nicht stillschweigend umgedeutet.

## Lizenz- und Reuse-Register

Die Commit-Spalte fixiert den in Phase 0 lokal geprüften Referenzstand. Eine spätere tatsächliche Codeübernahme benötigt erneut einen Diff- und Lizenzcheck sowie einen ADR-Nachtrag.

| Quelle | Lokaler Stand | Lizenz | Zulässige Nutzung | Bewusst nicht übernommen |
| --- | --- | --- | --- | --- |
| `tools/ebay-api` | `10.0.1`, `e20388bcf49c` | MIT | Contract-Fixtures und Sandbox-Prototypen | Kein zweites TypeScript-Backend, keine Domain-Ownership |
| `tools/ebay-mcp` | `1.14.3`, `fe295c3da1d6` | MIT | Tool-Schemas und Agent-DX als Referenz | Kein direkter eBay-Write-Bypass |
| `tools/em-ebay-repricer` | `0.1.0`, `ffef71e282b8` | MIT | Plan/Apply-, Freshness- und Pending-Decision-Muster | Keine Spree-, Elasticsearch- oder GCS-Annahmen |
| `tools/openoms` Anwendung | `b205e87a67b4` | Elastic License 2.0 | Nur Architektur-Research | Anwendung weder forken noch kopieren; SDK-Unterpakete nur nach separatem Lizenzcheck |
| `tools/wxt` | `68719cd3d0f2` | MIT | Referenz/Grundlage für Manifest-V3-Extension | Kein Copy-Paste in Phase 0, keine Secrets im Content Script |
| `tools/rembg` | `e14ee119f1d5` | MIT | Späterer lokaler Image-Adapter hinter vorhandenem Image-Port | Keine direkte Frontend-Anbindung |
| `tools/sdk-typescript` | `f8ced67a7e1e` | MIT | Temporal-Workflow-/Activity-SDK nach Aufnahme in das Hauptrepo | Kein produktiver Code im externen Lern-Scaffold |

## Geschützte Repository-Grenzen

| Pfad | Regel | Code |
| --- | --- | --- |
| `../MindBay/` | Nicht verwandtes Aktienanalyseprojekt; keine Importe oder Übernahmen | **Do-not-touch** |
| `../tools/openoms/` Anwendung | ELv2-Referenz; kein Fork und kein Anwendungscode im Hauptrepo | **Do-not-touch** |
| `../temporal-worker/` | Lern-Scaffold; produktive Workflows entstehen später im Hauptrepo | **Do-not-touch** |

## Festgelegte Architekturentscheidungen

- [ADR-0001: Nativer eBay Sell API Client im Go-Backend](adr/0001-ebay-go-client.md)
- [ADR-0002: Hybride Temporal- und Redis-Orchestrierung](adr/0002-temporal-hybrid-orchestration.md)
- [ADR-0003: Amazon.de als eigener Collector-Quelladapter](adr/0003-amazon-source-collector.md)
- [ADR-0004: WXT Companion Extension mit getrennter API-Audience](adr/0004-wxt-companion-extension.md)
