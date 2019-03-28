package main

import "io"
import "log"
import "fmt"
import "strings"
import "database/sql"
import "github.com/dustin/go-nntp/server"
import "github.com/dustin/go-nntp"
import "encoding/json"
import "net/textproto"
import "time"
import "io/ioutil"
import "strconv"
import "crypto/sha512"

type Backend struct {
	DB *sql.DB
	NodeName string
	AllowSignup bool
	dbType string
	username string
}

func NewBackend(t, u string) (b *Backend, e error) {
	b = &Backend{}
	b.dbType = t
	b.DB, e = sql.Open(t, u)
	return
}
func (b *Backend) Clone(u string) *Backend {
	b2 := *b
	b3 := &b2
	b3.username = u
	return b3
}
func (b *Backend) Init() (e error) {
	if b.dbType == "sqlite3" {
		b.DB.Exec("PRAGMA journal_mode=wal;")
	}
	_, e = b.DB.Exec(`
CREATE TABLE IF NOT EXISTS lastsync(uplink VARCHAR(256) PRIMARY KEY, date INTEGER);
CREATE TABLE IF NOT EXISTS users(username VARCHAR(256) PRIMARY KEY, password VARCHAR(256));
CREATE TABLE IF NOT EXISTS posts(id VARCHAR(256) PRIMARY KEY, groupname VARCHAR(256), subject VARCHAR(512), date INTEGER, headers VARCHAR(8192), body VARCHAR(65536), votes INTEGER);
CREATE TABLE IF NOT EXISTS groups(groupname VARCHAR(256) PRIMARY KEY, owner VARCHAR(256), info VARCHAR(1024), permissions VARCHAR(1), num INTEGER, date INTEGER);
INSERT OR IGNORE INTO groups VALUES('nntpit', 'ronsor@nntpit', 'NNTPIt Discussion', 'y', 0, 1);
CREATE TABLE IF NOT EXISTS blacklist(groupname VARCHAR(256) PRIMARY KEY);
`)
	return
}
func (b *Backend) AddGroup(grname string) (e error) {
	_, e = b.DB.Exec("INSERT INTO groups VALUES(?, 'nobody@nobody.test', ?, 'y', 0, 0);", grname, grname)
	return
}
func (b *Backend) SetSyncTime(uplink string, date int64) {
	b.DB.Exec("INSERT OR REPLACE INTO lastsync VALUES(?, ?)", uplink, date)
}
func (b *Backend) GetSyncTime(uplink string) (date int64) {
	rows, err := b.DB.Query("SELECT date FROM lastsync WHERE uplink = ?", uplink)
	if err != nil { return }
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&date)
	}
	return
}
func deserializeGroup(rows *sql.Rows) *nntp.Group {
	var name, owner, info, permissions string
	var num, date int64
	rows.Scan(&name, &owner, &info, &permissions, &num, &date)
	grp := &nntp.Group{
		Name: name,
		Description: info + "$!$!$" + owner,
		Count: date,
		Low: 0,
		High: num,
		Posting: nntp.PostingStatus('y'),
	}
	return grp
}
func deserializeArticle(rows *sql.Rows) *nntp.Article {
	var id, group, subject, /* date, */ hdr, body /*, votes */ string
	var rowid, date, votes int64
	var lines int
	var hdrdecode textproto.MIMEHeader
	rows.Scan(&rowid, &id, &group, &subject, &date, &hdr, &body, &votes)
	json.Unmarshal([]byte(hdr), &hdrdecode)
	hdrdecode.Set("Group-Index", fmt.Sprintf("%d", rowid))
	hdrdecode.Set("Votes", fmt.Sprintf("%d", votes))
	lines = len(body) / 80 // just guess
	art := &nntp.Article{
		Header: hdrdecode,
		Body: strings.NewReader(body),
		Bytes: len(body),
		Lines: lines,
	}
	return art
}
var grpRecent = &nntp.Group{
		Name: "recent",
		Count: 1,
		Low: 1,
		High: 1,
		Posting: nntp.PostingStatus('n'),
		Description: "The most recent posts.\n$!$!$",
		}
func (b *Backend) ListGroups(max int) ([]*nntp.Group, error) {
	out := []*nntp.Group{}
	rows, err := b.DB.Query("SELECT * FROM groups LIMIT ?", max)
	if err != nil { return nil, err }
	defer rows.Close()
	for rows.Next() {
		out = append(out, deserializeGroup(rows))
		out[len(out)-1].Description = "A group"
	}
	//out = append(out, grpRecent)
	return out, nil
}
func (b *Backend) GetArticlesSince(i int64) ([]*nntp.Article, error) {
	out := []*nntp.Article{}
	rows, err := b.DB.Query("SELECT rowid, * FROM posts WHERE date > ? ORDER BY date LIMIT 250", i)
	if err != nil { return out, err }
	defer rows.Close()
	for rows.Next() {
		art := deserializeArticle(rows)
		out = append(out, art)
	}
//	if len(out) == 0 {
//		return out, nntpserver.ErrInvalidArticleNumber
//	}
	return out, nil
}
func (b *Backend) IsBanned(name string) bool {
	ban := false
	rows, err := b.DB.Query("SELECT * FROM blacklist WHERE ? LIKE groupname", name)
	if err != nil { return false }
	for rows.Next() {
		ban = true
	}
	return ban
}
func (b *Backend) GetGroup(name string) (*nntp.Group, error) {
	if name == "recent" {
		grpRecent.High = time.Now().Unix()
		grp2 := *grpRecent
		return &grp2, nil
	}
	out := (*nntp.Group)(nil)
	rows, err := b.DB.Query("SELECT * FROM groups WHERE groupname LIKE ?", name)
	if err != nil { return nil, err }
	defer rows.Close()
	for rows.Next() {
		out = deserializeGroup(rows)
	}
	if out == nil {
		return nil, nntpserver.ErrNoSuchGroup
	}
	return out, nil
}
func (b *Backend) AllowPost() bool {
	return true
}
func (b *Backend) UpdateVote(a *nntp.Article, d int64) (err error) {
	_, err = b.DB.Exec("UPDATE posts SET votes = votes + ? WHERE id = ?", d, a.MessageID())
	return
}
func (b *Backend) Authenticate(user, pass string) (nntpserver.Backend, error) {
	s2 := strings.Split(user, ":")
	signup := false
	if b.AllowSignup && len(s2) == 2 && s2[0] == "signup" {
		signup = true
		user = s2[1]
	}
	rows, err := b.DB.Query("SELECT username, password FROM users WHERE username LIKE ?", user)
	pass = fmt.Sprintf("%x", sha512.Sum512([]byte(user + ":" + pass)))
	if err != nil { return nil, err }
	defer rows.Close()
	ok := signup
	for rows.Next() {
		var opass string
		rows.Scan(&user, &opass)
		if signup && opass != pass {
			ok = false
			break
		}
		if opass == pass {
			ok = true
			break
		}
	}
	if !ok {
		return nil, nntpserver.ErrAuthRejected
	}
	if signup {
		b.DB.Exec("INSERT INTO users VALUES(?, ?);", user, pass)
	}
	return b.Clone(user), nil
}
func (b *Backend) Authorized() bool {
	return b.username != ""
}
func (b *Backend) GetArticle(group *nntp.Group, id string) (*nntp.Article, error) {
	out := (*nntp.Article)(nil)
	var rows *sql.Rows
	var err error
	if intid, err := strconv.ParseInt(id, 10, 64); err == nil {
		gname := group.Name
		rows, err = b.DB.Query("SELECT rowid, * FROM posts WHERE groupname LIKE ? ORDER BY date ASC LIMIT 1 OFFSET ?", gname, intid)
	} else {
		if group != nil && group.Name != "recent" {
			gname := group.Name
			rows, err = b.DB.Query("SELECT rowid, * FROM posts WHERE groupname LIKE ? AND id = ?", gname, id)
		} else {
			rows, err = b.DB.Query("SELECT rowid, * FROM posts WHERE id = ?", id)
		}
	}
	if err != nil { return nil, err }
	defer rows.Close()
	for rows.Next() {
		out = deserializeArticle(rows)
	}
	if out == nil {
		return nil, nntpserver.ErrInvalidArticleNumber
	}
	return out, nil
}
func (b *Backend) GetArticles(group *nntp.Group, from, to int64) ([]nntpserver.NumberedArticle, error) {
	gname := group.Name
	out := []nntpserver.NumberedArticle{}
	APNDX := ""
	APNDX += "AND subject NOT LIKE 'Vote: %' AND subject NOT LIKE '' "
	if group.Description == "!UNIQUE" {
		APNDX += "AND subject NOT LIKE 'Re: %' "
	}
	rows, err := b.DB.Query("SELECT rowid, * FROM posts WHERE groupname LIKE ? " + APNDX + " ORDER BY date DESC LIMIT ? OFFSET ?", gname, to - from, from)
	if gname == "recent" {
		if err == nil { rows.Close() }
		if from >= 0 {
			rows, err = b.DB.Query("SELECT rowid, * FROM posts WHERE date > ? " + APNDX + " ORDER BY date DESC LIMIT ?", from, to)
		} else {
			rows, err = b.DB.Query("SELECT rowid, * FROM posts WHERE 1 = 1 " + APNDX + " ORDER BY date DESC LIMIT ? OFFSET ?", to, -(from)-1)
		}
	}
	if err != nil { return out, err }
	i := from
	defer rows.Close()
	for rows.Next() {
		if i > to { break }
		art := nntpserver.NumberedArticle{i, deserializeArticle(rows)}
		out = append(out, art)
		i++
	}
	return out, nil
}
func (b *Backend) Post(n *nntp.Article) error {
	if b.username != "" {
		n.Header.Set("From", b.username + "@" + b.NodeName)
	}
	if n.Header.Get("Message-ID") == "" {
		n.Header.Set("Message-ID", GUID() + "@" + b.NodeName)
	}
	if len(n.Header["Newsgroups"]) > 1 {
		return nntpserver.ErrPostingFailed
	}
	if _, err := strconv.ParseInt(n.Header.Get("Unix-Date"), 10, 64); err != nil {
		n.Header.Set("Unix-Date", fmt.Sprintf("%d", time.Now().Unix()))
	}
	date, _ := strconv.ParseInt(n.Header.Get("Unix-Date"), 10, 64);
	subj := n.Header.Get("Subject")
	id := n.Header.Get("Message-ID")
	votes := 1
	if n.Header.Get("Do") == "vote" {
		b.DB.Exec("UPDATE posts SET votes = votes + ? WHERE id = ?", votes, id)
		io.Copy(ioutil.Discard, n.Body)
		return &nntpserver.NNTPError{242, "Just voting"}
	}
	if b.IsBanned(n.Header.Get("From") + ":" + n.Header.Get("Newsgroups")) {
		return nntpserver.ErrPostingFailed
	}
	if n.Header.Get("Do") == "addgroup" {
		b.AddGroup(n.Header.Get("Newsgroups"))
		io.Copy(ioutil.Discard, n.Body)
		return &nntpserver.NNTPError{242, "Added group"}
	}
	data, _ := json.Marshal(n.Header)
	oa, err := b.GetArticle(nil, n.Header.Get("Message-ID"))
	if err == nil && id != "sidebar@" + n.Header.Get("Newsgroups") {
		return nntpserver.ErrPostingFailed
	}
	gallowed := append([]string{n.Header.Get("From")}, strings.Fields(n.Header.Get("Moderators"))...)
	if strings.HasPrefix(id, "sidebar@") && err == nil {
		ud, _ := strconv.ParseInt(oa.Header.Get("Unix-Date"), 10, 64)
		if date < ud {
			return nntpserver.ErrPostingFailed
		}
		allowed := append([]string{oa.Header.Get("From")}, strings.Split(oa.Header.Get("Moderators"), " ")...)
		ok := false
		for _, x := range allowed {
			if x == n.Header.Get("From") {
				ok = true
				break
			}
		}
		if !ok {
			return nntpserver.ErrPostingFailed
		}
	}
	slurp, _ := ioutil.ReadAll(n.Body)
	for _, g := range n.Header["Newsgroups"] {
		if id == "sidebar@" + g {
			b.DB.Exec("UPDATE groups SET info = ? WHERE groupname LIKE ?", string(slurp), g)
			b.DB.Exec("UPDATE groups SET owner = ? WHERE groupname LIKE ?", strings.Join(gallowed, " "), g)
		}
		b.DB.Exec("UPDATE groups SET num = num + 1 WHERE groupname LIKE ?", g)
		b.DB.Exec("UPDATE groups SET date = ? WHERE groupname LIKE ? AND date < ?", date, date)
		// format: &id, &group, &subject, &date, &hdr, &body, &votes
		_, err := b.DB.Exec("INSERT OR REPLACE INTO posts VALUES(?, ?, ?, ?, ?, ?, ?);", id, g, subj, date, data, string(slurp), votes)
		if err != nil {
			log.Println("Warning: POST failed:", err.Error())
		}
	}
	return nil
}
