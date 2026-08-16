# ADR-0002 Hybride Temporal- und Redis-Orchestrierung

Status: Accepted  
Datum: 2026-08-15

## Kontext

TradeMind betreibt heute Redis-Queues und DB-Leases unter anderem für `collect`, `productpublish`, `ordersync`, Bild- und Customer-Jobs sowie `taskreaper`. MindBay benötigt zusätzlich langlebige, wiederaufnehmbare Abläufe mit Timern, Signalen, Reconciliation und Kompensationen. Der Workspace enthält außerhalb des Hauptrepos ein TypeScript-Temporal-Starterprojekt, das kein Produkt- oder Datenmodell besitzt.

Eine sofortige Migration der Legacy-Worker würde funktionierende Douyin-/SEA-Flows gefährden. Ein unabhängiges Temporal-Backend würde hingegen Fachzustand und Writes duplizieren.

## Entscheidung

Wir verwenden ein hybrides Modell:

- Bestehende Redis-Worker, Queues und DB-Leases bleiben für bestehende Legacy-Jobs unverändert.
- Neue langlebige MindBay-Flows werden schrittweise mit Temporal umgesetzt; erster produktiver Flow ist `PublishListingWorkflow` in Phase 2.
- Der produktive TypeScript-Worker entsteht im Hauptrepo unter `trademind-ai/temporal/` und wird dann als eigener pnpm-Workspace registriert. `../temporal-worker/` bleibt Lern-Scaffold und wird nicht zum Produktivpfad.
- Workflow-Code ist deterministisch. Uhrzeit, Zufall, Netzwerk, Datenbank, eBay, Browser und LLM liegen ausschließlich in Activities.
- Activities verändern Fachzustand nur über versionierte interne Go-Commands/Services. Sie besitzen kein eigenes Repository und schreiben nicht direkt in MindBay-Tabellen.
- Activities erhalten Start-to-close-/Schedule-to-close-Timeouts, Heartbeats für lange Browseraktionen und begrenzte Retry-Policies nach interner Fehlerklasse. `AUTH_REQUIRED`, `UI_CHANGED`, `CAPTCHA`, Policy- und Validierungsfehler werden nicht blind wiederholt.
- Workflow-ID und Command-Idempotenz-Key werden stabil aus Workspace und Aggregat abgeleitet. Redis-Consumer und Temporal-Activity dürfen nie gleichzeitig Eigentümer derselben Mutation sein.
- Temporal Server wird frühestens in Phase 2 lokal per Docker aktiviert. `TEMPORAL_ADDRESS=localhost:7233` ist in Phase 0 nur ein sicherer Platzhalter; kein Prozess verbindet sich dadurch automatisch.
- Die im Blueprint festgelegten Workflow-Namen bleiben die einzige Namensfamilie: `DiscoveryWorkflow`, `ListingPreparationWorkflow`, `CalendarPlanningWorkflow`, `PublishListingWorkflow`, `MonitoringWorkflow`, `RepriceWorkflow`, `SaleReconciliationWorkflow`, `FulfillmentWorkflow`, `DailyOperationsWorkflow`.

## Alternativen

1. Alle Redis-Worker sofort nach Temporal migrieren: verworfen wegen hoher Regression und fehlendem Nutzen für kurze Legacy-Jobs.
2. Temporal als zweites Backend mit eigener Datenbank betreiben: verworfen wegen doppeltem Fachzustand und inkonsistenten Transaktionen.
3. Alle MindBay-Flows weiterhin nur über Redis/DB-Leases abbilden: verworfen, weil langlebige Timer, Signals, Replays und Kompensationen damit unnötig selbst implementiert würden.
4. Produktiven Worker außerhalb des Hauptrepos im Starter belassen: verworfen, weil Versionierung, CI und Architekturgrenzen auseinanderlaufen würden.

## Konsequenzen

- Zwei Orchestrierungsmechanismen existieren bewusst, aber mit getrenntem Ownership: Legacy kurzlebig auf Redis, neue langlebige MindBay-Flows auf Temporal.
- Phase 2 benötigt eine interne Go-Command-Grenze für Activities, Temporal-Test-Environment-Tests und Docker-/Health-Dokumentation.
- Eine spätere Migration einzelner Redis-Flows erfordert einen eigenen ADR-Nachtrag mit Cutover-, Idempotenz- und Rollback-Plan.
- Phase 0 fügt weder Temporal-Abhängigkeiten noch Runtime-Code hinzu.

## Quelle/Commit/Lizenz

- `tools/sdk-typescript`: Commit `f8ced67a7e1e`, MIT; vorgesehene SDK-Referenz für den späteren Worker.
- `../temporal-worker/`: lokales Lern-Scaffold; keine Übernahme in Phase 0 und keine eigenständige Produktdomäne.

## Bewusst nicht übernommen

- Keine `node_modules`, Lockfiles, Beispiel-Activities oder Beispiel-Workflows aus dem externen Starter.
- Keine direkte Datenbanklogik im Temporal-Worker.
- Keine Migration oder Abschaltung vorhandener Redis-Queues in Phase 0.
