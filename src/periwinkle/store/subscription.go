// Copyright 2015 Davis Webb
// Copyright 2015 Luke Shumaker
// Copyright 2015 Mark Pundmann

package store

import (
	"github.com/jinzhu/gorm"
	he "httpentity"
	"httpentity/util" // heutil
)

type Subscription struct {
	Address   UserAddress
	AddressId int64 `json:"-"`
	Group     Group
	GroupId   string `json:"group_id"`
}

func (o Subscription) dbSchema(db *gorm.DB) error {
	return db.CreateTable(&o).
		AddForeignKey("group_id", "groups(id)", "RESTRICT", "RESTRICT").
		AddForeignKey("address_id", "user_addresses(id)", "RESTRICT", "RESTRICT").
		Error
}

func GetSubscriptionsGroupById(db *gorm.DB, groupId string) []Subscription {
	var o []Subscription
	if result := db.Where("group_id = ?", groupId).Find(&o); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	return o
}

type t_dirSubscriptions struct {
	methods map[string]func(he.Request) he.Response
}

func newDirSubscriptions() t_dirSubscriptions {
	r := t_dirSubscriptions{}
	r.methods = map[string]func(he.Request) he.Response{
		"GET": func(req he.Request) he.Response {
			panic("Not yet implemented")
			return he.StatusOK(heutil.NetString("Not yet implemented"))
		},
	}
	return r
}

func (d t_dirSubscriptions) Methods() map[string]func(he.Request) he.Response {
	return d.methods
}

func (d t_dirSubscriptions) Subentity(user_id string, group_name string, req he.Request) he.Entity {
	//group_name = strings.ToLower(group_name)
	//db := req.Things["db"].(*gorm.DB)
	panic("Not yet implemented")
}
