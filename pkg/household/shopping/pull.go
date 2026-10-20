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

// conflictKeep is how far back a full sync's conflict log goes.
const conflictKeep = 30 * 24 * time.Hour

// ShoppingSyncResult is what a phone gets back from a sync: everything that
// changed since its cursor, or everything when Full is set.
type ShoppingSyncResult struct { //nolint:revive // stutters, but the type name is the OpenAPI schema name.
	Cursor int64 `json:"cursor" doc:"Send this as cursor next time."`
	Full   bool  `json:"full" doc:"True when this is a complete copy: replace everything stored locally. Otherwise only changes are included."`

	Lists              []*ShoppingList        `json:"lists" doc:"Lists (with their shops) that are new or changed."`
	DeletedListIDs     []int64                `json:"deleted_list_ids"`
	Categories         []*ShoppingCategory    `json:"categories"`
	DeletedCategoryIDs []int64                `json:"deleted_category_ids"`
	Catalog            []*ShoppingCatalogItem `json:"catalog"`
	DeletedCatalogIDs  []int64                `json:"deleted_catalog_ids"`
	Items              []*ShoppingItem        `json:"items"`
	DeletedItemUIDs    []string               `json:"deleted_item_uids"`
	Conflicts          []*ShoppingConflict    `json:"conflicts" doc:"New entries for the conflict log."`

	Rejected []*ShoppingSyncRejection `json:"rejected" doc:"Ops from this request that were dropped."`
}

func newSyncResult(cursor int64, full bool) *ShoppingSyncResult {
	return &ShoppingSyncResult{
		Cursor:             cursor,
		Full:               full,
		Lists:              []*ShoppingList{},
		DeletedListIDs:     []int64{},
		Categories:         []*ShoppingCategory{},
		DeletedCategoryIDs: []int64{},
		Catalog:            []*ShoppingCatalogItem{},
		DeletedCatalogIDs:  []int64{},
		Items:              []*ShoppingItem{},
		DeletedItemUIDs:    []string{},
		Conflicts:          []*ShoppingConflict{},
		Rejected:           []*ShoppingSyncRejection{},
	}
}

// PullChanges returns what changed after cursor. A cursor of 0, or one the
// server doesn't know (its database was restored from an older backup, say),
// gets a full copy.
func PullChanges(s *xorm.Session, a web.Auth, cursor int64) (*ShoppingSyncResult, error) {
	if err := requireMember(a); err != nil {
		return nil, err
	}
	latest, err := latestChangeID(s)
	if err != nil {
		return nil, err
	}
	if cursor <= 0 || cursor > latest {
		return fullCopy(s, latest)
	}

	result := newSyncResult(latest, false)
	changes := []*ShoppingChange{}
	if err := s.Where("id > ? AND id <= ?", cursor, latest).Find(&changes); err != nil {
		return nil, err
	}

	// Sort the changes into "load these" and "these are gone", per entity.
	changed := map[string][]int64{}
	for _, c := range changes {
		switch {
		case !c.Deleted:
			changed[c.Entity] = append(changed[c.Entity], c.EntityID)
		case c.Entity == entityList:
			result.DeletedListIDs = append(result.DeletedListIDs, c.EntityID)
		case c.Entity == entityCategory:
			result.DeletedCategoryIDs = append(result.DeletedCategoryIDs, c.EntityID)
		case c.Entity == entityCatalog:
			result.DeletedCatalogIDs = append(result.DeletedCatalogIDs, c.EntityID)
		case c.Entity == entityItem:
			result.DeletedItemUIDs = append(result.DeletedItemUIDs, c.UID)
		}
	}

	if err := result.load(s, func(entity string) rowFilter {
		ids := changed[entity]
		if len(ids) == 0 {
			return nil
		}
		return func(q *xorm.Session) *xorm.Session { return q.In("id", ids) }
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func fullCopy(s *xorm.Session, latest int64) (*ShoppingSyncResult, error) {
	result := newSyncResult(latest, true)
	since := time.Now().Add(-conflictKeep)
	err := result.load(s, func(entity string) rowFilter {
		if entity == entityConflict {
			return func(q *xorm.Session) *xorm.Session { return q.Where("created > ?", since) }
		}
		return func(q *xorm.Session) *xorm.Session { return q }
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// rowFilter narrows a query to the rows a sync sends.
type rowFilter func(q *xorm.Session) *xorm.Session

// load fills the result with the rows filterFor selects for each entity. A
// nil filter means "none of this entity". (The query is only started once we
// know it's wanted: xorm keeps conditions on the session until a Find.)
func (r *ShoppingSyncResult) load(s *xorm.Session, filterFor func(entity string) rowFilter) error {
	if f := filterFor(entityList); f != nil {
		if err := f(s.OrderBy("position ASC, id ASC")).Find(&r.Lists); err != nil {
			return err
		}
		if err := fillShops(s, r.Lists...); err != nil {
			return err
		}
	}
	if f := filterFor(entityCategory); f != nil {
		if err := f(s.OrderBy("position ASC, id ASC")).Find(&r.Categories); err != nil {
			return err
		}
	}
	if f := filterFor(entityCatalog); f != nil {
		if err := f(s.OrderBy("id ASC")).Find(&r.Catalog); err != nil {
			return err
		}
		if err := fillDetails(s, r.Catalog...); err != nil {
			return err
		}
	}
	if f := filterFor(entityItem); f != nil {
		if err := f(s.OrderBy("id ASC")).Find(&r.Items); err != nil {
			return err
		}
		if err := fillItemDetails(s, r.Items...); err != nil {
			return err
		}
	}
	if f := filterFor(entityConflict); f != nil {
		if err := f(s.OrderBy("id ASC")).Find(&r.Conflicts); err != nil {
			return err
		}
		if err := fillConflictUsers(s, r.Conflicts); err != nil {
			return err
		}
	}
	return nil
}
