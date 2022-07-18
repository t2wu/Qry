package mdl

import (
	"reflect"
	"strings"

	"github.com/stoewer/go-strcase"
)

// GetTableNameFromType get table name from the mdl reflect.type
func GetTableNameFromType(typ reflect.Type) string {
	m := reflect.New(typ).Interface().(IModel)
	return GetTableNameFromIModel(m)
}

func GetModelTypeNameFromIModel(m IModel) string {
	return reflect.TypeOf(m).Elem().Name()
}

func GetModelTableNameInModelIfValid(modelObj IModel, field string) (string, error) {
	typ, err := GetModelFieldTypeInModelIfValid(modelObj, field)
	if err != nil {
		return "", err
	}
	return GetTableNameFromType(typ), nil
}

// GetTableNameFromIModel gets table name from an IModel
func GetTableNameFromIModel(imodel IModel) string {
	var tableName string
	if m, ok := imodel.(IHasTableName); ok {
		tableName = m.TableName()
	} else {
		tableName = reflect.TypeOf(imodel).String()
		// If it is something like "XXX", we only want the stuff ater "."
		if strings.Contains(tableName, ".") {
			tableName = strings.Split(tableName, ".")[1]
		}

		tableName = strcase.SnakeCase(tableName)
	}

	// If it's a pointer, get rid of "*"
	tableName = strings.TrimPrefix(tableName, "*")

	return tableName
}
