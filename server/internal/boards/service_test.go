package boards

import (
 "database/sql"
 "testing"

 "github.com/mobbyg/otterlink/server/internal/db"
 _ "modernc.org/sqlite"
)

func TestBoardThreadReplyAndReadState(t *testing.T) {
 database,err:=sql.Open("sqlite",":memory:");if err!=nil{t.Fatal(err)};defer database.Close()
 database.SetMaxOpenConns(1)
 if err:=db.Initialize(database);err!=nil{t.Fatal(err)}
 if _,err:=database.Exec("INSERT INTO users(username,display_name,email,password_hash) VALUES('otter','Otter','otter@example.test','test-hash'),('friend','Friend','friend@example.test','test-hash')");err!=nil{t.Fatal(err)}
 service:=Service{DB:database};if err:=service.Initialize();err!=nil{t.Fatal(err)}
 var userID int64;if err:=database.QueryRow("SELECT id FROM users WHERE username='otter'").Scan(&userID);err!=nil{t.Fatal(err)}
 board,err:=service.CreateBoard("Retro Computing","Classic computers","Be kind.",userID);if err!=nil{t.Fatal(err)}
 user:=User{ID:userID,Username:"otter",DisplayName:"Otter"}
 thread,posts,err:=service.CreateThread(board.ID,user,"C64 setup","What is everyone running?");if err!=nil{t.Fatal(err)}
 if len(posts)!=1||thread.Subject!="C64 setup"{t.Fatalf("unexpected thread result: %#v %#v",thread,posts)}
 threads,err:=service.ListThreads(board.ID,userID);if err!=nil{t.Fatal(err)}
 if len(threads)!=1||threads[0].ReplyCount!=0{t.Fatalf("unexpected thread list: %#v",threads)}
 friend:=User{ID:userID+1,Username:"friend",DisplayName:"Friend"}
 reply,err:=service.Reply(thread.ID,friend,"A real C128D.");if err!=nil{t.Fatal(err)}
 if reply.Body!="A real C128D."{t.Fatalf("unexpected reply: %#v",reply)}
 threads,err=service.ListThreads(board.ID,userID);if err!=nil{t.Fatal(err)}
 if len(threads)!=1||threads[0].ReplyCount!=1||!threads[0].Unread{t.Fatalf("expected one unread reply: %#v",threads)}
 boards,err:=service.ListBoards(userID);if err!=nil{t.Fatal(err)};if len(boards)!=1||boards[0].UnreadCount!=1{t.Fatalf("expected one unread thread: %#v",boards)}
 if err:=service.MarkRead(thread.ID,userID,reply.ID);err!=nil{t.Fatal(err)}
 threads,err=service.ListThreads(board.ID,userID);if err!=nil{t.Fatal(err)};if len(threads)!=1||threads[0].Unread{t.Fatalf("expected thread to be marked read: %#v",threads)}
 loaded,loadedPosts,err:=service.GetThread(thread.ID,userID);if err!=nil{t.Fatal(err)}
 if loaded.ID!=thread.ID||len(loadedPosts)!=2{t.Fatalf("unexpected loaded thread: %#v %#v",loaded,loadedPosts)}
 if err:=service.SetThreadState(thread.ID,boolPtr(true),boolPtr(true));err!=nil{t.Fatal(err)}
 loaded,_,err=service.GetThread(thread.ID,userID);if err!=nil{t.Fatal(err)}
 if !loaded.Locked||!loaded.Sticky{t.Fatalf("thread moderation state not saved: %#v",loaded)}
 if _,err:=service.Reply(thread.ID,friend,"This should fail.");err==nil{t.Fatal("expected locked thread to reject replies")}
}
func boolPtr(v bool)*bool{return &v}
