package main

import (
	"net/http"
)

func (s *server) joinPublicConversation(w http.ResponseWriter, r *http.Request) {
	uid, cid := r.Context().Value(userIDKey).(string), r.PathValue("id")
	var c conversation
	err := s.db.QueryRow(r.Context(), `SELECT id::text,kind,title,created_at FROM conversations
		WHERE id=$1 AND is_public=TRUE AND kind IN ('group','channel')`, cid).Scan(&c.ID, &c.Kind, &c.Title, &c.CreatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "public group or channel not found")
		return
	}
	if _, err := s.db.Exec(r.Context(), `INSERT INTO conversation_members(conversation_id,user_id,role)
		VALUES($1,$2,'member') ON CONFLICT DO NOTHING`, cid, uid); err != nil {
		writeError(w, http.StatusInternalServerError, "could not join public group or channel")
		return
	}
	writeJSON(w, http.StatusOK, c)
}
