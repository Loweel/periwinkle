// Copyright 2015 Davis Webb
// Copyright 2015 Luke Shumaker
// Copyright 2015 Guntas Grewal

package store

import (
	"github.com/jinzhu/gorm"
	he "httpentity"
	"httpentity/util" // heutil
	"io"
	"jsonpatch"
	"strings"
)

var _ he.Entity = &Group{}
var _ he.NetEntity = &Group{}
var dirGroups he.Entity = newDirGroups()

// Model /////////////////////////////////////////////////////////////

type Group struct {
	Id            string         `json:"group_id"`
	Existence     int            `json:"existence"` // 1 -> public, 2 -> confirmed, 3 -> member	
	Read          int            `json:"read"`      // 1 -> public, 2 -> confirmed, 3 -> member	
	Post          int            `json:"post"`      // 1 -> public, 2 -> confirmed, 3 -> moderator
	Join          int            `json:"join"`      // 1 -> auto join, 2 -> confirm to join
	Addresses     []GroupAddress `json:"addresses"`
	Subscriptions []Subscription `json:"subscriptions"`
}

func (o Group) dbSchema(db *gorm.DB) error {
	return db.CreateTable(&o).Error
}

func (o Group) dbSeed(db *gorm.DB) error {
	return db.Create(&Group{
		Id:        "test",
		Existence: 1,
		Read:      1,
		Post:      1,
		Join:      1,
		Addresses: []GroupAddress{{
			Medium:  "twilio",
			Address: "add_twilio_phone_number",
		}},
		Subscriptions: []Subscription{},
	}).Error
}

type GroupAddress struct {
	Id      int64  `json:"group_address_id"`
	GroupId string `json:"group_id"`
	Medium  string `json:"medium"`
	Address string `json:"address"`
}

func (o GroupAddress) dbSchema(db *gorm.DB) error {
	return db.CreateTable(&o).
		AddForeignKey("group_id", "groups(id)", "RESTRICT", "RESTRICT").
		AddForeignKey("medium", "media(id)", "RESTRICT", "RESTRICT").
		AddUniqueIndex("uniqueness_idx", "medium", "address").
		Error
}

func GetGroupById(db *gorm.DB, id string) *Group {
	var o Group
	if result := db.First(&o, "id = ?", id); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	db.Model(&o).Related(&o.Addresses)
	db.Model(&o).Related(&o.Subscriptions)
	return &o
}

func GetGroupAddressByGroupId(db *gorm.DB, groupId string) []GroupAddress {
	var o []GroupAddress
	if result := db.Where("group_id = ?", groupId).Find(&o); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	return o
}

func GetGroupsByMember(db *gorm.DB, user User) []Group {
	// turn the User's list of addresses into a list of address IDs
	user_address_ids := make([]int64, len(user.Addresses))
	for i, user_address := range user.Addresses {
		user_address_ids[i] = user_address.Id
	}
	// use the list of address IDs to get a list of subscriptions
	var subscriptions []Subscription
	if len(user_address_ids) > 0 {
		if result := db.Where("address_id IN (?)", user_address_ids).Find(&subscriptions); result.Error != nil {
			if result.RecordNotFound() {
				return nil
			}
			panic(result.Error)
		}
	} else {
		subscriptions = make([]Subscription, 0)
	}
	// turn the list of subscriptions into a list of group IDs
	group_ids := make([]string, len(subscriptions))
	for i, subscription := range subscriptions {
		group_ids[i] = subscription.GroupId
	}
	// use the list of group IDs to get the groups
	var groups []Group
	if result := db.Where(group_ids).Find(&groups); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	// return them
	return groups
}

func GetGroupAddressesByMedium(db *gorm.DB, medium string) []GroupAddress {
	var o []GroupAddress
	if result := db.Where("medium = ?", medium).Find(&o); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	return o
}

func GetAllGroups(db *gorm.DB) []Group {
	var o []Group
	if result := db.Find(&o); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	return o
}

func NewGroup(db *gorm.DB, name string, existence int, read int, post int, join int) *Group {
	if name == "" {
		panic("name can't be empty")
	}
	o := Group{Id: name, Existence: existence, Read: read, Post: post, Join: join}
	if err := db.Create(&o).Error; err != nil {
		panic(err)
	}
	return &o
}

func (o *Group) Save(db *gorm.DB) {
	if err := db.Save(o).Error; err != nil {
		panic(err)
	}
}

func (o *Group) Subentity(name string, req he.Request) he.Entity {
	panic("TODO: API: (*Group).Subentity()")
}

func (o *Group) Methods() map[string]func(he.Request) he.Response {
	return map[string]func(he.Request) he.Response{
		"GET": func(req he.Request) he.Response {
			return he.StatusOK(o)
		},
		"PUT": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)

			var new_group Group
			httperr := safeDecodeJSON(req.Entity, &new_group)
			if httperr != nil {
				return *httperr
			}
			if o.Id != new_group.Id {
				return he.StatusConflict(heutil.NetString("Cannot change group id"))
			}
			*o = new_group
			o.Save(db)
			return he.StatusOK(o)
		},
		"PATCH": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)

			patch, ok := req.Entity.(jsonpatch.Patch)
			if !ok {
				return he.StatusUnsupportedMediaType(heutil.NetString("PATCH request must have a patch media type"))
			}
			var new_group Group
			err := patch.Apply(o, &new_group)
			if err != nil {
				return he.StatusConflict(heutil.NetPrintf("%v", err))
			}
			if o.Id != new_group.Id {
				return he.StatusConflict(heutil.NetString("Cannot change user id"))
			}
			*o = new_group
			o.Save(db)
			return he.StatusOK(o)
		},
		"DELETE": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)
			db.Delete(o)
			return he.StatusNoContent()
		},
	}
}

// View //////////////////////////////////////////////////////////////

func (o *Group) Encoders() map[string]func(io.Writer) error {
	return defaultEncoders(o)
}


// Directory ("Controller") //////////////////////////////////////////

type t_dirGroups struct {
	methods map[string]func(he.Request) he.Response
}

func newDirGroups() t_dirGroups {
	r := t_dirGroups{}
	r.methods = map[string]func(he.Request) he.Response{
		"GET": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)
			sess := req.Things["session"].(*Session)
			var groups []Group
			if sess == nil {
				groups = []Group{}
			} else {
				groups = GetAllGroups(db)
				// groups = GetGroupsByMember(db, *GetUserById(db, sess.UserId))
			}
			generic := make([]interface{}, len(groups))
			type EnumerateGroup struct {
				Id            string
				Existence     string
			        Read          string
				Post          string
				Join          string
				Addresses     []GroupAddress
				Subscriptions []Subscription
			}

			for i, group := range groups {
				var enum EnumerateGroup
				enum.Id = group.Id
				enum.Existence = Existence(group.Existence).String()
				enum.Read = Read(group.Read).String()
				enum.Post = Post(group.Post).String()
				enum.Join = Join(group.Join).String()
				enum.Addresses = group.Addresses
				enum.Subscriptions = group.Subscriptions
				generic[i] = enum
			}
			return he.StatusOK(heutil.NetList(generic))
		},
		"POST": func(req he.Request) he.Response {
			db := req.Things["db"].(*gorm.DB)
			type postfmt struct {
				Groupname string    `json:"groupname"`
				Existence Existence `json:"existence"`
				Read Read           `json:"read"`
				Post Post           `json:"post"`
				Join Join           `json:"join"`
			}
			var entity postfmt
			httperr := safeDecodeJSON(req.Entity, &entity)
			if httperr != nil {
				return *httperr
			}

			if entity.Groupname == "" {
				return he.StatusUnsupportedMediaType(heutil.NetString("groupname can't be emtpy"))
			}

			entity.Groupname = strings.ToLower(entity.Groupname)

			group := NewGroup(
				db,
				entity.Groupname,
				int(entity.Existence),
				int(entity.Read),
				int(entity.Post),
				int(entity.Join),
			)
			if group == nil {
				return he.StatusConflict(heutil.NetString("a group with that name already exists"))
			} else {
				return he.StatusCreated(r, group.Id, req)
			}
		},
	}
	return r
}

func (d t_dirGroups) Methods() map[string]func(he.Request) he.Response {
	return d.methods
}

func (d t_dirGroups) Subentity(name string, req he.Request) he.Entity {
	name = strings.ToLower(name)
	db := req.Things["db"].(*gorm.DB)
	// TODO: permissions check
	//sess := req.Things["session"].(*Session)
	return GetGroupById(db, name)
}
