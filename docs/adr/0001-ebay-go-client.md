# ADR-0001 Nativer eBay Sell API Client im Go-Backend

Status: Accepted  
Datum: 2026-08-15

## Kontext

TradeMind besitzt eine Go-Platform-Registry, Shop-/OAuth-Modelle, verschlüsselte Token-Persistenz, Idempotenz und Audit-Module. eBay ist in der Registry aktuell lediglich als `StatusPlanned` eingetragen; ein ausführender Adapter fehlt. Die lokalen Referenzen `tools/ebay-api` und `tools/ebay-mcp` sind TypeScript-Projekte. Die OpenOMS-Anwendung steht unter Elastic License 2.0.

MindBay benötigt später eBay Inventory, Offer, Fulfillment, Finances, Taxonomy/Metadata und Media. Plattformwrites müssen denselben Tenant-Scope, RBAC-, Audit- und Idempotenzpfad wie bestehende TradeMind-Commands verwenden.

## Entscheidung

Wir implementieren in Phase 2 einen nativen Go-HTTP-Client unter `backend/internal/providers/platform/ebay/` hinter den vorhandenen Platform-Provider-Verträgen. Domänenmodule rufen weder eBay REST noch ein TypeScript-SDK direkt auf.

- `EBAY_ENV` wählt ausschließlich `sandbox` oder `production`; Default ist `sandbox`.
- OAuth verwendet Authorization Code Grant. Client-Credentials bleiben in der verschlüsselten Plattformkonfiguration, Shop Access-/Refresh-Tokens im vorhandenen `shop_auth_tokens`-/Secrets-Pfad mit AES-GCM.
- Der Adapter fordert nur die für aktivierte Capabilities nötigen offiziellen Scopes an. Die Capability-Liste umfasst `sell.inventory`, `sell.fulfillment`, `sell.finances` sowie die erforderlichen Taxonomy-/Metadata-/Media-Berechtigungen; konkrete Scope-URIs werden bei Implementierung gegen die offizielle eBay-Spezifikation fixiert und getestet.
- Refresh erfolgt vor Ablauf mit einem per Shop und Workspace begrenzten Lock. Ein ungültiger Refresh Token führt zu `AUTH_REQUIRED`; Tokens und OAuth-Antworten werden nie vollständig geloggt.
- Jeder mutierende Call erhält einen stabilen internen Idempotenz-/Correlation-Kontext. Timeout oder unbekannter Remote-Erfolg erzeugt keinen zweiten SKU-/Offer-Versuch, sondern eine Reconciliation.
- Fehler werden zentral auf die Blueprint-Klassen abgebildet: 429/temporäre 5xx/Netzwerk auf `RETRYABLE_TRANSIENT`, 401/403 bzw. Tokenfehler auf `AUTH_REQUIRED`, unbekannte Response-Schemas auf `API_CONTRACT_CHANGED`, Policy-/Validierungsfehler auf `POLICY_BLOCKED` oder `TERMINAL_VALIDATION`, lokale Duplicate-Guards auf `IDEMPOTENCY_CONFLICT`.
- `tools/ebay-api` darf Golden-Fixtures und Sandbox-Prototypen liefern. `tools/ebay-mcp` darf Tool-Schemas inspirieren. Beide besitzen weder Fachzustand noch Schreibrechte im Produktivpfad.
- Die OpenOMS-Anwendung wird nicht kopiert oder geforkt. Separat lizenzierte Unterpakete benötigen vor einer Übernahme einen eigenen Lizenz- und API-Fit-Nachtrag.

## API-Kompatibilität

Bestehende Go-Routen unter `/api/v1` bleiben unverändert. Neue MindBay-Endpunkte werden später additiv in einer klar abgegrenzten `/v1/...`-Gruppe oder als kompatible additive Gruppe registriert. Es gibt in Phase 0 keine Route und keinen Plattformwrite.

## Übergreifender Phase-0-Anhang: Workspace und Tenant

`workspace_id` im MindBay-Blueprint ist semantisch identisch mit dem bestehenden `tenant_id`. Der aktuelle Service-Scope ist `int64` und wird über `backend/internal/pkg/adminperm` sowie `backend/internal/pkg/tenantquery` durchgesetzt.

Neue MindBay-Tabellen dürfen `workspace_id` als physischen Spaltennamen verwenden, müssen aber denselben `int64`-ID-Raum und dieselbe Service-Layer-Scope-Quelle nutzen. Bestehende Tabellen werden nicht umbenannt. Es wird kein dritter Scope-Key und keine parallele Workspace-Auflösung eingeführt.

## Alternativen

1. `tools/ebay-api` als zweites TypeScript-Backend betreiben: verworfen wegen doppelter Domain-, Secret-, RBAC- und Transaktionsverantwortung.
2. OpenOMS-Anwendung forken: verworfen wegen Lizenzgrenze und unpassender Gesamtarchitektur.
3. eBay-Aufrufe direkt in `productpublish`, `ordersync` und `webhook` verteilen: verworfen wegen duplizierter Auth-, Fehler- und Auditlogik.

## Konsequenzen

- Go bleibt alleiniger Eigentümer mutierender eBay-Operationen.
- Zusätzlicher Aufwand entsteht für REST-DTOs, Pagination, Fehler-Mapping, Rate-Limits und Contract-Fixtures.
- Sandbox und Production teilen Code, aber niemals Base URL, Credentials oder gespeicherte Tokens.
- Phase 2 muss Provider-, OAuth-, Contract-, Idempotenz- und Reconciliation-Tests ergänzen.

## Quelle/Commit/Lizenz

- `tools/ebay-api`: Version 10.0.1, Commit `e20388bcf49c`, MIT; nur Fixtures/Prototypen.
- `tools/ebay-mcp`: Version 1.14.3, Commit `fe295c3da1d6`, MIT; nur Tool-Schema-Referenz.
- `tools/openoms`: Commit `b205e87a67b4`, Elastic License 2.0; nur Architektur-Research.

## Bewusst nicht übernommen

- Kein TypeScript-Service als Listing-Eigentümer.
- Kein direkter MCP- oder Admin-Write zu eBay.
- Kein OpenOMS-Anwendungscode.
- Keine produktiven Credentials, OAuth-Flows oder Sandbox-Calls in Phase 0.
