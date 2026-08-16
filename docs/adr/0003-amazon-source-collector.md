# ADR-0003 Amazon.de als eigener Collector-Quelladapter

Status: Accepted  
Datum: 2026-08-15

## Kontext

Im Backend existiert `providers/platform/amazon/` als Amazon Selling Partner API Provider für Amazon als **Verkaufskanal**. Der Node-Collector besitzt dagegen den Port `CollectorProvider` und Quellen für 1688, AliExpress, Pinduoduo, Taobao/Tmall, Custom sowie Platzhalter. Amazon.de als Produktquelle fehlt.

MindBay benötigt Amazon.de-Suchergebnisse, Produktseiten, Varianten, Preis, Verfügbarkeit, ASIN, Bilder und belegte Produktfakten. Collector und Go-Service müssen dabei getrennte Verantwortungen behalten.

## Entscheidung

- Amazon.de wird in Phase 1 als eigener Adapter `collector/src/providers/sourceAmazon/` implementiert und additiv in `collector/src/providers/registry.ts` registriert.
- Der Adapter implementiert den bestehenden `CollectorProvider`; der Registry-Key wird `amazon` und die normalisierte Source `amazon.de`.
- Offizielle SP-API-/Product-Advertising-API-Daten werden genutzt, wenn passende Leseberechtigungen vorhanden sind. Playwright ist der gleichberechtigte Fallback; die Companion Extension kann kontrollierte Captures liefern.
- Der Collector erfasst, validiert und normalisiert. Er gibt normalisiertes JSON sowie einen begrenzten Rohdaten-/Artefaktverweis zurück und schreibt niemals direkt in die Domain-Datenbank.
- Der Go-Service unter dem bestehenden `collect`-/Produktpfad prüft Scope, dedupliziert mit `amazon.de + asin` und persistiert unveränderliche Snapshots. Ein Source Product zeigt auf den aktuellen Snapshot.
- Drittanbieterfelder bleiben im Amazon-Adapter. Gemeinsame Preis-, URL- und Qualitätslogik darf nur nach nachgewiesener Quellenunabhängigkeit in bestehende Normalizer verschoben werden.
- Crawlee darf später Queue, Dedupe und Politeness innerhalb des bestehenden Collectors ergänzen. Es entsteht kein zweiter Crawler-Service.
- `backend/internal/providers/platform/amazon/` bleibt für das MVP unverändert und wird nicht zum Quell-Collector umgebaut.
- Der spätere `AmazonRetailProvider` für Checkout ist ein separater Fulfillment-/Browser-Adapter und teilt weder dieses Source-Interface noch dessen Zustandsmodell.

## Alternativen

1. Den SP-API-Verkaufsprovider als Quelle erweitern: verworfen, weil Verkaufsplattform- und Lieferantenrollen vermischt würden.
2. Amazon direkt im Go-Service scrapen: verworfen, weil Browser-/Parser-Verantwortung aus dem bestehenden Collector herausgelöst würde.
3. Einen eigenständigen Crawlee-Service anlegen: verworfen wegen doppelter Queue-, Browser- und Normalisierungsarchitektur.
4. Die Extension direkt in PostgreSQL schreiben lassen: verworfen wegen Scope-, Validierungs- und Audit-Bypass.

## Konsequenzen

- Drei Amazon-Rollen bleiben ausdrücklich getrennt: Verkaufsplattform-Provider, Source Collector und späterer Retail-Fulfillment-Provider.
- Phase 1 benötigt Contract-Fixtures, URL-/ASIN-Normalisierung, fehlende-Felder-Tests, Rate-Limits und sichere Browser-Artefakte.
- Bestehende 1688-/PDD-/Taobao-/AliExpress-Quellen bleiben lauffähig, werden aber nicht zum MindBay-Default.

## Quelle/Commit/Lizenz

- Bestehender Repository-Code und `CollectorProvider`: Apache-2.0-Hauptrepository, unverändert weiterverwendet.
- `tools/crawlee` und Browser-Referenzen werden in Phase 0 nicht kopiert; Commit/Lizenz sind vor tatsächlicher Übernahme in einem ADR-Nachtrag zu fixieren.

## Bewusst nicht übernommen

- Keine Amazon-Implementierung, Credentials oder Live-Abfragen in Phase 0.
- Kein Datenbankclient im Collector.
- Kein Ausbau des Amazon-SP-API-Verkaufsproviders.
- Kein paralleler Crawler und kein Lieferanten-Domainmodell im Node-Prozess.
