package boards

import (
 "database/sql"
 "errors"
 "strings"
 "time"
)

type Service struct { DB *sql.DB }
type Board struct {
 ID int64 `json:"id"`
 Name string `json:"name"`
 Description string `json:"description"`
 Rules string `json:"rules"`
 CreatedAt string `json:"created_at"`
 ThreadCount int `json:"thread_count"`
 UnreadCount int `json:"unread_count"`
}
type Thread struct {
 ID int64 `json:"id"`
 BoardID int64 `json:"board_id"`
 Subject string `json:"subject"`
 Author string `json:"author"`
 CreatedAt string `json:"created_at"`
 LastPostAt string `json:"last_post_at"`
 ReplyCount int `json:"reply_count"`
 Locked bool `json:"locked"`
 Sticky bool `json:"sticky"`
 LastReadPostID int64 `json:"last_read_post_id"`
 LastPostID int64 `json:"last_post_id"`
 Unread bool `json:"unread"`
 CanModerate bool `json:"can_moderate,omitempty"`
 LastPostAuthor string `json:"last_post_author"`
}
type Post struct {
 ID int64 `json:"id"`
 ThreadID int64 `json:"thread_id"`
 AuthorID int64 `json:"author_id"`
 Author string `json:"author"`
 DisplayName string `json:"display_name"`
 Body string `json:"body"`
 CreatedAt string `json:"created_at"`
}
type User struct { ID int64; Username, DisplayName, Role string }

func (s Service) Initialize() error {
	statements := []string{
		"CREATE TABLE IF NOT EXISTS boards (id INTEGER PRIMARY KEY AUTOINCREMENT,name TEXT NOT NULL UNIQUE COLLATE NOCASE,description TEXT NOT NULL DEFAULT '',rules TEXT NOT NULL DEFAULT '',created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)",
		"CREATE TABLE IF NOT EXISTS board_moderators (board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,PRIMARY KEY(board_id,user_id))",
		"CREATE TABLE IF NOT EXISTS board_bans (board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,reason TEXT NOT NULL DEFAULT '',created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,PRIMARY KEY(board_id,user_id))",
		"CREATE TABLE IF NOT EXISTS board_threads (id INTEGER PRIMARY KEY AUTOINCREMENT,board_id INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,subject TEXT NOT NULL,author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,last_post_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,locked INTEGER NOT NULL DEFAULT 0 CHECK(locked IN(0,1)),sticky INTEGER NOT NULL DEFAULT 0 CHECK(sticky IN(0,1)))",
		"CREATE INDEX IF NOT EXISTS idx_board_threads_activity ON board_threads(board_id,sticky DESC,last_post_at DESC)",
		"CREATE TABLE IF NOT EXISTS board_posts (id INTEGER PRIMARY KEY AUTOINCREMENT,thread_id INTEGER NOT NULL REFERENCES board_threads(id) ON DELETE CASCADE,author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,body TEXT NOT NULL,created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)",
		"CREATE INDEX IF NOT EXISTS idx_board_posts_thread ON board_posts(thread_id,id)",
		"CREATE TABLE IF NOT EXISTS board_reads (user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,thread_id INTEGER NOT NULL REFERENCES board_threads(id) ON DELETE CASCADE,last_read_post_id INTEGER NOT NULL DEFAULT 0,last_read_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,PRIMARY KEY(user_id,thread_id))",
	}
	for _, statement := range statements {
		if _, err := s.DB.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}
func trim(v string,max int) string { v=strings.TrimSpace(v); if len([]rune(v))>max{return ""};return v }
func (s Service) ListBoards(userID int64)([]Board,error){
 rows,err:=s.DB.Query("SELECT b.id,b.name,b.description,b.rules,b.created_at,COUNT(DISTINCT t.id),SUM(CASE WHEN t.id IS NOT NULL AND COALESCE(r.last_read_post_id,0)<COALESCE(lp.id,0) THEN 1 ELSE 0 END) FROM boards b LEFT JOIN board_threads t ON t.board_id=b.id LEFT JOIN board_reads r ON r.thread_id=t.id AND r.user_id=? LEFT JOIN (SELECT thread_id,MAX(id) id FROM board_posts GROUP BY thread_id) lp ON lp.thread_id=t.id GROUP BY b.id ORDER BY b.name COLLATE NOCASE",userID)
 if err!=nil{return nil,err};defer rows.Close();out:=[]Board{}
 for rows.Next(){var b Board;var unread sql.NullInt64;if err:=rows.Scan(&b.ID,&b.Name,&b.Description,&b.Rules,&b.CreatedAt,&b.ThreadCount,&unread);err!=nil{return nil,err};b.UnreadCount=int(unread.Int64);out=append(out,b)}
 return out,rows.Err()
}
func(s Service) GetBoard(id int64)(Board,error){var b Board;err:=s.DB.QueryRow("SELECT id,name,description,rules,created_at FROM boards WHERE id=?",id).Scan(&b.ID,&b.Name,&b.Description,&b.Rules,&b.CreatedAt);return b,err}
func(s Service) CreateBoard(name,description,rules string,actor int64)(Board,error){
 name=trim(name,64);description=trim(description,500);rules=trim(rules,4000);if name==""{return Board{},errors.New("board name must be 1-64 characters")}
 res,err:=s.DB.Exec("INSERT INTO boards(name,description,rules,created_by) VALUES(?,?,?,?)",name,description,rules,actor);if err!=nil{if strings.Contains(strings.ToLower(err.Error()),"unique"){return Board{},errors.New("a board with that name already exists")};return Board{},err};id,err:=res.LastInsertId();if err!=nil{return Board{},err};return s.GetBoard(id)
}
func(s Service) UpdateBoard(id int64,name,description,rules string)(Board,error){name=trim(name,64);description=trim(description,500);rules=trim(rules,4000);if id<1||name==""{return Board{},errors.New("board id and a 1-64 character name are required")};_,err:=s.DB.Exec("UPDATE boards SET name=?,description=?,rules=?,updated_at=CURRENT_TIMESTAMP WHERE id=?",name,description,rules,id);if err!=nil{return Board{},err};return s.GetBoard(id)}
func(s Service) DeleteBoard(id int64)error{res,err:=s.DB.Exec("DELETE FROM boards WHERE id=?",id);if err!=nil{return err};n,_:=res.RowsAffected();if n==0{return sql.ErrNoRows};return nil}
func(s Service) IsModerator(board,user int64)bool{var n int;return s.DB.QueryRow("SELECT COUNT(*) FROM board_moderators WHERE board_id=? AND user_id=?",board,user).Scan(&n)==nil&&n>0}
func(s Service) SetModerator(board,user int64,enabled bool)error{if enabled{_,err:=s.DB.Exec("INSERT OR IGNORE INTO board_moderators(board_id,user_id) VALUES(?,?)",board,user);return err};_,err:=s.DB.Exec("DELETE FROM board_moderators WHERE board_id=? AND user_id=?",board,user);return err}
func(s Service) IsBanned(board,user int64)bool{var n int;return s.DB.QueryRow("SELECT COUNT(*) FROM board_bans WHERE board_id=? AND user_id=?",board,user).Scan(&n)==nil&&n>0}
func(s Service) Ban(board,user,actor int64,reason string)error{_,err:=s.DB.Exec("INSERT INTO board_bans(board_id,user_id,reason,created_by) VALUES(?,?,?,?) ON CONFLICT(board_id,user_id) DO UPDATE SET reason=excluded.reason,created_by=excluded.created_by,created_at=CURRENT_TIMESTAMP",board,user,trim(reason,500),actor);return err}
func(s Service) Unban(board,user int64)error{_,err:=s.DB.Exec("DELETE FROM board_bans WHERE board_id=? AND user_id=?",board,user);return err}
func(s Service) ListThreads(board,user int64)([]Thread,error){
 rows,err:=s.DB.Query("SELECT t.id,t.board_id,t.subject,u.username,t.created_at,t.last_post_at,(SELECT CASE WHEN COUNT(*)>0 THEN COUNT(*)-1 ELSE 0 END FROM board_posts p WHERE p.thread_id=t.id),t.locked,t.sticky,COALESCE(r.last_read_post_id,0),COALESCE((SELECT MAX(id) FROM board_posts p3 WHERE p3.thread_id=t.id),0),COALESCE((SELECT u2.username FROM board_posts p2 JOIN users u2 ON u2.id=p2.author_id WHERE p2.thread_id=t.id ORDER BY p2.id DESC LIMIT 1),u.username) FROM board_threads t JOIN users u ON u.id=t.author_id LEFT JOIN board_reads r ON r.thread_id=t.id AND r.user_id=? WHERE t.board_id=? ORDER BY t.sticky DESC,t.last_post_at DESC,t.id DESC",user,board)
 if err!=nil{return nil,err};defer rows.Close();out:=[]Thread{}
 for rows.Next(){var t Thread;var locked,sticky int;if err:=rows.Scan(&t.ID,&t.BoardID,&t.Subject,&t.Author,&t.CreatedAt,&t.LastPostAt,&t.ReplyCount,&locked,&sticky,&t.LastReadPostID,&t.LastPostID,&t.LastPostAuthor);err!=nil{return nil,err};t.Locked=locked!=0;t.Sticky=sticky!=0;t.Unread=t.LastPostID>t.LastReadPostID;out=append(out,t)}
 return out,rows.Err()
}
func(s Service) GetThread(id,user int64)(Thread,[]Post,error){
 var t Thread;var locked,sticky int
 err:=s.DB.QueryRow("SELECT t.id,t.board_id,t.subject,u.username,t.created_at,t.last_post_at,(SELECT CASE WHEN COUNT(*)>0 THEN COUNT(*)-1 ELSE 0 END FROM board_posts p WHERE p.thread_id=t.id),t.locked,t.sticky,COALESCE(r.last_read_post_id,0),COALESCE((SELECT MAX(id) FROM board_posts p3 WHERE p3.thread_id=t.id),0),COALESCE((SELECT u2.username FROM board_posts p2 JOIN users u2 ON u2.id=p2.author_id WHERE p2.thread_id=t.id ORDER BY p2.id DESC LIMIT 1),u.username) FROM board_threads t JOIN users u ON u.id=t.author_id LEFT JOIN board_reads r ON r.thread_id=t.id AND r.user_id=? WHERE t.id=?",user,id).Scan(&t.ID,&t.BoardID,&t.Subject,&t.Author,&t.CreatedAt,&t.LastPostAt,&t.ReplyCount,&locked,&sticky,&t.LastReadPostID,&t.LastPostID,&t.LastPostAuthor)
 if err!=nil{return Thread{},nil,err};t.Locked=locked!=0;t.Sticky=sticky!=0;t.Unread=t.LastPostID>t.LastReadPostID
 rows,err:=s.DB.Query("SELECT p.id,p.thread_id,p.author_id,u.username,u.display_name,p.body,p.created_at FROM board_posts p JOIN users u ON u.id=p.author_id WHERE p.thread_id=? ORDER BY p.id",id);if err!=nil{return Thread{},nil,err};defer rows.Close();posts:=[]Post{}
 for rows.Next(){var p Post;if err:=rows.Scan(&p.ID,&p.ThreadID,&p.AuthorID,&p.Author,&p.DisplayName,&p.Body,&p.CreatedAt);err!=nil{return Thread{},nil,err};posts=append(posts,p)}
 return t,posts,rows.Err()
}
func(s Service) CreateThread(board int64,user User,subject,body string)(Thread,[]Post,error){
 subject=trim(subject,120);body=trim(body,10000);if subject==""{return Thread{},nil,errors.New("subject must be 1-120 characters")};if body==""{return Thread{},nil,errors.New("post body must be 1-10000 characters")};if user.Role != "admin" && s.IsBanned(board,user.ID){return Thread{},nil,errors.New("you are banned from this board")};if _,err:=s.GetBoard(board);err!=nil{return Thread{},nil,errors.New("board not found")}
 tx,err:=s.DB.Begin();if err!=nil{return Thread{},nil,err};defer tx.Rollback();now:=time.Now().UTC().Format(time.RFC3339)
 res,err:=tx.Exec("INSERT INTO board_threads(board_id,subject,author_id,created_at,last_post_at) VALUES(?,?,?,?,?)",board,subject,user.ID,now,now);if err!=nil{return Thread{},nil,err};id,err:=res.LastInsertId();if err!=nil{return Thread{},nil,err};post,err:=tx.Exec("INSERT INTO board_posts(thread_id,author_id,body,created_at) VALUES(?,?,?,?)",id,user.ID,body,now);if err!=nil{return Thread{},nil,err};postID,err:=post.LastInsertId();if err!=nil{return Thread{},nil,err}
 _,err=tx.Exec("INSERT INTO board_reads(user_id,thread_id,last_read_post_id,last_read_at) VALUES(?,?,?,?)",user.ID,id,postID,now);if err!=nil{return Thread{},nil,err};if err=tx.Commit();err!=nil{return Thread{},nil,err};return s.GetThread(id,user.ID)
}
func (s Service) Reply(thread int64, user User, body string) (Post, error) {
	body = trim(body, 10000)
	if body == "" {
		return Post{}, errors.New("reply must be 1-10000 characters")
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return Post{}, err
	}
	defer tx.Rollback()

	var board int64
	var locked int
	if err := tx.QueryRow("SELECT board_id,locked FROM board_threads WHERE id=?", thread).Scan(&board, &locked); err != nil {
		return Post{}, err
	}
	if locked != 0 {
		return Post{}, errors.New("this thread is locked")
	}

	var banned int
	if err := tx.QueryRow("SELECT COUNT(*) FROM board_bans WHERE board_id=? AND user_id=?", board, user.ID).Scan(&banned); err != nil {
		return Post{}, err
	}
	if banned > 0 && user.Role != "admin" {
		return Post{}, errors.New("you are banned from this board")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := tx.Exec(
		"INSERT INTO board_posts(thread_id,author_id,body,created_at) VALUES(?,?,?,?)",
		thread, user.ID, body, now,
	)
	if err != nil {
		return Post{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Post{}, err
	}
	if _, err := tx.Exec("UPDATE board_threads SET last_post_at=? WHERE id=?", now, thread); err != nil {
		return Post{}, err
	}
	if _, err := tx.Exec(
		"INSERT INTO board_reads(user_id,thread_id,last_read_post_id,last_read_at) VALUES(?,?,?,?) "+
			"ON CONFLICT(user_id,thread_id) DO UPDATE SET last_read_post_id=excluded.last_read_post_id,last_read_at=excluded.last_read_at",
		user.ID, thread, id, now,
	); err != nil {
		return Post{}, err
	}

	var post Post
	if err := tx.QueryRow(
		"SELECT p.id,p.thread_id,p.author_id,u.username,u.display_name,p.body,p.created_at "+
			"FROM board_posts p JOIN users u ON u.id=p.author_id WHERE p.id=?",
		id,
	).Scan(&post.ID, &post.ThreadID, &post.AuthorID, &post.Author, &post.DisplayName, &post.Body, &post.CreatedAt); err != nil {
		return Post{}, err
	}
	if err := tx.Commit(); err != nil {
		return Post{}, err
	}
	return post, nil
}

func(s Service) MarkRead(thread,user,last int64)error{
 var max int64;if err:=s.DB.QueryRow("SELECT COALESCE(MAX(id),0) FROM board_posts WHERE thread_id=?",thread).Scan(&max);err!=nil{return err};if last<=0||last>max{last=max}
 _,err:=s.DB.Exec("INSERT INTO board_reads(user_id,thread_id,last_read_post_id,last_read_at) VALUES(?,?,?,CURRENT_TIMESTAMP) ON CONFLICT(user_id,thread_id) DO UPDATE SET last_read_post_id=MAX(board_reads.last_read_post_id,excluded.last_read_post_id),last_read_at=CURRENT_TIMESTAMP",user,thread,last);return err
}
func (s Service) SetThreadState(thread int64, locked, sticky *bool) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if locked != nil {
		value := 0
		if *locked {
			value = 1
		}
		if _, err := tx.Exec("UPDATE board_threads SET locked=? WHERE id=?", value, thread); err != nil {
			return err
		}
	}
	if sticky != nil {
		value := 0
		if *sticky {
			value = 1
		}
		if _, err := tx.Exec("UPDATE board_threads SET sticky=? WHERE id=?", value, thread); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func(s Service) DeleteThread(thread int64)error{res,err:=s.DB.Exec("DELETE FROM board_threads WHERE id=?",thread);if err!=nil{return err};n,_:=res.RowsAffected();if n==0{return sql.ErrNoRows};return nil}
func (s Service) DeletePost(post int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var thread int64
	if err := tx.QueryRow("SELECT thread_id FROM board_posts WHERE id=?", post).Scan(&thread); err != nil {
		return err
	}
	var first int64
	if err := tx.QueryRow("SELECT MIN(id) FROM board_posts WHERE thread_id=?", thread).Scan(&first); err != nil {
		return err
	}
	if post == first {
		if _, err := tx.Exec("DELETE FROM board_threads WHERE id=?", thread); err != nil {
			return err
		}
		return tx.Commit()
	}
	if _, err := tx.Exec("DELETE FROM board_posts WHERE id=?", post); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"UPDATE board_threads SET last_post_at=COALESCE((SELECT created_at FROM board_posts WHERE thread_id=? ORDER BY id DESC LIMIT 1),created_at) WHERE id=?",
		thread, thread,
	); err != nil {
		return err
	}
	return tx.Commit()
}
