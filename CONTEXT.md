# TokenTelemetry

High-performance single-binary telemetry and analytics engine for local AI coding agents.

## Language

### Core Entities

**Session**:
A complete agent conversation or execution recorded in a transcript on disk.
_Avoid_: Thread, chat, run, task

**Message Turn**:
A single discrete exchange in a session containing role, model, token usage, tool invocations, and cost metrics.
_Avoid_: Message, step, interaction, prompt-response

**Subagent Run**:
An autonomous subagent execution spawned and linked to a parent orchestrator session.
_Avoid_: Child session, sidechain, worker task

**Transcript**:
The raw on-disk log file written by an AI agent and passively parsed by the scanner.
_Avoid_: History log, audit trail, raw dump

**Synthetic Transcript**:
A deterministically generated test transcript file used by test fixtures to simulate agent activity.
_Avoid_: Mock log, dummy data, fake session

**Agent Parser**:
A format-specific extraction routine that detects and transforms an agent's native transcript structure into standard session and message turn models.
_Avoid_: Adapter, translator, decoder, driver

**Scanner Checkpoint**:
Persisted file state tracking modification timestamp, byte offset, and content hash to enable efficient incremental scanning and avoid duplicate processing.
_Avoid_: Bookmark, read pointer, watermark, state marker

**Turn Metric Heuristic**:
A fallback estimation ratio used to approximate token counts from character lengths when a transcript omits native token usage metrics.
_Avoid_: Token guess, rough count, proxy token, fuzzy metric

### Pricing & Metrics

**Gross Cost**:
The theoretical cost of LLM consumption assuming all prompt tokens are charged at standard input rates.
_Avoid_: Base price, un-discounted cost

**Net Cost**:
The actual billable cost taking into account prompt cache hit discounts and cache creation premiums.
_Avoid_: Discounted price, true cost, final bill

**Pricing Override**:
A user-defined model rate persisted in SQLite that takes precedence over embedded catalog rates.
_Avoid_: Custom rate, custom price, manual fee

**Power Estimation**:
Calculated electrical power consumption and electricity cost based on hardware profile TDP and local utility rates.
_Avoid_: Hardware cost, energy bill, machine usage

**Reasoning Tokens**:
LLM tokens consumed to produce hidden chain-of-thought, recorded as a distinct metric from visible output tokens. Whether a provider bills them inside output or on top of it is provider-specific.
_Avoid_: Thinking tokens, thought tokens

**Billable Output Tokens**:
The output token count a provider actually charges for, derived from output and reasoning tokens under the provider's reasoning-inclusion semantics. Single source of truth for pricing and reporting.
_Avoid_: Effective output tokens, charged output

### Architecture & Distributed Topology

**Collector** (`tt`):
The local client command-line utility running on a developer's workstation that watches agent transcript directories, parses session turns, renders interactive terminal UI/status, and transmits ingestion batches to the Hub.
_Avoid_: Agent, Daemon, Scraper, Sniffer, Local Server

**Hub** (`tt-server`):
The centralized telemetry backend server deployed to Kubernetes or a host server, responsible for REST/SSE APIs, storage persistence, analytics aggregation, and hosting the embedded Astro Web UI.
_Avoid_: Master, Central Engine, Aggregator, Backend

**Ingestion Batch**:
A structured payload of newly discovered or incrementally updated sessions and message turns transmitted from Collector to Hub over HTTP.
_Avoid_: Sync packet, telemetry push, log payload, event dump

**Client Batch Buffer**:
An in-memory queue within the collector that aggregates and debounces parsed sessions before transmitting them in an ingestion batch.
_Avoid_: Transmission queue, local buffer, staging cache, spool

## Support & Scope

**First-Class Agent**:
An agent harness backed by a standing support commitment from the maintainers, whose ingestion gaps and bugs take priority. All other agents are demand-driven and added only when a user provides evidence of real usage.
_Avoid_: Supported agent, priority agent, tier-one agent
