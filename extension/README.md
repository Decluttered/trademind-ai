# MindBay Companion Extension

WXT/Manifest-V3-Extension für kontrollierte Amazon.de-Captures. Das Content Script sendet nur die öffentliche Produkt-URL und einen Idempotenzschlüssel an den Service Worker. Token und Backend-Konfiguration liegen in `storage.session` mit `TRUSTED_CONTEXTS`; Cookies, Amazon-Sitzungsdaten, Admin-JWTs und Plattform-Credentials werden weder gelesen noch an AI-Dienste übertragen.

1. Im Admin über `POST /v1/extension-tokens` ein 15 Minuten gültiges Capture-Token ausstellen.
2. Die Extension-Optionen öffnen und Token plus Backend-URL koppeln.
3. Auf einer Amazon.de-Produktseite „In MindBay erfassen“ wählen.

`pnpm --filter @trademind/extension build` erstellt das MV3-Bundle. Es gibt keine eBay-Publish-, Checkout- oder Datenbankfunktion.
