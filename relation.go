package graphor

import (
	"fmt"
	"log"
	"strings"
)

type Relation interface {
	Query
	Add(child Model, facets ...map[string]interface{})
	Remove(child Model)
	Clear()
	Set(child Model, facets ...map[string]interface{})
}

type relation struct {
	query
	Parent         Model
	RelationSchema RelationSchema
	FacetsFilter   []string
	SortedByFacet  bool
}

func BuildRelation(parent Model, rs RelationSchema) Relation {
	qRelation := `
	{
		q(func: uid(<#{uid}>)) {
			#{edge} #{sorting} #{facets} #{facets_filter} #{filter} #{take} { #{body} }	
		}
	}`

	q := build(qRelation, rs.SchemaFunc(), map[string]interface{}{
		"uid":  parent.GetUid(),
		"edge": rs.Edge,
	})

	r := &relation{
		query:          *q,
		Parent:         parent,
		RelationSchema: rs,
		FacetsFilter:   []string{},
	}

	return r
}

func (r *relation) SetSortOption(key string, order string) Query {
	r.SortKey = key
	r.SortOrder = order

	_, ok := r.RelationSchema.Facets[key]
	r.SortedByFacet = ok

	return r
}

func (r *relation) IsOrderAsc() bool {
	return r.query.IsOrderAsc()
}

func (r *relation) GetSortKey() string {
	return r.SortKey
}

func (r *relation) Take(count int) Query {
	r.query.Take(count)
	return r
}

func (r *relation) Where(field string, op string, value interface{}) Query {
	if facet, ok := r.RelationSchema.Facets[field]; ok { // facets
		r.FacetsFilter = append(r.FacetsFilter, fmt.Sprintf("%s(%s, %s)", op, facet.Edge, eval(value)))
	} else {
		r.query.Where(field, op, value)
	}

	return r
}

func (r *relation) Between(field string, left interface{}, right interface{}) Query {
	return r.Where(field, "ge", left).Where(field, "lt", right)
}

func (r *relation) Has(edge string) Query {
	r.query.Has(edge)
	return r
}

func (r *relation) HasNot(edge string) Query {
	r.query.HasNot(edge)
	return r
}

func (r *relation) Or(filters ...(func(q Query) Query)) Query {
	r.query.Or(filters...)
	return r
}

func (r *relation) Regex(field, regex string) Query {
	r.query.Regex(field, regex)
	return r
}

func (r *relation) Scope(filter func(q Query) Query) Query {
	r.query.Scope(filter)
	return r
}

func (r *relation) Identify(uids ...string) Query {
	r.query.Identify(uids...)
	return r
}

func (r *relation) Debug() Query {
	r.query.Debug()
	return r
}

func (r *relation) Execute() ([]interface{}, error) {
	facets := []string{}

	if r.SortedByFacet {
		facets = append(facets, fmt.Sprintf("order%s: %s", r.SortOrder, r.RelationSchema.Facets[r.SortKey].Edge))
		r.Args["sorting"] = ""
	} else {
		r.Args["sorting"] = fmt.Sprintf("(order%s: %s)", r.SortOrder, r.SortKey)
	}

	for name, f := range r.RelationSchema.Facets {
		facets = append(facets, fmt.Sprintf("%s: %s", name, f.Edge))
	}

	if len(facets) > 0 {
		r.Args["facets"] = "@facets(" + strings.Join(facets, ", ") + ")"
	} else {
		r.Args["facets"] = ""
	}

	if len(r.FacetsFilter) > 0 {
		r.query.Args["facets_filter"] = "@facets(" + strings.Join(r.FacetsFilter, " and ") + ")"
	} else {
		r.query.Args["facets_filter"] = ""
	}

	if r.TakeCount > 0 {
		r.Args["take"] = fmt.Sprintf("(first: %d)", r.TakeCount)
	}

	return r.query.Execute()
}

func (r *relation) First() (QueryData, error) {
	res, err := r.Take(1).Execute()
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	children := res[0].(map[string]interface{})[r.RelationSchema.Edge].([]interface{})
	if len(children) == 0 {
		return nil, nil
	}

	return r.Schema.Decode(children[0]), nil
}

func (r *relation) All() ([]QueryData, error) {
	res, err := r.Execute()
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return []QueryData{}, nil
	}

	children := res[0].(map[string]interface{})[r.RelationSchema.Edge].([]interface{})
	dataList := []QueryData{}
	for _, child := range children {
		dataList = append(dataList, r.Schema.Decode(child))
	}

	return dataList, nil
}

func (r *relation) Paging(since interface{}, until interface{}, count int) Query {
	index := r.SortKey

	if !isEmpty(since) {
		if r.IsOrderAsc() {
			r.Where(index, "ge", since)
		} else {
			r.Where(index, "le", since)
		}
	}

	if !isEmpty(until) {
		if r.IsOrderAsc() {
			r.Where(index, "lt", until)
		} else {
			r.Where(index, "gt", until)
		}
	}

	return r.Take(count)
}

func (r *relation) Exists() (bool, error) {
	dataList, err := r.Take(1).All()
	return len(dataList) > 0, err
}

func (r *relation) Count() (int, error) {
	return r.query.Count()
}

func (r *relation) add(child Model, facets ...map[string]interface{}) {
	if r.Parent == nil || r.Parent.isEmpty() {
		log.Print("Relation.Add failed: Parent is empty.")
		return
	}

	if child == nil || child.isEmpty() {
		log.Print("Relation.Add failed: Child is empty.")
		return
	}

	if IsReversed(r.RelationSchema.Edge) {
		log.Print("Relation.Add failed: Can't add to reversed edge.")
		return
	}

	fields := []string{}
	fields = append(fields, fmt.Sprintf(`"uid": %q`, child.GetUid()))

	if len(facets) > 0 {
		for name, value := range facets[0] {
			fields = append(fields, fmt.Sprintf(`"%s|%s": %s`, r.RelationSchema.Edge, r.RelationSchema.Facets[name].Edge, eval(value)))
		}
	}

	q := fmt.Sprintf(`{
		"uid": %q,
		%q: {
			%s
		}
	}`, r.Parent.GetUid(), r.RelationSchema.Edge, strings.Join(fields, ",\n"))

	db().Insert(q)
}

func (r *relation) Add(child Model, facets ...map[string]interface{}) {
	if !r.RelationSchema.HasMany {
		log.Print("Relation.Add failed: Don't use Relation.Add for 'hasOne' relation. Use Relation.Set instead.")
		return
	}

	r.add(child, facets...)
}

func (r *relation) Remove(child Model) {
	if r.Parent == nil || !r.Parent.isSaved() {
		log.Print("Relation.Remove failed: Parent is empty or not saved.")
		return
	}

	if child == nil || !child.isSaved() {
		log.Print("Relation.Remove failed: Child is empty or not saved.")
		return
	}

	if IsReversed(r.RelationSchema.Edge) {
		log.Print("Relation.Remove failed: Can't remove reversed edge.")
		return
	}

	if !r.RelationSchema.HasMany {
		log.Print("Relation.Remove failed: Don't use Relation.Remove for 'hasOne' relation. Use Relation.Clear instead.")
		return
	}

	q := fmt.Sprintf(`{
		"uid": %q,
		%q: {"uid": %q}
	}`, r.Parent.GetUid(), r.RelationSchema.Edge, child.GetUid())

	db().Delete(q)
}

func (r *relation) Clear() {
	if r.Parent == nil || !r.Parent.isSaved() {
		return
	}

	if IsReversed(r.RelationSchema.Edge) {
		log.Print("Relation.Clear failed: Can't clear reversed edge.")
	}

	q := fmt.Sprintf(`
		{
			"uid": %q,
			%q: null
		}
	`, r.Parent.GetUid(), r.RelationSchema.Edge)

	db().Delete(q)
}

func (r *relation) Set(child Model, facets ...map[string]interface{}) {
	if r.RelationSchema.HasMany {
		log.Print("Relation.Set failed: Don't use Relation.Set for 'hasMany' relation. Use Relation.Add instead.")
	}

	r.Clear()
	r.add(child, facets...)
}
