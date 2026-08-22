# Detaillierter Implementierungsplan

## Programmregeln vor Phase 1

1. Von `dev` in einem `feat/*`-Branch arbeiten; der aktuelle Branch und uncommitted Admin-Änderungen werden vor jedem Workstream geprüft.
2. Bestehende MindBay-Routen `/v1` sowie Legacy `/api/v1` bleiben kompatibel. Jede neue Write-Route erhält Auth, Tenant-Scope, DTO-Validation, Idempotency und Audit.
3. Keine Live-Provider-Schreiboperation, bevor ein manueller Sandbox-Abnahmelauf erfolgreich war. Standard ist `DRY_RUN` plus Sandbox.
4. Jeder neuen Tabelle/Queue/Provider liegen Migration-, State-Machine-, Fake-Adapter-, Contract- und betroffene Admin-Tests zugrunde. DB/Redis-Tests nur mit isolierten Test-URLs.

## Phase 0 — belastbare Baseline und Entscheidungsprotokoll

**Lieferobjekte:** ein kurzer ADR-Nachtrag zu Produktgrenze und Fulfillment-Modell; keine Produktcode-Massenänderung.

- Bestehende MindBay Unit-, Contract-, Collector- und Admin-E2E-Suites in CI bewahren; lokal nur die direkt betroffenen statischen Checks ausführen.
- Konfiguration und Dokumentation auf Wahrheit prüfen: `AUTOMATION_MODE`, `EBAY_ENV`, Temporal-Service-Token, OAuth RuName, Marketplace `EBAY_DE`.
- Eine Sandbox-Checkliste definieren: OAuth, Privileges, Policies, Location, Kategorie, Aspekte, GPSR, eindeutige SKU, Reconcile nach Timeout.
- Produktentscheidung festhalten: zunächst operator-gesteuertes Amazon-Retail-Fulfillment oder nur Listing/Monitoring. Automatischer Checkout ist kein impliziter Teil dieser Roadmap.

**Done wenn:** Der aktuelle Flow und seine Grenzen dokumentiert sind; keine Tests, Provider oder Legacy-Pfade wurden vorschnell gelöscht.

## Phase 1 — eBay-Publication abschließen und härten

**Owner:** `shop`, `publication`, `listingstudio`, `providers/platform/ebay`, Temporal.

1. OAuth- und Shop-Connection durch Fake-HTTP-Tests absichern: State-Bindung, verschlüsselte Speicherung, Scope, Token-Refresh/reauthorization, 401/403 und keine Tokenlogs.
2. Taxonomy-, Policies- und Location-Synchronisation als Cache mit `fetched_at`/Freshness implementieren. Die Readiness-Prüfung blockiert bei fehlenden oder veralteten Pflichtdaten.
3. Publication-Payload vervollständigen: SKU, condition, title, description, images, aspects, category, cents/currency, quantity, policies und location.
4. Happy-Path und Unknown-Result-Verhalten testen: Inventory Item -> Offer -> Publish; bei Timeout/5xx erst Reconcile per SKU/Offer-ID, danach höchstens begrenzter Retry.
5. Admin zeigt Sandbox und DRY_RUN deutlich; `Approve` verlangt die vorhandene explizite Freigabe und sendet genau einen idempotenten Command.

**Tests:** Go Provider mit `httptest`; Publication Service-State-Machine; Migrationstest; `/v1/calendar`-/`/v1/publications`-Contract; vorhandene MindBay-Phase-2 E2E aktualisieren.  
**Done wenn:** Ein manuell freigegebener Sandbox-Run erzeugt genau ein Inventory Item, Offer, Listing und lokales Listing; ein wiederholter Apply/Retry erzeugt kein Duplikat.

## Phase 2 — Source-Freshness, Bestand und Preisgrundlage

**Owner:** `sourceAmazon`, `catalog`, `collect`, `inventorysync`, `monitoring`, `listingcalc`.

1. Amazon-Parser und offizielle Lese-API als austauschbare Source-Strategien führen. Marketplace wird nie geraten; rohe fremde Strukturen enden am Collector-Port.
2. Snapshot-Felder/Normalisierung festlegen: source URL/ID, EUR-Cents, Versand, Availability/Stock Hint, Verkäufer, Lieferfenster, Variantendaten, Capture-Fehler und Capture-Zeit.
3. Stale-Policy implementieren: Scheduler/Queue triggert nur Erfassung; Service schreibt neuen immutable Snapshot. Fehlgeschlagene Captures verändern den letzten guten Snapshot nicht, erzeugen aber Alert/Task.
4. Inventory-Policy konfigurieren: bei `OUT_OF_STOCK` eBay-Quantity auf null; beim Re-stock nur definierter Puffer. Jede Änderung läuft über vorhandene Publication-/SKU-Referenzen und den eBay-Provider.
5. Price-Input deterministisch machen: Kosten, optional source shipping, eBay-Gebührenannahme, Zielmarge. Alle Werte in Cents; kein unbemerkter Übergang auf Legacy-Floats.

**Tests:** Collector fixtures für Preis/Verfügbarkeit/fehlende Felder; Service für stale/out-of-stock; Redis/Queue- und idempotente Inventory-Task-Tests; Provider-Fake für Bulk/Single-Update.  
**Done wenn:** Ein geänderter Source-Snapshot eine nachvollziehbare, deduplizierte Inventory-/Preisentscheidung erzeugt und DRY_RUN niemals eBay mutiert.

## Phase 3 — Repricing-Entscheidungen produktreif machen

**Owner:** `monitoring`, eBay provider, Admin Monitoring.

1. Bestehende `PriceRule` als immutable Version beibehalten. Regel enthält Min-Marge, Rundung, Cooldown, maximale Änderung, Source-Freshness, Automation-Level und Blockgründe.
2. Monitor Run bindet Source Snapshot, Marketplace Listing Snapshot und Regelversion. Ergebnis ist `NO_CHANGE`, `PROPOSED`, `AUTO_APPLIED`, `BLOCKED_MARGIN`, `BLOCKED_POLICY` oder `BLOCKED_COOLDOWN`.
3. `Apply` verlangt Idempotency Key, prüft Aktualität erneut und schreibt Request-/Response-Artefakte redigiert. Bei unbekanntem Ergebnis wird das Offer gelesen/reconciled, nicht blind erneut geschrieben.
4. UI liefert Inboxes für proposed/blocked/failed, den Rechenweg in Cents und die Regelversion. Expected und Actual Profit bleiben getrennt.

**Done wenn:** Jede Preisänderung auf einen Snapshot und eine Regelversion zurückführbar ist und ein paralleler/erneuter Apply nicht doppelt aktualisiert.

## Phase 4 — Orders und tatsächlicher Gewinn

**Owner:** `ordersync`, `order`, `orderexception`, NEW `profit`, eBay provider.

1. Fulfillment-API-Ingestion mittels Cursor, Workspace/Shop-Scope und idempotentem Upsert. Rohpayload bleibt referenziert/redigiert; keine ungebundenen SKU-Matches als Erfolg markieren.
2. Ausnahmen sichtbar machen: unbekannte SKU, Teilzustellung, Storno, Rückerstattung, verspätete Source-Beschaffung. Operator-Kommandos bekommen Audit und Statusmachine.
3. `order_financials` und `order_fee_lines` additiv migrieren. Fee Lines werden per eBay-Transaktions-ID dedupliziert; Berechnung ist versioniert und wiederholbar.
4. Actual Profit entsteht nur aus Sale-/Fee-Daten plus dokumentierten Kosten. Eine Schätzung bleibt Expected, nie „tatsächlich“.

**Tests:** Cursor- und Upsert-Duplikate, Ausnahmen, fehlende Gebühren, Rückerstattung, gleiche externe Fee zweimal, Transaction Rollback, Profit-Report Contract.  
**Done wenn:** Eine eBay-Order von Item bis Fee Line nachvollziehbar ist und die Summe der Fee Lines/Costs den ausgewiesenen Gewinn erklärt.

## Phase 5 — Research und Candidate Inbox

**Owner:** NEW `research`, `catalog`, `collect`, Admin.

1. `research_runs` als asynchrone, begrenzte Suche anlegen; Request enthält Quelle, Query, Filter und Budget/Limits.
2. `research_candidates` referenziert Quelle/Snapshot; ein Kandidat wird erst nach Review/Import zum Catalog Product. Dedup erfolgt per Marketplace + stabilem Source-Identifier, nicht nur Titel.
3. Score als erklärbare Komponenten speichern: Demand, Competition, Margin Estimate, Compliance-/Brand-Risk, Data Completeness. Keine KI-Score-Blackbox ohne Inputs.
4. Kandidaten-UI liefert Filter, Faktenquellen, Blockgründe, Import als idempotenten Command und Loading/Empty/Error/Readonly.

**Done wenn:** Ein Run erzeugt reproduzierbare Kandidaten, doppelte Imports sind sicher und ein Operator sieht, warum ein Kandidat empfohlen oder blockiert ist.

## Phase 6 — Produktkalender und UX-Konsolidierung

**Owner:** `publication`, MindBay Admin pages.

1. `calendar_slot`/`publication_job` statt einer neuen Schedule-Tabelle als Kern verwenden. Nur falls echte Serienplanung über den vorhandenen Slotumfang hinausgeht, ein dünnes `schedule_template` einführen, das Slots erzeugt und keine eBay-Logik enthält.
2. Preview ist pure; Apply ist idempotent und konfliktfrei. Tageslimit, Zeitfenster und Zeitzone werden im Workspace/Shop konfiguriert und bei Apply materialisiert.
3. Endnutzer-Navigation: Dashboard, Research, Products, Listings, Calendar, Orders, Profit, Automation, eBay Account, Settings. Task Center, Workers, Logs und Ops bleiben intern/advanced.
4. Jede Page folgt existierenden `TmPageContainer`-/`TmProTable`-/Token-Konventionen, hat fünf Viewports und keinen Root-horizontal-overflow.

**Done wenn:** Ein Nutzer kann Ready Listings previewen, sicher reservieren, Status/Fehler sehen und keine Seite suggeriert ein eBay-Ergebnis vor tatsächlichem Publish.

## Phase 7 — Invoice Archive und Agent Gateway

**Owner:** NEW `invoice`, NEW `agent`, Storage Provider.

1. Google Drive erst als Storage-Provider hinter vorhandener Storage-Schnittstelle implementieren: OAuth/Credentials verschlüsselt, keine Drive-URL als einzige Wahrheit, idempotenter Monatsjob, Checksumme und Retention-Policy.
2. Archive sammelt nur freigegebene, tenant-scoped Daten und ist wiederholbar. Dateien/Exportdaten werden über `files`/Storage referenziert.
3. Agent startet read-only. Alle Tools benutzen Services, Scope und Audit; Write-Tools sind explizit allowlisted, idempotent und durch gleiche Freigaberegeln wie die UI geschützt.
4. Agent-Tool-Ausgaben sind strukturierte, redigierte Domänenantworten. Kein unbounded Prompt, keine ungeprüfte Tool-Kette, keine Credentials im Toolkontext.

**Done wenn:** Ein Monatsarchiv trotz Wiederholung nur einmal erzeugt wird und ein Agent keine Aktion ausführen kann, die ein gleichberechtigter UI-Nutzer nicht mit denselben Freigaben ausführen dürfte.

## Phase 8 — kontrollierter Legacy-Rückbau

Erst wenn der eBay-first Flow, Tests und Produktionserfahrung stabil sind: Legacy-Provider aus Produktkonfiguration und Navigation entfernen, Referenzen/Migrationen/Tests bewusst migrieren und anschließend physisch löschen. Kein großflächiges Löschen in einem Feature-PR.

## Durchgehende Qualitäts- und Abnahmekriterien

- Neue Provider, Queue-, Scheduler-, Migrations- und Cross-Module-Änderungen bekommen Deep Review: Zustandsübergänge, Transaktionen, Retry, Timeout, Rate Limit, Lock/Lease, Idempotenz, Scope, Tokenredaktion und Rollback.
- API: Handler, Service, DTO, Berechtigung, Contract Fixture, Admin Service/Mock und Dokumentation gemeinsam ändern.
- Admin: vorhandene E2E-Suites erweitern; Schreibrequests per Write Guard deklarieren, Cancel sendet null, Confirm exakt einen Request.
- Lokal: nur passende Checks wie `pnpm check:dev`, `pnpm check:ui-copy --strict`, `pnpm build:admin`, `pnpm build:collector`, `pnpm architecture:check` und gezielte Go-Tests. Vollständige Regression wird über GitHub Actions ausgeführt.
- Manuelle Abnahme bleibt erforderlich für Sandbox/Production OAuth, eBay Policies/Locations, Publish, Real Orders, Finance und UX.

## Hauptrisiken und Gegenmaßnahmen

| Risiko | Gegenmaßnahme |
| --- | --- |
| Doppeltes Listing nach Timeout | Stabiler SKU/Offer-Reconcile vor Retry; Publication als einziger Write-Owner. |
| Stale Amazon-Daten verursachen Fehlpreise | Freshness-Gate, Snapshot-Referenz, Block statt Auto-Apply. |
| Falsche Compliance-/Item-Specifics | Pflichtaspekte/GPSR blockieren; Operator bestätigt, AI erfindet nicht. |
| Gewinnzahlen sind irreführend | Expected und Actual getrennt; Fee Lines und Quellenversionen behalten. |
| Agent umgeht Sicherheit | Service-only Tools, RBAC, Audit, Idempotenz, approvals, read-first rollout. |
| Frühes Löschen schadet Legacy/CI | zuerst ausblenden, erst nach Ersatz samt Testmigration löschen. |
