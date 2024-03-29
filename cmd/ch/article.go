package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"html/template"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/globalsign/mgo/bson"
)

type ArticlesView struct {
	Articles       []*Article
	ParentArticle  *Article
	Next, Prev     int64
	NoNext, NoPrev bool
	ShowIP         bool
	SearchTerm     string
	Title          string
}

type Article struct {
	ID           bson.ObjectId `bson:"_id"`
	Parent       bson.ObjectId `bson:"parent,omitempty"`
	Replies      int64         `bson:"replies_count"`
	Announcement bool          `bson:"announcement,omitempty"`
	Locked       bool          `bson:"locked,omitempty"`
	Title        string        `bson:"title"`
	Content      string        `bson:"content"`
	Author       uint64        `bson:"author"`
	IP           uint64        `bson:"ip"`
	Images       []string      `bson:"images"`
	Tags         []string      `bson:"tags"`
	CreateTime   int64         `bson:"create_time"`
	ReplyTime    int64         `bson:"reply_time"`
}

func NewArticle(title, content string, author uint64, images []string, tags []string, ip uint64) *Article {
	return &Article{
		ID:         bson.NewObjectId(),
		Title:      expandText(title),
		Content:    content,
		Author:     author,
		Images:     images,
		Tags:       tags,
		IP:         ip,
		CreateTime: time.Now().UnixNano() / 1e3,
		ReplyTime:  time.Now().UnixNano() / 1e3,
	}
}

func (a *Article) DisplayID() string {
	return objectIDToDisplayID(a.ID)
}

func (a *Article) DisplayParentID() string {
	return objectIDToDisplayID(a.Parent)
}

func (a *Article) CreateTimeString() string {
	return formatTime(a.CreateTime)
}

func (a *Article) ReplyTimeString() string {
	return formatTime(a.ReplyTime)
}

func (a *Article) AuthorString() string {
	n := strconv.FormatUint(a.Author, 36)
	if isAdmin(a.Author) {
		n += "*"
	}
	return n
}

func (a *Article) IPString() string {
	x := *(*[8]byte)(unsafe.Pointer(&a.IP))
	p := make([]byte, 0, 32)
	s := 0
	for i, x := range x {
		if (x >= '0' && x <= '9') || x == '.' || x == 0 {
			s++
		}
		p = strconv.AppendInt(p, int64(x), 10)
		if i < 7 {
			p = append(p, '.')
		}
	}
	if s == 8 {
		return string(bytes.TrimRight(x[:], "\x00"))
	}
	return string(p)
}

func (a *Article) ContentHTML() template.HTML {
	return template.HTML(sanText(a.Content))
}

func (a *Article) TitleString() string {
	return collapseText(a.Title)
}

func (a *Article) FirstImage() string {
	if len(a.Images) > 0 {
		return config.ImageDomain + "/i/" + a.Images[0]
	}
	return ""
}

func (a *Article) String() string {
	return a.DisplayID() + "-" + a.Title + "-" + strconv.FormatInt(a.ReplyTime, 10) + "-" + a.DisplayParentID()
}

func authorNameToHash(n string) uint64 {
	x := hmac.New(sha1.New, []byte(config.Key)).Sum([]byte(n + config.Key))
	return binary.BigEndian.Uint64(x)
}

func objectIDToDisplayID(id bson.ObjectId) string {
	if id == "" {
		return ""
	}
	x := []byte(id)
	for i := 0; i < 11; i++ {
		x[i] += x[11] * x[11]
	}
	e3 := func(b []byte) string {
		return strconv.FormatUint(uint64(b[0])<<40+uint64(b[1])<<32+
			uint64(b[2])<<24+uint64(b[3])<<16+
			uint64(b[4])<<8+uint64(b[5])<<0, 8)
	}
	return e3(x) + "." + e3(x[6:])
}

func displayIDToObejctID(id string) bson.ObjectId {
	if id == "" {
		return ""
	}
	idx := strings.Index(id, ".")
	if idx == -1 {
		return ""
	}
	x := make([]byte, 16)
	v1, _ := strconv.ParseUint(id[:idx], 8, 64)
	v2, _ := strconv.ParseUint(id[idx+1:], 8, 64)
	binary.BigEndian.PutUint64(x[8:], v2)
	binary.BigEndian.PutUint64(x[2:], v1)
	x = x[4:]
	for i := 0; i < 11; i++ {
		x[i] -= x[11] * x[11]
	}
	return bson.ObjectId(x)
}

func formatTime(t int64) string {
	x, now := time.Unix(0, t*1000), time.Now()
	if now.YearDay() == x.YearDay() && now.Year() == x.Year() {
		return x.Format("15:04:05")
	}
	if now.Year() == x.Year() {
		return x.Format("01-02 15:04:05")
	}
	return x.Format("2006-01-02 15:04:05")
}
