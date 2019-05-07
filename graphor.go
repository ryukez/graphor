package graphor

import (
	"fmt"
	"strings"

	"github.com/nosukeru/graphor/auth"
	"github.com/nosukeru/graphor/database"
	"github.com/nosukeru/graphor/errors"
	"github.com/nosukeru/graphor/timestamp"
)

type graphorInterface interface {
	Auth() auth.Auth
	DB() database.Database
	Index() int
	Save(model Model, schema Schema)
	Delete(model Model)
	HardDelete(model Model)
	Mutate(execute func() error) error
	ClearDatabase() error
	MigrateDatabase(body string) error
	ReverseEdge(edge string) string
	IsReversed(edge string) bool
	BaseMigrations(schemaList []Schema) string
}

type graphor struct {
	Mutates    []Model
	IndexCount int
	Database   database.Database
	_Auth      auth.Auth
}

func newGraphor() (graphorInterface, error) {
	db, err := database.NewDatabase()
	auth := auth.NewAuth()

	return &graphor{[]Model{}, 0, db, auth}, err
}

func (g *graphor) Auth() auth.Auth {
	return g._Auth
}

func (g *graphor) DB() database.Database {
	return g.Database
}

func (g *graphor) Index() int {
	g.IndexCount = (g.IndexCount + 1) % 10e8
	return g.IndexCount
}

func (g *graphor) Save(model Model, schema Schema) {
	if model == nil {
		return
	}

	if model.isEmpty() {
		model.SetUid(fmt.Sprintf("_:model%d", g.Index()))
		g.Mutates = append(g.Mutates, model)
		model.setCreatedAt(timestamp.Now())
	}

	model.setUpdatedAt(timestamp.Now())

	all := map[string]interface{}{}
	cast(model, &all)

	// omit non-fields
	partial := map[string]interface{}{}
	fields := schema.Fields

	for _, field := range fields {
		if v, ok := all[field]; ok {
			partial[field] = v
		}
	}
	partial["uid"] = model.GetUid()
	partial["tag"] = schema.Tag
	partial["updated_at"] = model.GetUpdatedAt()

	if model.GetCreatedAt() > 0 {
		partial["created_at"] = model.GetCreatedAt()
	}

	if model.GetDeletedAt() > 0 {
		partial["deleted_at"] = model.GetDeletedAt()
	}

	q := toJSON(partial)
	g.Database.Insert(q)
}

func (g *graphor) Delete(model Model) {
	if model == nil {
		return
	}

	model.setDeletedAt(timestamp.Now())

	partial := map[string]interface{}{
		"uid":        model.GetUid(),
		"deleted_at": model.GetDeletedAt(),
	}

	q := toJSON(partial)
	g.Database.Insert(q)
}

func (g *graphor) HardDelete(model Model) {
	if model == nil || !model.isSaved() {
		return
	}

	q := fmt.Sprintf(`{"uid": %q}`, model.GetUid())
	g.Database.Delete(q)
}

func (g *graphor) Mutate(execute func() error) error {
	g.Mutates = []Model{}
	g.Database.InitMutation()

	err := execute()
	if err != nil {
		return err
	}

	res, err := g.Database.RunMutation()
	if err != nil {
		return err
	}

	for _, model := range g.Mutates {
		if model.isNew() {
			uid, ok := res[model.GetUid()[2:]]
			if !ok {
				return errors.New(errors.NoUidReturned, "Mutate failed: No uid returned.")
			}

			model.SetUid(uid)
		}
	}

	return nil
}

func (g *graphor) ClearDatabase() error {
	return g.Database.Clear()
}

func (g *graphor) MigrateDatabase(body string) error {
	return g.Database.Migrate(body)
}

func (g *graphor) ReverseEdge(edge string) string {
	if edge[0] == '~' {
		return edge[1:]
	}
	return "~" + edge
}

func (g *graphor) IsReversed(edge string) bool {
	return edge[0] == '~'
}

func (g *graphor) BaseMigrations(schemaList []Schema) string {
	// Edges
	hasReverse := map[string]bool{}
	for _, schema := range schemaList {
		edges := []string{}

		for _, b := range schema.Booleans {
			edges = append(edges, b.Edge)
		}

		for _, r := range schema.Relations {
			edges = append(edges, r.Edge)
		}

		for _, edge := range edges {
			if g.IsReversed(edge) {
				hasReverse[g.ReverseEdge(edge)] = true
			} else {
				_, ok := hasReverse[edge]
				if !ok {
					hasReverse[edge] = false
				}
			}
		}
	}

	edges := []string{}
	for name, r := range hasReverse {
		reverse := ""
		if r {
			reverse = "@reverse"
		}
		edges = append(edges, fmt.Sprintf("%s: uid %s .", name, reverse))
	}

	migrationBody := strings.Join(edges, "\n")

	// Model
	migrationBody += `
		tag: int @index(int) .
		created_at: int @index(int) .
		updated_at: int @index(int) .
		deleted_at: int .
	`

	return migrationBody
}
