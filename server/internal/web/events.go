package web

import (
    "database/sql"
    "errors"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/mobbyg/otterlink/server/internal/accounts"
    "github.com/mobbyg/otterlink/server/internal/chat"
    "github.com/mobbyg/otterlink/server/internal/events"
)

type eventRequest struct {
    Title string `json:"title"`
    Description string `json:"description"`
    EventType string `json:"event_type"`
    TargetType string `json:"target_type"`
    TargetID int64 `json:"target_id"`
    StartAt string `json:"start_at"`
    EndAt string `json:"end_at"`
    AllDay bool `json:"all_day"`
}

func (s *Server) eventList(w http.ResponseWriter,r *http.Request) {
    if _,ok:=s.user(r);!ok { http.Error(w,"unauthorized",http.StatusUnauthorized);return }
    year,_:=strconv.Atoi(r.URL.Query().Get("year"));month,_:=strconv.Atoi(r.URL.Query().Get("month"))
    if year<1 { year=0 }; if month<1||month>12 { month=0 }
    if year==0 { yearNow,monthNow,_:=timeNowUTC(); year=yearNow;month=monthNow }
    if month==0 { http.Error(w,"month is required",http.StatusBadRequest);return }
    list,err:=s.Events.List(year,month);if err!=nil{http.Error(w,err.Error(),http.StatusBadRequest);return}
    writeJSON(w,http.StatusOK,map[string]any{"events":list,"year":year,"month":month})
}

func timeNowUTC()(int,int,int){ t:=time.Now().UTC();return t.Year(),int(t.Month()),t.Day() }

func (s *Server) eventCreate(w http.ResponseWriter,r *http.Request) {
    user,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",http.StatusUnauthorized);return}
    var req eventRequest;if !decodeJSON(w,r,&req){return}
    if err:=s.authorizeEvent(user,req.EventType,req.TargetType,req.TargetID);err!=nil{http.Error(w,err.Error(),http.StatusForbidden);return}
    e,err:=s.Events.Create(req.Title,req.Description,req.EventType,user.Username,req.TargetType,req.TargetID,req.StartAt,req.EndAt,req.AllDay)
    if err!=nil{http.Error(w,err.Error(),http.StatusBadRequest);return}
    writeJSON(w,http.StatusCreated,e)
}

func (s *Server) eventUpdate(w http.ResponseWriter,r *http.Request) {
    user,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",http.StatusUnauthorized);return}
    id,ok:=parseEventID(r);if !ok{http.Error(w,"invalid event id",http.StatusBadRequest);return}
    existing,err:=s.Events.Get(id);if errors.Is(err,sql.ErrNoRows){http.Error(w,"event not found",http.StatusNotFound);return};if err!=nil{http.Error(w,"unable to load event",500);return}
    if !s.canEditEvent(user,existing){http.Error(w,"you do not have permission to edit this event",http.StatusForbidden);return}
    var req eventRequest;if !decodeJSON(w,r,&req){return}
    if err:=s.authorizeEvent(user,req.EventType,req.TargetType,req.TargetID);err!=nil{http.Error(w,err.Error(),http.StatusForbidden);return}
    e,err:=s.Events.Update(id,req.Title,req.Description,req.EventType,req.TargetType,req.TargetID,req.StartAt,req.EndAt,req.AllDay)
    if err!=nil{http.Error(w,err.Error(),http.StatusBadRequest);return};writeJSON(w,http.StatusOK,e)
}

func (s *Server) eventDelete(w http.ResponseWriter,r *http.Request) {
    user,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",http.StatusUnauthorized);return}
    id,ok:=parseEventID(r);if !ok{http.Error(w,"invalid event id",http.StatusBadRequest);return}
    existing,err:=s.Events.Get(id);if errors.Is(err,sql.ErrNoRows){http.Error(w,"event not found",http.StatusNotFound);return};if err!=nil{http.Error(w,"unable to load event",500);return}
    if !s.canEditEvent(user,existing){http.Error(w,"you do not have permission to delete this event",http.StatusForbidden);return}
    if err:=s.Events.Delete(id);err!=nil{http.Error(w,err.Error(),500);return};w.WriteHeader(http.StatusNoContent)
}

func parseEventID(r *http.Request)(int64,bool){id,err:=strconv.ParseInt(r.PathValue("eventID"),10,64);return id,err==nil&&id>0}

func (s *Server) authorizeEvent(user accounts.User,eventType,targetType string,targetID int64) error {
    eventType=strings.ToLower(strings.TrimSpace(eventType));targetType=strings.ToLower(strings.TrimSpace(targetType))
    if user.Role=="admin" { if eventType!="public"&&eventType!="community"&&eventType!="server"{return errors.New("invalid event type")}; if eventType=="server"&&targetType!="none"{return errors.New("server events cannot have a target")}; return nil }
    if eventType=="server" { return errors.New("only server administrators can create server events") }
    if targetType=="none" { if eventType=="community"{return errors.New("community events require a community target")}; return nil }
    if targetType!="chat" {return errors.New("invalid event target")}
    channel,err:=s.chatChannelByID(targetID);if err!=nil{return errors.New("target community not found")}
    role,roleErr:=s.Chat.Role(targetID,user.ID)
    if eventType=="community" {
        if roleErr!=nil || (role!="mod"&&role!="original_mod") {return errors.New("only a community moderator can create community events")}
        if !channel.Permanent {return errors.New("community events require a permanent community")}
        return nil
    }
    if !channel.Permanent {
        if roleErr!=nil {return errors.New("you must be a member of the community chat to host an event there")}
        return nil
    }
    if roleErr==nil && (role=="mod"||role=="original_mod") {return nil}
    return errors.New("you can only host public events in non-permanent community chats unless you moderate that community")
}

func (s *Server) canEditEvent(user accounts.User,e events.Event) bool {
    if user.Role=="admin" {return true}
    if strings.EqualFold(user.Username,e.CreatedBy) {
        if e.EventType=="public" {return true}
        if e.EventType=="community" && e.TargetType=="chat" { role,err:=s.Chat.Role(e.TargetID,user.ID);return err==nil&&(role=="mod"||role=="original_mod") }
    }
    if e.EventType=="community"&&e.TargetType=="chat" {
        role,err:=s.Chat.Role(e.TargetID,user.ID);return err==nil&&(role=="mod"||role=="original_mod")
    }
    return false
}

func (s *Server) chatChannelByID(id int64)(chat.Channel,error){
    channels,err:=s.Chat.ListChannels();if err!=nil{return chat.Channel{},err}
    for _,c:=range channels{if c.ID==id{return c,nil}}
    return chat.Channel{},sql.ErrNoRows
}
