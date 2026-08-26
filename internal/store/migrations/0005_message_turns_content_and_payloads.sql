-- Migration 0005: Expand message_turns with content, thinking, reasoning_effort, and tool payloads
ALTER TABLE message_turns ADD COLUMN content TEXT DEFAULT '';
ALTER TABLE message_turns ADD COLUMN thinking TEXT DEFAULT '';
ALTER TABLE message_turns ADD COLUMN reasoning_effort TEXT DEFAULT '';
ALTER TABLE message_turns ADD COLUMN tool_calls_json TEXT DEFAULT '[]';
ALTER TABLE message_turns ADD COLUMN tool_results_json TEXT DEFAULT '[]';
ALTER TABLE message_turns ADD COLUMN raw_payload_json TEXT DEFAULT '';
