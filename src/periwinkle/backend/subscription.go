// Copyright 2015 Davis Webb
// Copyright 2015 Luke Shumaker
// Copyright 2015 Mark Pundmann

package backend

import (
	"github.com/jinzhu/gorm"
)

type Subscription struct {
	Address   UserAddress `json:"addresses"`
	AddressID int64       `json:"-"`
	Group     Group       `json:"group"`
	GroupID   string      `json:"group_id"`
	Confirmed bool        `json:"confirmed"`
}

func (o Subscription) dbSchema(db *gorm.DB) error {
	return db.CreateTable(&o).
		AddForeignKey("group_id", "groups(id)", "CASCADE", "RESTRICT").
		AddForeignKey("address_id", "user_addresses(id)", "CASCADE", "RESTRICT").
		Error
}

func GetSubscriptionsGroupByID(db *gorm.DB, groupID string) []Subscription {
	var o []Subscription
	if result := db.Where("group_id = ?", groupID).Find(&o); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		panic(result.Error)
	}
	return o
}

func IsSubscribed(db *gorm.DB, userID string, group Group) bool {
	subscriptions := GetSubscriptionsGroupByID(db, group.ID)
	addressIDs := make([]int64, len(subscriptions))
	for i, subscription := range subscriptions {
		addressIDs[i] = subscription.AddressID
	}
	var addresses []UserAddress
	if len(addressIDs) > 0 {
		if result := db.Where("id IN (?)", addressIDs).Find(&addresses); result.Error != nil {
			if !result.RecordNotFound() {
				panic("cant find any subscriptions corresponding user address")
			}
		}
	} else {
		// no subscriptions so user cannot possibly be subscribed
		return false
	}
	for _, address := range addresses {
		if address.UserID == userID {
			return true
		}
	}
	// could not find user in subscribed user addresses, therefore, he/she isn't subscribed
	return false
}

func IsAdmin(db *gorm.DB, userID string, group Group) bool {
	subscriptions := GetSubscriptionsGroupByID(db, group.ID)
	addressIDs := make([]int64, len(subscriptions))
	for i, subscription := range subscriptions {
		addressIDs[i] = subscription.AddressID
	}
	var addresses []UserAddress
	if len(addressIDs) > 0 {
		if result := db.Where("id IN (?)", addressIDs).Find(&addresses); result.Error != nil {
			if !result.RecordNotFound() {
				panic("cant find any subscriptions corresponding user address")
			}
		}
	} else {
		// no subscriptions so user cannot possibly be subscribed
		return false
	}
	for _, address := range addresses {
		if address.UserID == userID && address.Medium == "admin" {
			return true
		}
	}
	// could not find user in subscribed user addresses, therefore, he/she isn't subscribed
	return false
}