package graphor

import (
	"github.com/nosukeru/graphor/auth"
	"github.com/nosukeru/graphor/database"
)

var __graphor graphorInterface

func InitializeGraphor() error {
	var err error
	__graphor, err = newGraphor()
	return err
}

func Auth() auth.Auth {
	return __graphor.Auth()
}

func db() database.Database {
	return __graphor.DB()
}

func Save(model Model, schema Schema) {
	__graphor.Save(model, schema)
}

func Delete(model Model) {
	__graphor.Delete(model)
}

func HardDelete(model Model) {
	__graphor.HardDelete(model)
}

func Mutate(execute func() error) error {
	return __graphor.Mutate(execute)
}

func ClearDatabase() error {
	return __graphor.ClearDatabase()
}

func MigrateDatabase(body string) error {
	return __graphor.MigrateDatabase(body)
}

func ReverseEdge(edge string) string {
	return __graphor.ReverseEdge(edge)
}

func IsReversed(edge string) bool {
	return __graphor.IsReversed(edge)
}

func BaseMigrations(schemaList []Schema) string {
	return __graphor.BaseMigrations(schemaList)
}

func DecodeString(x interface{}) string {
	return decodeString(x)
}

func DecodeInt(x interface{}) int {
	return decodeInt(x)
}
