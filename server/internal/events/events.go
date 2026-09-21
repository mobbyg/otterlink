package events

import (
    "database/sql"
    "errors"
    "fmt"
    "strings"
    "time"
)

const maxFuture = 24 * time.Month

type Event struct {
    ID          int64  `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    EventType   string `json:"event_type"`
    CreatedBy   string `json:"created_by"`
    TargetType  string `json:"target_type"`
    TargetID    int64  `json:"target_id,omitempty"`
    StartAt     string `json:"start_at"`
    EndAt       string `json:"end_at"`
    AllDay      bool   `json:"all_day"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
}

type Service struct { DB *sql.DB }

func (s Service) Create(title, description, eventType, creator, targetType string, targetID int64, startAt, endAt string, allDay bool) (Event, error) {
    if err := validate(title, description, eventType, targetType, targetID, startAt, endAt, allDay); err != nil { return Event{}, err }
    result, err := s.DB.Exec(`INSERT INTO events (title, description, event_type, created_by, target_type, target_id, start_at, end_at, all_day)
        VALUES (?, ?, ?, ?, ?, NULLIF(?, 0), ?, ?, ?)`, strings.TrimSpace(title), strings.TrimSpace(description),
        eventType, strings.TrimSpace(creator), targetType, targetID, startAt, endAt, allDay)
    if err != nil { return Event{}, fmt.Errorf("create event: %w", err) }
    id, err := result.LastInsertId(); if err != nil { return Event{}, err }
    return s.Get(id)
}

func (s Service) Get(id int64) (Event, error) {
    var e Event
    var target sql.NullInt64
    var allDay int
    err := s.DB.QueryRow(`SELECT id,title,description,event_type,created_by,target_type,target_id,start_at,end_at,all_day,created_at,updated_at FROM events WHERE id=?`, id).
        Scan(&e.ID,&e.Title,&e.Description,&e.EventType,&e.CreatedBy,&e.TargetType,&target,&e.StartAt,&e.EndAt,&allDay,&e.CreatedAt,&e.UpdatedAt)
    if err != nil { return Event{}, err }
    if target.Valid { e.TargetID = target.Int64 }
    e.AllDay = allDay != 0
    return e,nil
}

func (s Service) List(year, month int) ([]Event,error) {
    if month < 1 || month > 12 { return nil, errors.New("invalid month") }
    start := time.Date(year,time.Month(month),1,0,0,0,0,time.UTC)
    end := start.AddDate(0,1,0)
    rows, err := s.DB.Query(`SELECT id FROM events WHERE start_at < ? AND end_at >= ? ORDER BY start_at, id`,
        end.Format(time.RFC3339), start.Format(time.RFC3339))
    if err != nil { return nil,err }
    defer rows.Close()
    result:=make([]Event,0)
    for rows.Next(){ var id int64; if err:=rows.Scan(&id);err!=nil{return nil,err}; e,err:=s.Get(id);if err!=nil{return nil,err}; result=append(result,e) }
    return result,rows.Err()
}

func (s Service) Update(id int64, title, description, eventType, targetType string, targetID int64, startAt, endAt string, allDay bool) (Event,error) {
    if err:=validate(title,description,eventType,targetType,targetID,startAt,endAt,allDay);err!=nil{return Event{},err}
    result,err:=s.DB.Exec(`UPDATE events SET title=?,description=?,event_type=?,target_type=?,target_id=NULLIF(?,0),start_at=?,end_at=?,all_day=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`,
        strings.TrimSpace(title),strings.TrimSpace(description),eventType,targetType,targetID,startAt,endAt,allDay,id)
    if err!=nil{return Event{},fmt.Errorf("update event: %w",err)}
    n,_:=result.RowsAffected();if n==0{return Event{},sql.ErrNoRows}
    return s.Get(id)
}

func (s Service) Delete(id int64) error {
    result,err:=s.DB.Exec(`DELETE FROM events WHERE id=?`,id);if err!=nil{return err}
    n,_:=result.RowsAffected();if n==0{return sql.ErrNoRows};return nil
}

func validate(title,description,eventType,targetType string,targetID int64,startAt,endAt string,allDay bool) error {
    if strings.TrimSpace(title)=="" || len([]rune(title))>120{return errors.New("event title must be 1-120 characters")}
    if len([]rune(description))>2000{return errors.New("event description exceeds 2000 characters")}
    if eventType!="public" && eventType!="community" && eventType!="server"{return errors.New("invalid event type")}
    if eventType=="server" && targetType!="none"{return errors.New("server events cannot have a target")}
    if targetType!="none" && targetType!="chat"{return errors.New("invalid event target")}
    if targetType=="none" && targetID!=0{return errors.New("target id is required only for targeted events")}
    if targetType=="chat" && targetID<1{return errors.New("chat target id is required")}
    if allDay {
        if len(startAt)!=10 || len(endAt)!=10 { return errors.New("all-day events require YYYY-MM-DD dates") }
        if endAt<startAt{return errors.New("event end must not be before its start")}
        startDate,err:=time.Parse("2006-01-02",startAt);if err!=nil{return errors.New("invalid all-day start date")}
        if _,err:=time.Parse("2006-01-02",endAt);err!=nil{return errors.New("invalid all-day end date")}
        if startDate.After(time.Now().UTC().AddDate(2,0,0)){return errors.New("events may only be scheduled up to 2 years ahead")}
    } else {
        start,err:=time.Parse(time.RFC3339,startAt);if err!=nil{return errors.New("invalid start time")}
        end,err:=time.Parse(time.RFC3339,endAt);if err!=nil{return errors.New("invalid end time")}
        if end.Before(start){return errors.New("event end must not be before its start")}
        if start.After(time.Now().UTC().AddDate(2,0,0)){return errors.New("events may only be scheduled up to 2 years ahead")}
    }
    return nil
}
