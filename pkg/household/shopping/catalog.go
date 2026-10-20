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
	"strings"
	"time"

	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// ShoppingCatalogItem is the household's memory of an item: every name ever
// added, plus a pre-loaded set of common items. It remembers the category and
// the last quantity and unit used, how often it was added (for autocomplete),
// an optional photo, and which shops don't stock it.
type ShoppingCatalogItem struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID   int64  `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this catalogue entry."`
	Name string `xorm:"varchar(250) not null" json:"name" valid:"runelength(1|250)" minLength:"1" maxLength:"250" doc:"The item's name as it's shown, e.g. Vegemite."`
	// NameKey is the lower-cased, trimmed name: "Milk" and " milk" are one entry.
	NameKey    string  `xorm:"varchar(250) not null unique" json:"-"`
	CategoryID int64   `xorm:"bigint not null default 0 index" json:"category_id" minimum:"0" doc:"The category new list items with this name go into. 0 means uncategorised."`
	Quantity   float64 `xorm:"double not null default 0" json:"quantity" minimum:"0" doc:"The quantity used last time; offered again when the item is re-added. 0 means none."`
	Unit       string  `xorm:"varchar(50) not null default ''" json:"unit" maxLength:"50" doc:"The unit used last time."`

	TimesAdded int64      `xorm:"bigint not null default 0" json:"times_added" readOnly:"true" doc:"How many times the item has been added to a list. Autocomplete ranks frequent items first."`
	LastAdded  *time.Time `xorm:"datetime null" json:"last_added" readOnly:"true" doc:"When the item was last added to a list; null if never."`

	PhotoFileID int64 `xorm:"bigint not null default 0" json:"photo_id" readOnly:"true" doc:"Changes whenever the photo is replaced, so clients can cache photos by it. 0 means no photo."`
	HasPhoto    bool  `xorm:"-" json:"has_photo" readOnly:"true" doc:"Whether the item has a photo. Fetch it from /shopping/catalog/{id}/photo."`

	NotStockedShopIDs []int64 `xorm:"-" json:"not_stocked_shop_ids" doc:"Shops that don't stock this item. It is hidden on those shops' tabs. Replaced as a whole on update."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this entry was created."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this entry was last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*ShoppingCatalogItem) TableName() string {
	return "shopping_catalog_items"
}

// ShoppingNotStocked marks a catalogue item as not sold at a shop.
type ShoppingNotStocked struct { //nolint:revive // stutters, but matches the other table names.
	ID            int64 `xorm:"bigint autoincr not null unique pk"`
	CatalogItemID int64 `xorm:"bigint not null unique(item_shop)"`
	ShopID        int64 `xorm:"bigint not null unique(item_shop) index"`
}

// TableName sets the database table name.
func (*ShoppingNotStocked) TableName() string {
	return "shopping_not_stocked"
}

// nameKey normalises a name for matching: trimmed and lower-cased.
func nameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func getCatalogItem(s *xorm.Session, id int64) (*ShoppingCatalogItem, error) {
	c := &ShoppingCatalogItem{}
	exists, err := s.Where("id = ?", id).Get(c)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCatalogItemDoesNotExist{CatalogItemID: id}
	}
	return c, nil
}

func findCatalogItemByName(s *xorm.Session, name string) (*ShoppingCatalogItem, bool, error) {
	c := &ShoppingCatalogItem{}
	exists, err := s.Where("name_key = ?", nameKey(name)).Get(c)
	return c, exists, err
}

// useCatalogItem links an item to its catalogue entry by name, creating the
// entry if the name is new, and teaches the entry the item's category,
// quantity and unit.
//
// newUse is true when the item is being put on a list: then the item's blank
// fields are filled from last time and the use is counted for autocomplete.
// Edits pass false, so clearing a quantity on purpose isn't undone.
func useCatalogItem(s *xorm.Session, item *ShoppingItem, newUse bool) error {
	entry, exists, err := findCatalogItemByName(s, item.Name)
	if err != nil {
		return err
	}
	if !exists {
		entry = &ShoppingCatalogItem{Name: strings.TrimSpace(item.Name), NameKey: nameKey(item.Name)}
		if _, err := s.Insert(entry); err != nil {
			return err
		}
		if err := recordChange(s, entityCatalog, entry.ID, "", false); err != nil {
			return err
		}
	}

	// Blanks on a newly added item come from last time.
	if newUse {
		if item.CategoryID == 0 {
			item.CategoryID = entry.CategoryID
		}
		if item.Quantity == 0 && item.Unit == "" {
			item.Quantity, item.Unit = entry.Quantity, entry.Unit
		}
	}

	item.CatalogItemID = entry.ID

	// And this time is remembered for next time.
	learned := *entry
	if item.CategoryID != 0 {
		learned.CategoryID = item.CategoryID
	}
	learned.Quantity, learned.Unit = item.Quantity, item.Unit
	unchanged := learned.CategoryID == entry.CategoryID && learned.Quantity == entry.Quantity && learned.Unit == entry.Unit
	if !newUse && unchanged {
		// Nothing to write: a tick or a note edit leaves the catalogue alone.
		// Fewer writes matters on SQLite, which only allows one writer at a time.
		return nil
	}

	cols := []string{"category_id", "quantity", "unit"}
	if newUse {
		now := time.Now()
		learned.TimesAdded++
		learned.LastAdded = &now
		cols = append(cols, "times_added", "last_added")
	}
	if _, err = s.ID(entry.ID).Cols(cols...).Update(&learned); err != nil {
		return err
	}
	return recordChange(s, entityCatalog, entry.ID, "", false)
}

// catalogDetails holds the per-entry extras that items show: photo and
// not-stocked shops.
type catalogDetails struct {
	hasPhoto   bool
	notStocked []int64
}

// loadCatalogDetails fetches photo flags and not-stocked shops for many
// catalogue entries in two queries, keyed by catalogue id.
func loadCatalogDetails(s *xorm.Session, ids []int64) (map[int64]*catalogDetails, error) {
	details := make(map[int64]*catalogDetails, len(ids))
	for _, id := range ids {
		details[id] = &catalogDetails{notStocked: []int64{}}
	}
	if len(ids) == 0 {
		return details, nil
	}

	entries := []*ShoppingCatalogItem{}
	if err := s.In("id", ids).Cols("id", "photo_file_id").Find(&entries); err != nil {
		return nil, err
	}
	for _, e := range entries {
		details[e.ID].hasPhoto = e.PhotoFileID != 0
	}

	marks := []*ShoppingNotStocked{}
	if err := s.In("catalog_item_id", ids).OrderBy("shop_id ASC").Find(&marks); err != nil {
		return nil, err
	}
	for _, m := range marks {
		details[m.CatalogItemID].notStocked = append(details[m.CatalogItemID].notStocked, m.ShopID)
	}
	return details, nil
}

// fillDetails sets the derived fields on catalogue entries.
func fillDetails(s *xorm.Session, entries ...*ShoppingCatalogItem) error {
	ids := make([]int64, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	details, err := loadCatalogDetails(s, ids)
	if err != nil {
		return err
	}
	for _, e := range entries {
		e.HasPhoto = details[e.ID].hasPhoto
		e.NotStockedShopIDs = details[e.ID].notStocked
	}
	return nil
}

// setNotStocked replaces the entry's not-stocked shops with shopIDs, which
// must all exist.
func setNotStocked(s *xorm.Session, catalogItemID int64, shopIDs []int64) error {
	unique := make([]int64, 0, len(shopIDs))
	seen := map[int64]bool{}
	for _, id := range shopIDs {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	if len(unique) > 0 {
		count, err := s.In("id", unique).Count(&ShoppingShop{})
		if err != nil {
			return err
		}
		if count != int64(len(unique)) {
			return ErrShopDoesNotExist{}
		}
	}

	if _, err := s.Where("catalog_item_id = ?", catalogItemID).Delete(&ShoppingNotStocked{}); err != nil {
		return err
	}
	for _, shopID := range unique {
		if _, err := s.Insert(&ShoppingNotStocked{CatalogItemID: catalogItemID, ShopID: shopID}); err != nil {
			return err
		}
	}
	return recordChange(s, entityCatalog, catalogItemID, "", false)
}

// CanRead lets every household member read every catalogue entry.
func (c *ShoppingCatalogItem) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if err := requireMember(a); err != nil {
		return false, 0, err
	}
	stored, err := getCatalogItem(s, c.ID)
	if err != nil {
		return false, 0, err
	}
	*c = *stored
	return true, int(permissionAdmin), nil
}

// CanCreate lets members add catalogue entries.
func (c *ShoppingCatalogItem) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	return true, nil
}

// CanUpdate lets members change any entry.
func (c *ShoppingCatalogItem) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteCatalogItem(s, c.ID, a)
}

// CanDelete lets members remove any entry.
func (c *ShoppingCatalogItem) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteCatalogItem(s, c.ID, a)
}

func canWriteCatalogItem(s *xorm.Session, id int64, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getCatalogItem(s, id); err != nil {
		return false, err
	}
	return true, nil
}

// checkNameFree fails if another entry already has this name.
func checkNameFree(s *xorm.Session, name string, exceptID int64) error {
	other, exists, err := findCatalogItemByName(s, name)
	if err != nil {
		return err
	}
	if exists && other.ID != exceptID {
		return ErrCatalogItemExists{Name: other.Name}
	}
	return nil
}

// Create adds a catalogue entry directly (without putting it on a list).
func (c *ShoppingCatalogItem) Create(s *xorm.Session, _ web.Auth) error {
	if err := checkNameFree(s, c.Name, 0); err != nil {
		return err
	}
	if err := checkCategory(s, c.CategoryID); err != nil {
		return err
	}
	c.ID = 0
	c.Name = strings.TrimSpace(c.Name)
	c.NameKey = nameKey(c.Name)
	c.TimesAdded = 0
	c.LastAdded = nil
	c.PhotoFileID = 0
	if _, err := s.Insert(c); err != nil {
		return err
	}
	if err := setNotStocked(s, c.ID, c.NotStockedShopIDs); err != nil {
		return err
	}
	return fillDetails(s, c)
}

// ReadOne fills in photo and not-stocked details; CanRead loaded the row.
func (c *ShoppingCatalogItem) ReadOne(s *xorm.Session, _ web.Auth) error {
	return fillDetails(s, c)
}

// ReadAll returns catalogue entries, most often and most recently added
// first, optionally filtered by a name search.
func (c *ShoppingCatalogItem) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (any, int, int64, error) {
	if err := requireMember(a); err != nil {
		return nil, 0, 0, err
	}

	query := func() *xorm.Session {
		q := s.Table("shopping_catalog_items")
		if key := nameKey(search); key != "" {
			q = q.Where("name_key LIKE ?", "%"+key+"%")
		}
		return q
	}

	total, err := query().Count(&ShoppingCatalogItem{})
	if err != nil {
		return nil, 0, 0, err
	}

	entries := []*ShoppingCatalogItem{}
	// "last_added IS NULL" sorts never-added entries last on every database
	// (they disagree on where NULLs go in a DESC sort).
	err = paginate(query(), page, perPage).
		OrderBy("times_added DESC, last_added IS NULL, last_added DESC, name_key ASC").
		Find(&entries)
	if err != nil {
		return nil, 0, 0, err
	}
	if err := fillDetails(s, entries...); err != nil {
		return nil, 0, 0, err
	}
	return entries, len(entries), total, nil
}

// Update renames the entry or changes its category, last quantity/unit and
// not-stocked shops. Usage counts and the photo are not writable here.
func (c *ShoppingCatalogItem) Update(s *xorm.Session, _ web.Auth) error {
	if err := checkNameFree(s, c.Name, c.ID); err != nil {
		return err
	}
	if err := checkCategory(s, c.CategoryID); err != nil {
		return err
	}
	c.Name = strings.TrimSpace(c.Name)
	c.NameKey = nameKey(c.Name)
	if _, err := s.ID(c.ID).Cols("name", "name_key", "category_id", "quantity", "unit").Update(c); err != nil {
		return err
	}
	if err := recordChange(s, entityCatalog, c.ID, "", false); err != nil {
		return err
	}
	// A nil slice means the client didn't send the field (e.g. an old
	// client); leave the marks alone then. [] clears them.
	if c.NotStockedShopIDs != nil {
		if err := setNotStocked(s, c.ID, c.NotStockedShopIDs); err != nil {
			return err
		}
	}
	stored, err := getCatalogItem(s, c.ID)
	if err != nil {
		return err
	}
	*c = *stored
	return fillDetails(s, c)
}

// Delete removes the entry, its not-stocked marks and its photo. Items on
// lists keep their name; they just lose the link.
func (c *ShoppingCatalogItem) Delete(s *xorm.Session, _ web.Auth) error {
	stored, err := getCatalogItem(s, c.ID)
	if err != nil {
		return err
	}
	if err := deletePhotoFile(s, stored.PhotoFileID); err != nil {
		return err
	}
	if _, err := s.Where("catalog_item_id = ?", c.ID).Delete(&ShoppingNotStocked{}); err != nil {
		return err
	}
	items := []*ShoppingItem{}
	if err := s.Where("catalog_item_id = ?", c.ID).Cols("id", "uid").Find(&items); err != nil {
		return err
	}
	_, err = s.Table("shopping_items").Where("catalog_item_id = ?", c.ID).Update(map[string]any{"catalog_item_id": 0})
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := recordItemChange(s, item, false); err != nil {
			return err
		}
	}
	if _, err = s.ID(c.ID).Delete(&ShoppingCatalogItem{}); err != nil {
		return err
	}
	return recordChange(s, entityCatalog, c.ID, "", true)
}

// deletePhotoFile removes a stored photo; a file that's already gone is fine.
func deletePhotoFile(s *xorm.Session, fileID int64) error {
	if fileID == 0 {
		return nil
	}
	f := &files.File{ID: fileID}
	if err := f.Delete(s); err != nil && !files.IsErrFileDoesNotExist(err) {
		return err
	}
	return nil
}
