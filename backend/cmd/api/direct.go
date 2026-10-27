package main

import (
 "net/http"
 "strings"

 "github.com/jackc/pgx/v5"
)

type directConversationRequest struct { Username string `json:"username"` }

func (s *server) searchUsers(w http.ResponseWriter,r *http.Request) {
 q:=strings.TrimSpace(r.URL.Query().Get("q"))
 if len([]rune(q))<2 || len([]rune(q))>50 { writeError(w,400,"search query must contain 2-50 characters");return }
 uid:=r.Context().Value(userIDKey).(string)
 rows,err:=s.db.Query(r.Context(),`SELECT id::text,username,display_name,created_at FROM users WHERE id<>$1 AND (username ILIKE '%' || $2 || '%' OR display_name ILIKE '%' || $2 || '%') ORDER BY username LIMIT 25`,uid,q)
 if err!=nil{writeError(w,500,"could not search users");return};defer rows.Close()
 result:=make([]user,0)
 for rows.Next(){var u user;if err:=rows.Scan(&u.ID,&u.Username,&u.DisplayName,&u.CreatedAt);err!=nil{writeError(w,500,"could not read search results");return};result=append(result,u)}
 if rows.Err()!=nil{writeError(w,500,"could not read search results");return};writeJSON(w,200,result)
}

func (s *server) createDirectConversation(w http.ResponseWriter,r *http.Request) {
 var in directConversationRequest
 if !decodeJSON(w,r,&in){return}
 target:=strings.ToLower(strings.TrimSpace(in.Username))
 if target=="" || len(target)>32 {writeError(w,400,"valid username is required");return}
 me:=r.Context().Value(userIDKey).(string)
 var other string
 err:=s.db.QueryRow(r.Context(),`SELECT id::text FROM users WHERE username=$1`,target).Scan(&other)
 if err!=nil{writeError(w,404,"user not found");return}
 if other==me{writeError(w,400,"you cannot start a direct chat with yourself");return}

 tx,err:=s.db.Begin(r.Context());if err!=nil{writeError(w,500,"could not create direct chat");return};defer tx.Rollback(r.Context())
 var c conversation
 err=tx.QueryRow(r.Context(),`SELECT c.id::text,c.kind,c.title,c.created_at FROM conversations c JOIN conversation_members a ON a.conversation_id=c.id AND a.user_id=$1 JOIN conversation_members b ON b.conversation_id=c.id AND b.user_id=$2 WHERE c.kind='direct' AND (SELECT count(*) FROM conversation_members m WHERE m.conversation_id=c.id)=2 LIMIT 1`,me,other).Scan(&c.ID,&c.Kind,&c.Title,&c.CreatedAt)
 if err==nil {if err=tx.Commit(r.Context());err!=nil{writeError(w,500,"could not open direct chat");return};writeJSON(w,200,c);return}
 if err!=pgx.ErrNoRows{writeError(w,500,"could not find direct chat");return}
 err=tx.QueryRow(r.Context(),`INSERT INTO conversations(id,kind,created_by) VALUES(gen_random_uuid(),'direct',$1) RETURNING id::text,kind,title,created_at`,me).Scan(&c.ID,&c.Kind,&c.Title,&c.CreatedAt)
 if err==nil{_,err=tx.Exec(r.Context(),`INSERT INTO conversation_members(conversation_id,user_id,role) VALUES($1,$2,'member'),($1,$3,'member')`,c.ID,me,other)}
 if err!=nil{writeError(w,500,"could not create direct chat");return}
 if err=tx.Commit(r.Context());err!=nil{writeError(w,500,"could not save direct chat");return}
 writeJSON(w,http.StatusCreated,c)
}

