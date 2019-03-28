package main

import mnntp "github.com/dustin/go-nntp"
import "github.com/chrisfarms/nntp"
import "log"
import "strings"
import "time"
import "net/textproto"
func StartSync(b *Backend) {
	uplinks := strings.Split(*uplinks + *defaultuplinks, ",")
	for {
		for _, v := range uplinks {
			if v == "" { continue }
			t := b.GetSyncTime(v)
			srv, err := nntp.Dial("tcp", v)
			if err != nil {
				continue
			}
			myart, _ := b.GetArticlesSince(t)
			gadded := map[string]bool{}
			for _, msg := range myart {
				if gadded[msg.Header.Get("Newsgroups")] { continue }
				addgrp := textproto.MIMEHeader{}
				addgrp.Set("Newsgroups", msg.Header.Get("Newsgroups"))
				gadded[addgrp.Get("Newsgroups")] = true
				addgrp.Set("Do", "addgroup")
				srv.Post(&nntp.Article{
					Header: addgrp, Body: strings.NewReader("") })
			}
			for _, msg := range myart {
				addgrp := textproto.MIMEHeader{}
				addgrp.Set("Newsgroups", msg.Header.Get("Newsgroups"))
				addgrp.Set("Do", "addgroup")
				srv.Post(&nntp.Article{
					Header: addgrp, Body: strings.NewReader("") })
				srv.Post(&nntp.Article{msg.Header,msg.Body})
			}

			_, _, _, err = srv.Group("recent")
			if err != nil {
				srv.Quit()
				continue
			}
			ov, err := srv.Overview(int(t), 1000000000)
			if err != nil {
				srv.Quit()
				continue
			}
			for _, msg := range ov {
				if b.IsBanned(msg.From + ":*") { continue }
				if _, err2 := b.GetArticle(nil, msg.MessageId); err2 == nil { continue }
				art, err := srv.Article(msg.MessageId)
				if err != nil { log.Println("Warning: Sync: Article:", err); continue }
				if !gadded[art.Header["Newsgroups"][0]] {
					gadded[art.Header["Newsgroups"][0]] = true
					b.AddGroup(art.Header["Newsgroups"][0])
				}
				err = b.Post(&mnntp.Article{Header:textproto.MIMEHeader(art.Header),Body:art.Body})
				if err != nil {
					log.Println("Warning: Sync: Post:", err)
				}
			}
			b.SetSyncTime(v, time.Now().Unix())
			srv.Quit()
		}
		time.Sleep(60*time.Second)
	}
}
