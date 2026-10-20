// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package shopping

import (
	"time"

	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// ShoppingItem is one line on a shopping list: "2 L milk", "500 g mince",
// "school shoes (size 3)".
type ShoppingItem struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID         int64   `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this item."`
	ListID     int64   `xorm:"bigint not null index" json:"list_id" readOnly:"true" doc:"The list this item is on. Taken from the URL; ignored in the body."`
	Name       string  `xorm:"varchar(250) not null" json:"name" valid:"runelength(1|250)" minLength:"1" maxLength:"250" doc:"What to buy, e.g. Milk."`
	Quantity   float64 `xorm:"double not null default 0" json:"quantity" minimum:"0" doc:"How many or how much, read together with unit. 0 means no quantity."`
	Unit       string  `xorm:"varchar(50) not null default ''" json:"unit" maxLength:"50" doc:"Unit for quantity, e.g. g, kg, L, packs. Empty for a plain count."`
	Note       string  `xorm:"text null" json:"note" maxLength:"2000" doc:"Free-text note, e.g. the brand or a size."`
	CategoryID int64   `xorm:"bigint not null default 0 index" json:"category_id" minimum:"0" doc:"The category this item is grouped under. 0 means uncategorised."`
	Done       bool    `xorm:"bool not null default false" json:"done" doc:"Whether the item has been ticked off."`
	// A pointer so "not done" is JSON null instead of a zero date.
	DoneAt   *time.Time `xorm:"datetime null" json:"done_at" readOnly:"true" doc:"When the item was ticked off; null while not done. Set by the server."`
	Position float64    `xorm:"double not null default 0" json:"position" doc:"Order within its category; lower comes first."`

	CreatedByID int64      `xorm:"bigint not null default 0" json:"-"`
	CreatedBy   *user.User `xorm:"-" json:"created_by" readOnly:"true" doc:"The member who added this item."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this item was added."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this item was last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*ShoppingItem) TableName() string {
	return "shopping_items"
}

// getItem loads an item and checks it really is on the list from the URL.
func getItem(s *xorm.Session, listID, id int64) (*ShoppingItem, error) {
	item := &ShoppingItem{}
	exists, err := s.Where("id = ? AND list_id = ?", id, listID).Get(item)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrItemDoesNotExist{ItemID: id}
	}
	return item, nil
}

// CanRead lets every household member read every item.
func (i *ShoppingItem) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if err := requireMember(a); err != nil {
		return false, 0, err
	}
	stored, err := getItem(s, i.ListID, i.ID)
	if err != nil {
		return false, 0, err
	}
	*i = *stored
	return true, int(permissionAdmin), nil
}

// CanCreate lets members add items to any existing list.
func (i *ShoppingItem) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getListByID(s, i.ListID); err != nil {
		return false, err
	}
	return true, nil
}

// CanUpdate lets members change any item.
func (i *ShoppingItem) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteItem(s, i.ListID, i.ID, a)
}

// CanDelete lets members remove any item.
func (i *ShoppingItem) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteItem(s, i.ListID, i.ID, a)
}

func canWriteItem(s *xorm.Session, listID, id int64, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getItem(s, listID, id); err != nil {
		return false, err
	}
	return true, nil
}

// checkCategory rejects category ids that don't exist; 0 (none) is fine.
func checkCategory(s *xorm.Session, categoryID int64) error {
	if categoryID == 0 {
		return nil
	}
	_, err := getCategoryByID(s, categoryID)
	return err
}

// setDoneAt keeps done_at in step with done: stamped when ticked, cleared
// when unticked, left alone otherwise.
func (i *ShoppingItem) setDoneAt(wasDone bool, previous *time.Time) {
	switch {
	case i.Done && !wasDone:
		now := time.Now()
		i.DoneAt = &now
	case !i.Done:
		i.DoneAt = nil
	default:
		i.DoneAt = previous
	}
}

// Create adds an item to the list in ListID.
func (i *ShoppingItem) Create(s *xorm.Session, a web.Auth) error {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	if err := checkCategory(s, i.CategoryID); err != nil {
		return err
	}
	i.ID = 0
	i.CreatedByID = u.ID
	i.CreatedBy = u
	i.setDoneAt(false, nil)
	_, err = s.Insert(i)
	return err
}

// ReadOne fills in the creator; CanRead already loaded the row.
func (i *ShoppingItem) ReadOne(s *xorm.Session, _ web.Auth) error {
	creators, err := loadCreators(s, []int64{i.CreatedByID})
	if err != nil {
		return err
	}
	i.CreatedBy = creators[i.CreatedByID]
	return nil
}

// ReadAll returns the items on ListID: not-done first, then by position.
// The frontend groups them by category.
func (i *ShoppingItem) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (any, int, int64, error) {
	if err := requireMember(a); err != nil {
		return nil, 0, 0, err
	}
	if _, err := getListByID(s, i.ListID); err != nil {
		return nil, 0, 0, err
	}

	query := func() *xorm.Session {
		q := s.Table("shopping_items").Where("list_id = ?", i.ListID)
		if search != "" {
			q = q.And("name LIKE ?", "%"+search+"%")
		}
		return q
	}

	total, err := query().Count(&ShoppingItem{})
	if err != nil {
		return nil, 0, 0, err
	}

	items := []*ShoppingItem{}
	err = paginate(query(), page, perPage).OrderBy("done ASC, position ASC, id ASC").Find(&items)
	if err != nil {
		return nil, 0, 0, err
	}

	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.CreatedByID)
	}
	creators, err := loadCreators(s, ids)
	if err != nil {
		return nil, 0, 0, err
	}
	for _, item := range items {
		item.CreatedBy = creators[item.CreatedByID]
	}

	return items, len(items), total, nil
}

// Update replaces the editable fields. The item can't move lists; list_id
// and created_by are never written.
func (i *ShoppingItem) Update(s *xorm.Session, a web.Auth) error {
	stored, err := getItem(s, i.ListID, i.ID)
	if err != nil {
		return err
	}
	if err := checkCategory(s, i.CategoryID); err != nil {
		return err
	}
	i.setDoneAt(stored.Done, stored.DoneAt)

	_, err = s.ID(i.ID).
		Cols("name", "quantity", "unit", "note", "category_id", "done", "done_at", "position").
		Update(i)
	if err != nil {
		return err
	}
	return i.reload(s, a)
}

func (i *ShoppingItem) reload(s *xorm.Session, a web.Auth) error {
	stored, err := getItem(s, i.ListID, i.ID)
	if err != nil {
		return err
	}
	*i = *stored
	return i.ReadOne(s, a)
}

// Delete removes the item.
func (i *ShoppingItem) Delete(s *xorm.Session, _ web.Auth) error {
	_, err := s.Where("id = ? AND list_id = ?", i.ID, i.ListID).Delete(&ShoppingItem{})
	return err
}
