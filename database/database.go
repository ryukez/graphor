package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dgraph-io/dgo"
	"github.com/dgraph-io/dgo/protos/api"
	"github.com/nosukeru/graphor/errors"
	"google.golang.org/grpc"
)

type Database interface {
	Clear() error
	Migrate(body string) error
	Insert(q string)
	Delete(q string)
	InitMutation()
	RunMutation() (map[string]string, error)
	Query(q string) ([]interface{}, error)
}

type mutation struct {
	Insertions []string
	Deletions  []string
}

type database struct {
	Client   *dgo.Dgraph
	Mutation *mutation
}

func NewDatabase() (Database, error) {
	d, err := grpc.Dial("localhost:9080", grpc.WithInsecure())
	if err != nil {
		return nil, errors.New(errors.ConnectionRefused, err.Error())
	}

	c := dgo.NewDgraphClient(api.NewDgraphClient(d))
	m := new(mutation)

	return &database{c, m}, nil
}

func (db *database) Clear() error {
	ctx := context.Background()

	err := db.Client.Alter(ctx, &api.Operation{
		DropAll: true,
	})

	if err != nil {
		return errors.New(errors.DropDBFailed, err.Error())
	}
	return nil
}

func (db *database) Migrate(body string) error {
	ctx := context.Background()

	err := db.Client.Alter(ctx, &api.Operation{
		Schema: body,
	})

	if err != nil {
		return errors.New(errors.MigrationFailed, err.Error()).Add("migrationBody", body)
	}
	return nil
}

func (db *database) Insert(q string) {
	db.Mutation.Insertions = append(db.Mutation.Insertions, q)
}

func (db *database) Delete(q string) {
	db.Mutation.Deletions = append(db.Mutation.Deletions, q)
}

func (db *database) InitMutation() {
	db.Mutation = new(mutation)
}

func (db *database) RunMutation() (map[string]string, error) {
	ctx := context.Background()

	txn := db.Client.NewTxn()
	defer txn.Discard(ctx)

	// delete
	if len(db.Mutation.Deletions) > 0 {
		mu := new(api.Mutation)
		deletion := fmt.Sprintf("[%s]", strings.Join(db.Mutation.Deletions, ","))

		mu.DeleteJson = []byte(deletion)
		_, err := txn.Mutate(ctx, mu)

		if err != nil {
			return nil, errors.New(errors.InsertionFailed, err.Error()).Add("deletion", deletion)
		}
	}

	// set
	uids := map[string]string{}

	if len(db.Mutation.Insertions) > 0 {
		mu := new(api.Mutation)
		insertion := fmt.Sprintf("[%s]", strings.Join(db.Mutation.Insertions, ","))

		mu.SetJson = []byte(insertion)
		res, err := txn.Mutate(ctx, mu)

		if err != nil {
			return nil, errors.New(errors.DeletionFailed, err.Error()).Add("insertion", insertion)
		}

		uids = res.Uids
	}

	err := txn.Commit(ctx)
	if err != nil {
		return uids, errors.New(errors.MutationCommitFailed, err.Error())
	}
	return uids, nil
}

func (db *database) Query(q string) ([]interface{}, error) {
	ctx := context.Background()
	txn := db.Client.NewTxn()
	defer txn.Discard(ctx)

	res, err := txn.Query(ctx, q)
	if err != nil {
		return nil, errors.New(errors.QueryFailed, err.Error()).Add("q", q)
	}

	var r interface{}
	err = json.Unmarshal(res.Json, &r)

	if err != nil {
		return nil, errors.New(errors.UnmarshalizeFailed, err.Error()).Add("body", string(res.Json))
	}

	data := r.(map[string]interface{})["q"]
	if data == nil {
		return []interface{}{}, nil
	}

	results := data.([]interface{})

	// groupby
	if len(results) > 0 {
		if ar, ok := results[0].(map[string]interface{})["@groupby"]; ok {
			return ar.([]interface{}), nil
		}
	}

	return results, nil
}
