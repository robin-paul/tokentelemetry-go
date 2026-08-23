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
