-- A group or channel cannot have multiple owners during role changes.
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_members_single_owner
  ON conversation_members(conversation_id)
  WHERE role = 'owner';
