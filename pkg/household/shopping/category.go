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
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// ShoppingCategory groups items on a list (Fruit & Veg, Dairy, Frozen...).
// One category order is shared by every list and every shop.
type ShoppingCategory struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	ID       int64   `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this category."`
	Title    string  `xorm:"varchar(250) not null" json:"title" valid:"runelength(1|250)" minLength:"1" maxLength:"250" doc:"The category name, e.g. Dairy & Eggs."`
	Position float64 `xorm:"double not null default 0" json:"position" doc:"Order of the category's group on a list; lower comes first."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this category was created."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this category was last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*ShoppingCategory) TableName() string {
	return "shopping_categories"
}

func getCategoryByID(s *xorm.Session, id int64) (*ShoppingCategory, error) {
	c := &ShoppingCategory{}
	exists, err := s.Where("id = ?", id).Get(c)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCategoryDoesNotExist{CategoryID: id}
	}
	return c, nil
}

// CanRead lets every household member read every category.
func (c *ShoppingCategory) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if err := requireMember(a); err != nil {
		return false, 0, err
	}
	stored, err := getCategoryByID(s, c.ID)
	if err != nil {
		return false, 0, err
	}
	*c = *stored
	return true, int(permissionAdmin), nil
}

// CanCreate lets every household member add categories.
func (c *ShoppingCategory) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	return household.IsMember(a), nil
}

// CanUpdate lets every household member rename or reorder any category.
func (c *ShoppingCategory) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteCategory(s, c.ID, a)
}

// CanDelete lets every household member delete any category.
func (c *ShoppingCategory) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return canWriteCategory(s, c.ID, a)
}

func canWriteCategory(s *xorm.Session, id int64, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getCategoryByID(s, id); err != nil {
		return false, err
	}
	return true, nil
}

// Create adds a category, at the end unless a position is given.
func (c *ShoppingCategory) Create(s *xorm.Session, _ web.Auth) (err error) {
	c.ID = 0
	if c.Position == 0 {
		if c.Position, err = nextPosition(s, "shopping_categories"); err != nil {
			return err
		}
	}
	if _, err = s.Insert(c); err != nil {
		return err
	}
	return recordChange(s, entityCategory, c.ID, "", false)
}

// ReadOne has nothing to add: CanRead already loaded the row.
func (c *ShoppingCategory) ReadOne(_ *xorm.Session, _ web.Auth) error {
	return nil
}

// ReadAll returns every category in display order.
func (c *ShoppingCategory) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (any, int, int64, error) {
	if err := requireMember(a); err != nil {
		return nil, 0, 0, err
	}

	query := func() *xorm.Session {
		q := s.Table("shopping_categories")
		if search != "" {
			q = q.Where("title LIKE ?", "%"+search+"%")
		}
		return q
	}

	total, err := query().Count(&ShoppingCategory{})
	if err != nil {
		return nil, 0, 0, err
	}

	categories := []*ShoppingCategory{}
	err = paginate(query(), page, perPage).OrderBy("position ASC, id ASC").Find(&categories)
	if err != nil {
		return nil, 0, 0, err
	}
	return categories, len(categories), total, nil
}

// Update changes the title and position.
func (c *ShoppingCategory) Update(s *xorm.Session, _ web.Auth) error {
	if _, err := s.ID(c.ID).Cols("title", "position").Update(c); err != nil {
		return err
	}
	stored, err := getCategoryByID(s, c.ID)
	if err != nil {
		return err
	}
	*c = *stored
	return recordChange(s, entityCategory, c.ID, "", false)
}

// Delete removes the category. Its items and catalogue entries become
// uncategorised (category 0) rather than being deleted.
func (c *ShoppingCategory) Delete(s *xorm.Session, _ web.Auth) error {
	items := []*ShoppingItem{}
	if err := s.Where("category_id = ?", c.ID).Cols("id", "uid").Find(&items); err != nil {
		return err
	}
	var entries []int64
	if err := s.Table("shopping_catalog_items").Where("category_id = ?", c.ID).Cols("id").Find(&entries); err != nil {
		return err
	}
	for _, table := range []string{"shopping_items", "shopping_catalog_items"} {
		_, err := s.Table(table).
			Where("category_id = ?", c.ID).
			Update(map[string]any{"category_id": 0})
		if err != nil {
			return err
		}
	}
	if _, err := s.ID(c.ID).Delete(&ShoppingCategory{}); err != nil {
		return err
	}
	for _, item := range items {
		if err := recordItemChange(s, item, false); err != nil {
			return err
		}
	}
	if err := recordChanges(s, entityCatalog, entries); err != nil {
		return err
	}
	return recordChange(s, entityCategory, c.ID, "", true)
}
