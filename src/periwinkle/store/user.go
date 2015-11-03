// Copyright 2015 Davis Webb
// Copyright 2015 Guntas Grewal
// Copyright 2015 Luke Shumaker

package store

import (
	"encoding/json"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
	he "httpentity"
	"httpentity/util" // heutil
	"io"
	"jsonpatch"
	"periwinkle/util" // putil
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

func (o User) dbSchema(db *gorm.DB) error {
	return db.CreateTable(&o).Error
}

type UserAddress struct {
	Id      int64  `json:"-"`
	UserId  string `json:"-"`
	Medium  string `json:"medium"`
	Address string `json:"address"`
}

func (o UserAddress) dbSchema(db *gorm.DB) error {
	return db.CreateTable(&o).
		AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").
		AddForeignKey("medium", "media(id)", "RESTRICT", "RESTRICT").
		AddUniqueIndex("uniqueness_idx", "medium", "address").
		Error
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

func NewUser(db *gorm.DB, name string, password string, email string) User {
	if name == "" {
		panic("name can't be empty")
	}
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
	return o
}

func (o *User) Save(db *gorm.DB) {
	if err := db.Save(o).Error; err != nil {
		panic(err)
	}
}

func (o *User) Subentity(name string, req he.Request) he.Entity {
	return nil
}

func (o *User) patchPassword(patch *jsonpatch.Patch) putil.HTTPError {
	// this is in the running for the grossest code I've ever
	// written, but I think it's the best way to do it --lukeshu
	type patchop struct {
		Op    string
		Path  string
		Value string
	}
	str, err := json.Marshal(patch)
	if err != nil {
		panic(err)
	}
	var ops []patchop
	err = json.Unmarshal(str, &ops)
	if err != nil {
		return nil
	}
	out_ops := make([]patchop, 0, len(ops))
	checkedpass := false
	for _, op := range ops {
		if op.Path == "/password" {
			switch op.Op {
			case "test":
				if !o.CheckPassword(op.Value) {
					return putil.HTTPErrorf(409, "old password didn't match")
				}
				checkedpass = true
			case "replace":
				if !checkedpass {
					return putil.HTTPErrorf(409, "you must submit and old password (using 'test') before setting a new one")
				}
				o.SetPassword(op.Value)
			default:
				return putil.HTTPErrorf(415, "you may only 'set' or 'replace' the password")
			}
		} else {
			out_ops = append(out_ops, op)
		}
	}
	str, err = json.Marshal(out_ops)
	if err != nil {
		panic(err)
	}
	var out jsonpatch.JSONPatch
	err = json.Unmarshal(str, &out)
	if err != nil {
		panic(out)
	}
	*patch = out
	return nil
}

func (o *User) Methods() map[string]func(he.Request) he.Response {
	return map[string]func(he.Request) he.Response{
		"GET": func(req he.Request) he.Response {
			return he.StatusOK(o)
		},
		"PUT": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)
			sess := req.Things["session"].(*Session)
			if sess.UserId != o.Id {
				return he.StatusForbidden(heutil.NetString("Unauthorized user"))
			}
			var new_user User
			err := safeDecodeJSON(req.Entity, &new_user)
			if err != nil {
				return err.Response()
			}
			if o.Id != new_user.Id {
				return he.StatusConflict(heutil.NetString("Cannot change user id"))
			}
			*o = new_user
			o.Save(db)
			return he.StatusOK(o)
		},
		"PATCH": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)
			sess := req.Things["session"].(*Session)
			if sess.UserId != o.Id {
				return he.StatusForbidden(heutil.NetString("Unauthorized user"))
			}
			patch, ok := req.Entity.(jsonpatch.Patch)
			if !ok {
				return putil.HTTPErrorf(415, "PATCH request must have a patch media type").Response()
			}
			httperr := o.patchPassword(&patch)
			if httperr != nil {
				return httperr.Response()
			}
			var new_user User
			err := patch.Apply(o, &new_user)
			if err != nil {
				return putil.HTTPErrorf(409, "%v", err).Response()
			}
			if o.Id != new_user.Id {
				return he.StatusConflict(heutil.NetString("Cannot change user id"))
			}
			*o = new_user
			o.Save(db)
			return he.StatusOK(o)
		},
		"DELETE": func(req he.Request) he.Response {
                        db := req.Things["db"].(*gorm.DB)
			db.Delete(o)
			return he.StatusGone(heutil.NetString("User has been deleted"))
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
			type postfmt struct {
				Username             string `json:"username"`
				Email                string `json:"email"`
				Password             string `json:"password"`
				PasswordVerification string `json:"password_verification,omitempty"`
			}
			var entity postfmt
			httperr := safeDecodeJSON(req.Entity, &entity)
			if httperr != nil {
				return httperr.Response()
			}

			if entity.Username == "" || entity.Email == "" || entity.Password == "" {
				return he.StatusUnsupportedMediaType(heutil.NetString("username, email, and password can't be emtpy"))
			}

			if entity.PasswordVerification != "" {
				if entity.Password != entity.PasswordVerification {
					// Passwords don't match
					return he.StatusConflict(heutil.NetString("password and password_verification don't match"))
				}
			}

			entity.Username = strings.ToLower(entity.Username)

			user := NewUser(db, entity.Username, entity.Password, entity.Email)
			req.Things["user"] = user
			return he.StatusCreated(r, user.Id, req)
		},
	}
	return r
}

func (d t_dirUsers) Methods() map[string]func(he.Request) he.Response {
	return d.methods
}

func (d t_dirUsers) Subentity(name string, req he.Request) he.Entity {
	name = strings.ToLower(name)
	sess := req.Things["session"].(*Session)
	if sess == nil && req.Method == "POST" {
		user, ok := req.Things["user"].(User)
		if !ok {
			return nil
		}
		if user.Id == name {
			return &user
		}
		return nil
	} else if sess.UserId != name {
		return nil
	}
	db := req.Things["db"].(*gorm.DB)
	return GetUserById(db, name)
}
