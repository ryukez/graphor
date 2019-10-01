package graphor

import (
	"fmt"
	"strings"
)

type QueryData map[string]interface{}

type Query interface {
	SetSortOption(key string, order string) Query
	IsOrderAsc() bool
	GetSortKey() string
	Take(take int) Query
	Where(field string, op string, value interface{}) Query
	Between(field string, left interface{}, right interface{}) Query
	Has(edge string) Query
	HasNot(edge string) Query
	Or(filters ...(func(q Query) Query)) Query
	Regex(field, regex string) Query
	Scope(filter func(q Query) Query) Query
	Identify(uids ...string) Query
	Debug() Query
	Execute() ([]interface{}, error)
	First() (QueryData, error)
	All() ([]QueryData, error)
	Get(interface{}) error
	Paging(since interface{}, until interface{}, count int) Query
	Exists() (bool, error)
	Count() (int, error)
}

type query struct {
	Base           string
	Args           map[string]interface{}
	Filters        []string
	SortKey        string
	SortOrder      string
	TakeCount      int
	OnlyNotDeleted bool
	IsDebug        bool
	Schema         Schema
}

func build(qStr string, schema Schema, args map[string]interface{}) *query {
	q := new(query)
	q.Base = qStr
	q.Args = args
	q.Filters = []string{}
	q.SortKey = "created_at"
	q.SortOrder = "desc"
	q.TakeCount = 0
	q.Schema = schema

	return q
}

func BuildRawQuery(qStr string, schema Schema, args map[string]interface{}) Query {
	return build(qStr, schema, args)
}

func BuildQuery(schema Schema) Query {
	qAll := `
	{
		q(func: eq(tag, #{tag}), #{sorting}#{take}) #{filter} { #{body} }
	}`

	return build(qAll, schema, map[string]interface{}{})
}

func (q *query) SetSortOption(key string, order string) Query {
	q.SortKey = key
	q.SortOrder = order
	return q
}

func (q *query) IsOrderAsc() bool {
	return q.SortOrder == "asc"
}

func (q *query) GetSortKey() string {
	return q.SortKey
}

func (q *query) Take(count int) Query {
	q.TakeCount = count
	return q
}

func (q *query) Where(field string, op string, value interface{}) Query {
	q.Filters = append(q.Filters, fmt.Sprintf("%s(%s, %s)", op, field, eval(value)))
	return q
}

func (q *query) Between(field string, left interface{}, right interface{}) Query {
	return q.Where(field, "ge", left).Where(field, "lt", right)
}

func (q *query) Has(edge string) Query {
	q.Filters = append(q.Filters, fmt.Sprintf("has(%s)", edge))
	return q
}

func (q *query) HasNot(edge string) Query {
	q.Filters = append(q.Filters, fmt.Sprintf("not has(%s)", edge))
	return q
}

func (q *query) Or(filters ...(func(q Query) Query)) Query {
	conditions := []string{}

	for _, filter := range filters {
		qf := filter(&query{
			Filters: []string{},
		}).(*query)

		conditions = append(conditions, "("+strings.Join(qf.Filters, " and ")+")")
	}

	q.Filters = append(q.Filters, "("+strings.Join(conditions, " or ")+")")
	return q
}

func (q *query) Regex(field, regex string) Query {
	q.Filters = append(q.Filters, fmt.Sprintf("regexp(%s, %s)", field, regex))
	return q
}

func (q *query) Scope(filter func(q Query) Query) Query {
	return filter(q)
}

func (q *query) Identify(uids ...string) Query {
	if len(uids) == 0 || !isValidUid(uids...) {
		uids = []string{"0x0"} // dummy uid
	}

	q.Filters = append(q.Filters, fmt.Sprintf("uid(<%s>)", strings.Join(uids, ">, <")))
	return q
}

func (q *query) Debug() Query {
	q.IsDebug = true
	return q
}

func (q *query) generate() string {
	query := q.Base
	for name, value := range q.Args {
		// eval
		s := ""
		switch v := value.(type) {
		case int:
			s = fmt.Sprintf("%d", v)
		case bool:
			s = fmt.Sprintf("%t", v)
		case string:
			s = v
		}

		query = strings.Replace(query, fmt.Sprintf("#{%s}", name), s, -1)
	}

	if q.IsDebug {
		print(query)
	}

	return query
}

func (q *query) Execute() ([]interface{}, error) {
	filters := append(q.Filters, "not has(deleted_at)")
	filter := fmt.Sprintf("@filter(%s)", strings.Join(filters, " and "))

	args := q.Args
	args["tag"] = q.Schema.Tag

	if !keyExists(args, "sorting") {
		args["sorting"] = fmt.Sprintf("order%s: %s", q.SortOrder, q.SortKey)
	}

	if !keyExists(args, "take") {
		if q.TakeCount > 0 {
			args["take"] = fmt.Sprintf(", first: %d", q.TakeCount)
		} else {
			args["take"] = ""
		}
	}
	args["filter"] = filter
	args["body"] = q.Schema.Build()

	return db().Query(q.generate())
}

func (q *query) First() (QueryData, error) {
	res, err := q.Take(1).Execute()
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return q.Schema.Decode(res[0]), nil
}

func (q *query) All() ([]QueryData, error) {
	res, err := q.Execute()
	if err != nil {
		return nil, err
	}

	dataList := []QueryData{}
	for _, obj := range res {
		dataList = append(dataList, q.Schema.Decode(obj))
	}

	return dataList, nil
}

func (q *query) Get(x interface{}) error {
	body, err := q.Execute()
	if err != nil {
		return err
	}

	cast(body, x)
	return nil
}

func (q *query) Paging(since interface{}, until interface{}, count int) Query {
	index := q.SortKey

	if !isEmpty(since) {
		if q.IsOrderAsc() {
			q.Where(index, "ge", since)
		} else {
			q.Where(index, "le", since)
		}
	}

	if !isEmpty(until) {
		if q.IsOrderAsc() {
			q.Where(index, "lt", until)
		} else {
			q.Where(index, "gt", until)
		}
	}

	return q.Take(count)
}

func (q *query) Exists() (bool, error) {
	data, err := q.Take(1).All()
	return len(data) > 0, err
}

func (q *query) Count() (int, error) {
	q.Schema = CountSchema(q.Schema.Tag)

	type Data struct {
		Count int
	}

	data := []Data{}
	err := q.Get(&data)
	if err != nil {
		return 0, err
	}

	return data[0].Count, err
}
