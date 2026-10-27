package main

import (
 "bytes"
 "context"
 "encoding/json"
 "fmt"
 "net/http"
 "net/http/httptest"
 "os"
 "strings"
 "testing"
 "time"

 "github.com/gorilla/websocket"
 "github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthAndDirectMessageIntegration(t *testing.T) {
 dsn:=os.Getenv("TEST_DATABASE_URL")
 if dsn=="" { t.Skip("TEST_DATABASE_URL is not configured") }
 ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel()
 pool,err:=pgxpool.New(ctx,dsn);if err!=nil{t.Fatalf("open test database: %v",err)};defer pool.Close()
 if err:=pool.Ping(ctx);err!=nil{t.Fatalf("ping test database: %v",err)}
 if err:=runMigrations(ctx,pool);err!=nil{t.Fatalf("run migrations: %v",err)}
 suffix:=fmt.Sprintf("%d",time.Now().UnixNano())
 usernameA,usernameB:="tester"+suffix,"friend"+suffix
 defer func(){
  cleanupCtx:=context.Background()
  _,_=pool.Exec(cleanupCtx,`DELETE FROM conversations WHERE created_by IN (SELECT id FROM users WHERE username=$1 OR username=$2)`,usernameA,usernameB)
  _,_=pool.Exec(cleanupCtx,`DELETE FROM users WHERE username=$1 OR username=$2`,usernameA,usernameB)
 }()
 api:=&server{db:pool,hub:newRealtimeHub()}
 h:=api.routes()
 cookieA:=registerForTest(t,h,usernameA,"Alice Example")
 cookieB:=registerForTest(t,h,usernameB,"Bob Example")
 directBody,_:=json.Marshal(map[string]string{"username":usernameB})
 direct:=httptest.NewRequest(http.MethodPost,"/api/v1/conversations/direct",bytes.NewReader(directBody));direct.AddCookie(cookieA)
 directRec:=httptest.NewRecorder();h.ServeHTTP(directRec,direct)
 if directRec.Code!=http.StatusCreated {t.Fatalf("create direct conversation: got %d: %s",directRec.Code,directRec.Body.String())}
 var c conversation
 if err:=json.Unmarshal(directRec.Body.Bytes(),&c);err!=nil{t.Fatalf("decode conversation: %v",err)}
 messageBody,_:=json.Marshal(map[string]string{"body":"Hello from integration test"})
 send:=httptest.NewRequest(http.MethodPost,"/api/v1/conversations/"+c.ID+"/messages",bytes.NewReader(messageBody));send.AddCookie(cookieA)
 sendRec:=httptest.NewRecorder();h.ServeHTTP(sendRec,send)
 if sendRec.Code!=http.StatusCreated {t.Fatalf("send message: got %d: %s",sendRec.Code,sendRec.Body.String())}
 read:=httptest.NewRequest(http.MethodGet,"/api/v1/conversations/"+c.ID+"/messages",nil);read.AddCookie(cookieB)
 readRec:=httptest.NewRecorder();h.ServeHTTP(readRec,read)
 if readRec.Code!=http.StatusOK {t.Fatalf("read message as recipient: got %d: %s",readRec.Code,readRec.Body.String())}
 var messages []message
 if err:=json.Unmarshal(readRec.Body.Bytes(),&messages);err!=nil{t.Fatalf("decode messages: %v",err)}
 if len(messages)!=1||messages[0].Body!="Hello from integration test"{t.Fatalf("unexpected persisted messages: %#v",messages)}

 testServer:=httptest.NewServer(h);defer testServer.Close()
 headers:=http.Header{};headers.Add("Cookie","pm_session="+cookieB.Value)
 wsURL:="ws"+strings.TrimPrefix(testServer.URL,"http")+"/api/v1/ws?conversationId="+c.ID
 wsConn,_,err:=websocket.DefaultDialer.Dial(wsURL,headers);if err!=nil{t.Fatalf("connect websocket recipient: %v",err)};defer wsConn.Close()
 _=wsConn.SetReadDeadline(time.Now().Add(5*time.Second))
 secondBody,_:=json.Marshal(map[string]string{"body":"Instant websocket delivery"})
 secondReq:=httptest.NewRequest(http.MethodPost,"/api/v1/conversations/"+c.ID+"/messages",bytes.NewReader(secondBody));secondReq.AddCookie(cookieA)
 secondRec:=httptest.NewRecorder();h.ServeHTTP(secondRec,secondReq)
 if secondRec.Code!=http.StatusCreated{t.Fatalf("send websocket test message: got %d: %s",secondRec.Code,secondRec.Body.String())}
 var event realtimeEvent
 if err:=wsConn.ReadJSON(&event);err!=nil{t.Fatalf("read websocket event: %v",err)}
 if event.Type!="message"||event.Message.Body!="Instant websocket delivery"{t.Fatalf("unexpected websocket event: %#v",event)}
}

func registerForTest(t *testing.T,h http.Handler,username,displayName string)*http.Cookie {
 t.Helper()
 payload,_:=json.Marshal(map[string]string{"username":username,"displayName":displayName,"password":"test-password-strong"})
 req:=httptest.NewRequest(http.MethodPost,"/api/v1/auth/register",bytes.NewReader(payload))
 rec:=httptest.NewRecorder();h.ServeHTTP(rec,req)
 if rec.Code!=http.StatusCreated {t.Fatalf("register %s: got %d: %s",username,rec.Code,rec.Body.String())}
 for _,cookie:=range rec.Result().Cookies(){if cookie.Name=="pm_session"{return cookie}}
 t.Fatalf("registration for %s did not set session cookie",username);return nil
}
