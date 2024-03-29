package main

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func handleNewPostView(g *gin.Context) {
	var pl = struct {
		UUID      string
		Reply     string
		Abstract  string
		Challenge string
		Tags      []string

		RTitle, RAuthor, RContent, RTags, EError string
	}{
		UUID:      makeCSRFToken(g),
		Challenge: makeChallengeToken(),
		RTitle:    g.Query("title"),
		RContent:  g.Query("content"),
		RTags:     g.Query("tags"),
		RAuthor:   g.Query("author"),
		EError:    g.Query("error"),
		Tags:      config.Tags,
	}

	if id := g.Param("id"); id != "0" {
		pl.Reply = id
		a, err := m.GetArticle(displayIDToObejctID(id))
		if err != nil {
			log.Println(err)
			g.AbortWithStatus(400)
			return
		}
		if a.Title != "" {
			pl.Abstract = softTrunc(a.TitleString(), 50)
		} else {
			pl.Abstract = softTrunc(a.Content, 50)
		}
	}

	if id, _ := g.Cookie("id"); id != "" && pl.RAuthor == "" {
		pl.RAuthor = id
	}

	g.HTML(200, "newpost.html", pl)
}

func hashIP(g *gin.Context) uint64 {
	buf := make([]byte, 8)
	ip := g.ClientIP()
	ip2 := net.ParseIP(ip)
	if len(ip2) == net.IPv4len {
		copy(buf, ip2[:3]) // first 3 bytes
	} else if len(ip2) == net.IPv6len {
		if len(ip2.To4()) == net.IPv4len {
			copy(buf, ip2.To4()[:3]) // first 3 bytes
		} else {
			copy(buf, ip2) // first 8 byte
		}
	} else {
		copy(buf, ip)
	}
	return binary.LittleEndian.Uint64(buf)
}

func handleNewPostAction(g *gin.Context) {
	var (
		reply     = displayIDToObejctID(g.PostForm("reply"))
		answer    = g.PostForm("answer")
		challenge = g.PostForm("challenge")
		uuid      = g.PostForm("uuid")
		ip        = hashIP(g)
		content   = softTrunc(g.PostForm("content"), int(config.MaxContent))
		title     = softTrunc(g.PostForm("title"), 100)
		author    = softTrunc(g.PostForm("author"), 32)
		tags      = splitTags(softTrunc(g.PostForm("tags"), 128))
		image, _  = g.FormFile("image")
		redir     = func(a, b string) {
			q := encodeQuery(a, b, "author", author, "content", content, "title", title, "tags", strings.Join(tags, " "))
			if reply == "" {
				g.Redirect(302, "/new/0?"+q)
			} else {
				g.Redirect(302, "/new/"+objectIDToDisplayID(reply)+"?"+q)
			}
		}
	)

	if !g.GetBool("ip-ok") {
		redir("error", "guard/cooling-down")
		return
	}

	if g.PostForm("refresh") != "" {
		redir("", "")
		return
	}

	if !isCSRFTokenValid(g, uuid) {
		redir("", "")
		return
	}

	if !isChallengeTokenValid(challenge, answer) && !isAdmin(author) {
		log.Println(g.ClientIP(), "challenge failed")
		redir("error", "guard/failed-captcha")
		return
	}

	if author == "" {
		author = g.ClientIP()
	}

	if image == nil && len(content) < int(config.MinContent) {
		redir("error", "content/too-short")
		return
	}

	var imagek []string
	if image != nil {
		if config.ImageDisabled && !isAdmin(author) {
			redir("error", "image/disabled")
			return
		}

		f, err := image.Open()
		if err != nil {
			redir("error", "image/open-error: "+err.Error())
			return
		}
		buf, _ := ioutil.ReadAll(io.LimitReader(f, 1024*1024))
		f.Close()

		if !isValidImage(buf) {
			redir("error", "image/invalid-format")
			return
		}

		localpath, displaypath := getImageLocalTmpPath(image.Filename, buf)
		if err := ioutil.WriteFile(localpath, buf, 0700); err != nil {
			redir("error", "image/write-error: "+err.Error())
			return
		}
		imagek = []string{displaypath}
	}

	var err error
	if reply != "" {
		err = m.PostReply(reply, NewArticle("", content, authorNameToHash(author), imagek, nil, ip))
	} else {
		if len(title) < int(config.MinContent) {
			redir("error", "title/too-short")
			return
		}
		a := NewArticle(title, content, authorNameToHash(author), imagek, tags, ip)
		err = m.PostArticle(a)
		reply = a.ID
	}
	if err != nil {
		redir("error", "internal/error: "+err.Error())
		return
	}

	g.Redirect(302, "/p/"+objectIDToDisplayID(reply)+"?p=-1")
}
