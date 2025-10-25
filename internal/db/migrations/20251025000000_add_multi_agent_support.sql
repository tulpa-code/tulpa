-- +goose Up
-- +goose StatementBegin
-- Add agent fields to sessions table for multi-agent support
ALTER TABLE sessions ADD COLUMN active_agent_id TEXT NOT NULL DEFAULT 'coder';
ALTER TABLE sessions ADD COLUMN agent_history TEXT NOT NULL DEFAULT '[]';

-- Add agent_id column to messages table to track which agent generated each message
ALTER TABLE messages ADD COLUMN agent_id TEXT;

-- Backfill existing messages to assume they were from the coder agent
UPDATE messages 
SET agent_id = 'coder' 
WHERE agent_id IS NULL AND role = 'assistant';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove agent fields from sessions table
ALTER TABLE sessions DROP COLUMN active_agent_id;
ALTER TABLE sessions DROP COLUMN agent_history;

-- Remove agent_id from messages table
ALTER TABLE messages DROP COLUMN agent_id;
-- +goose StatementEnd