// Copyright 2015 Davis Webb
// Copyright 2015 Guntas Grewal
// Copyright 2015 Luke Shumaker

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
	he "httpentity"
	"httpentity/util"
	"io"
	"jsonpatch"
	"strings"
)

var _ he.Entity = &User{}
var _ he.NetEntity = &User{}
var dirUsers he.Entity = newDirUsers()

// Model /////////////////////////////////////////////////////////////

type User struct {
	Id        string        `json:"user_id"`
	FullName  string        `json:"fullname"`
	PwHash    []byte        `json:"-"`
	Addresses []UserAddress `json:"addresses"`
}

func (o User) schema(db *gorm.DB) {
	db.CreateTable(&o)
}

type UserAddress struct {
	Id      int64  `json:"-"`
	UserId  string `json:"-"`
	Medium  string `json:"medium"`
	Address string `json:"address"`
}

func (o UserAddress) schema(db *gorm.DB) {
	table := db.CreateTable(&o)
	table.AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT")
	table.AddForeignKey("medium", "media(id)", "RESTRICT", "RESTRICT")
	table.AddUniqueIndex("uniqueness_idx", "medium", "address")
}

func GetUserById(db *gorm.DB, id string) *User {
	id = strings.ToLower(id)
	var o User
	if result := db.First(&o, "id = ?", id); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	db.Model(&o).Related(&o.Addresses)
	return &o
}

func GetUserByAddress(db *gorm.DB, medium string, address string) *User {
	var o User
	result := db.Joins("inner join user_addresses on user_addresses.user_id=users.id").Where("user_addresses.medium=? and user_addresses.address=?", medium, address).Find(&o)
	if result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	db.Model(&o).Related(&o.Addresses)
	return &o
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), -1)
	u.PwHash = hash
	return err
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword(u.PwHash, []byte(password))
	return err == nil
}

var BadPasswordErr = errors.New("Password was incorrect")

func (u *User) UpdatePassword(db *gorm.DB, newPass string, oldPass string) error {
	if !u.CheckPassword(oldPass) {
		return BadPasswordErr
	}
	if err := u.SetPassword(newPass); err != nil {
		return err
	}
	u.Save(db)
	return nil
}

func (u *User) UpdateEmail(db *gorm.DB, newEmail string, pw string) {
	if !u.CheckPassword(pw) {
		panic("Password was incorrect")
	}
	for _, addr := range u.Addresses {
		if addr.Medium == "email" {
			addr.Address = newEmail
			if err := db.Save(&addr).Error; err != nil {
				panic(err)
			}
		}
	}
}

func NewUser(db *gorm.DB, name string, password string, email string) *User {
	o := User{
		Id:        name,
		FullName:  "",
		Addresses: []UserAddress{{Medium: "email", Address: email}},
	}
	if err := o.SetPassword(password); err != nil {
		panic(err)
	}
	if err := db.Create(&o).Error; err != nil {
		panic(err)
	}
	return &o
}

func (o *User) Save(db *gorm.DB) {
	if err := db.Save(o).Error; err != nil {
		panic(err)
	}
}

func (o *User) Subentity(name string, req he.Request) he.Entity {
	return nil
}

func (o *User) Methods() map[string]func(he.Request) he.Response {
	return map[string]func(he.Request) he.Response{
		"GET": func(req he.Request) he.Response {
			return he.StatusOK(o)
		},
		"PUT": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)
			// TODO: permissions
			entity, ok := req.Entity.(json.Decoder)
			if !ok {
				return he.StatusUnsupportedMediaType(heutil.NetString("415: PUT request must have a document media type"))
			}
			var new_user User
			err := entity.Decode(&new_user)
			if err != nil {
				return he.StatusConflict(heutil.NetString(fmt.Sprintf("409 Conflict: %v", err)))
			}
			// TODO: check that .Id didn't change.
			*o = new_user
			o.Save(db)
			return he.StatusOK(o)
		},
		"PATCH": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)
			// TODO: permissions
			patch, ok := req.Entity.(jsonpatch.Patch)
			if !ok {
				return he.StatusUnsupportedMediaType(heutil.NetString("415: PATCH request must have a patch media type"))
			}
			var new_user User
			err := patch.Apply(o, &new_user)
			if err != nil {
				return he.StatusConflict(heutil.NetString(fmt.Sprintf("409 Conflict: %v", err)))
			}
			// TODO: check that .Id didn't change.
			*o = new_user
			o.Save(db)
			return he.StatusOK(o)
		},
		"DELETE": func(req he.Request) he.Response {
			panic("TODO: API: (*User).Methods()[\"DELETE\"]")
		},
	}
}

// View //////////////////////////////////////////////////////////////

func (o *User) Encoders() map[string]func(io.Writer) error {
	return defaultEncoders(o)
}

// Directory ("Controller") //////////////////////////////////////////

type t_dirUsers struct {
	methods map[string]func(he.Request) he.Response
}

func newDirUsers() t_dirUsers {
	r := t_dirUsers{}
	r.methods = map[string]func(he.Request) he.Response{
		"POST": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)
			badbody := he.StatusUnsupportedMediaType(heutil.NetString("submitted body not what expected"))
			hash, ok := req.Entity.(map[string]interface{}); if !ok { return badbody }
			username, ok := hash["username"].(string)      ; if !ok { return badbody }
			email   , ok := hash["email"].(string)         ; if !ok { return badbody }
			password, ok := hash["password"].(string)      ; if !ok { return badbody }

			if password2, ok := hash["password_verification"].(string); ok {
				if password != password2 {
					// Passwords don't match
					return he.StatusConflict(heutil.NetString("password and password_verification don't match"))
				}
			}

			username = strings.ToLower(username)

			user := NewUser(db, username, password, email)
			if user == nil {
				return he.StatusConflict(heutil.NetString("either that username or password is already taken"))
			} else {
				return he.StatusCreated(r, username, req)
			}
		},
	}
	return r
}

func (d t_dirUsers) Methods() map[string]func(he.Request) he.Response {
	return d.methods
}

func (d t_dirUsers) Subentity(name string, req he.Request) he.Entity {
	sess := req.Things["session"].(*Session)
	if sess == nil {
		db := req.Things["db"].(*gorm.DB)
		hash, ok := req.Entity.(map[string]interface{}); if !ok { return nil }
		username, ok := hash["username"].(string)      ; if !ok { return nil }
		password, ok := hash["password"].(string)      ; if !ok { return nil }
		var user *User
		user = GetUserById(db, username)
		if !user.CheckPassword(password) {
			return nil
		} else {
			return user
		}
	} else if sess.UserId != name {
		return nil
	}
	db := req.Things["db"].(*gorm.DB)
	return GetUserById(db, name)
}
