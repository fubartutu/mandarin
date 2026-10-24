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

package recipes

import (
	"strings"
	"time"
	"unicode/utf8"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/household/menu"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// Limits that the API schema can't express on nested values.
const (
	maxIngredients   = 300
	maxSteps         = 200
	maxTags          = 20
	maxLineLength    = 500
	maxStepLength    = 5000
	maxTagLength     = 50
	maxGroupLength   = 250
	maxImages        = 4
	imageMaxEdge     = 1600 // big enough for a tablet on the kitchen bench
	importedImageMax = 15 << 20
)

// Recipe is one recipe in the household's recipe box.
//
// Ingredients, steps, tags and image ids are stored as JSON in one column
// each (the `json` xorm tag): they're always read and written with the
// recipe, never searched on their own, so separate tables would add joins
// for nothing.
type Recipe struct {
	ID          int64  `xorm:"bigint autoincr not null unique pk" json:"id" readOnly:"true" doc:"The unique, numeric id of this recipe."`
	Title       string `xorm:"varchar(250) not null" json:"title" valid:"runelength(1|250)" minLength:"1" maxLength:"250" doc:"The recipe's name."`
	SourceURL   string `xorm:"'source_url' varchar(2000) not null default ''" json:"source_url" maxLength:"2000" doc:"Where the recipe came from; empty for one typed in."`
	Description string `xorm:"text null" json:"description" maxLength:"5000" doc:"A short introduction."`

	Servings     float64 `xorm:"double not null default 0" json:"servings" minimum:"0" doc:"How many people the amounts are for; 0 when unknown. Used to scale ingredients."`
	Yields       string  `xorm:"varchar(100) not null default ''" json:"yields" maxLength:"100" doc:"What the recipe makes, as the source puts it, e.g. 16 cookies."`
	PrepMinutes  int     `xorm:"int not null default 0" json:"prep_minutes" minimum:"0"`
	CookMinutes  int     `xorm:"int not null default 0" json:"cook_minutes" minimum:"0"`
	TotalMinutes int     `xorm:"int not null default 0" json:"total_minutes" minimum:"0"`

	Ingredients []*RecipeIngredient `xorm:"text null json" json:"ingredients" doc:"In order. Each line's amount and name are worked out by the server from its text."`
	Steps       []string            `xorm:"text null json" json:"steps" doc:"The method, one step per entry, in order."`
	Tags        []string            `xorm:"text null json" json:"tags" doc:"E.g. Mains, Italian."`
	Notes       string              `xorm:"text null" json:"notes" maxLength:"10000" doc:"The household's own notes, e.g. what to change next time."`

	ImageIDs  []int64  `xorm:"'image_ids' text null json" json:"image_ids" readOnly:"true" doc:"The recipe's pictures, first is the main one. Manage them under /recipes/{id}/images."`
	ImageURLs []string `xorm:"-" json:"image_urls,omitempty" readOnly:"true" doc:"Import previews only: pictures found on the page (or paperless:documents/{id}/image for a scan's page). After saving, fetch the wanted ones with POST /recipes/{id}/images/fetch."`

	// Import previews only: how the recipe was read, so the app can say how
	// much to trust it and offer the AI.
	ImportMethod   string `xorm:"-" json:"import_method,omitempty" readOnly:"true" doc:"Import previews only: schema (the page's recipe data), rules (simple rules on a scan's text) or ai."`
	ImportComplete bool   `xorm:"-" json:"import_complete,omitempty" readOnly:"true" doc:"Import previews only: false when the rules found no whole recipe and it needs checking closely."`
	AIAvailable    bool   `xorm:"-" json:"ai_available,omitempty" readOnly:"true" doc:"Import previews only: whether the scraper has an AI model set up."`
	ImportNote     string `xorm:"-" json:"import_note,omitempty" readOnly:"true" doc:"Import previews only: why the AI wasn't used when the rules fell short, e.g. a refused API key."`

	CreatedByID int64      `xorm:"bigint not null default 0" json:"-"`
	CreatedBy   *user.User `xorm:"-" json:"created_by" readOnly:"true" doc:"The member who added this recipe."`

	Created time.Time `xorm:"created not null" json:"created" readOnly:"true" doc:"When this recipe was added."`
	Updated time.Time `xorm:"updated not null" json:"updated" readOnly:"true" doc:"When this recipe was last changed."`

	web.CRUDable    `xorm:"-" json:"-"`
	web.Permissions `xorm:"-" json:"-"`
}

// TableName sets the database table name.
func (*Recipe) TableName() string {
	return "recipes"
}

func getRecipe(s *xorm.Session, id int64) (*Recipe, error) {
	r := &Recipe{}
	exists, err := s.Where("id = ?", id).Get(r)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrRecipeDoesNotExist{RecipeID: id}
	}
	r.emptyNotNull()
	return r, nil
}

// emptyNotNull turns missing JSON columns into empty lists, so the API
// always sends [] rather than null.
func (r *Recipe) emptyNotNull() {
	if r.Ingredients == nil {
		r.Ingredients = []*RecipeIngredient{}
	}
	if r.Steps == nil {
		r.Steps = []string{}
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
	if r.ImageIDs == nil {
		r.ImageIDs = []int64{}
	}
}

// CanRead lets every household member read every recipe, with admin permission.
func (r *Recipe) CanRead(s *xorm.Session, a web.Auth) (bool, int, error) {
	if err := requireMember(a); err != nil {
		return false, 0, err
	}
	stored, err := getRecipe(s, r.ID)
	if err != nil {
		return false, 0, err
	}
	*r = *stored
	return true, int(permissionAdmin), nil
}

// CanCreate lets every household member add recipes.
func (r *Recipe) CanCreate(_ *xorm.Session, a web.Auth) (bool, error) {
	return household.IsMember(a), nil
}

// CanUpdate lets every household member change any recipe.
func (r *Recipe) CanUpdate(s *xorm.Session, a web.Auth) (bool, error) {
	return canWrite(s, r.ID, a)
}

// CanDelete lets every household member delete any recipe.
func (r *Recipe) CanDelete(s *xorm.Session, a web.Auth) (bool, error) {
	return canWrite(s, r.ID, a)
}

func canWrite(s *xorm.Session, id int64, a web.Auth) (bool, error) {
	if err := requireMember(a); err != nil {
		return false, err
	}
	if _, err := getRecipe(s, id); err != nil {
		return false, err
	}
	return true, nil
}

// normalize trims the recipe, drops blank lines, checks the limits, and
// gives ingredients that haven't been parsed (by the scraper, see
// ParseIngredients) an amount and name from the simple parser.
func (r *Recipe) normalize() error {
	r.Title = strings.TrimSpace(r.Title)
	if n := utf8.RuneCountInString(r.Title); n < 1 || n > 250 {
		return ErrInvalidRecipe{Reason: "The title must be 1–250 characters."}
	}
	r.SourceURL = strings.TrimSpace(r.SourceURL)
	r.Yields = strings.TrimSpace(r.Yields)

	ingredients := make([]*RecipeIngredient, 0, len(r.Ingredients))
	for _, ing := range r.Ingredients {
		if ing == nil {
			continue
		}
		ing.Text = strings.Join(strings.Fields(ing.Text), " ")
		ing.Group = strings.TrimSpace(ing.Group)
		if ing.Text == "" {
			continue
		}
		if utf8.RuneCountInString(ing.Text) > maxLineLength || utf8.RuneCountInString(ing.Group) > maxGroupLength {
			return ErrInvalidRecipe{Reason: "An ingredient line is too long."}
		}
		if ing.Name == "" {
			ing.parseSimply()
		}
		ingredients = append(ingredients, ing)
	}
	if len(ingredients) > maxIngredients {
		return ErrInvalidRecipe{Reason: "A recipe can have at most 300 ingredients."}
	}
	r.Ingredients = ingredients

	r.Steps = trimmed(r.Steps)
	if len(r.Steps) > maxSteps {
		return ErrInvalidRecipe{Reason: "A recipe can have at most 200 steps."}
	}
	for _, step := range r.Steps {
		if utf8.RuneCountInString(step) > maxStepLength {
			return ErrInvalidRecipe{Reason: "A step is too long."}
		}
	}

	r.Tags = uniqueTags(r.Tags)
	if len(r.Tags) > maxTags {
		return ErrInvalidRecipe{Reason: "A recipe can have at most 20 tags."}
	}
	for _, tag := range r.Tags {
		if utf8.RuneCountInString(tag) > maxTagLength {
			return ErrInvalidRecipe{Reason: "A tag is too long."}
		}
	}
	r.emptyNotNull()
	return nil
}

// trimmed returns the non-blank entries, trimmed.
func trimmed(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// uniqueTags trims tags and drops repeats, ignoring case.
func uniqueTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range trimmed(tags) {
		key := strings.ToLower(tag)
		if !seen[key] {
			seen[key] = true
			out = append(out, tag)
		}
	}
	return out
}

// Create adds a recipe; the caller becomes its creator. Images are added
// afterwards, through the image endpoints.
func (r *Recipe) Create(s *xorm.Session, a web.Auth) error {
	u, err := user.GetFromAuth(a)
	if err != nil {
		return err
	}
	if err := r.normalize(); err != nil {
		return err
	}
	r.ID = 0
	r.CreatedByID = u.ID
	r.CreatedBy = u
	r.ImageIDs = []int64{}
	r.ImageURLs = nil
	_, err = s.Insert(r)
	return err
}

// ReadOne fills in the creator; CanRead already loaded the row.
func (r *Recipe) ReadOne(s *xorm.Session, _ web.Auth) error {
	return fillCreators(s, r)
}

func fillCreators(s *xorm.Session, recipes ...*Recipe) error {
	ids := []int64{}
	for _, r := range recipes {
		if r.CreatedByID > 0 {
			ids = append(ids, r.CreatedByID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	users, err := user.GetUsersByIDs(s, ids)
	if err != nil {
		return err
	}
	for _, r := range recipes {
		r.CreatedBy = users[r.CreatedByID]
	}
	return nil
}

// ReadAll returns the household's recipes by title. The search matches the
// title, tags and ingredients, so "chicken" finds every chicken recipe.
func (r *Recipe) ReadAll(s *xorm.Session, a web.Auth, search string, page int, perPage int) (any, int, int64, error) {
	if err := requireMember(a); err != nil {
		return nil, 0, 0, err
	}
	cond := builder.NewCond()
	if search = strings.TrimSpace(search); search != "" {
		cond = builder.Or(db.ILIKE("title", search), db.ILIKE("tags", search), db.ILIKE("ingredients", search))
	}

	total, err := s.Where(cond).Count(&Recipe{})
	if err != nil {
		return nil, 0, 0, err
	}
	recipes := []*Recipe{}
	q := s.Where(cond).OrderBy("title ASC, id ASC")
	if limit, start := limitFromPage(page, perPage); limit > 0 {
		q = q.Limit(limit, start)
	}
	if err := q.Find(&recipes); err != nil {
		return nil, 0, 0, err
	}
	for _, recipe := range recipes {
		recipe.emptyNotNull()
	}
	if err := fillCreators(s, recipes...); err != nil {
		return nil, 0, 0, err
	}
	return recipes, len(recipes), total, nil
}

// Update replaces everything but the images and creator.
func (r *Recipe) Update(s *xorm.Session, a web.Auth) error {
	if err := r.normalize(); err != nil {
		return err
	}
	_, err := s.ID(r.ID).
		Cols("title", "source_url", "description", "servings", "yields", "prep_minutes", "cook_minutes",
			"total_minutes", "ingredients", "steps", "tags", "notes").
		Update(r)
	if err != nil {
		return err
	}
	stored, err := getRecipe(s, r.ID)
	if err != nil {
		return err
	}
	*r = *stored
	return r.ReadOne(s, a)
}

// Delete removes the recipe and its images. Menu entries planned with it
// become notes with its title, in the same transaction.
func (r *Recipe) Delete(s *xorm.Session, _ web.Auth) error {
	stored, err := getRecipe(s, r.ID)
	if err != nil {
		return err
	}
	if err := menu.RecipeDeleted(s, stored.ID, stored.Title); err != nil {
		return err
	}
	for _, id := range stored.ImageIDs {
		if err := deleteFile(s, id); err != nil {
			return err
		}
	}
	_, err = s.ID(r.ID).Delete(&Recipe{})
	return err
}

func deleteFile(s *xorm.Session, fileID int64) error {
	f := &files.File{ID: fileID}
	if err := f.Delete(s); err != nil && !files.IsErrFileDoesNotExist(err) {
		return err
	}
	return nil
}
