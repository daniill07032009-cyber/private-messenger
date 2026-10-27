package main

import (
 "encoding/json"
 "sync"
 "time"

 "github.com/gorilla/websocket"
)

type realtimeEvent struct {
 Type string `json:"type"`
 Message message `json:"message"`
}
type realtimeClient struct {
 conn *websocket.Conn
 send chan []byte
 conversationID string
 hub *realtimeHub
}
type realtimeHub struct {
 mu sync.RWMutex
 rooms map[string]map[*realtimeClient]struct{}
}
func newRealtimeHub()*realtimeHub{return &realtimeHub{rooms:make(map[string]map[*realtimeClient]struct{})}}
func(h *realtimeHub)add(c *realtimeClient){
 h.mu.Lock();defer h.mu.Unlock()
 if h.rooms[c.conversationID]==nil{h.rooms[c.conversationID]=make(map[*realtimeClient]struct{})}
 h.rooms[c.conversationID][c]=struct{}{}
}
func(h *realtimeHub)remove(c *realtimeClient){
 h.mu.Lock();defer h.mu.Unlock()
 if room:=h.rooms[c.conversationID];room!=nil{
  if _,ok:=room[c];ok{delete(room,c);close(c.send)}
  if len(room)==0{delete(h.rooms,c.conversationID)}
 }
}
func(h *realtimeHub)broadcast(conversationID string,m message){
 payload,err:=json.Marshal(realtimeEvent{Type:"message",Message:m});if err!=nil{return}
 h.mu.RLock();defer h.mu.RUnlock()
 for client:=range h.rooms[conversationID]{select{case client.send<-payload:default:}}
}
func(c *realtimeClient)writePump(){
 ticker:=time.NewTicker(25*time.Second)
 defer func(){ticker.Stop();c.hub.remove(c);_=c.conn.Close()}()
 for{
  select{
  case payload,ok:=<-c.send:
   _=c.conn.SetWriteDeadline(time.Now().Add(10*time.Second))
   if !ok{_=c.conn.WriteMessage(websocket.CloseMessage,[]byte{});return}
   if err:=c.conn.WriteMessage(websocket.TextMessage,payload);err!=nil{return}
  case <-ticker.C:
   _=c.conn.SetWriteDeadline(time.Now().Add(10*time.Second))
   if err:=c.conn.WriteMessage(websocket.PingMessage,nil);err!=nil{return}
  }
 }
}
