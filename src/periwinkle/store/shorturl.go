// Copyright 2015 Luke Shumaker
// Copyright 2015 Davis Webb

package store

import (
	"database/sql"
	he "httpentity"
	//"math/rand"
	"net/url"
)

var _ he.Entity = &ShortUrl{}
var dirShortUrls he.Entity = newDirShortUrls()

// Model /////////////////////////////////////////////////////////////

type ShortUrl struct {
	Id   string
	Dest *url.URL
}
/*
func randomByte(size int) []byte {
	byteSize := size
	var table = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	var gen []byte
	for i := 0 ; i < byteSize; i++{
		gen[i] = table[rand.Intn(61)]
	}
	return gen
}
*/
func newShortURL(u *url.URL) *ShortUrl {
	s := &ShortUrl{
		Id:   string(randomByte(5)),
		Dest: u,
	}
	err := s.Save()
	if err != nil {
		return nil
	}
	return s
}

func (s *ShortUrl) Save() error {
	// TODO
	panic("not implemented")
}

func GetShortUrlById(con DB, id string) *ShortUrl {
	var s ShortUrl
	err := con.QueryRow("SELECT shortURL FROM shortURL WHERE id=?", id).Scan(&s)
	switch {
	case err == sql.ErrNoRows:
		return nil
	case err != nil:
		panic(err)
	default:
		return &s
	}
}

func (o *ShortUrl) Subentity(name string, req he.Request) he.Entity {
	return nil
}

func (o *ShortUrl) Methods() map[string]he.Handler {
	panic("not implemented")
}

// Directory ("Controller") //////////////////////////////////////////

type t_dirShortUrls struct {
	methods map[string]he.Handler
}

func newDirShortUrls() t_dirShortUrls {
	r := t_dirShortUrls{}
	r.methods = map[string]he.Handler{}
	return r
}

func (d t_dirShortUrls) Methods() map[string]he.Handler {
	return d.methods
}

func (d t_dirShortUrls) Subentity(name string, request he.Request) he.Entity {
	return GetShortUrlById(nil /*TODO*/, name)
}
