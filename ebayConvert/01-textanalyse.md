# Analyse des gelieferten Konzepts

## Kurzurteil

Der Text trifft die wichtigste strategische Entscheidung richtig: TradeMind sollte als **Commerce- und Operations-Fundament** weiterverwendet werden, nicht als Oberfläche, die man vollständig in Velno umbenennt. Er unterschätzt jedoch den aktuellen Stand der Codebase: Mehrere als `NEW` beschriebene Bausteine sind inzwischen implementiert und dürfen nicht doppelt gebaut werden.

## Was am Text fachlich stark ist

1. Die Trennung zwischen sichtbarem SaaS und technischem Fundament ist richtig. Ein neues, fokussiertes UX kann auf bestehenden Auth-, Tenant-, Worker-, Queue-, Provider-, Audit- und Sicherheitsmechanismen aufsetzen.
2. Die Warnung vor einem zweiten Scheduler ist zentral richtig. Zeitplanung darf nur Publication Jobs erzeugen; sie darf nie eigene eBay-Write-Logik enthalten.
3. Die Reihenfolge „einen vollständigen Product-to-eBay-Listing-Loop zuerst“ ist richtig. Research, Kalender-Politur und Agent-UX haben wenig Wert, wenn OAuth, Policies, Readiness, Publish-Reconcile und Fehlerwiederherstellung nicht belastbar sind.
4. Source-Preis/Verfügbarkeit müssen historisiert werden. Repricing und Stock-Updates auf einem überschriebenen `products`-Feld wären nicht erklärbar oder auditierbar.
5. Gewinn gehört in eine nachvollziehbare Ledger-/Kostenstruktur und nicht in ein einziges veränderbares `orders.profit`-Feld.
6. Ein Agent muss über schmale Services/Tools arbeiten. Direkte Datenbankabfragen oder direkte Plattform-Schreibzugriffe wären ein Umgehen von Berechtigung, Audit, Readiness und Idempotenz.

## Was korrigiert werden muss

| Aussage im Text | Verifizierter Stand | Konsequenz |
| --- | --- | --- |
| `EbayProvider` sei neu | `providers/platform/ebay` existiert: OAuth, Taxonomy, Inventory Item, Offer, Publish und Reconcile. | Härten und erweitern, nicht neu anlegen. |
| Listing Calendar sei neu | `modules/publication` besitzt `calendar_slot`, Preview, idempotentes Apply, Publication Job und Temporal-Starter. | Bestehenden Kalender zum Produkt-Kalender ausbauen. |
| Amazon-Collector sei neu | `collector/src/providers/sourceAmazon` und `modules/catalog` existieren. | Parser/Erfassung weiterentwickeln und rechtlich/operativ absichern. |
| Repricer/Profit seien neu | `modules/monitoring` besitzt Monitor Runs, immutable Price Rules, Price Decisions und Profit Ledger. | Tatsächliche Orders/Fees addieren; keine zweite Pricing-Domain schaffen. |
| `productpublish` sei das eBay-Herzstück | Für eBay ist dieser Redis-Pfad bewusst deaktiviert. Der exklusive eBay-Write-Owner ist `modules/publication` plus Temporal. | `productpublish` nicht in eBay umbiegen. |
| Legacy-Provider sofort löschen | Mehrere Legacy-Pfade und Tests existieren noch. | Zuerst aus der MindBay-Navigation/Registry-Konfiguration ausblenden, erst nach Ersatz und Testmigration entfernen. |

## Präzisierte Aufwandsschätzung

Die Prozentzahlen im Text sind als Denkmodell brauchbar, aber nicht als Projektplanung. Der realistische Aufwand hängt vor allem von drei Produktentscheidungen ab:

- **Fulfillment-Modell:** Nur eBay-Listing/Monitoring ist deutlich kleiner als vollautomatischer Amazon-Retail-Checkout mit Tracking, Storno und Kundenkommunikation.
- **Datenqualität und Compliance:** Amazon-Sourcedaten, Marken-/Bildrechte, GPSR, EAN/MPN und eBay-Item-Specifics erzeugen den größten manuellen Review-Anteil.
- **Automation Policy:** Ein DRY_RUN-/Freigabe-Modell ist wesentlich schneller sicher auslieferbar als sofortiges Live-Autopublishing und Auto-Repricing.

Als Planungsannahme ist daher sinnvoll: Das Fundament spart viel Infrastrukturarbeit, aber es beseitigt nicht die eBay- und Dropshipping-spezifische Facharbeit. Das Vorhaben ist ein fokussiertes Produktprogramm, keine Umbenennung eines bestehenden Admins.

## Produktentscheidung

Das Zielprodukt sollte eine klare Trennung haben:

```
Velno-artige Produktoberfläche
    -> Research, Listings, Kalender, Orders, Gewinn, Automationen
TradeMind Operations-Fundament
    -> Auth, Tenant, Credentials, Provider, Jobs, Audit, Health, Storage
MindBay-Domain
    -> Amazon Snapshot, Listing Studio, Publication/Temporal, Monitoring
```

Diese Schichten dürfen nicht gegeneinander dupliziert werden. Neue UX darf TradeMind nicht wie ein ERP wirken lassen; neue Domainlogik darf jedoch auch nicht die bewährten Sicherheitsgrenzen umgehen.
