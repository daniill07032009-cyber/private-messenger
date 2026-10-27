package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGroupRolesVisibilityAndInvitesIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	ownerName, memberName, discovererName := "owner"+suffix, "member"+suffix, "discoverer"+suffix
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM conversations WHERE created_by IN (SELECT id FROM users WHERE username=$1 OR username=$2 OR username=$3)`, ownerName, memberName, discovererName)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE username=$1 OR username=$2 OR username=$3`, ownerName, memberName, discovererName)
	}()

	api := &server{db: pool, hub: newRealtimeHub()}
	h := api.routes()
	ownerCookie := registerForTest(t, h, ownerName, "Group Owner")
	memberCookie := registerForTest(t, h, memberName, "Group Member")
	discovererCookie := registerForTest(t, h, discovererName, "Public Group Explorer")

	created := performJSONRequest(t, h, ownerCookie, http.MethodPost, "/api/v1/conversations", map[string]string{"kind": "group", "title": "Integration Group"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create group: got %d: %s", created.Code, created.Body.String())
	}
	var group conversation
	if err := json.Unmarshal(created.Body.Bytes(), &group); err != nil {
		t.Fatalf("decode group: %v", err)
	}

	settingsResponse := performJSONRequest(t, h, ownerCookie, http.MethodGet, "/api/v1/conversations/"+group.ID+"/settings", nil)
	if settingsResponse.Code != http.StatusOK {
		t.Fatalf("load group settings: got %d: %s", settingsResponse.Code, settingsResponse.Body.String())
	}
	var settings conversationSettings
	if err := json.Unmarshal(settingsResponse.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode group settings: %v", err)
	}
	if settings.CurrentRole != "owner" || settings.IsPublic || settings.ID != group.ID {
		t.Fatalf("unexpected initial group settings: %#v", settings)
	}

	inviteResponse := performJSONRequest(t, h, ownerCookie, http.MethodPost, "/api/v1/conversations/"+group.ID+"/invite", map[string]string{})
	if inviteResponse.Code != http.StatusCreated {
		t.Fatalf("create invite: got %d: %s", inviteResponse.Code, inviteResponse.Body.String())
	}
	var invite struct { Code string `json:"code"` }
	if err := json.Unmarshal(inviteResponse.Body.Bytes(), &invite); err != nil || len(invite.Code) != 48 {
		t.Fatalf("invalid invite response: %s", inviteResponse.Body.String())
	}

	joined := performJSONRequest(t, h, memberCookie, http.MethodPost, "/api/v1/invites/"+invite.Code+"/join", map[string]string{})
	if joined.Code != http.StatusOK {
		t.Fatalf("join group using invite: got %d: %s", joined.Code, joined.Body.String())
	}

	membersResponse := performJSONRequest(t, h, ownerCookie, http.MethodGet, "/api/v1/conversations/"+group.ID+"/members", nil)
	if membersResponse.Code != http.StatusOK {
		t.Fatalf("list members: got %d: %s", membersResponse.Code, membersResponse.Body.String())
	}
	var members []groupMember
	if err := json.Unmarshal(membersResponse.Body.Bytes(), &members); err != nil {
		t.Fatalf("decode members: %v", err)
	}
	var memberID string
	for _, m := range members {
		if m.Username == memberName {
			memberID = m.UserID
		}
	}
	if memberID == "" || len(members) != 2 {
		t.Fatalf("expected owner and joined member, got %#v", members)
	}

	promoted := performJSONRequest(t, h, ownerCookie, http.MethodPatch, "/api/v1/conversations/"+group.ID+"/members/"+memberID, map[string]string{"role": "admin"})
	if promoted.Code != http.StatusOK {
		t.Fatalf("promote member: got %d: %s", promoted.Code, promoted.Body.String())
	}
	transferred := performJSONRequest(t, h, ownerCookie, http.MethodPost, "/api/v1/conversations/"+group.ID+"/transfer-owner", map[string]string{"userId": memberID})
	if transferred.Code != http.StatusOK {
		t.Fatalf("transfer ownership: got %d: %s", transferred.Code, transferred.Body.String())
	}

	forbidden := performJSONRequest(t, h, ownerCookie, http.MethodPatch, "/api/v1/conversations/"+group.ID+"/visibility", map[string]bool{"isPublic": true})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("former owner should not control visibility: got %d: %s", forbidden.Code, forbidden.Body.String())
	}
	visible := performJSONRequest(t, h, memberCookie, http.MethodPatch, "/api/v1/conversations/"+group.ID+"/visibility", map[string]bool{"isPublic": true})
	if visible.Code != http.StatusOK {
		t.Fatalf("new owner should control visibility: got %d: %s", visible.Code, visible.Body.String())
	}
	publicList := performJSONRequest(t, h, memberCookie, http.MethodGet, "/api/v1/conversations/public", nil)
	if publicList.Code != http.StatusOK {
		t.Fatalf("list public groups: got %d: %s", publicList.Code, publicList.Body.String())
	}
	var publicGroups []publicConversation
	if err := json.Unmarshal(publicList.Body.Bytes(), &publicGroups); err != nil {
		t.Fatalf("decode public groups: %v", err)
	}
	found := false
	for _, c := range publicGroups {
		if c.ID == group.ID && c.IsPublic {
			found = true
		}
	}
	if !found {
		t.Fatalf("public group not returned in discovery: %#v", publicGroups)
	}
	joinedPublic := performJSONRequest(t, h, discovererCookie, http.MethodPost, "/api/v1/conversations/"+group.ID+"/join-public", map[string]string{})
	if joinedPublic.Code != http.StatusOK {
		t.Fatalf("join public group: got %d: %s", joinedPublic.Code, joinedPublic.Body.String())
	}
	publicMembers := performJSONRequest(t, h, discovererCookie, http.MethodGet, "/api/v1/conversations/"+group.ID+"/members", nil)
	if publicMembers.Code != http.StatusOK {
		t.Fatalf("list members after public join: got %d: %s", publicMembers.Code, publicMembers.Body.String())
	}
	var joinedMembers []groupMember
	if err := json.Unmarshal(publicMembers.Body.Bytes(), &joinedMembers); err != nil {
		t.Fatalf("decode members after public join: %v", err)
	}
	if len(joinedMembers) != 3 {
		t.Fatalf("expected public join to add third member, got %#v", joinedMembers)
	}

	createdChannel := performJSONRequest(t, h, ownerCookie, http.MethodPost, "/api/v1/conversations", map[string]string{"kind": "channel", "title": "Admin Only Channel"})
	if createdChannel.Code != http.StatusCreated {
		t.Fatalf("create channel: got %d: %s", createdChannel.Code, createdChannel.Body.String())
	}
	var channel conversation
	if err := json.Unmarshal(createdChannel.Body.Bytes(), &channel); err != nil {
		t.Fatalf("decode channel: %v", err)
	}
	channelInviteResponse := performJSONRequest(t, h, ownerCookie, http.MethodPost, "/api/v1/conversations/"+channel.ID+"/invite", map[string]string{})
	if channelInviteResponse.Code != http.StatusCreated {
		t.Fatalf("create channel invite: got %d: %s", channelInviteResponse.Code, channelInviteResponse.Body.String())
	}
	var channelInvite struct { Code string `json:"code"` }
	if err := json.Unmarshal(channelInviteResponse.Body.Bytes(), &channelInvite); err != nil {
		t.Fatalf("decode channel invite: %v", err)
	}
	channelJoin := performJSONRequest(t, h, memberCookie, http.MethodPost, "/api/v1/invites/"+channelInvite.Code+"/join", map[string]string{})
	if channelJoin.Code != http.StatusOK {
		t.Fatalf("join channel: got %d: %s", channelJoin.Code, channelJoin.Body.String())
	}
	memberPost := performJSONRequest(t, h, memberCookie, http.MethodPost, "/api/v1/conversations/"+channel.ID+"/messages", map[string]string{"body": "member post should be denied"})
	if memberPost.Code != http.StatusForbidden {
		t.Fatalf("channel member should not publish: got %d: %s", memberPost.Code, memberPost.Body.String())
	}
	ownerPost := performJSONRequest(t, h, ownerCookie, http.MethodPost, "/api/v1/conversations/"+channel.ID+"/messages", map[string]string{"body": "owner post should be allowed"})
	if ownerPost.Code != http.StatusCreated {
		t.Fatalf("channel owner should publish: got %d: %s", ownerPost.Code, ownerPost.Body.String())
	}

}

func performJSONRequest(t *testing.T, h http.Handler, cookie *http.Cookie, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode request: %v", err)
		}
		body = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, body)
	req.AddCookie(cookie)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
