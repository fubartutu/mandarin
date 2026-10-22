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

	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// ShoppingShop is a shop a list is bought at, e.g. Aldi or Supabarn for the
// Groceries list. A shop isn't a separate list: it's a view of its list that
// hides the items the shop doesn't stock. Ticking an item at one shop ticks it
// everywhere, because it's the same item.
type ShoppingShop struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID       int64   `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this shop."`
	ListID   int64   `xorm:"bigint not null index" json:"list_id" readOnly:"true" doc:"The list this shop belongs to. Taken from the URL; ignored in the body."`
	Title    string  `xorm:"varchar(250) not null" json:"title" valid:"runelength(1|250)" minLength:"1" maxLength:"250" doc:"The shop's name, e.g. Aldi."`
	Position float64 `xorm:"double not null default 0" json:"position" doc:"Order of the shop's tab; lower comes first."`
	// CategoryOrder is the shop's aisle order. Stored as JSON.
	CategoryOrder []int64 `xorm:"text null json" json:"category_order" doc:"Category ids in the order the shop's aisles come; its tab groups items this way. Categories not in it follow in the household order. Unknown ids and repeats are dropped."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this shop was added."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this shop was last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*ShoppingShop) TableName() string {
	return "shopping_shops"
}

func getShop(s *xorm.Session, listID, id int64) (*ShoppingShop, error) {
	shop := &ShoppingShop{}
	exists, err := s.Where("id = ? AND list_id = ?", id, listID).Get(shop)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrShopDoesNotExist{ShopID: id}
	}
	shop.fillOrder()
	return shop, nil
}

// fillOrder makes a missing aisle order [] rather than null.
func (sh *ShoppingShop) fillOrder() {
	if sh.CategoryOrder == nil {
		sh.CategoryOrder = []int64{}
	}
}

// cleanOrder drops categories that don't exist and repeats from the aisle
// order, keeping the rest in the order given.
func (sh *ShoppingShop) cleanOrder(s *xorm.Session) error {
	order := []int64{}
	if len(sh.CategoryOrder) > 0 {
		var known []int64
		if err := s.Table("shopping_categories").In("id", sh.CategoryOrder).Cols("id").Find(&known); err != nil {
			return err
		}
		exists := make(map[int64]bool, len(known))
		for _, id := range known {
			exists[id] = true
		}
		for _, id := range sh.CategoryOrder {
			if exists[id] {
				order = append(order, id)
				delete(exists, id)
			}
		}
	}
	sh.CategoryOrder = order
	return nil
}

// shopsForLists returns each list's shops in tab order, keyed by list id.
func shopsForLists(s *xorm.Session, listIDs []int64) (map[int64][]*ShoppingShop, error) {
	byList := make(map[int64][]*ShoppingShop, len(listIDs))
	if len(listIDs) == 0 {
		return byList, nil
	}
	shops := []*ShoppingShop{}
	if err := s.In("list_id", listIDs).OrderBy("position ASC, id ASC").Find(&shops); err != nil {
		return nil, err
	}
	for _, shop := range shops {
		shop.fillOrder()
		byList[shop.ListID] = append(byList[shop.ListID], shop)
	}
	return byList, nil
}

// CanRead lets every household member read every shop.
func (sh *ShoppingShop) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if err := requireMember(a); err != nil {
		return false, 0, err
	}
	stored, err := getShop(s, sh.ListID, sh.ID)
	if err != nil {
		return false, 0, err
	}
	*sh = *stored
	return true, int(permissionAdmin), nil
}

// CanCreate lets members add shops to any existing list.
func (sh *ShoppingShop) CanCreate(s *xorm.Session, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getListByID(s, sh.ListID); err != nil {
		return false, err
	}
	return true, nil
}

// CanUpdate lets members rename or reorder any shop.
func (sh *ShoppingShop) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteShop(s, sh.ListID, sh.ID, a)
}

// CanDelete lets members remove any shop.
func (sh *ShoppingShop) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteShop(s, sh.ListID, sh.ID, a)
}

func canWriteShop(s *xorm.Session, listID, id int64, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getShop(s, listID, id); err != nil {
		return false, err
	}
	return true, nil
}

// Create adds a shop to the end of its list's tabs.
func (sh *ShoppingShop) Create(s *xorm.Session, _ web.Auth) (err error) {
	sh.ID = 0
	if sh.Position == 0 {
		if sh.Position, err = nextPosition(s.Where("list_id = ?", sh.ListID), "shopping_shops"); err != nil {
			return err
		}
	}
	if err = sh.cleanOrder(s); err != nil {
		return err
	}
	if _, err = s.Insert(sh); err != nil {
		return err
	}
	// Phones get shops as part of their list.
	return recordChange(s, entityList, sh.ListID, "", false)
}

// ReadOne has nothing to add: CanRead already loaded the row.
func (sh *ShoppingShop) ReadOne(_ *xorm.Session, _ web.Auth) error {
	return nil
}

// ReadAll returns the shops of ListID in tab order.
func (sh *ShoppingShop) ReadAll(s *xorm.Session, a web.Auth, _ string, page int, perPage int) (any, int, int64, error) {
	if err := requireMember(a); err != nil {
		return nil, 0, 0, err
	}
	if _, err := getListByID(s, sh.ListID); err != nil {
		return nil, 0, 0, err
	}
	total, err := s.Where("list_id = ?", sh.ListID).Count(&ShoppingShop{})
	if err != nil {
		return nil, 0, 0, err
	}
	shops := []*ShoppingShop{}
	err = paginate(s.Where("list_id = ?", sh.ListID), page, perPage).OrderBy("position ASC, id ASC").Find(&shops)
	if err != nil {
		return nil, 0, 0, err
	}
	for _, shop := range shops {
		shop.fillOrder()
	}
	return shops, len(shops), total, nil
}

// Update changes the title, position and aisle order.
func (sh *ShoppingShop) Update(s *xorm.Session, _ web.Auth) error {
	if err := sh.cleanOrder(s); err != nil {
		return err
	}
	if _, err := s.ID(sh.ID).Cols("title", "position", "category_order").Update(sh); err != nil {
		return err
	}
	stored, err := getShop(s, sh.ListID, sh.ID)
	if err != nil {
		return err
	}
	*sh = *stored
	return recordChange(s, entityList, sh.ListID, "", false)
}

// Delete removes the shop and every "not stocked here" mark for it. Items
// that were only for this shop become items for any shop.
func (sh *ShoppingShop) Delete(s *xorm.Session, _ web.Auth) error {
	if err := deleteShops(s, []int64{sh.ID}); err != nil {
		return err
	}
	return recordChange(s, entityList, sh.ListID, "", false)
}

// deleteShops removes shops and their not-stocked marks, and frees the items
// that were only for them. The caller logs the change to the shops' list.
func deleteShops(s *xorm.Session, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	// The catalogue entries that lose a mark change too.
	var marked []int64
	if err := s.Table("shopping_not_stocked").In("shop_id", ids).Distinct("catalog_item_id").Find(&marked); err != nil {
		return err
	}
	if _, err := s.In("shop_id", ids).Delete(&ShoppingNotStocked{}); err != nil {
		return err
	}
	if _, err := s.In("id", ids).Delete(&ShoppingShop{}); err != nil {
		return err
	}
	if err := freeShopItems(s, ids); err != nil {
		return err
	}
	return recordChanges(s, entityCatalog, marked)
}

// freeShopItems sets items that were only for the given shops to any shop.
// The change counts as made now, so it wins over an offline edit that
// still names the shop.
func freeShopItems(s *xorm.Session, shopIDs []int64) error {
	items := []*ShoppingItem{}
	if err := s.In("shop_id", shopIDs).Find(&items); err != nil {
		return err
	}
	now := time.Now()
	for _, item := range items {
		item.ShopID = 0
		item.FieldTimes = item.FieldTimes.with([]string{fieldShop}, now, 0)
		if _, err := s.ID(item.ID).Cols("shop_id", "field_times").Update(item); err != nil {
			return err
		}
		if err := recordItemChange(s, item, false); err != nil {
			return err
		}
	}
	return nil
}
