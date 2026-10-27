package main

import (
 "net/http"
 "time"

 "github.com/gorilla/websocket"
)

func(s *server)websocket(w http.ResponseWriter,r *http.Request){
 uid:=r.Context().Value(userIDKey).(string)
 cid:=r.URL.Query().Get("conversationId")
 if cid==""||!s.isMember(r.Context(),cid,uid){http.Error(w,"conversation not found",http.StatusNotFound);return}
 upgrader:=websocket.Upgrader{
  ReadBufferSize:1024,
  WriteBufferSize:1024,
  CheckOrigin:func(req *http.Request)bool{
   origin:=req.Header.Get("Origin")
   if origin==""{return true}
   return s.webOrigin!=""&&origin==s.webOrigin
  },
 }
 conn,err:=upgrader.Upgrade(w,r,nil);if err!=nil{return}
 client:=&realtimeClient{conn:conn,send:make(chan []byte,64),conversationID:cid,hub:s.hub}
 if client.hub==nil{_=conn.WriteControl(websocket.CloseMessage,websocket.FormatCloseMessage(websocket.CloseTryAgainLater,"real-time service unavailable"),time.Now().Add(time.Second));_=conn.Close();return}
 s.hub.add(client)
 conn.SetReadLimit(1024)
 _=conn.SetReadDeadline(time.Now().Add(70*time.Second))
 conn.SetPongHandler(func(string)error{return conn.SetReadDeadline(time.Now().Add(70*time.Second))})
 go client.writePump()
 for{
  if _,_,err:=conn.ReadMessage();err!=nil{break}
 }
 s.hub.remove(client)
 _=conn.Close()
}
