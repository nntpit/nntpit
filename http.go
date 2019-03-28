package main
import "net/textproto"
import "github.com/golang-commonmark/markdown"
import "log"
import "github.com/dustin/go-nntp"
import "github.com/dustin/go-nntp/server"
import "io/ioutil"
import "strconv"
import "github.com/ararog/timeago"
import "html"
import "io"
import "net/http"
import "strings"
import "github.com/julienschmidt/httprouter"
import "fmt"
import "time"
import "sort"
import "flag"
var md = markdown.New(markdown.XHTMLOutput(true), markdown.Nofollow(true))
var fmtabout = flag.String("html.about", "", "File to use for about page template.")
var fmtheader = flag.String("html.header", "", "File to use for header template.")
var fmtfooter = flag.String("html.footer", "", "File to use for footer template.")
var fmtcss = flag.String("css.style", "", "File to use for CSS styles.")
var coinhive = flag.String("captcha.coinhive", "", "Site key for Coinhive captcha. Leave blank to disable.")
var hostilecaptcha = flag.Bool("captcha.hostile", false, "Make the captcha more hostile: require it at logins and when submitting posts. (Please don't do this.)")
func _useFmtOr(fn string, tpl string) string {
	if fn != "" {
		fd, err := ioutil.ReadFile(fn)
		if err != nil { log.Fatal(err) }
		tpl = string(fd)
	}
	return tpl
}
func InitTemplates() {
	ABOUT_TXT = _useFmtOr(*fmtabout, ABOUT_TXT)
	CSS = _useFmtOr(*fmtcss, CSS)
	HEADER_FMT = _useFmtOr(*fmtheader, HEADER_FMT)
	FOOTER_FMT = _useFmtOr(*fmtfooter, FOOTER_FMT)
}
var ABOUT_TXT = `
<section style='margin-left: 15px;'>
<h2>About</h2>
<article>
<h2>$host</h2>
<p>$host is a node on the NNTPIt network. Posts, links, and groups are shared
between other networks. If you obtain an account on this node, you can post to
the network and your posts will be distributed to all peers.</p>
<p>As a side note, upvotes are not shared between peers.</p>
<h2>Features</h2>
<p>$host fully supports posts formatted with markdown.</p>
<h2>License</h2>
<p>$host is powered by the <code>go-nntpit</code> software.</p>
<p>go-nntpit is licensed under the MIT license. go-nntpit (C) 2018 ronsor.</p>
</article>
</section>
`
var COINHIVE_CAPTCHA = `
	<script src="https://authedmine.com/lib/captcha.min.js" async></script>
	<div class="coinhive-captcha" 
		data-hashes="1024" 
		data-key="$coinhive"
		data-whitelabel="false"
		data-disable-elements="input[type=submit]"
	>
		<em>Loading Captcha...<br>
		If it doesn't load, please disable uBlock or similar software!</em>
	</div>
`
var HEADER_FMT = `
<html>
	<head>
		<title>$title</title>
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<link rel="stylesheet" type="text/css" href="/style.css" />
		<script>
			function toggleSidebar() {
				var sb = document.querySelectorAll('aside')[0];
				var opts = { none: 'block', block: 'none', '': 'block' };
				var newstyle = opts[sb.style.display];
				sb.style.display = newstyle;
			}
		</script>
	</head>
	<body>
		<header><a href='/' class='linktitle'><h1>$host</h1></a>
		<span class='only-group'><a class='btn primarybtn' href='/g/$title?sort=hot'>Hot</a> <a class='btn' href='/g/$title?sort=new'>New</a></span>
		</header>
		<main>
		<a class='btn sidebtn' id='show-sidebar-link' onClick='toggleSidebar()'>Toggle Sidebar</a>
`
var FOOTER_FMT = `
		</main>
		<footer>
			<p><a href='/about'>About</a></p>
			(C) 2018 $host. All rights reserved.
		</footer>
	</body>
</html>
`
var CSS = `
.only-group {
	display: none;
}
body {
	background: #f4f4f4;
	color: #404040;
	margin: 0;
	font-family: sans-serif;
	font-size: 11pt;
}
hr {
	border: none;
	border-top: 1px solid #dcdcdc;
}
a {
	color: #4aabe7;
}
.linktitle, .subtitle {
	color: #404040;
	text-decoration: none;
}
.linktitle:hover, .subtitle:hover {
	color: #4aabe7;
}
.newpost, .newpost input {
	font-size: 13pt;
}
main {
	margin-top: 16px;
	min-height: 60%;
	width: calc(100% - 250px);
	background: #fefefe;
	border-radius: 2px;
	border: 1px solid #dcdcdc;
	margin-left: auto;
	margin-right: auto;
}
main ul {
	list-style-type: none;
}
aside {
	float: right;
	width: 250px;
	padding: 5px;
}
header {
	padding-top: 5px;
	padding-bottom: 5px;
	padding-left: 125px;
	padding-right: 125px;
	display: block;
	background: #fefefe;
	width: calc(100% - 250px);
	min-height: 70px;
	border-bottom: 1px solid #dcdcdc;
	border-top: 1px solid #dcdcdc;
}
footer {
	text-align: center;
	display: block;
}
.voter {
	display: inline-block;
	text-align: center;
	float: left;
	height: 48px;
	max-width: 16px;
	padding-right: 5px;
	padding-left: 5px;
}
.primarybtn {
	background: linear-gradient(to bottom,#bbe5ff,#a6ddff) !important;
	border: 1px solid #70b7e3 !important;
}
.sidebtn {
	width: 90%;
	margin: auto;
	text-align: center;
}
.btn {
	background: linear-gradient(to bottom,#f8f8f8,#fafafa);
	border: 1px solid #d0d0d0;
	text-decoration: none;
	color: #444;
	display: inline-block;
	padding: 7px;
}
.loginheader {
	width: calc(100% - 25px);
	margin: auto;
	margin-top: 5px;
	margin-bottom: 5px;
	text-align: right;
	background: #f4f4f4;
	padding: 5px;
	border-radius: 2px;
	border: 1px solid #dcdcdc;
}
.votercount a {
	display: block;
	text-decoration: none;
}
.votercount {
	border-radius: 4px;
	border: 1px solid #dcdcdc;
	display: inline-block;
	float: left;
	text-align: center;
	padding: 5px;
	font-size: 9pt;
	margin-right: 5px;
	background: #fefefe;
}
article {
	background: #f4f4f4;
	padding: 5px;
	width: calc(100% - 300px);
	border-radius: 2px;
	border: 1px solid #dcdcdc;
}
#show-sidebar-link { display: none; }
@media all and (max-width: 640px) {
	main {
		width: calc(100% - 20px);
	}
	header {
		padding-left: 5px;
		padding-right: 5px;
		width: calc(100% - 10px);
	}
	#show-sidebar-link { display: block; margin-top: 5px; }
	aside { display: none; width: calc(100% - 25px); float: none; }
	article { width: calc(100% - 30px); }
}
`
func ExpandVars(s string, title string) string {
	s = strings.Replace(s, "$title", title, -1)
	s = strings.Replace(s, "$host", *hostname, -1)
	s = strings.Replace(s, "$coinhive", *coinhive, -1)
	return s
}
func show404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	io.WriteString(w, ExpandVars(HEADER_FMT, "404 Not Found"))
	io.WriteString(w, "<center><h2>404</h2><br/>We can't find what you're looking for.</center>")
	io.WriteString(w, ExpandVars(FOOTER_FMT, "404 Not Found"))
}
func show401(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(401)
	io.WriteString(w, ExpandVars(HEADER_FMT, "401 Unauthorized"))
	io.WriteString(w, "<center><h2>401</h2><br/>Please login first.</center>")
	io.WriteString(w, ExpandVars(FOOTER_FMT, "401 Unauthorized"))
}
func voteWidget(b *Backend, a *nntp.Article) string {
	id := a.MessageID()
	vt := strconvOr(a.Header.Get("Votes"), 1)
	return fmt.Sprintf(`<span class='votercount'><a href='/vote/%s/1'>&#9650;</a> %d <a href='/vote/%s/-1'>&#9660;</a></span>`, id, vt, id)
}
func strconvOr(a string, b int64) (c int64) {
	c, e := strconv.ParseInt(a, 0, 64)
	if e != nil {
		c = b
	}
	return
}
type ArticleTree struct {
	Depth int
	*nntp.Article
}
func collectArticleTree(b *Backend, g *nntp.Group, root string, from, to int64, depth int) (out []ArticleTree) {
	out = []ArticleTree{}
	arts, err := b.GetArticles(g, from, to)
	if err != nil {
		return
	}
	for _, v := range arts {
		a := v.Article
		if a.Header.Get("References") == root {
			out = append(out, ArticleTree{Article:a,Depth:depth})
			out = append(out, collectArticleTree(b, g, a.MessageID(), from, to, depth+1)...)
		}
	}
	return
}
func showLoginHeader(b *Backend) string {
	if b.username != "" {
		return fmt.Sprintf("<section class='loginheader'><strong>%s</strong></section>", html.EscapeString(b.username))
	}
	return fmt.Sprintf("<section class='loginheader'><a href='/login'>Login</a> or <a href='/signup'>Signup</a> to vote and post.</section>")
}
func genSidebar(b *Backend, g *nntp.Group, p *nntp.Article) (o string) {
	o  = "<aside>"
	gd := strings.Split(g.Description, "$!$!$")
	if p != nil {
	o += "Submission by " + html.EscapeString(p.Header.Get("From")) + "<br>"
	o += "<hr/>"
	}
	if b.username != "" {
	n := g.Name
	if g.Name == "recent" { n = "nntpit" }
	o += "<p><a href='/g/" + n + "/newpost' class='btn sidebtn primarybtn'>New Post</a></p>"
	o += "<p><a href='/addgroup' class='btn sidebtn primarybtn'>New Group</a></p>"
	}
	o += "<h2><a class='subtitle' href='/g/" + g.Name + "'>" + html.EscapeString(g.Name) + "</a></h2>"
	o += "<hr/>"
	o += md.RenderToString([]byte(gd[0]))
	if len(gd) == 1 {
		gd = []string{"",""}
	}
	ow := strings.Split(gd[1], " ")
	o += "<hr/>"
	o += "<span style='font-size: 9pt'><i>owned by " + ow[0] + "</i></span>"
	o += "<hr/>"
	o += "<b>Moderators:</b><br>"
	o += "<ul>"
	ismod := false
	for _, v := range ow {
	o += "<li>" + v + "</li>"
	if v == b.username + "@" + b.NodeName {
		ismod = true
	}
	}
	o += "</ul>"
	if ismod {
	o += "<p><a href='/g/" + g.Name + "/sidebar' class='btn sidebtn primarybtn'>Group Settings</a></p>"
	}
	o += "</aside>"
	return
}
func GetUserBackend(b *Backend, w http.ResponseWriter, r *http.Request, req bool) (*Backend, error) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		if req {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"Login to your account\"")
			return b, fmt.Errorf("Authentication required")
		}
		return b, nil
	}
	nb, err := b.Authenticate(user, pass)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Login to your account\"")
		return nil, err
	}
	return nb.(*Backend), nil
}
func SignupGetUserBackend(b *Backend, w http.ResponseWriter, r *http.Request, req bool) (*Backend, error) {
	user, pass, ok := r.BasicAuth()
	user = "signup:" + user
	if !ok {
		if req {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"Signup for an account\"")
			return b, fmt.Errorf("Authentication required")
		}
		return b, nil
	}
	nb, err := b.Authenticate(user, pass)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Signup for an account\"")
		return nil, err
	}
	return nb.(*Backend), nil
}
func StartHttp(b *Backend, listen string, admin string) {
	router := httprouter.New()
	router.GET("/style.css", func (w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(CSS))
	})
	router.GET("/login", func (w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ub, err := GetUserBackend(b, w, r, true)
		if err != nil {
			show401(w, r)
			return
		}
		w.Header().Set("Location", r.Header.Get("Referer"))
		w.WriteHeader(302)
		_ = ub
	})
	router.GET("/signup", func (w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if *coinhive != "" && r.FormValue("coinhive-captcha-token") == "" {
			io.WriteString(w, ExpandVars(HEADER_FMT, "Signup"))
			io.WriteString(w, `<div style='margin: 5px'><h2>Captcha</h2><p>The node owner has configured a Coinhive captcha. Please solve the captcha to sign up.</p>`)
			io.WriteString(w, `<form action="/signup" method="GET">` + ExpandVars(COINHIVE_CAPTCHA, "CoinhiveCaptchaModule") + `<p><input type=submit value="Continue"></p></form></div>`)
			io.WriteString(w, ExpandVars(FOOTER_FMT, "Signup"))
			return
		}
		ub, err := SignupGetUserBackend(b, w, r, true)
		if err != nil {
			show401(w, r)
			return
		}
		w.Header().Set("Location", r.Header.Get("Referer"))
		w.WriteHeader(302)
		_ = ub
	})
	newpostfn := func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		b, err := GetUserBackend(b, w, r, true)
		if err != nil {
			show401(w, r)
			return
		}
		grp := ps.ByName("group")
		grpo, err := b.GetGroup(grp)
		if err != nil {
			show404(w, r)
			return
		}
		errflag := " "
		if r.Method == "POST" {
			if r.FormValue("subject") == "" {
				errflag = "A title is required"
				goto out
			}
			if len(r.FormValue("message")) == 0 {
				errflag = "Text is required"
				goto out
			}
			pid := GUID() + "@" + b.NodeName
			hdr := textproto.MIMEHeader{}
			hdr.Set("Message-Id", pid)
			hdr.Set("Subject", r.FormValue("subject"))
			hdr.Set("Newsgroups", grpo.Name)
			art := &nntp.Article{
				Header: hdr,
				Body: strings.NewReader(r.FormValue("message")),
			}
			err := b.Post(art)
			if err != nil {
				errflag = err.Error()
				goto out
			}
			w.Header().Set("Location", "/g/" + grpo.Name + "/post/" + pid)
			w.WriteHeader(302)
			return
		}
		out:
		io.WriteString(w, ExpandVars(HEADER_FMT, "New Post"))
		io.WriteString(w, showLoginHeader(b))
		io.WriteString(w, genSidebar(b, grpo, nil))
		fmt.Fprintf(w, `<form action="/g/%s/newpost" method="POST" class='newpost'>
			<p><strong style='color:red'>%s</strong></p>
			<p><strong>Title:</strong><br>
			<input type='text' name='subject'><p>
			<p><strong>Text:</strong><br>
			<textarea style='width: 100%%; max-width: 500px; height: 300px;' name='message'></textarea></p>
			<p>Markdown (with images!) is supported here. Go ahead and try it.
				<details>
				<summary>Help?</summary>
				<code>Add a link: [title](http://url)<br>
				Show a header: ## myheader<br>
				Insert an image: ![description](http://image/link.jpg)</code>
				</details></p>
			<p><input class=btn type=submit value=Submit></p></form>`, grpo.Name, errflag)
		io.WriteString(w, ExpandVars(FOOTER_FMT, "New Post"))
	}
	router.POST("/g/:group/newpost", newpostfn)
	router.GET("/g/:group/newpost", newpostfn)
	router.POST("/g/:group/post/:id/reply", func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		b, err := GetUserBackend(b, w, r, true)
		if err != nil {
			show401(w, r)
			return
		}
		grp := ps.ByName("group")
		grpo, err := b.GetGroup(grp)
		if err != nil {
			show404(w, r)
			return
		}
		art := ps.ByName("id")
		arto, err := b.GetArticle(grpo, art)
		if err != nil {
			show404(w, r)
		}
		errflag := " "
		if r.Method == "POST" {
			if len(r.FormValue("message")) == 0 {
				errflag = "Text is required"
				goto out
			}
			pid := GUID() + "@" + b.NodeName
			hdr := textproto.MIMEHeader{}
			hdr.Set("Message-Id", pid)
			hdr.Set("References", art)
			hdr.Set("Subject", "Re: " + arto.Header.Get("Subject"))
			hdr.Set("Newsgroups", grpo.Name)
			art := &nntp.Article{
				Header: hdr,
				Body: strings.NewReader(r.FormValue("message")),
			}
			err := b.Post(art)
			if err != nil {
				errflag = err.Error()
				goto out
			}
			w.Header().Set("Location", r.Header.Get("Referer") + "#" + arto.MessageID())
			w.WriteHeader(302)
			return
		}
		out:
		io.WriteString(w, ExpandVars(HEADER_FMT, "New Post"))
		io.WriteString(w, showLoginHeader(b))
		io.WriteString(w, genSidebar(b, grpo, nil))
		fmt.Fprintf(w, `<p><strong style='color:red'>%s</strong></p>`, errflag)
		io.WriteString(w, ExpandVars(FOOTER_FMT, "New Post"))

	})
	router.GET("/vote/:id/:ct", func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		b, err := GetUserBackend(b, w, r, true)
		if err != nil {
			show401(w, r)
			return
		}
		art := ps.ByName("id")
		arto, err := b.GetArticle(nil, art)
		if err != nil {
			show404(w, r)
		}
		dir := ps.ByName("ct")
		dct := strconvOr(dir,0)
		errflag := " "
		if r.Method == "GET" {
			if b.UpdateVote(arto, dct) != nil {
				errflag = "Error"
				goto out
			}
			w.Header().Set("Location", r.Header.Get("Referer"))
			w.WriteHeader(302)
			return
		}
		out:
		io.WriteString(w, ExpandVars(HEADER_FMT, "Error"))
		io.WriteString(w, showLoginHeader(b))
		fmt.Fprintf(w, `<p><strong style='color:red'>%s</strong></p>`, errflag)
		io.WriteString(w, ExpandVars(FOOTER_FMT, "Error"))

	})
	router.GET("/g/:group/post/:post", func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		b, err := GetUserBackend(b, w, r, false)
		if err != nil {
			show401(w, r)
			return
		}
		grp := ps.ByName("group")
		grpo, err := b.GetGroup(grp)
		if err != nil {
			show404(w, r)
			return
		}
		pst := ps.ByName("post")
		psto, err := b.GetArticle(grpo, pst)
		if err != nil {
			show404(w, r)
			return
		}
		io.WriteString(w, ExpandVars(HEADER_FMT, psto.Header.Get("Subject")))
		io.WriteString(w, showLoginHeader(b))
		io.WriteString(w, genSidebar(b, grpo, psto))
		bod, _ := ioutil.ReadAll(psto.Body)
		io.WriteString(w, "<section style='margin-left: 15px'>"+voteWidget(b,psto))
		io.WriteString(w, "<strong style='font-size: 15pt;'>" + html.EscapeString(psto.Header.Get("Subject")) + "</strong><br>")
		ago, _ := timeago.TimeAgoFromNowWithTime(time.Unix(strconvOr(psto.Header.Get("Unix-Date"), 1), 0))
		by := html.EscapeString(psto.Header.Get("From"))
		fmt.Fprintf(w, `<span style='font-size: 9pt'>submitted %s by <strong>%s</strong></span><br>`, strings.ToLower(ago), by)
		io.WriteString(w, "<article>" + md.RenderToString(bod) + "</article></section>")
		fmt.Fprintf(w, `<form action='/g/%s/post/%s/reply' method='POST' style='padding: 10px;'>
				<textarea style='width: 100%%; min-width: 300px; max-width: 400px; height: 120px;' name='message'></textarea>
				<p><input class=btn type=submit value='Comment'></p>
				</form>`, grpo.Name, psto.MessageID())
		io.WriteString(w, "<section class='comments'>")
		gid := strconvOr(psto.Header.Get("Group-Index"), grpo.High-100)
		comments := collectArticleTree(b, grpo, psto.MessageID(), gid-100, gid+100, 0)
		for _, v := range comments {
			cbod, _ := ioutil.ReadAll(v.Article.Body)
			ago, _ := timeago.TimeAgoFromNowWithTime(time.Unix(strconvOr(v.Article.Header.Get("Unix-Date"), 1), 0))
			fmt.Fprintf(w, `<section class='comment' style='padding-left: %dpx' id='%s'>
						<span style='font-size: 9pt'><strong>%s</strong> %s</span>
						<p>%s</p>
						<details class='reply_form'>
							<summary style='font-size: 9pt'>reply</summary>
							<form action='/g/%s/post/%s/reply' method='POST'>
								<textarea style='width: 100%%; min-width: 300px; max-width: 400px; height: 110px; padding: 0;' name='message'></textarea>
								<br/>
								<p style=''><input class=btn type=submit value='Reply'></p>
							</form>
						</details>
					</section>`, (v.Depth + 1) * 15, v.Article.MessageID(), html.EscapeString(v.Article.Header.Get("From")), strings.ToLower(ago), md.RenderToString(cbod),
							grpo.Name, v.Article.MessageID())
		}
		if b.username == "" {
			io.WriteString(w, "<style>.reply_form{display:none;}</style>")
		}
		io.WriteString(w, "</section>")
		io.WriteString(w, ExpandVars(FOOTER_FMT, psto.Header.Get("Subject")))
	})
	groupsidebar := func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		b, err := GetUserBackend(b, w, r, true)
		if err != nil {
			show401(w, r)
			return
		}
		grp := ps.ByName("group")
		grpo, err := b.GetGroup(grp)
		if err != nil {
			show404(w, r)
			return
		}
		errflag := ""
		if r.Method == "POST" {
			if len(r.FormValue("mods")) < 10 {
				errflag = "You can't remove all moderators!"
				goto skip
			}
			hdr := textproto.MIMEHeader{}
			hdr.Set("Newsgroups", grp)
			hdr.Set("From", b.username + "@" + b.NodeName)
			hdr.Set("Message-Id", "sidebar@" + grp)
			mods := strings.Fields(r.FormValue("mods"))
			if len(mods) > 1 {
				mods = mods[1:]
			} else {
				mods = []string{}
			}
			hdr.Set("Moderators", strings.Join(mods, " "))
			bod := strings.NewReader(r.FormValue("sidebar"))
			art := &nntp.Article{Body:bod,Header:hdr}
			err := b.Post(art)
			if err != nil {
				errflag = "You aren't allowed to do that!"
			}
			grpo, err = b.GetGroup(grpo.Name)
		}
		skip:
		io.WriteString(w, ExpandVars(HEADER_FMT, "Sidebar"))
		io.WriteString(w, showLoginHeader(b))
		io.WriteString(w, genSidebar(b, grpo, nil))
		io.WriteString(w, "<section style='margin: 15px;'><p><strong>" + errflag + "</strong></p>")
		io.WriteString(w, "<form method='POST' action='/g/" + grp + "/sidebar'>")
		io.WriteString(w, "<p><strong>Sidebar Text</strong><br/>")
		io.WriteString(w, "<textarea style='width: 400px; height: 300px;' name='sidebar'>")
		gd := strings.Split(grpo.Description, "$!$!$")
		io.WriteString(w, html.EscapeString(gd[0]))
		io.WriteString(w, "</textarea></p>")
		io.WriteString(w, "<p><strong>Moderator List</strong><br/>")
		io.WriteString(w, "Add or remove user addresses (username@node) to change the list. Don't remove yourself!<br/>")
		io.WriteString(w, "<textarea style='width: 400px; height: 300px;' name='mods'>")
		io.WriteString(w, html.EscapeString(strings.Replace(gd[1], " ", "\n", -1)))
		io.WriteString(w, "</textarea>")
		io.WriteString(w, "</p><p><input type=submit value='Save Settings'>")
		io.WriteString(w, "</p>")
		io.WriteString(w, "</form>")
		io.WriteString(w, "</section>")
		io.WriteString(w, ExpandVars(FOOTER_FMT, "Sidebar"))
	}
	router.GET("/g/:group/sidebar", groupsidebar)
	router.POST("/g/:group/sidebar", groupsidebar)
	addgroup := func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		b, err := GetUserBackend(b, w, r, true)
		if err != nil {
			show401(w, r)
			return
		}
		errflag := ""
		if r.Method == "POST" {
			grp := r.FormValue("group")
			grpo, err := b.GetGroup(grp)
			errflag := ""
			if err != nil {
				errflag = "Group exists"
			}
			if grp == "" {
				errflag = "No group specified"
			}
			b.AddGroup(grp)
			hdr := textproto.MIMEHeader{}
			hdr.Set("Newsgroups", grp)
			hdr.Set("From", b.username + "@" + b.NodeName)
			hdr.Set("Message-Id", "sidebar@" + grp)
			mods := strings.Fields(r.FormValue("mods"))
			if len(mods) > 1 {
				mods = mods[1:]
			} else {
				mods = []string{}
			}
			hdr.Set("Moderators", "")
			bod := strings.NewReader(r.FormValue("sidebar"))
			art := &nntp.Article{Body:bod,Header:hdr}
			err = b.Post(art)
			if err != nil {
				errflag = "You aren't allowed to do that!"
			}
			if errflag == "" {
				w.Header().Set("Location", "/g/" + grpo.Name)
				w.WriteHeader(302)
				return
			}

		}
		io.WriteString(w, ExpandVars(HEADER_FMT, "Add Group"))
		io.WriteString(w, showLoginHeader(b))
		io.WriteString(w, `<section style='margin: 15px'><strong>` + errflag + `</strong><form action='/addgroup' method='POST'><p>
				<strong>Name of new group:</strong> <input type=text name='group' size=30></p>
				<p><input type=submit value='Add My Group'></p></form></section>`)
		io.WriteString(w, ExpandVars(FOOTER_FMT, "Add Group"))

	}
	router.GET("/addgroup", addgroup)
	router.POST("/addgroup", addgroup)
	router.GET("/g/:group", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		b, err := GetUserBackend(b, w, r, false)
		if err != nil {
			show401(w, r)
			return
		}
		sortmode := r.FormValue("sort")
		if sortmode == "" { sortmode = "hot" }
		grp := ps.ByName("group")
		var from, to int64
		off := strconvOr(r.FormValue("start"), 0)
		grpo, err := b.GetGroup(grp)
		if err != nil {
			show404(w, r)
			return
		}
		if grp == "recent" {
			from = -1 + -off
			to = 50
		} else {
			from = grpo.High - 50 - off
			if from < 0 {
				from = 0
			}
			to = grpo.High - off
		}
		io.WriteString(w, ExpandVars(HEADER_FMT, grp))
		io.WriteString(w, showLoginHeader(b))
		io.WriteString(w, genSidebar(b, grpo, nil))
		io.WriteString(w, "<ul>")
		grpo.Description = "!UNIQUE"
		total := 0
		tart := []nntpserver.NumberedArticle{}
		retry:
		art, err := b.GetArticles(grpo, from, to)
		if err != nil {
			io.WriteString(w, "<b>Something went wrong!</b><br/><br/>")
			log.Println("HTTP", err)
		}
		for _, v := range art {
			if total > 10 { break }
			tart = append(tart, v)
			total++
		}
		if total < 10 && grpo.High > 10 && from != 0 {
			from -= 50
			if from < 0 {
				from = 0
				to -= from
			}
			to -= 50
			goto retry
		}
		if sortmode == "hot" {
			sort.Slice(tart, func (i, j int) bool {
				a := strconvOr(tart[i].Article.Header.Get("Votes"), 1)
				b := strconvOr(tart[j].Article.Header.Get("Votes"), 1)
				return a > b
			})
		}
		for _, v := range tart {
			a := v.Article
			tago, _ := timeago.TimeAgoFromNowWithTime(time.Unix(strconvOr(a.Header.Get("Unix-Date"), 1), 0))
			fmt.Fprintf(w, `
				<li class='linkitem'>
					%s
					<a class='linktitle' href='/g/%s/post/%s'><strong style='font-size: 13pt;'>%s</strong></a><br/>
					<span style='font-size: 9pt'>submitted %s by %s to <a href='/g/%s'>%s</a><br/>
					<a href='/g/%s/post/%s?comments=1'>comments</a> - share</span><p></p>
				</li>`, voteWidget(b,a), a.Header.Get("Newsgroups"), a.MessageID(), 
					html.EscapeString(a.Header.Get("Subject")), strings.ToLower(tago), html.EscapeString(a.Header.Get("From")), 
					a.Header.Get("Newsgroups"), a.Header.Get("Newsgroups"), a.Header.Get("Newsgroups"), a.MessageID())
		}
		io.WriteString(w, "</ul>")
		if from > 0 {
			fmt.Fprintf(w, "&nbsp;<a href='?start=%d' class='btn'>Back</a>", from - 10)
		}
		if (from + int64(total)) < grpo.High {
			fmt.Fprintf(w, "&nbsp;<a href='?start=%d' class='btn'>Next</a>", from + 10)
		}
		io.WriteString(w, "<style>.only-group { display: inline; }</style>")
		io.WriteString(w, ExpandVars(FOOTER_FMT, grp))
	})
	router.GET("/about", func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		io.WriteString(w, ExpandVars(HEADER_FMT, "About"))
		io.WriteString(w, ExpandVars(ABOUT_TXT, "About"))
		io.WriteString(w, ExpandVars(FOOTER_FMT, "About"))
	})
	router.GET("/", func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Location", "/g/recent")
		w.WriteHeader(302)
		w.Write([]byte("<a href='/g/recent'>Follow</a>"))
	})
	http.ListenAndServe(listen, router)
}
