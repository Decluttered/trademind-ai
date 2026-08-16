# ADR-0004 WXT Companion Extension mit getrennter API-Audience

Status: Accepted  
Datum: 2026-08-15

## Kontext

MindBay soll Produktkontext auf Amazon.de und später kontrollierten eBay-Seiten erfassen können. Das Hauptrepo besitzt noch kein Extension-Paket. `tools/wxt` steht als MIT-lizenzierte Manifest-V3-Referenz zur Verfügung. Admin-JWT, Plattform-Credentials und Browser-Sessions dürfen nicht in Content Scripts gelangen.

## Entscheidung

- Die Companion Extension wird ab Phase 1 mit WXT, Manifest V3, TypeScript und React unter `trademind-ai/extension/` entwickelt.
- Phase 0 registriert nur ein dependency-freies, privates pnpm-Workspace-Scaffold. Es enthält noch keinen WXT-Code, Build und Browser-Entrypoint.
- Die Extension darf Amazon-/eBay-Seitenkontext lesen, dem Nutzer den erfassten Inhalt anzeigen und Captures ausschließlich an eine versionierte Go-Extension-API senden.
- Die Extension schreibt nicht in PostgreSQL, ruft keine eBay Sell API auf und führt keinen Amazon-Checkout aus.
- Extension-Tokens erhalten eine eigene Audience und minimale Capture-Scopes. Sie sind weder Admin-JWT noch eBay-/Amazon-Token. Kurzlebige Tokens liegen nur im Extension-Service-Worker; Content Scripts erhalten keine Plattform-Secrets.
- Host Permissions werden auf die tatsächlich unterstützten Amazon.de-/eBay.de-Seiten begrenzt. Neue Hosts erfordern Review und eine dokumentierte Capability.
- Capture-Commands tragen Workspace-Scope, Correlation-ID und Idempotenz-Key. Der Go-Service validiert URL, Payload-Größe, Produktidentität und Berechtigung vor jeder Persistenz.
- WXT wird als Framework-Abhängigkeit erst in Phase 1 aus dem öffentlichen Paket-Registry aufgenommen. `tools/wxt` dient als geprüfte Referenz, nicht als manuell kopierter Fork.

## Alternativen

1. Plain Manifest V3 ohne Framework: möglich, aber verworfen zugunsten konsistenter TypeScript-/React-Builds und Browser-Zielkonfiguration.
2. Extension als Teil des Admin-Bundles: verworfen wegen anderer Runtime, Permissions und Token-Audience.
3. Direkte eBay-/DB-Aufrufe aus der Extension: verworfen wegen Secret-, RBAC-, Audit- und Idempotenz-Bypass.
4. Den vollständigen `tools/wxt`-Quellbaum kopieren: verworfen wegen Wartungs- und Update-Drift.

## Konsequenzen

- Das neue Workspace-Paket bleibt in Phase 0 absichtlich ohne Runtime-Abhängigkeiten und Verhalten.
- Phase 1 muss Extension-Architekturregeln, API-Contract, Token-Ausgabe/-Widerruf, Content-Script-Isolation und Mock-E2E-Tests ergänzen.
- Eine Store-Veröffentlichung, Signierung oder Browser-Distribution ist nicht Bestandteil von Phase 0.

## Quelle/Commit/Lizenz

- `tools/wxt`: Commit `68719cd3d0f2`, MIT; Framework- und Strukturreferenz.

## Bewusst nicht übernommen

- Kein WXT-Quellcode und keine Dependency in Phase 0.
- Keine Admin-, eBay- oder Amazon-Credentials.
- Keine direkten Plattformwrites, keine Datenbankverbindung und kein Checkout.
