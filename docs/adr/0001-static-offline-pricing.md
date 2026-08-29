# Pricing stays offline: static embedded catalog, no runtime network sync

Codeburn refreshes model rates by fetching the LiteLLM catalog from GitHub every 24 hours. TokenTelemetry-Go keeps the compile-time embedded `pricing_data.json` plus user-defined `pricing_overrides` as the only rate sources: the README promises offline cost calculation, the tool is a single binary with no network dependency surface, and the one observed pricing failure (`gemini-3.1-pro-preview` billing $0.00) was a resolver normalization bug (dots vs dashes), not staleness — the embedded catalog already contained the model.

## Considered Options

- **Static embedded catalog + overrides** — accepted.
- **Opt-in manual refresh (`tt pricing refresh` → disk cache)** — deferred; re-entry trigger is a real staleness incident on a first-class agent that resolver fixes cannot address.
- **Codeburn-style automatic background sync** — rejected; violates the offline principle and adds a network dependency to a tool whose value proposition includes fully local operation.

## Consequences

New and preview models need a dataset rebuild or a user override until the next release. Model ID normalization (`NormalizeModelID`) is the front line of pricing robustness and must be actively maintained.