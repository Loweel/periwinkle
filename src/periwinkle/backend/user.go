// Copyright 2015 Davis Webb
// Copyright 2015 Guntas Grewal
// Copyright 2015 Luke Shumaker

package backend

import (
	"locale"
	"strings"

	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string        `json:"user_id"`
	FullName  string        `json:"fullname"`
	PwHash    []byte        `json:"-"`
	Addresses []UserAddress `json:"addresses"`
}

func (o User) dbSchema(db *gorm.DB) locale.Error {
	return locale.UntranslatedError(db.CreateTable(&o).Error)
}

type UserAddress struct {
	// TODO: add a "verified" boolean
	ID            int64          `json:"-"`
	UserID        string         `json:"-"      sql:"type:varchar(255) REFERENCES users(id) ON DELETE CASCADE  ON UPDATE RESTRICT"`
	Medium        string         `json:"medium" sql:"type:varchar(255) REFERENCES media(id) ON DELETE RESTRICT ON UPDATE RESTRICT"`
	Address       string         `json:"address"`
	SortOrder     uint64         `json:"sort_order"`
	Confirmed     bool           `json:"confirmed"`
	Subscriptions []Subscription `json:"subscriptions"`
}

func (o UserAddress) dbSchema(db *gorm.DB) locale.Error {
	return locale.UntranslatedError(db.CreateTable(&o).
		AddUniqueIndex("address_idx", "medium", "address").
		//AddUniqueIndex("user_idx", "user_id", "sort_order").
		Error)
}

func (addr UserAddress) AsEmailAddress() string {
	if addr.Medium == "email" {
		return addr.Address
	} else {
		return addr.Address + "@" + addr.Medium + ".gateway"
	}
}

func (u *User) populate(db *gorm.DB) {
	db.Where(`user_id = ? AND medium != "noop" AND medium != "admin"`, u.ID).Model(UserAddress{}).Find(&u.Addresses)
	addressIDs := make([]int64, len(u.Addresses))
	for i, address := range u.Addresses {
		addressIDs[i] = address.ID
	}
	var subscriptions []Subscription
	if len(addressIDs) > 0 {
		if result := db.Where("address_id IN (?)", addressIDs).Find(&subscriptions); result.Error != nil {
			if !result.RecordNotFound() {
				dbError(result.Error)
			}
		}
	} else {
		subscriptions = make([]Subscription, 0)
	}
	for i := range u.Addresses {
		u.Addresses[i].Subscriptions = []Subscription{}
		for _, subscription := range subscriptions {
			if u.Addresses[i].ID == subscription.AddressID {
				u.Addresses[i].Subscriptions = append(u.Addresses[i].Subscriptions, subscription)
			}
		}
	}
}

func (u *User) GetUserSubscriptions(db *gorm.DB) []Subscription {
	db.Model(u).Related(&u.Addresses)
	addressIDs := make([]int64, len(u.Addresses))
	for i, address := range u.Addresses {
		addressIDs[i] = address.ID
	}
	var subscriptions []Subscription
	if len(addressIDs) > 0 {
		if result := db.Where("address_id IN (?)", addressIDs).Find(&subscriptions); result.Error != nil {
			if !result.RecordNotFound() {
				dbError(result.Error)
			}
		}
	} else {
		subscriptions = make([]Subscription, 0)
	}
	for _, subscription := range subscriptions {
		db.Model(subscription).Related(&subscription.Group)
	}
	return subscriptions
}

func GetAddressByIDAndMedium(db *gorm.DB, id string, medium string) *UserAddress {
	id = strings.ToLower(id)
	var o UserAddress
	if result := db.Where("user_id=? and medium=?", id, medium).First(&o); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		dbError(result.Error)
	}
	return &o
}

func GetUserByID(db *gorm.DB, id string) *User {
	id = strings.ToLower(id)
	var o User
	if result := db.First(&o, "id = ?", id); result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		dbError(result.Error)
	}
	o.populate(db)
	return &o
}

func GetUserByAddress(db *gorm.DB, medium string, address string) *User {
	var o User
	result := db.Joins("INNER JOIN user_addresses ON user_addresses.user_id=users.id").Where("user_addresses.medium=? and user_addresses.address=?", medium, address).Find(&o)
	if result.Error != nil {
		if result.RecordNotFound() {
			return nil
		}
		dbError(result.Error)
	}
	o.populate(db)
	return &o
}

func (u *User) SetPassword(password string) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), -1)
	if err != nil {
		panic(err) // Luke says this is OK
	}
	u.PwHash = hash
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword(u.PwHash, []byte(password))
	return err == nil
}

func NewUser(db *gorm.DB, name string, password string, email string) User {
	if name == "" {
		programmerError("User name can't be empty")
	}
	o := User{
		ID:        name,
		FullName:  "",
		Addresses: []UserAddress{{Medium: "email", Address: email, Confirmed: false}},
	}
	o.SetPassword(password)
	if err := db.Create(&o).Error; err != nil {
		dbError(err)
	}
	return o
}

func NewUserAddress(db *gorm.DB, userID string, medium string, address string, confirmed bool) UserAddress {
	o := UserAddress{
		UserID:        userID,
		Medium:        medium,
		Address:       address,
		Subscriptions: make([]Subscription, 0),
		Confirmed:     confirmed,
	}
	if err := db.Create(&o).Error; err != nil {
		dbError(err)
	}
	return o
}

func (usr *User) Save(db *gorm.DB) {
	if usr.Addresses != nil {
		var oldAddresses []UserAddress
		db.Model(usr).Related(&oldAddresses)

		deleteAddressIDs := []int64{}
		for o := range oldAddresses {
			oldAddr := &oldAddresses[o]
			match := false
			for n := range usr.Addresses {
				newAddr := &usr.Addresses[n]
				if newAddr.Medium == oldAddr.Medium && newAddr.Address == oldAddr.Address {
					newAddr.ID = oldAddr.ID
					match = true
				}
			}
			if !match {
				deleteAddressIDs = append(deleteAddressIDs, oldAddr.ID)
			}
		}

		if err := db.Save(usr).Error; err != nil {
			dbError(err)
		}
		if len(deleteAddressIDs) > 0 {
			if err := db.Where("id IN (?)", deleteAddressIDs).Delete(UserAddress{}).Error; err != nil {
				dbError(err)
			}
		}
	} else {
		if err := db.Save(usr).Error; err != nil {
			dbError(err)
		}
	}
}
