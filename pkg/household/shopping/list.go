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

	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// ShoppingList is one household shopping list, e.g. "Groceries" or "Kmart".
//
// The struct tags do several jobs at once: `xorm` describes the database
// column, `json` the API field name, `valid` is checked by Vikunja's
// validator, and `doc`/`minLength`/`readOnly` feed the /api/v2 OpenAPI schema
// the frontend client is generated from.
type ShoppingList struct { //nolint:revive // "shopping.ShoppingList" stutters, but the type name is the OpenAPI schema name.
	ID       int64   `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this shopping list."`
	Title    string  `xorm:"varchar(250) not null" json:"title" valid:"runelength(1|250)" minLength:"1" maxLength:"250" doc:"The name of the list, e.g. Groceries."`
	Position float64 `xorm:"double not null default 0" json:"position" doc:"Sort order among lists; lower comes first."`

	CreatedByID int64      `xorm:"bigint not null default 0" json:"-"`
	CreatedBy   *user.User `xorm:"-" json:"created_by" readOnly:"true" doc:"The member who created this list. Null for lists the server created."`

	Shops []*ShoppingShop `xorm:"-" json:"shops" readOnly:"true" doc:"The shops this list is bought at, in tab order. Manage them under /shopping/lists/{list}/shops."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this list was created."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this list was last changed."`

	// Embedding these interfaces satisfies them without writing stub methods;
	// the methods we do write below take precedence.
	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*ShoppingList) TableName() string {
	return "shopping_lists"
}

func getListByID(s *xorm.Session, id int64) (*ShoppingList, error) {
	l := &ShoppingList{}
	exists, err := s.Where("id = ?", id).Get(l)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrListDoesNotExist{ListID: id}
	}
	return l, nil
}

// load replaces the receiver with the stored list, so later steps (ReadOne,
// Update) work on the real row. *l = *stored copies the whole struct.
func (l *ShoppingList) load(s *xorm.Session, a web.Auth) error {
	if err := requireMember(a); err != nil {
		return err
	}
	stored, err := getListByID(s, l.ID)
	if err != nil {
		return err
	}
	*l = *stored
	return nil
}

// CanRead lets every household member read every list, with admin permission.
func (l *ShoppingList) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if err := l.load(s, a); err != nil {
		return false, 0, err
	}
	return true, int(permissionAdmin), nil
}

// CanCreate lets every household member create lists.
func (l *ShoppingList) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	return household.IsMember(a), nil
}

// CanUpdate lets every household member change any list.
func (l *ShoppingList) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteList(s, l.ID, a)
}

// CanDelete lets every household member delete any list.
func (l *ShoppingList) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteList(s, l.ID, a)
}

func canWriteList(s *xorm.Session, id int64, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getListByID(s, id); err != nil {
		return false, err
	}
	return true, nil
}

// Create adds a new list; the caller becomes its creator.
func (l *ShoppingList) Create(s *xorm.Session, a web.Auth) error {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	l.ID = 0
	l.CreatedByID = u.ID
	l.CreatedBy = u
	l.Shops = []*ShoppingShop{}
	if l.Position == 0 {
		if l.Position, err = nextPosition(s, "shopping_lists"); err != nil {
			return err
		}
	}
	if _, err = s.Insert(l); err != nil {
		return err
	}
	return recordChange(s, entityList, l.ID, "", false)
}

// ReadOne fills in the creator; CanRead already loaded the row.
func (l *ShoppingList) ReadOne(s *xorm.Session, _ web.Auth) error {
	creators, err := loadCreators(s, []int64{l.CreatedByID})
	if err != nil {
		return err
	}
	l.CreatedBy = creators[l.CreatedByID]
	return fillShops(s, l)
}

// fillShops sets Shops on each list ([] rather than null when there are none).
func fillShops(s *xorm.Session, lists ...*ShoppingList) error {
	ids := make([]int64, 0, len(lists))
	for _, list := range lists {
		ids = append(ids, list.ID)
	}
	byList, err := shopsForLists(s, ids)
	if err != nil {
		return err
	}
	for _, list := range lists {
		list.Shops = byList[list.ID]
		if list.Shops == nil {
			list.Shops = []*ShoppingShop{}
		}
	}
	return nil
}

// ReadAll returns every household list, optionally filtered by title.
func (l *ShoppingList) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (any, int, int64, error) {
	if err := requireMember(a); err != nil {
		return nil, 0, 0, err
	}

	// A small closure so the count and the page use the same filter.
	query := func() *xorm.Session {
		q := s.Table("shopping_lists")
		if search != "" {
			q = q.Where("title LIKE ?", "%"+search+"%")
		}
		return q
	}

	total, err := query().Count(&ShoppingList{})
	if err != nil {
		return nil, 0, 0, err
	}

	lists := []*ShoppingList{}
	err = paginate(query(), page, perPage).OrderBy("position ASC, id ASC").Find(&lists)
	if err != nil {
		return nil, 0, 0, err
	}

	ids := make([]int64, 0, len(lists))
	for _, list := range lists {
		ids = append(ids, list.CreatedByID)
	}
	creators, err := loadCreators(s, ids)
	if err != nil {
		return nil, 0, 0, err
	}
	for _, list := range lists {
		list.CreatedBy = creators[list.CreatedByID]
	}
	if err := fillShops(s, lists...); err != nil {
		return nil, 0, 0, err
	}

	return lists, len(lists), total, nil
}

// Update changes the title and position. Columns not listed in Cols are never
// written, so a client can't overwrite created_by.
func (l *ShoppingList) Update(s *xorm.Session, a web.Auth) error {
	if _, err := s.ID(l.ID).Cols("title", "position").Update(l); err != nil {
		return err
	}
	if err := recordChange(s, entityList, l.ID, "", false); err != nil {
		return err
	}
	if err := l.load(s, a); err != nil {
		return err
	}
	return l.ReadOne(s, a)
}

// Delete removes the list with its items and shops.
func (l *ShoppingList) Delete(s *xorm.Session, _ web.Auth) error {
	items := []*ShoppingItem{}
	if err := s.Where("list_id = ?", l.ID).Cols("id", "uid").Find(&items); err != nil {
		return err
	}
	if err := deleteItems(s, items...); err != nil {
		return err
	}
	var shopIDs []int64
	if err := s.Table("shopping_shops").Where("list_id = ?", l.ID).Cols("id").Find(&shopIDs); err != nil {
		return err
	}
	if err := deleteShops(s, shopIDs); err != nil {
		return err
	}
	if _, err := s.ID(l.ID).Delete(&ShoppingList{}); err != nil {
		return err
	}
	return recordChange(s, entityList, l.ID, "", true)
}

// ClearDoneItems deletes the ticked items on the list and returns how many
// went. Callers must check CanUpdate first; this is not a generic CRUD action.
func (l *ShoppingList) ClearDoneItems(s *xorm.Session) (int64, error) {
	items := []*ShoppingItem{}
	if err := s.Where("list_id = ? AND done = ?", l.ID, true).Cols("id", "uid").Find(&items); err != nil {
		return 0, err
	}
	return int64(len(items)), deleteItems(s, items...)
}
