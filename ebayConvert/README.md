# eBayConvert: Umsetzungsplan fuer ein Velno-artiges Produkt

Stand: 2026-08-22  
Scope: Amazon.de als Quelle, eBay.de als Zielmarkt; keine Erweiterung auf weitere Marktplätze.

## Ergebnis in einem Satz

TradeMind ist die geeignete Basis, aber der Umbau ist **kein Reskin**: Das bestehende MindBay-Fundament wird gezielt fertiggestellt und um Research, echte Verkauf-/Finanzdaten, Amazon-Retail-Fulfillment, Rechnungsarchiv und eine kontrollierte Agent-Schnittstelle ergänzt.

## Dokumentensatz

| Dokument | Zweck |
| --- | --- |
| [01-textanalyse.md](01-textanalyse.md) | Prüft die gelieferten Aussagen, trennt richtige Strategie von veraltetem Ist-Stand und leitet Entscheidungen ab. |
| [02-codebase-audit.md](02-codebase-audit.md) | Verifizierter Ist-Zustand der aktuellen Codebase einschließlich vorhandener MindBay-Implementierung. |
| [03-zielarchitektur.md](03-zielarchitektur.md) | Zielgrenzen, Datenfluss, Datenmodell und Sicherheits-/Betriebsregeln. |
| [04-modulmatrix.md](04-modulmatrix.md) | Konkrete KEEP / EXTEND / LEAVE / NEW-Matrix für Backend, Provider, Collector und Admin. |
| [05-implementierungsplan.md](05-implementierungsplan.md) | Sequenzierter Implementierungsplan mit Lieferobjekten, Tests, Gates und Risiken. |

## Entscheidungslogik

Die richtige Frage ist nicht, ob nachher noch die sichtbare TradeMind-Oberfläche übrig bleibt. Entscheidend ist, welche langlebigen Risiken nicht neu erfunden werden müssen: Tenant-Scope, Auth, verschlüsselte Credentials, idempotente Commands, Audit-Log, Provider-Grenzen, Worker-Health und Publish-Recovery. Diese Grundlagen sind vorhanden und sollten erhalten bleiben.

Die bereits realisierte MindBay-Linie ist ebenfalls kein theoretischer Vorschlag: Amazon-Collector, Catalog-Snapshots, Listing Studio, Calendar/Publication, nativer eBay-Provider und Monitoring/Repricing existieren. Der Plan baut darauf auf und ersetzt ihn nicht durch eine zweite parallele Domain.

## Nicht-Ziele

- Kein physisches Löschen von Legacy-Plattformen in der ersten Umsetzungswelle.
- Kein zweiter eBay-Publish-Worker neben `modules/publication` und Temporal.
- Kein direkter eBay- oder Datenbankzugriff durch Admin, Collector oder Agent.
- Keine automatische Live-Veröffentlichung, automatische Bestellung oder KI-Aktion ohne ausdrückliche, auditierte Freigabe.
- Keine Nutzung fremder Amazon-Bilder oder Inhalte ohne Quellenkennzeichnung und Operatorentscheidung.

## Startpunkt und Schutz der Arbeitskopie

Die Analyse wurde auf Branch `ebayamazon` durchgeführt. Bereits vorhandene, fremde Änderungen liegen in mehreren Dateien unter `admin/src/**`; dieser Dokumentensatz berührt keine davon. Er ist ein Planungsartefakt, keine Produktimplementierung.
