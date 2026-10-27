ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS invite_code TEXT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_conversations_public
  ON conversations(kind, created_at DESC) WHERE is_public = TRUE;
CREATE INDEX IF NOT EXISTS idx_conversations_invite_code
  ON conversations(invite_code) WHERE invite_code IS NOT NULL;
