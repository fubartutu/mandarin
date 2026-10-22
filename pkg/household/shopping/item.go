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

	"github.com/google/uuid"
	"xorm.io/xorm"
)

// ShoppingItem is one line on a shopping list: "2 L milk", "500 g mince",
// "school shoes (size 3)".
type ShoppingItem struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID int64 `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this item."`
	// UID is made by the phone that adds the item, so an item added offline
	// has an id before the server has seen it.
	UID        string  `xorm:"'uid' varchar(36) not null unique" json:"uid" readOnly:"true" doc:"The item's id across devices: a UUID made by the app that added it. Sync uses it; the numeric id is for these REST endpoints."`
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
	ShopID   int64      `xorm:"bigint not null default 0 index" json:"shop_id" minimum:"0" doc:"The only shop to buy this at, e.g. a brand only Supabarn has: it's hidden on the other shops' tabs. 0 means any shop."`

	CatalogItemID     int64   `xorm:"bigint not null default 0 index" json:"catalog_item_id" readOnly:"true" doc:"The catalogue entry this item was matched to by name. Set by the server."`
	NotStockedShopIDs []int64 `xorm:"-" json:"not_stocked_shop_ids" readOnly:"true" doc:"Shops that don't stock this item (from its catalogue entry); hide it on those shops' tabs."`
	HasPhoto          bool    `xorm:"-" json:"has_photo" readOnly:"true" doc:"Whether the item's catalogue entry has a photo."`

	CreatedByID int64      `xorm:"bigint not null default 0" json:"-"`
	CreatedBy   *user.User `xorm:"-" json:"created_by" readOnly:"true" doc:"The member who added this item."`

	// FieldTimes says when each field last changed, so sync can let the
	// latest change win per field (see sync.go).
	FieldTimes fieldStamps `xorm:"text null json" json:"-"`

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

// checkShop rejects shop ids that aren't shops of listID; 0 (any) is fine.
func checkShop(s *xorm.Session, listID, shopID int64) error {
	if shopID == 0 {
		return nil
	}
	_, err := getShop(s, listID, shopID)
	return err
}

// checkCategory rejects category ids that don't exist; 0 (none) is fine.
func checkCategory(s *xorm.Session, categoryID int64) error {
	if categoryID == 0 {
		return nil
	}
	_, err := getCategoryByID(s, categoryID)
	return err
}

// setDoneAt keeps done_at in step with done: set to now when ticked,
// cleared when unticked, left alone otherwise.
func (i *ShoppingItem) setDoneAt(wasDone bool, previous *time.Time, now time.Time) {
	switch {
	case i.Done && !wasDone:
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
	if err := checkShop(s, i.ListID, i.ShopID); err != nil {
		return err
	}
	i.UID = uuid.NewString()
	return i.insert(s, u, time.Now())
}

// insert adds a new item made by u at time at. Sync passes the time the
// item was added on the phone.
func (i *ShoppingItem) insert(s *xorm.Session, u *user.User, at time.Time) error {
	// Fills blank category/quantity/unit from last time and remembers this use.
	if err := useCatalogItem(s, i, true); err != nil {
		return err
	}
	i.ID = 0
	i.CreatedByID = u.ID
	i.CreatedBy = u
	i.setDoneAt(false, nil, at)
	i.FieldTimes = stampAll(at, u.ID)
	if _, err := s.Insert(i); err != nil {
		return err
	}
	if err := recordItemChange(s, i, false); err != nil {
		return err
	}
	if OnItemAdded != nil {
		if err := OnItemAdded(s, i.ListID, i.Name, u.ID); err != nil {
			return err
		}
	}
	return fillItemDetails(s, i)
}

// OnItemAdded is called in the same transaction whenever an item is added
// to a list (not when an add is merged into an existing item). The push
// package uses it for shopping summaries; it's a variable so this package
// doesn't import that one.
var OnItemAdded func(s *xorm.Session, listID int64, name string, actorID int64) error

// fillItemDetails copies photo and not-stocked info from the items'
// catalogue entries.
func fillItemDetails(s *xorm.Session, items ...*ShoppingItem) error {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		if item.CatalogItemID != 0 {
			ids = append(ids, item.CatalogItemID)
		}
	}
	details, err := loadCatalogDetails(s, ids)
	if err != nil {
		return err
	}
	for _, item := range items {
		item.NotStockedShopIDs = []int64{}
		if d, ok := details[item.CatalogItemID]; ok {
			item.HasPhoto = d.hasPhoto
			item.NotStockedShopIDs = d.notStocked
		}
	}
	return nil
}

// ReadOne fills in the creator; CanRead already loaded the row.
func (i *ShoppingItem) ReadOne(s *xorm.Session, _ web.Auth) error {
	creators, err := loadCreators(s, []int64{i.CreatedByID})
	if err != nil {
		return err
	}
	i.CreatedBy = creators[i.CreatedByID]
	return fillItemDetails(s, i)
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
	if err := fillItemDetails(s, items...); err != nil {
		return nil, 0, 0, err
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
	if err := checkShop(s, i.ListID, i.ShopID); err != nil {
		return err
	}
	i.UID = stored.UID // read-only, so not in the body
	now := time.Now()
	i.setDoneAt(stored.Done, stored.DoneAt, now)
	// The fields that differ count as changed now, by this member.
	i.FieldTimes = stored.FieldTimes.with(changedFields(stored, i), now, a.GetID())
	if err := i.save(s); err != nil {
		return err
	}
	return i.reload(s, a)
}

// save writes the editable fields of an existing item and logs the change.
func (i *ShoppingItem) save(s *xorm.Session) error {
	// Relinks the item if it was renamed, and teaches the catalogue the edit.
	if err := useCatalogItem(s, i, false); err != nil {
		return err
	}
	_, err := s.ID(i.ID).
		Cols("name", "quantity", "unit", "note", "category_id", "done", "done_at", "position", "shop_id", "catalog_item_id", "field_times").
		Update(i)
	if err != nil {
		return err
	}
	return recordItemChange(s, i, false)
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
	stored, err := getItem(s, i.ListID, i.ID)
	if err != nil {
		return err
	}
	return deleteItems(s, stored)
}

// deleteItems removes items and leaves a deletion in the change log for each.
func deleteItems(s *xorm.Session, items ...*ShoppingItem) error {
	for _, item := range items {
		if _, err := s.ID(item.ID).Delete(&ShoppingItem{}); err != nil {
			return err
		}
		if err := recordItemChange(s, item, true); err != nil {
			return err
		}
	}
	return nil
}
