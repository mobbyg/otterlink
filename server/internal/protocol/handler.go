package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/presence"
)

type DefaultHandler struct {
	Accounts accounts.Service
	Presence *presence.Service
	Events   func(Message)
	mu sync.Mutex
	connections map[uint64]int64
}

type loginPayload struct { Username string `json:"username"`; Password string `json:"password"` }
type tokenPayload struct { Token string `json:"token"` }
type chatSendPayload struct { Token string `json:"token"`; Message string `json:"message"` }
type chatMessage struct { From chatUser `json:"from"`; Message string `json:"message"`; Timestamp string `json:"timestamp"` }
type chatUser struct { ID int64 `json:"id"`; Username string `json:"username"`; DisplayName string `json:"display_name"` }

func (h *DefaultHandler) Handle(ctx context.Context, msg Message) Message {
	switch msg.Service { case ServiceSession: return h.handleSession(ctx,msg); case ServicePresence: return h.handlePresence(msg); case ServiceChat: return h.handleChat(msg); default: return ErrorResponse(msg.ID,ErrNotFound,"service or action not found") }
}
func (h *DefaultHandler) handleSession(ctx context.Context,msg Message) Message {
	switch msg.Action {
	case "ping": return SuccessResponse(msg.ID,ServiceSession,"pong",map[string]string{"status":"ok"})
	case "login":
		var input loginPayload; if err:=decodePayload(msg.Payload,&input); err!=nil || strings.TrimSpace(input.Username)=="" || input.Password=="" { return ErrorResponse(msg.ID,ErrBadRequest,"username and password are required") }
		user,token,err:=h.Accounts.Authenticate(input.Username,input.Password); if err!=nil { return ErrorResponse(msg.ID,ErrInvalidCredentials,"invalid username or password") }; h.markOnline(ctx,user)
		return SuccessResponse(msg.ID,ServiceSession,"login",map[string]interface{}{"user":user,"token":token})
	case "whoami": user,err:=h.authenticate(msg); if err!=nil{return ErrorResponse(msg.ID,ErrUnauthorized,err.Error())}; return SuccessResponse(msg.ID,ServiceSession,"whoami",user)
	case "logout":
		token,err:=tokenFromPayload(msg.Payload); if err!=nil{return ErrorResponse(msg.ID,ErrUnauthorized,"valid session token required")}; user,err:=h.Accounts.FromToken(token); if err!=nil{return ErrorResponse(msg.ID,ErrUnauthorized,"invalid session")}; if err:=h.Accounts.Logout(token);err!=nil{return ErrorResponse(msg.ID,ErrServer,"unable to end session")}; h.markOffline(ctx,user.ID); return SuccessResponse(msg.ID,ServiceSession,"logout",map[string]string{"status":"ok"})
	default:return ErrorResponse(msg.ID,ErrNotFound,"service or action not found")
	}
}
func (h *DefaultHandler) handlePresence(msg Message) Message {
	user,err:=h.authenticate(msg); if err!=nil{return ErrorResponse(msg.ID,ErrUnauthorized,"valid session token required")}; if h.Presence==nil{return ErrorResponse(msg.ID,ErrServer,"presence service unavailable")}
	switch msg.Action { case "list":return SuccessResponse(msg.ID,ServicePresence,"list",map[string]interface{}{"users":h.Presence.List()}); case "get": entry,ok:=h.Presence.Get(user.ID); if !ok{return SuccessResponse(msg.ID,ServicePresence,"get",map[string]interface{}{"online":false})}; return SuccessResponse(msg.ID,ServicePresence,"get",entry); default:return ErrorResponse(msg.ID,ErrNotFound,"service or action not found") }
}
func (h *DefaultHandler) handleChat(msg Message) Message {
	user,err:=h.authenticate(msg); if err!=nil{return ErrorResponse(msg.ID,ErrUnauthorized,"valid session token required")}; if msg.Action!="send"{return ErrorResponse(msg.ID,ErrNotFound,"service or action not found")}
	var input chatSendPayload; if err:=decodePayload(msg.Payload,&input);err!=nil||strings.TrimSpace(input.Message)==""{return ErrorResponse(msg.ID,ErrBadRequest,"message is required")}; message:=strings.TrimSpace(input.Message); if len([]rune(message))>2000{return ErrorResponse(msg.ID,ErrBadRequest,"message exceeds 2000 characters")}
	event:=Message{Type:TypeEvent,Service:ServiceChat,Action:"message",Payload:chatMessage{From:chatUser{ID:user.ID,Username:user.Username,DisplayName:user.DisplayName},Message:message,Timestamp:time.Now().UTC().Format(time.RFC3339)}}; if h.Events!=nil{h.Events(event)}; return SuccessResponse(msg.ID,ServiceChat,"send",map[string]string{"status":"sent"})
}
func (h *DefaultHandler) markOnline(ctx context.Context,user accounts.User){id:=connectionID(ctx);if id==0||h.Presence==nil{return};h.mu.Lock();if h.connections==nil{h.connections=make(map[uint64]int64)};if old,ok:=h.connections[id];ok&&old!=user.ID{h.mu.Unlock();h.markOffline(ctx,old);h.mu.Lock()};h.connections[id]=user.ID;h.mu.Unlock();entry,becameOnline:=h.Presence.OnlineConnection(user,id);if becameOnline&&h.Events!=nil{h.Events(Message{Type:TypeEvent,Service:ServicePresence,Action:"online",Payload:entry})}}
func (h *DefaultHandler) markOffline(ctx context.Context,userID int64){id:=connectionID(ctx);if id==0||h.Presence==nil{return};h.mu.Lock();if h.connections!=nil{delete(h.connections,id)};h.mu.Unlock();entry,becameOffline:=h.Presence.OfflineConnection(userID,id);if becameOffline&&h.Events!=nil{h.Events(Message{Type:TypeEvent,Service:ServicePresence,Action:"offline",Payload:entry})}}
func (h *DefaultHandler) OnDisconnect(ctx context.Context){id:=connectionID(ctx);if id==0{return};h.mu.Lock();userID,ok:=h.connections[id];h.mu.Unlock();if ok{h.markOffline(ctx,userID)}}
func (h *DefaultHandler) authenticate(msg Message)(accounts.User,error){token,err:=tokenFromPayload(msg.Payload);if err!=nil{return accounts.User{},err};return h.Accounts.FromToken(token)}
func decodePayload(payload interface{},target interface{})error{data,err:=json.Marshal(payload);if err!=nil{return err};return json.Unmarshal(data,target)}
func tokenFromPayload(payload interface{})(string,error){var input tokenPayload;if err:=decodePayload(payload,&input);err!=nil{return "",err};input.Token=strings.TrimSpace(input.Token);if input.Token==""{return "",errors.New("missing session token")};return input.Token,nil}
