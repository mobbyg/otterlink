package web

import (
 "database/sql"
 "errors"
 "net/http"
 "strconv"
 "strings"

 "github.com/mobbyg/otterlink/server/internal/accounts"
 "github.com/mobbyg/otterlink/server/internal/boards"
)

func boardID(r *http.Request,key string)(int64,bool){id,err:=strconv.ParseInt(r.PathValue(key),10,64);return id,err==nil&&id>0}
func boardUser(u accounts.User) boards.User{return boards.User{ID:u.ID,Username:u.Username,DisplayName:u.DisplayName,Role:u.Role}}
func (s *Server) boardList(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return}
 items,err:=s.Boards.ListBoards(u.ID);if err!=nil{http.Error(w,"unable to list boards",500);return};writeJSON(w,200,map[string]any{"boards":items})
}
func (s *Server) boardThreads(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"boardID");if !ok{http.Error(w,"invalid board id",400);return}
 if u.Role!="admin"&&s.Boards.IsBanned(id,u.ID){http.Error(w,"you are banned from this board",403);return}
 b,err:=s.Boards.GetBoard(id);if errors.Is(err,sql.ErrNoRows){http.Error(w,"board not found",404);return};if err!=nil{http.Error(w,"unable to load board",500);return}
 threads,err:=s.Boards.ListThreads(id,u.ID);if err!=nil{http.Error(w,"unable to list threads",500);return};writeJSON(w,200,map[string]any{"board":b,"threads":threads})
}
func (s *Server) boardCreateThread(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"boardID");if !ok{http.Error(w,"invalid board id",400);return}
 if u.Role!="admin"&&s.Boards.IsBanned(id,u.ID){http.Error(w,"you are banned from this board",403);return}
 if _,err:=s.Boards.GetBoard(id);errors.Is(err,sql.ErrNoRows){http.Error(w,"board not found",404);return}else if err!=nil{http.Error(w,"unable to load board",500);return}
 var req struct{Subject string `json:"subject"`;Body string `json:"body"`};if !decodeJSON(w,r,&req){return}
 thread,posts,err:=s.Boards.CreateThread(id,boardUser(u),req.Subject,req.Body);if err!=nil{http.Error(w,err.Error(),400);return};writeJSON(w,201,map[string]any{"thread":thread,"posts":posts})
}
func (s *Server) boardGetThread(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"threadID");if !ok{http.Error(w,"invalid thread id",400);return}
 thread,posts,err:=s.Boards.GetThread(id,u.ID);if errors.Is(err,sql.ErrNoRows){http.Error(w,"thread not found",404);return};if err!=nil{http.Error(w,"unable to load thread",500);return}
 if u.Role!="admin"&&s.Boards.IsBanned(thread.BoardID,u.ID){http.Error(w,"you are banned from this board",403);return}
 if err:=s.Boards.MarkRead(id,u.ID,thread.LastPostID);err!=nil{http.Error(w,"unable to update read state",500);return}
 thread.LastReadPostID=thread.LastPostID;thread.Unread=false;thread.CanModerate=u.Role=="admin"||s.Boards.IsModerator(thread.BoardID,u.ID)
 writeJSON(w,200,map[string]any{"thread":thread,"posts":posts})
}
func (s *Server) boardReply(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"threadID");if !ok{http.Error(w,"invalid thread id",400);return}
 thread,_,err:=s.Boards.GetThread(id,u.ID);if errors.Is(err,sql.ErrNoRows){http.Error(w,"thread not found",404);return};if err!=nil{http.Error(w,"unable to load thread",500);return}
 if u.Role!="admin"&&s.Boards.IsBanned(thread.BoardID,u.ID){http.Error(w,"you are banned from this board",403);return}
 if thread.Locked{http.Error(w,"this thread is locked",409);return}
 var req struct{Body string `json:"body"`};if !decodeJSON(w,r,&req){return}
 post,err:=s.Boards.Reply(id,boardUser(u),req.Body);if err!=nil{http.Error(w,err.Error(),400);return};writeJSON(w,201,post)
}
func (s *Server) boardMarkRead(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"threadID");if !ok{http.Error(w,"invalid thread id",400);return}
 thread,_,err:=s.Boards.GetThread(id,u.ID);if errors.Is(err,sql.ErrNoRows){http.Error(w,"thread not found",404);return};if err!=nil{http.Error(w,"unable to load thread",500);return}
 if u.Role!="admin"&&s.Boards.IsBanned(thread.BoardID,u.ID){http.Error(w,"you are banned from this board",403);return}
 if err:=s.Boards.MarkRead(id,u.ID,thread.LastPostID);err!=nil{http.Error(w,"unable to update read state",500);return};w.WriteHeader(http.StatusNoContent)
}
func (s *Server) adminBoards(w http.ResponseWriter,r *http.Request){
 admin,ok:=s.requireAdmin(w,r);if !ok{return}
 switch r.Method{
 case http.MethodGet:
  items,err:=s.Boards.ListBoards(admin.ID);if err!=nil{http.Error(w,"unable to list boards",500);return};writeJSON(w,200,map[string]any{"boards":items})
 case http.MethodPost:
  var req struct{Name string `json:"name"`;Description string `json:"description"`;Rules string `json:"rules"`};if !decodeJSON(w,r,&req){return}
  b,err:=s.Boards.CreateBoard(req.Name,req.Description,req.Rules,admin.ID);if err!=nil{http.Error(w,err.Error(),400);return}
  _=s.Accounts.LogAudit(admin,"admin.board_create",b.Name,"success","Board created",nil);writeJSON(w,201,b)
 }
}
func (s *Server) adminBoard(w http.ResponseWriter,r *http.Request){
 admin,ok:=s.requireAdmin(w,r);if !ok{return};id,ok:=boardID(r,"boardID");if !ok{http.Error(w,"invalid board id",400);return}
 if r.Method==http.MethodDelete{b,err:=s.Boards.GetBoard(id);if err!=nil{http.Error(w,"board not found",404);return};if err=s.Boards.DeleteBoard(id);err!=nil{http.Error(w,"unable to delete board",500);return};_=s.Accounts.LogAudit(admin,"admin.board_delete",b.Name,"success","Board and its threads removed",nil);w.WriteHeader(204);return}
 var req struct{Name string `json:"name"`;Description string `json:"description"`;Rules string `json:"rules"`};if !decodeJSON(w,r,&req){return}
 b,err:=s.Boards.UpdateBoard(id,req.Name,req.Description,req.Rules);if err!=nil{http.Error(w,err.Error(),400);return};_=s.Accounts.LogAudit(admin,"admin.board_update",b.Name,"success","Board settings updated",nil);writeJSON(w,200,b)
}
func (s *Server) adminBoardModerator(w http.ResponseWriter,r *http.Request){
 admin,ok:=s.requireAdmin(w,r);if !ok{return};board,ok:=boardID(r,"boardID");if !ok{http.Error(w,"invalid board id",400);return}
 username:=strings.TrimSpace(r.PathValue("username"));target,err:=s.Accounts.GetByUsername(username);if err!=nil{http.Error(w,"user not found",404);return}
 var req struct{Enabled bool `json:"enabled"`};if !decodeJSON(w,r,&req){return}
 if err:=s.Boards.SetModerator(board,target.ID,req.Enabled);err!=nil{http.Error(w,"unable to update board moderator",500);return}
 action:="removed";if req.Enabled{action="appointed"};_=s.Accounts.LogAudit(admin,"admin.board_moderator",username,"success",action+" board moderator",nil);writeJSON(w,200,map[string]any{"ok":true})
}
func (s *Server) boardModerateThread(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"threadID");if !ok{http.Error(w,"invalid thread id",400);return}
 thread,_,err:=s.Boards.GetThread(id,u.ID);if err!=nil{http.Error(w,"thread not found",404);return}
 if u.Role!="admin"&&!s.Boards.IsModerator(thread.BoardID,u.ID){http.Error(w,"board moderator access required",403);return}
 var req struct{Locked *bool `json:"locked"`;Sticky *bool `json:"sticky"`};if !decodeJSON(w,r,&req){return}
 if req.Locked==nil&&req.Sticky==nil{http.Error(w,"locked or sticky must be provided",400);return}
 if err:=s.Boards.SetThreadState(id,req.Locked,req.Sticky);err!=nil{http.Error(w,"unable to update thread",500);return}
 details:="board_id="+strconv.FormatInt(thread.BoardID,10);if req.Locked!=nil{details+="; locked="+strconv.FormatBool(*req.Locked)};if req.Sticky!=nil{details+="; sticky="+strconv.FormatBool(*req.Sticky)};_=s.Accounts.LogAudit(u,"board.thread_moderate",strconv.FormatInt(id,10),"success",details,nil)
 writeJSON(w,200,map[string]any{"ok":true})
}
func (s *Server) boardBan(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"boardID");if !ok{http.Error(w,"invalid board id",400);return}
 if u.Role!="admin"&&!s.Boards.IsModerator(id,u.ID){http.Error(w,"board moderator access required",403);return}
 target,err:=s.Accounts.GetByUsername(strings.TrimSpace(r.PathValue("username")));if err!=nil{http.Error(w,"user not found",404);return}
 var req struct{Action string `json:"action"`;Reason string `json:"reason"`};if !decodeJSON(w,r,&req){return}
 switch strings.ToLower(strings.TrimSpace(req.Action)){case "ban":if u.Role!="admin"&&target.Role=="admin"{http.Error(w,"board moderators cannot ban administrators",403);return};err=s.Boards.Ban(id,target.ID,u.ID,req.Reason);case "unban":err=s.Boards.Unban(id,target.ID);default:http.Error(w,"action must be ban or unban",400);return}
 if err!=nil{http.Error(w,"unable to update board ban",500);return}
 action:="board.user_unban";if strings.EqualFold(strings.TrimSpace(req.Action),"ban"){action="board.user_ban"};details:="board_id="+strconv.FormatInt(id,10);_=s.Accounts.LogAudit(u,action,target.Username,"success",details,&target.ID)
 writeJSON(w,200,map[string]any{"ok":true})
}

func (s *Server) boardDeleteThread(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"threadID");if !ok{http.Error(w,"invalid thread id",400);return}
 thread,_,err:=s.Boards.GetThread(id,u.ID);if err!=nil{http.Error(w,"thread not found",404);return}
 if u.Role!="admin"&&!s.Boards.IsModerator(thread.BoardID,u.ID){http.Error(w,"board moderator access required",403);return}
 if err:=s.Boards.DeleteThread(id);err!=nil{http.Error(w,"unable to delete thread",500);return};_=s.Accounts.LogAudit(u,"board.thread_delete",thread.Subject,"success","board_id="+strconv.FormatInt(thread.BoardID,10)+"; thread_id="+strconv.FormatInt(id,10),nil);w.WriteHeader(http.StatusNoContent)
}
func (s *Server) boardDeletePost(w http.ResponseWriter,r *http.Request){
 u,ok:=s.user(r);if !ok{http.Error(w,"unauthorized",401);return};id,ok:=boardID(r,"postID");if !ok{http.Error(w,"invalid post id",400);return}
 var threadID,boardID int64
 if err:=s.Boards.DB.QueryRow("SELECT p.thread_id,t.board_id FROM board_posts p JOIN board_threads t ON t.id=p.thread_id WHERE p.id=?",id).Scan(&threadID,&boardID);err!=nil{http.Error(w,"post not found",404);return}
 if u.Role!="admin"&&!s.Boards.IsModerator(boardID,u.ID){http.Error(w,"board moderator access required",403);return}
 if err:=s.Boards.DeletePost(id);err!=nil{http.Error(w,"unable to delete post",500);return};_=s.Accounts.LogAudit(u,"board.post_delete",strconv.FormatInt(id,10),"success","board_id="+strconv.FormatInt(boardID,10)+"; thread_id="+strconv.FormatInt(threadID,10)+"; post_id="+strconv.FormatInt(id,10),nil);w.WriteHeader(http.StatusNoContent)
}
