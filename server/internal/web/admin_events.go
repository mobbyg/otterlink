package web

import (
    "database/sql"
    "errors"
    "net/http"
    "strconv"
    "time"
)

func (s *Server) adminEvents(w http.ResponseWriter,r *http.Request){
    if _,ok:=s.requireAdmin(w,r);!ok{return}
    year,_:=strconv.Atoi(r.URL.Query().Get("year"));month,_:=strconv.Atoi(r.URL.Query().Get("month"))
    if year<1||month<1||month>12 { t:=time.Now().UTC();year=t.Year();month=int(t.Month()) }
    list,err:=s.Events.List(year,month);if err!=nil{http.Error(w,err.Error(),http.StatusBadRequest);return}
    writeJSON(w,http.StatusOK,map[string]any{"events":list,"year":year,"month":month})
}

func (s *Server) adminEventCreate(w http.ResponseWriter,r *http.Request){
    admin,ok:=s.requireAdmin(w,r);if !ok{return}
    var req eventRequest;if !decodeJSON(w,r,&req){return}
    e,err:=s.Events.Create(req.Title,req.Description,"server",admin.Username,"none",0,req.StartAt,req.EndAt,req.AllDay)
    if err!=nil{http.Error(w,err.Error(),http.StatusBadRequest);return}
    writeJSON(w,http.StatusCreated,e)
}

func (s *Server) adminEventUpdate(w http.ResponseWriter,r *http.Request){
    if _,ok:=s.requireAdmin(w,r);!ok{return}
    id,ok:=parseEventID(r);if !ok{http.Error(w,"invalid event id",http.StatusBadRequest);return}
    existing,err:=s.Events.Get(id);if errors.Is(err,sql.ErrNoRows){http.Error(w,"event not found",http.StatusNotFound);return};if err!=nil{http.Error(w,"unable to load event",500);return}
    var req eventRequest;if !decodeJSON(w,r,&req){return}
    eventType:=existing.EventType
    if eventType=="server" { req.EventType="server";req.TargetType="none";req.TargetID=0 }
    e,err:=s.Events.Update(id,req.Title,req.Description,eventType,req.TargetType,req.TargetID,req.StartAt,req.EndAt,req.AllDay)
    if err!=nil{http.Error(w,err.Error(),http.StatusBadRequest);return};writeJSON(w,http.StatusOK,e)
}

func (s *Server) adminEventDelete(w http.ResponseWriter,r *http.Request){
    if _,ok:=s.requireAdmin(w,r);!ok{return}
    id,ok:=parseEventID(r);if !ok{http.Error(w,"invalid event id",http.StatusBadRequest);return}
    if err:=s.Events.Delete(id);errors.Is(err,sql.ErrNoRows){http.Error(w,"event not found",http.StatusNotFound);return}else if err!=nil{http.Error(w,err.Error(),500);return}
    w.WriteHeader(http.StatusNoContent)
}
