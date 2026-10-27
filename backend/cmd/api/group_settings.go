package main

import (
	"net/http"
)

type conversationSettings struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	IsPublic    bool   `json:"isPublic"`
	CurrentRole string `json:"currentRole"`
	OwnerUserID string `json:"ownerUserId"`
	HasInvite   bool   `json:"hasInvite"`
}

func (s *server) getConversationSettings(w http.ResponseWriter, r *http.Request) {
	uid, cid := r.Context().Value(userIDKey).(string), r.PathValue("id")
	var settings conversationSettings
	err := s.db.QueryRow(r.Context(), `SELECT c.id::text,c.kind,COALESCE(c.title,''),c.is_public,actor.role,
		COALESCE(owner.user_id::text,''),c.invite_code IS NOT NULL
		FROM conversations c
		JOIN conversation_members actor ON actor.conversation_id=c.id AND actor.user_id=$2
		LEFT JOIN conversation_members owner ON owner.conversation_id=c.id AND owner.role='owner'
			AND owner.conversation_id=c.id
		WHERE c.id=$1 AND c.kind IN ('group','channel')`, cid, uid).Scan(
		&settings.ID, &settings.Kind, &settings.Title, &settings.IsPublic, &settings.CurrentRole, &settings.OwnerUserID, &settings.HasInvite)
	if err != nil {
		writeError(w, http.StatusNotFound, "group or channel not found")
		return
	}
	writeJSON(w, http.StatusOK, settings)
}
