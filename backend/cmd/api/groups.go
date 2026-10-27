package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

type groupMember struct {
	UserID      string    `json:"userId"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joinedAt"`
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

type transferOwnershipRequest struct {
	UserID string `json:"userId"`
}

type visibilityRequest struct {
	IsPublic bool `json:"isPublic"`
}

type publicConversation struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	IsPublic    bool      `json:"isPublic"`
	MemberCount int       `json:"memberCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (s *server) listConversationMembers(w http.ResponseWriter, r *http.Request) {
	uid, cid := r.Context().Value(userIDKey).(string), r.PathValue("id")
	if !s.isMember(r.Context(), cid, uid) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT u.id::text,u.username,u.display_name,cm.role,cm.joined_at
		FROM conversation_members cm JOIN users u ON u.id=cm.user_id
		WHERE cm.conversation_id=$1 ORDER BY CASE cm.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END, u.username`, cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list members")
		return
	}
	defer rows.Close()
	out := make([]groupMember, 0)
	for rows.Next() {
		var m groupMember
		if err := rows.Scan(&m.UserID, &m.Username, &m.DisplayName, &m.Role, &m.JoinedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read members")
			return
		}
		out = append(out, m)
	}
	if rows.Err() != nil {
		writeError(w, http.StatusInternalServerError, "could not read members")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) updateMemberRole(w http.ResponseWriter, r *http.Request) {
	uid, cid, targetID := r.Context().Value(userIDKey).(string), r.PathValue("id"), r.PathValue("memberID")
	var in updateMemberRoleRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	in.Role = strings.TrimSpace(in.Role)
	if in.Role != "admin" && in.Role != "member" {
		writeError(w, http.StatusBadRequest, "role must be admin or member")
		return
	}
	var kind, actorRole string
	err := s.db.QueryRow(r.Context(), `SELECT c.kind,cm.role FROM conversations c
		JOIN conversation_members cm ON cm.conversation_id=c.id AND cm.user_id=$2 WHERE c.id=$1`, cid, uid).Scan(&kind, &actorRole)
	if err != nil || (kind != "group" && kind != "channel") {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if actorRole != "owner" {
		writeError(w, http.StatusForbidden, "only the owner can change member roles")
		return
	}
	if targetID == uid {
		writeError(w, http.StatusBadRequest, "owner role cannot be changed here")
		return
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE conversation_members SET role=$3
		WHERE conversation_id=$1 AND user_id=$2 AND role <> 'owner'`, cid, targetID, in.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update member role")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "member not found or owner role is protected")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"userId": targetID, "role": in.Role})
}

func (s *server) transferOwnership(w http.ResponseWriter, r *http.Request) {
	uid, cid := r.Context().Value(userIDKey).(string), r.PathValue("id")
	var in transferOwnershipRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	in.UserID = strings.TrimSpace(in.UserID)
	if in.UserID == "" || in.UserID == uid {
		writeError(w, http.StatusBadRequest, "a different member userId is required")
		return
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not transfer ownership")
		return
	}
	defer tx.Rollback(r.Context())
	var kind string
	var role string
	err = tx.QueryRow(r.Context(), `SELECT c.kind,cm.role FROM conversations c
		JOIN conversation_members cm ON cm.conversation_id=c.id AND cm.user_id=$2
		WHERE c.id=$1 FOR UPDATE OF c`, cid, uid).Scan(&kind, &role)
	if err != nil || (kind != "group" && kind != "channel") {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only the owner can transfer ownership")
		return
	}
	var targetExists bool
	if err = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=$1 AND user_id=$2)`, cid, in.UserID).Scan(&targetExists); err != nil || !targetExists {
		writeError(w, http.StatusBadRequest, "new owner must already be a member")
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE conversation_members SET role='admin' WHERE conversation_id=$1 AND user_id=$2`, cid, uid); err != nil {
		writeError(w, http.StatusInternalServerError, "could not transfer ownership")
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE conversation_members SET role='owner' WHERE conversation_id=$1 AND user_id=$2`, cid, in.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not transfer ownership")
		return
	}
	if _, err = tx.Exec(r.Context(), `UPDATE conversations SET created_by=$2 WHERE id=$1`, cid, in.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update conversation owner")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save ownership transfer")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ownerUserId": in.UserID, "previousOwnerRole": "admin"})
}

func (s *server) updateConversationVisibility(w http.ResponseWriter, r *http.Request) {
	uid, cid := r.Context().Value(userIDKey).(string), r.PathValue("id")
	var in visibilityRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	var role string
	err := s.db.QueryRow(r.Context(), `SELECT role FROM conversation_members WHERE conversation_id=$1 AND user_id=$2`, cid, uid).Scan(&role)
	if err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only the owner can change visibility")
		return
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE conversations SET is_public=$2 WHERE id=$1 AND kind IN ('group','channel')`, cid, in.IsPublic)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update visibility")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "group or channel not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"isPublic": in.IsPublic})
}

func (s *server) createInvite(w http.ResponseWriter, r *http.Request) {
	uid, cid := r.Context().Value(userIDKey).(string), r.PathValue("id")
	var role string
	err := s.db.QueryRow(r.Context(), `SELECT role FROM conversation_members WHERE conversation_id=$1 AND user_id=$2`, cid, uid).Scan(&role)
	if err != nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if role != "owner" {
		writeError(w, http.StatusForbidden, "only the owner can create or rotate invite links")
		return
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate invite link")
		return
	}
	code := hex.EncodeToString(raw)
	tag, err := s.db.Exec(r.Context(), `UPDATE conversations SET invite_code=$2 WHERE id=$1 AND kind IN ('group','channel')`, cid, code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save invite link")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "group or channel not found")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"code": code})
}

func (s *server) listPublicConversations(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT c.id::text,c.kind,c.title,c.is_public,
		(SELECT count(*)::int FROM conversation_members cm WHERE cm.conversation_id=c.id),c.created_at
		FROM conversations c WHERE c.is_public=TRUE AND c.kind IN ('group','channel')
		ORDER BY c.created_at DESC LIMIT 100`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list public groups and channels")
		return
	}
	defer rows.Close()
	out := make([]publicConversation, 0)
	for rows.Next() {
		var c publicConversation
		if err := rows.Scan(&c.ID, &c.Kind, &c.Title, &c.IsPublic, &c.MemberCount, &c.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read public groups and channels")
			return
		}
		out = append(out, c)
	}
	if rows.Err() != nil {
		writeError(w, http.StatusInternalServerError, "could not read public groups and channels")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) joinInvite(w http.ResponseWriter, r *http.Request) {
	uid, code := r.Context().Value(userIDKey).(string), strings.TrimSpace(r.PathValue("code"))
	if len(code) != 48 {
		writeError(w, http.StatusNotFound, "invite link not found")
		return
	}
	var cid, kind string
	err := s.db.QueryRow(r.Context(), `SELECT id::text,kind FROM conversations WHERE invite_code=$1 AND kind IN ('group','channel')`, code).Scan(&cid, &kind)
	if err != nil {
		writeError(w, http.StatusNotFound, "invite link not found")
		return
	}
	_, err = s.db.Exec(r.Context(), `INSERT INTO conversation_members(conversation_id,user_id,role) VALUES($1,$2,'member') ON CONFLICT DO NOTHING`, cid, uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not join conversation")
		return
	}
	var c conversation
	err = s.db.QueryRow(r.Context(), `SELECT id::text,kind,title,created_at FROM conversations WHERE id=$1`, cid).Scan(&c.ID, &c.Kind, &c.Title, &c.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load conversation")
		return
	}
	writeJSON(w, http.StatusOK, c)
}
