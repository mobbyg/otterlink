package events

import (
    "database/sql"
    "testing"
    _ "modernc.org/sqlite"
    "github.com/mobbyg/otterlink/server/internal/db"
)

func TestCreateAndUpdateEvent(t *testing.T) {
    database,err:=sql.Open("sqlite",":memory:");if err!=nil{t.Fatal(err)};defer database.Close()
    if err:=db.Initialize(database);err!=nil{t.Fatal(err)}
    s:=Service{DB:database}
    e,err:=s.Create("Game Night","Come play!","public","alice","none",0,"2026-10-01T19:00:00Z","2026-10-01T21:00:00Z",false)
    if err!=nil{t.Fatal(err)}
    if e.ID<1 || e.CreatedBy!="alice"{t.Fatalf("unexpected event: %+v",e)}
    e,err=s.Update(e.ID,"Game Night 2","Updated","public","none",0,e.StartAt,e.EndAt,false);if err!=nil{t.Fatal(err)}
    if e.Title!="Game Night 2"{t.Fatalf("unexpected title %q",e.Title)}
    if err:=s.Delete(e.ID);err!=nil{t.Fatal(err)}
}
