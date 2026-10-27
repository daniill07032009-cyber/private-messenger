-- A group or channel must have exactly one owner after creation/transfer.
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_members_single_owner
  ON conversation_members(conversation_id)
  WHERE role = 'owner';
