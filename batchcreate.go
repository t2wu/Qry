package qry

import (
	"reflect"

	"github.com/t2wu/qry/mdl"
	"github.com/t2wu/qry/qtag"
)

type BatchCreateData struct {
	toProcess map[string][]mdl.IModel // key table name, values imodels
}

func gatherModelToCreate(v reflect.Value, data *BatchCreateData) error {
	for i := 0; i < v.NumField(); i++ {
		t := qtag.GetQryTag(v.Type().Field(i).Tag)

		if t == qtag.QryTagPeg {
			switch v.Field(i).Kind() {
			case reflect.Struct:
				m := v.Field(i).Addr().Interface().(mdl.IModel)
				if m.GetID() != nil { // could be embedded struct that never get initialiezd
					fieldTableName := mdl.GetTableNameFromIModel(m)
					if _, ok := data.toProcess[fieldTableName]; ok {
						ms := data.toProcess[fieldTableName]
						ms = append(ms, m)
						data.toProcess[fieldTableName] = ms
					} else {
						arr := make([]mdl.IModel, 1)
						arr[0] = m
						data.toProcess[fieldTableName] = arr
					}

					// Traverse into it
					if err := gatherModelToCreate(v.Field(i), data); err != nil {
						return err
					}
				}
			case reflect.Slice:
				typ := v.Type().Field(i).Type.Elem()
				m, _ := reflect.New(typ).Interface().(mdl.IModel)
				fieldTableName := mdl.GetTableNameFromIModel(m)
				for j := 0; j < v.Field(i).Len(); j++ {
					imodel := v.Field(i).Index(j).Addr().Interface().(mdl.IModel)
					if _, ok := data.toProcess[fieldTableName]; ok {
						ms := data.toProcess[fieldTableName]
						ms = append(ms, imodel)
						data.toProcess[fieldTableName] = ms
					} else {
						data.toProcess[fieldTableName] = []mdl.IModel{imodel}
					}

					// Can it be a pointer type inside?, then unbox it in the next recursion
					if err := gatherModelToCreate(v.Field(i).Index(j), data); err != nil {
						return err
					}
				}
			case reflect.Ptr:
				if !isNil(v.Field(i)) && !isNil(v.Field(i).Elem()) &&
					v.Field(i).IsValid() && v.Field(i).Elem().IsValid() {
					imodel := v.Field(i).Interface().(mdl.IModel)
					fieldTableName := mdl.GetTableNameFromIModel(imodel)

					if _, ok := data.toProcess[fieldTableName]; ok {
						ms := data.toProcess[fieldTableName]
						ms = append(ms, imodel)
						data.toProcess[fieldTableName] = ms
					} else {
						data.toProcess[fieldTableName] = []mdl.IModel{imodel}
					}

					if err := gatherModelToCreate(v.Field(i).Elem(), data); err != nil {
						return err
					}
				}
			}
		}
		/* what should happen in many to many?
		else if strings.HasPrefix(t, "pegassoc-manytomany") {
			// We're deleting. And now we have a many to many in here
			// Remove the many to many
			var m mdl.IModel
			switch v.Field(i).Kind() {
			case reflect.Struct:
				m = v.Field(i).Addr().Interface().(mdl.IModel)
			case reflect.Slice:
				typ := v.Type().Field(i).Type.Elem()
				m = reflect.New(typ).Interface().(mdl.IModel)
			case reflect.Ptr:
				m = v.Elem().Interface().(mdl.IModel)
			}
			if err := removeManyToManyAssociationTableElem(db, m); err != nil {
				return err
			}
		}
		*/
	}
	return nil
}
