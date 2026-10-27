package main

import (
 "context"
 "crypto/rand"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "os"
 "regexp"
 "strings"
 "time"

 "github.com/jackc/pgx/v5/pgxpool"
 "golang.org/x/crypto/bcrypt"
)

type server struct { db *pgxpool.Pool; webOrigin string; hub *realtimeHub }
type ctxKey string
const userIDKey ctxKey = "userID"
type user struct { ID string `json:"id"`; Username string `json:"username"`; DisplayName string `json:"displayName"`; CreatedAt time.Time `json:"createdAt"` }
type conversation struct { ID string `json:"id"`; Kind string `json:"kind"`; Title *string `json:"title,omitempty"`; CreatedAt time.Time `json:"createdAt"` }
type message struct { ID string `json:"id"`; ConversationID string `json:"conversationId"`; SenderID string `json:"senderId"`; Body string `json:"body"`; CreatedAt time.Time `json:"createdAt"` }
type credentials struct { Username string `json:"username"`; DisplayName string `json:"displayName"`; Password string `json:"password"` }
type createConversationRequest struct { Kind string `json:"kind"`; Title string `json:"title"` }
type sendMessageRequest struct { Body string `json:"body"` }
var usernamePattern=regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

func (s *server) routes() http.Handler {
 mux:=http.NewServeMux()
 mux.HandleFunc("GET /healthz",s.health)
 mux.HandleFunc("GET /api/v1",func(w http.ResponseWriter,r *http.Request){writeJSON(w,200,map[string]string{"name":"Private Messenger API","version":"0.2.0","status":"ready"})})
 mux.HandleFunc("POST /api/v1/auth/register",s.register)
 mux.HandleFunc("POST /api/v1/auth/login",s.login)
 mux.Handle("POST /api/v1/auth/logout",s.requireAuth(http.HandlerFunc(s.logout)))
 mux.Handle("GET /api/v1/auth/me",s.requireAuth(http.HandlerFunc(s.me)))
 mux.Handle("GET /api/v1/users/search",s.requireAuth(http.HandlerFunc(s.searchUsers)))
 mux.Handle("POST /api/v1/conversations/direct",s.requireAuth(http.HandlerFunc(s.createDirectConversation)))
 mux.Handle("GET /api/v1/conversations",s.requireAuth(http.HandlerFunc(s.listConversations)))
 mux.Handle("GET /api/v1/ws",s.requireAuth(http.HandlerFunc(s.websocket)))
 mux.Handle("POST /api/v1/conversations",s.requireAuth(http.HandlerFunc(s.createConversation)))
 mux.Handle("GET /api/v1/conversations/public",s.requireAuth(http.HandlerFunc(s.listPublicConversations)))
 mux.Handle("GET /api/v1/conversations/{id}/members",s.requireAuth(http.HandlerFunc(s.listConversationMembers)))
 mux.Handle("PATCH /api/v1/conversations/{id}/members/{memberID}",s.requireAuth(http.HandlerFunc(s.updateMemberRole)))
 mux.Handle("POST /api/v1/conversations/{id}/transfer-owner",s.requireAuth(http.HandlerFunc(s.transferOwnership)))
 mux.Handle("PATCH /api/v1/conversations/{id}/visibility",s.requireAuth(http.HandlerFunc(s.updateConversationVisibility)))
 mux.Handle("POST /api/v1/conversations/{id}/invite",s.requireAuth(http.HandlerFunc(s.createInvite)))
 mux.Handle("POST /api/v1/invites/{code}/join",s.requireAuth(http.HandlerFunc(s.joinInvite)))
 mux.Handle("GET /api/v1/conversations/{id}/messages",s.requireAuth(http.HandlerFunc(s.listMessages)))
 mux.Handle("POST /api/v1/conversations/{id}/messages",s.requireAuth(http.HandlerFunc(s.sendMessage)))
 return s.cors(securityHeaders(mux))
}
func(s *server)health(w http.ResponseWriter,r *http.Request){ctx,cancel:=context.WithTimeout(r.Context(),2*time.Second);defer cancel();if err:=s.db.Ping(ctx);err!=nil{writeError(w,503,"database unavailable");return};writeJSON(w,200,map[string]string{"status":"ok"})}
func(s *server)register(w http.ResponseWriter,r *http.Request){
 var in credentials;if !decodeJSON(w,r,&in){return};in.Username=strings.ToLower(strings.TrimSpace(in.Username));in.DisplayName=strings.TrimSpace(in.DisplayName)
 if !usernamePattern.MatchString(in.Username){writeError(w,400,"username must be 3-32 characters: letters, numbers, underscore");return}
 if in.DisplayName==""||len([]rune(in.DisplayName))>80{writeError(w,400,"displayName is required and must be at most 80 characters");return}
 if len(in.Password)<10||len(in.Password)>72{writeError(w,400,"password must be between 10 and 72 bytes");return}
 hash,err:=bcrypt.GenerateFromPassword([]byte(in.Password),bcrypt.DefaultCost);if err!=nil{writeError(w,500,"could not create account");return}
 var u user
 err=s.db.QueryRow(r.Context(),`INSERT INTO users(id,username,display_name,password_hash) VALUES(gen_random_uuid(),$1,$2,$3) RETURNING id::text,username,display_name,created_at`,in.Username,in.DisplayName,string(hash)).Scan(&u.ID,&u.Username,&u.DisplayName,&u.CreatedAt)
 if err!=nil{if strings.Contains(err.Error(),"users_username_key"){writeError(w,409,"username is already taken")}else{writeError(w,500,"could not create account")};return}
 if err=s.startSession(w,r,u.ID);err!=nil{writeError(w,500,"account created but session could not be started");return};writeJSON(w,201,u)
}
func(s *server)login(w http.ResponseWriter,r *http.Request){
 var in credentials;if !decodeJSON(w,r,&in){return};username:=strings.ToLower(strings.TrimSpace(in.Username))
 var id,hash string;err:=s.db.QueryRow(r.Context(),`SELECT id::text,password_hash FROM users WHERE username=$1`,username).Scan(&id,&hash)
 if err!=nil||bcrypt.CompareHashAndPassword([]byte(hash),[]byte(in.Password))!=nil{writeError(w,401,"invalid username or password");return}
 if err:=s.startSession(w,r,id);err!=nil{writeError(w,500,"could not start session");return};u,err:=s.getUser(r.Context(),id);if err!=nil{writeError(w,500,"could not load user");return};writeJSON(w,200,u)
}
func(s *server)logout(w http.ResponseWriter,r *http.Request){if c,err:=r.Cookie("pm_session");err==nil{h:=sha256.Sum256([]byte(c.Value));_,_=s.db.Exec(r.Context(),`DELETE FROM sessions WHERE token_hash=$1`,hex.EncodeToString(h[:]))};http.SetCookie(w,&http.Cookie{Name:"pm_session",Value:"",Path:"/",HttpOnly:true,Secure:secureCookies(),SameSite:http.SameSiteLaxMode,MaxAge:-1});w.WriteHeader(204)}
func(s *server)me(w http.ResponseWriter,r *http.Request){u,err:=s.getUser(r.Context(),r.Context().Value(userIDKey).(string));if err!=nil{writeError(w,500,"could not load user");return};writeJSON(w,200,u)}
func(s *server)listConversations(w http.ResponseWriter,r *http.Request){
 uid:=r.Context().Value(userIDKey).(string);rows,err:=s.db.Query(r.Context(),`SELECT c.id::text,c.kind,CASE WHEN c.kind='direct' THEN COALESCE(peer.display_name,peer.username,'Личная переписка') ELSE c.title END AS display_title,c.created_at FROM conversations c JOIN conversation_members cm ON cm.conversation_id=c.id LEFT JOIN conversation_members other ON other.conversation_id=c.id AND other.user_id<>$1 AND c.kind='direct' LEFT JOIN users peer ON peer.id=other.user_id WHERE cm.user_id=$1 ORDER BY (SELECT max(m.created_at) FROM messages m WHERE m.conversation_id=c.id) DESC NULLS LAST,c.created_at DESC LIMIT 200`,uid)
 if err!=nil{writeError(w,500,"could not list conversations");return};defer rows.Close();out:=make([]conversation,0)
 for rows.Next(){var c conversation;if err:=rows.Scan(&c.ID,&c.Kind,&c.Title,&c.CreatedAt);err!=nil{writeError(w,500,"could not read conversations");return};out=append(out,c)}
 if rows.Err()!=nil{writeError(w,500,"could not read conversations");return};writeJSON(w,200,out)
}
func(s *server)createConversation(w http.ResponseWriter,r *http.Request){
 var in createConversationRequest;if !decodeJSON(w,r,&in){return};in.Kind=strings.TrimSpace(in.Kind);in.Title=strings.TrimSpace(in.Title)
 if in.Kind!="group"&&in.Kind!="channel"{writeError(w,400,"kind must be group or channel");return};if in.Title==""||len([]rune(in.Title))>120{writeError(w,400,"title is required and must be at most 120 characters");return}
 uid:=r.Context().Value(userIDKey).(string);tx,err:=s.db.Begin(r.Context());if err!=nil{writeError(w,500,"could not create conversation");return};defer tx.Rollback(r.Context())
 var c conversation;err=tx.QueryRow(r.Context(),`INSERT INTO conversations(id,kind,title,created_by) VALUES(gen_random_uuid(),$1,$2,$3) RETURNING id::text,kind,title,created_at`,in.Kind,in.Title,uid).Scan(&c.ID,&c.Kind,&c.Title,&c.CreatedAt)
 if err==nil{_,err=tx.Exec(r.Context(),`INSERT INTO conversation_members(conversation_id,user_id,role) VALUES($1,$2,'owner')`,c.ID,uid)};if err==nil{err=tx.Commit(r.Context())};if err!=nil{writeError(w,500,"could not create conversation");return};writeJSON(w,201,c)
}
func(s *server)listMessages(w http.ResponseWriter,r *http.Request){
 uid,cid:=r.Context().Value(userIDKey).(string),r.PathValue("id");if !s.isMember(r.Context(),cid,uid){writeError(w,404,"conversation not found");return}
 rows,err:=s.db.Query(r.Context(),`SELECT id::text,conversation_id::text,sender_id::text,body,created_at FROM (SELECT id,conversation_id,sender_id,body,created_at FROM messages WHERE conversation_id=$1 ORDER BY created_at DESC LIMIT 100) recent ORDER BY created_at ASC`,cid)
 if err!=nil{writeError(w,500,"could not load messages");return};defer rows.Close();out:=make([]message,0)
 for rows.Next(){var m message;if err:=rows.Scan(&m.ID,&m.ConversationID,&m.SenderID,&m.Body,&m.CreatedAt);err!=nil{writeError(w,500,"could not read messages");return};out=append(out,m)}
 if rows.Err()!=nil{writeError(w,500,"could not read messages");return};writeJSON(w,200,out)
}
func(s *server)sendMessage(w http.ResponseWriter,r *http.Request){
 var in sendMessageRequest;if !decodeJSON(w,r,&in){return};in.Body=strings.TrimSpace(in.Body);if in.Body==""||len([]rune(in.Body))>4000{writeError(w,400,"message must contain 1-4000 characters");return}
 uid,cid:=r.Context().Value(userIDKey).(string),r.PathValue("id");if !s.isMember(r.Context(),cid,uid){writeError(w,404,"conversation not found");return}
 var m message;err:=s.db.QueryRow(r.Context(),`INSERT INTO messages(id,conversation_id,sender_id,body) VALUES(gen_random_uuid(),$1,$2,$3) RETURNING id::text,conversation_id::text,sender_id::text,body,created_at`,cid,uid,in.Body).Scan(&m.ID,&m.ConversationID,&m.SenderID,&m.Body,&m.CreatedAt)
 if err!=nil{writeError(w,500,"could not save message");return};if s.hub!=nil{s.hub.broadcast(cid,m)};writeJSON(w,201,m)
}
func(s *server)isMember(ctx context.Context,cid,uid string)bool{var ok bool;err:=s.db.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM conversation_members WHERE conversation_id=$1 AND user_id=$2)`,cid,uid).Scan(&ok);return err==nil&&ok}
func(s *server)requireAuth(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
 c,err:=r.Cookie("pm_session");if err!=nil{writeError(w,401,"authentication required");return};h:=sha256.Sum256([]byte(c.Value));var uid string
 err=s.db.QueryRow(r.Context(),`SELECT user_id::text FROM sessions WHERE token_hash=$1 AND expires_at>now()`,hex.EncodeToString(h[:])).Scan(&uid);if err!=nil{writeError(w,401,"authentication required");return}
 next.ServeHTTP(w,r.WithContext(context.WithValue(r.Context(),userIDKey,uid)))
})}
func(s *server)startSession(w http.ResponseWriter,r *http.Request,uid string)error{
 raw:=make([]byte,32);if _,err:=rand.Read(raw);err!=nil{return err};token:=hex.EncodeToString(raw);h:=sha256.Sum256([]byte(token))
 if _,err:=s.db.Exec(r.Context(),`INSERT INTO sessions(id,user_id,token_hash,expires_at) VALUES(gen_random_uuid(),$1,$2,now()+interval '30 days')`,uid,hex.EncodeToString(h[:]));err!=nil{return err}
 http.SetCookie(w,&http.Cookie{Name:"pm_session",Value:token,Path:"/",HttpOnly:true,Secure:secureCookies(),SameSite:http.SameSiteLaxMode,Expires:time.Now().Add(30*24*time.Hour),MaxAge:30*24*60*60});return nil
}
func(s *server)getUser(ctx context.Context,id string)(user,error){var u user;err:=s.db.QueryRow(ctx,`SELECT id::text,username,display_name,created_at FROM users WHERE id=$1`,id).Scan(&u.ID,&u.Username,&u.DisplayName,&u.CreatedAt);return u,err}
func(s *server)cors(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
 origin:=r.Header.Get("Origin");if s.webOrigin!=""&&origin==s.webOrigin{w.Header().Set("Access-Control-Allow-Origin",origin);w.Header().Set("Access-Control-Allow-Credentials","true");w.Header().Add("Vary","Origin")}
 if r.Method==http.MethodOptions{w.Header().Set("Access-Control-Allow-Methods","GET, POST, PATCH, OPTIONS");w.Header().Set("Access-Control-Allow-Headers","Content-Type");w.WriteHeader(204);return};next.ServeHTTP(w,r)
})}
func decodeJSON(w http.ResponseWriter,r *http.Request,dst any)bool{r.Body=http.MaxBytesReader(w,r.Body,1<<20);dec:=json.NewDecoder(r.Body);dec.DisallowUnknownFields();if err:=dec.Decode(dst);err!=nil{writeError(w,400,"invalid JSON request");return false};var extra any;if err:=dec.Decode(&extra);!errors.Is(err,io.EOF){writeError(w,400,"request must contain one JSON object");return false};return true}
func writeJSON(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.WriteHeader(status);_=json.NewEncoder(w).Encode(v)}
func writeError(w http.ResponseWriter,status int,message string){writeJSON(w,status,map[string]string{"error":message})}
func securityHeaders(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Header().Set("X-Content-Type-Options","nosniff");w.Header().Set("Referrer-Policy","no-referrer");w.Header().Set("X-Frame-Options","DENY");w.Header().Set("Cache-Control","no-store");next.ServeHTTP(w,r)})}
func secureCookies()bool{return strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")),"true")}
