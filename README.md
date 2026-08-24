# tokentelemetry-go

High-performance single-binary Go rewrite of TokenTelemetry with embedded Astro frontend.

TokenTelemetry passively monitors local AI coding agent session logs, parses message turns and token metrics, calculates financial and electricity costs offline, and provides both an interactive terminal interface (`tt`) and a centralized web dashboard (`tt-server`).

---

## 📚 Documentation & Architecture Guides

- **[Ubiquitous Language Glossary (`CONTEXT.md`)](CONTEXT.md)**: Standardized domain terms (*Session*, *Message Turn*, *Agent Parser*, *Scanner Checkpoint*, *Client Batch Buffer*, etc.).
- **[Collector Architecture Guide (`docs/collector-architecture.md`)](docs/collector-architecture.md)**: Deep dive into the `tt` CLI collector, background watching, concurrency, offline pricing engine, and ingestion buffering.
- **[Transcript Parsing, Storage Schema & UI Guide (`docs/transcript-parsing-and-schema.md`)](docs/transcript-parsing-and-schema.md)**: Full parser breakdown for 18+ agent harnesses, SQLite schema definition, Astro/React frontend data flow, and the **Agent Harness Schema Change Playbook**.

---

## ⚡ Quick Start

### 1. Build Binaries

```bash
# Build collector (bin/tt) and hub server (bin/tt-server)
make build
```

### 2. Run the Hub Server

```bash
./bin/tt-server
# Hub API and embedded Astro UI available at http://localhost:8000
```

### 3. Run the Collector CLI

```bash
# Continuous watching mode (monitors ~/.gemini, ~/.claude, etc.)
./bin/tt watch

# One-off sweep and summary
./bin/tt scan

# Dry-run inspection without streaming to Hub
./bin/tt scan --dry-run

# Test end-to-end connectivity with a synthetic session
./bin/tt send --synthetic
```

---

## 🧪 Testing

```bash
# Run unit and integration tests across all packages
go test ./...
```
