package qry

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	uuid "github.com/satori/go.uuid"
	"github.com/t2wu/qry/datatype"
	"github.com/t2wu/qry/mdl"
	"github.com/t2wu/qry/qtag"

	"gorm.io/gorm"
)

// Remove strategy
// If pegassoc, record it as dissociate and you're done.
// If pegged, record it as to be removed., traverse into the struct. If then encounter a peg, record it
// to be removed and need to traverse into it further, if encounter a pegassoc, record it as to be
// dissociated.
// If no pegassoc and pegged under this struct, return.

type modelAndIds struct {
	modelObj mdl.IModel
	ids      []interface{} // to send to Gorm need to be interface not *datatype.UUID
}

type cargo struct {
	toProcess map[string]modelAndIds
}

func DeleteModelFixManyToManyAndPegAndPegAssoc(db *gorm.DB, modelObj mdl.IModel) error {
	if err := removeManyToManyAssociationTableElem(db, modelObj); err != nil {
		return err
	}

	// Delete nested field
	// Not yet support two-level of nested field
	v := reflect.Indirect(reflect.ValueOf(modelObj))

	peg := make(map[string]modelAndIds) // key: name of the table to be deleted, val: list of ids
	car := cargo{toProcess: peg}

	if err := markForDelete(db, v, car); err != nil {
		return err
	}

	// Now actually delete
	for tblName := range car.toProcess {
		if err := db.Table(tblName).Delete(car.toProcess[tblName].modelObj, car.toProcess[tblName].ids).Error; err != nil {
			return err
		}
	}

	return nil

}

// TODO: if there is a "pegassoc-manytomany" inside a pegged struct
// and we're deleting the pegged struct, the many-to-many relationship needs to be removed
func markForDelete(db *gorm.DB, v reflect.Value, car cargo) error {
	for i := 0; i < v.NumField(); i++ {
		t := qtag.GetQryTag(v.Type().Field(i).Tag)
		// t := pegPegassocOrPegManyToMany(v.Type().Field(i).Tag)
		if t == qtag.QryTagPeg {
			switch v.Field(i).Kind() {
			case reflect.Struct:
				m := v.Field(i).Addr().Interface().(mdl.IModel)
				if m.GetID() != nil { // could be embedded struct that never get initialiezd
					fieldTableName := mdl.GetTableNameFromIModel(m)
					if _, ok := car.toProcess[fieldTableName]; ok {
						mids := car.toProcess[fieldTableName]
						mids.ids = append(mids.ids, m.GetID())
						car.toProcess[fieldTableName] = mids
					} else {
						arr := make([]interface{}, 1)
						arr[0] = m.GetID()
						car.toProcess[fieldTableName] = modelAndIds{modelObj: m, ids: arr}
					}

					// Traverse into it
					if err := markForDelete(db, v.Field(i), car); err != nil {
						return err
					}
				}
			case reflect.Slice:
				typ := v.Type().Field(i).Type.Elem()
				m, _ := reflect.New(typ).Interface().(mdl.IModel)
				fieldTableName := mdl.GetTableNameFromIModel(m)
				for j := 0; j < v.Field(i).Len(); j++ {
					if _, ok := car.toProcess[fieldTableName]; ok {
						mids := car.toProcess[fieldTableName]
						mids.ids = append(mids.ids, v.Field(i).Index(j).Addr().Interface().(mdl.IModel).GetID())
						car.toProcess[fieldTableName] = mids
					} else {
						arr := make([]interface{}, 1)
						arr[0] = v.Field(i).Index(j).Addr().Interface().(mdl.IModel).GetID()
						car.toProcess[fieldTableName] = modelAndIds{modelObj: m, ids: arr}
					}

					// Can it be a pointer type inside?, then unbox it in the next recursion
					if err := markForDelete(db, v.Field(i).Index(j), car); err != nil {
						return err
					}
				}
			case reflect.Ptr:
				// Need to dereference and get the struct id before traversing in
				if !isNil(v.Field(i)) && !isNil(v.Field(i).Elem()) &&
					v.Field(i).IsValid() && v.Field(i).Elem().IsValid() {
					imodel := v.Field(i).Interface().(mdl.IModel)
					fieldTableName := mdl.GetTableNameFromIModel(imodel)
					id := imodel.GetID()

					if _, ok := car.toProcess[fieldTableName]; ok {
						mids := car.toProcess[fieldTableName]
						mids.ids = append(mids.ids, id)
						car.toProcess[fieldTableName] = mids
					} else {
						arr := make([]interface{}, 1)
						arr[0] = id
						car.toProcess[fieldTableName] = modelAndIds{modelObj: imodel, ids: arr}
					}

					if err := markForDelete(db, v.Field(i).Elem(), car); err != nil {
						return err
					}
				}
			}
		} else if t == qtag.QryTagPegAssocMany2Many {
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
	}
	return nil
}

func removeManyToManyAssociationTableElem(db *gorm.DB, modelObj mdl.IModel) error {
	// many to many, here we remove the entry in the actual immediate table
	// because that's actually the link table. Thought we don't delete the
	// Model table itself
	v := reflect.Indirect(reflect.ValueOf(modelObj))
	for i := 0; i < v.NumField(); i++ {
		tagAndFields := qtag.GetQryTagAndField(v.Type().Field(i).Tag)
		if len(tagAndFields) == 0 {
			continue
		}
		tagAndField := tagAndFields[0]

		if tagAndField.Tag == qtag.QryTagPegAssocMany2Many {
			// many to many, here we remove the entry in the actual immediate table
			// because that's actually the link table. Thought we don't delete the
			// Model table itself

			// The normal Delete(mdl, ids) doesn't quite work because
			// I don't have access to the mdl, it's not registered as typestring
			// nor part of the field type. It's a joining table between many to many

			linkTableName := tagAndField.Field
			typ := v.Type().Field(i).Type.Elem() // Get the type of the element of slice
			m2, _ := reflect.New(typ).Interface().(mdl.IModel)
			fieldTableName := mdl.GetTableNameFromIModel(m2)
			selfTableName := mdl.GetTableNameFromIModel(modelObj)

			fieldVal := v.Field(i)
			if fieldVal.Len() >= 1 {
				uuidStmts := strings.Repeat("?,", fieldVal.Len())
				uuidStmts = uuidStmts[:len(uuidStmts)-1]

				allIds := make([]interface{}, 0, 10)
				allIds = append(allIds, modelObj.GetID().String())
				for j := 0; j < fieldVal.Len(); j++ {
					idToDel := fieldVal.Index(j).FieldByName("ID").Interface().(*uuid.UUID)
					allIds = append(allIds, idToDel.String())
				}

				stmt := fmt.Sprintf("DELETE FROM \"%s\" WHERE \"%s\" = ? AND \"%s\" IN (%s)",
					linkTableName, selfTableName+"_id", fieldTableName+"_id", uuidStmts)
				err := db.Exec(stmt, allIds...).Error
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkIDsShouldNotBeFound(db *gorm.DB, nestedIModels []mdl.IModel) error {
	if len(nestedIModels) > 0 {
		ids := make([]*uuid.UUID, 0)
		for _, nestedIModel := range nestedIModels {
			id := nestedIModel.GetID()
			if id != nil {
				ids = append(ids, id)
			}
		}

		if len(ids) == 0 {
			// nothing to check
			return nil
		}

		tableName := mdl.GetTableNameFromIModel(nestedIModels[0])
		var count int64
		err := db.Table(tableName).Where("id IN (?)", ids).Count(&count).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err // some real error
		}
		if count != 0 {
			return fmt.Errorf("id of embedded pegged object already exists")
		}
	}

	return nil
}

// Why does this need to check if checkIDsShouldNotBeFound?
// If user input some ID that's already in DB I am to reject here.
// But if I left it to gorm it could be rejected as well, I just need to catch it
// and turn it into "id of embedded pegged object already exists"
func CheckPeggedFieldsHasNoExistingID(db *gorm.DB, modelObj mdl.IModel) error {
	v := reflect.Indirect(reflect.ValueOf(modelObj))

	for i := 0; i < v.NumField(); i++ {
		tag := qtag.GetQryTag(v.Type().Field(i).Tag)
		if tag == qtag.QryTagPeg {
			fieldVal := v.Field(i)
			switch fieldVal.Kind() {
			case reflect.Slice:
				// Loop through the slice
				ms := make([]mdl.IModel, 0)
				for j := 0; j < fieldVal.Len(); j++ {
					nestedModel := fieldVal.Index(j).Addr().Interface()
					nestedIModel, ok := nestedModel.(mdl.IModel)
					if ok {
						ms = append(ms, nestedIModel)
					}
				}

				if err := checkIDsShouldNotBeFound(db, ms); err != nil {
					return err
				}

				for j := 0; j < len(ms); j++ {
					nestedIModel := ms[j]
					// Traverse into it
					if err := CheckPeggedFieldsHasNoExistingID(db, nestedIModel); err != nil {
						return err
					}
				}
			case reflect.Ptr:
				nestedModel := v.Field(i).Interface()
				nestedIModel, ok := nestedModel.(mdl.IModel)
				if ok && !isNil(nestedIModel) {
					if nestedIModel.GetID() != nil {
						if nestedIModel.GetID() != nil {
							if err := checkIDsShouldNotBeFound(db, []mdl.IModel{nestedIModel}); err != nil {
								return err
							}
						}
					}
					// Traverse into it
					if err := CheckPeggedFieldsHasNoExistingID(db, nestedIModel); err != nil {
						return err
					}
				}

			case reflect.Struct:
				nestedModel := v.Field(i).Addr().Interface()
				nestedIModel, ok := nestedModel.(mdl.IModel)
				if ok {
					if nestedIModel.GetID() != nil {
						if err := checkIDsShouldNotBeFound(db, []mdl.IModel{nestedIModel}); err != nil {
							return err
						}
					}

					// Traverse into it
					if err := CheckPeggedFieldsHasNoExistingID(db, nestedIModel); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// CreatePeggedAssocFields :-
func CreatePeggedAssocFields(db *gorm.DB, modelObj mdl.IModel) (err error) {
	v := reflect.Indirect(reflect.ValueOf(modelObj))
	for i := 0; i < v.NumField(); i++ {
		tag := qtag.GetQryTag(v.Type().Field(i).Tag)
		// columnName := v.Type().Field(i).Name
		if tag == qtag.QryTagPegAssoc {
			fieldVal := v.Field(i)
			switch fieldVal.Kind() {
			case reflect.Slice:
				// Loop through the slice
				for j := 0; j < fieldVal.Len(); j++ {
					// nestedModelID := fieldVal.Index(j).FieldByName("ID").Interface().(*datatype.UUID)
					nestedModel := fieldVal.Index(j).Addr().Interface()
					nestedIModel, ok := nestedModel.(mdl.IModel)
					if ok && nestedIModel.GetID() != nil {
						tableName := mdl.GetTableNameFromIModel(modelObj)
						correspondingColumnName := tableName + "_id"
						// Where clause is not needed when the embedded is a struct, but if it's a pointer to struct then it's needed
						if err := db.Model(nestedModel).Where("id = ?", nestedModel.(mdl.IModel).GetID()).
							Update(correspondingColumnName, modelObj.GetID()).Error; err != nil {
							return err
						}
					}

					// // this loops forever unlike update, why?
					// if err = db.Set("gorm:association_autoupdate", true).Model(modelObj).Association(columnName).Append(nestedModel).Error; err != nil {
					// 	return err
					// }
				}
			case reflect.Ptr:
				nestedModel := v.Field(i).Interface()
				nestedIModel, ok := nestedModel.(mdl.IModel)
				if ok && !isNil(nestedIModel) && nestedIModel.GetID() != nil {
					tableName := mdl.GetTableNameFromIModel(modelObj)
					correspondingColumnName := tableName + "_id"
					// Where clause is not needed when the embedded is a struct, but if it's a pointer to struct then it's needed
					if err := db.Model(nestedModel).Where("id = ?", nestedModel.(mdl.IModel).GetID()).Update(correspondingColumnName, modelObj.GetID()).Error; err != nil {
						return err
					}
				}
			case reflect.Struct:
				nestedModel := v.Field(i).Addr().Interface()
				nestedIModel, ok := nestedModel.(mdl.IModel)
				if ok && nestedIModel.GetID() != nil {
					tableName := mdl.GetTableNameFromIModel(modelObj)
					correspondingColumnName := tableName + "_id"
					// Where clause is not needed when the embedded is a struct, but if it's a pointer to struct then it's needed
					if err := db.Model(nestedModel).Where("id = ?", nestedModel.(mdl.IModel).GetID()).
						Update(correspondingColumnName, modelObj.GetID()).Error; err != nil {
						return err
					}
				}
			default:
				// embedded object is considered part of the structure, so no removal
			}
		}
	}
	return nil
}

func isNil(a interface{}) bool {
	defer func() { recover() }()
	return a == nil || reflect.ValueOf(a).IsNil()
}

// TODO: Need to make this more efficient by handling multiple tables at the same time like we do on create
func UpdateNestedFields(db *gorm.DB, oldModelObj mdl.IModel, newModelObj mdl.IModel) (err error) {
	// Indirect is dereference
	// Interface() is extract content than re-wrap to interface
	// Since reflect.New() returns pointer to the object, after reflect.ValueOf
	// We need to deference it, hence "Indirect", now v1 and v2 are the actual object, not
	// ptr to objects
	v1 := reflect.Indirect(reflect.ValueOf(oldModelObj))
	v2 := reflect.Indirect(reflect.ValueOf(newModelObj))

	for i := 0; i < v1.NumField(); i++ {
		tagNFields := qtag.GetQryTagAndField(v1.Type().Field(i).Tag)
		if len(tagNFields) != 0 { // peg, pegassoc, or pegassoc-many2many
			tagNField := tagNFields[0]

			fieldVal1 := v1.Field(i)
			fieldVal2 := v2.Field(i)

			set1 := datatype.NewSetString()
			set2 := datatype.NewSetString()

			oriM := make(map[string]interface{})
			newM := make(map[string]interface{})

			switch fieldVal1.Kind() {
			case reflect.Slice:
				// Loop through the slice
				for j := 0; j < fieldVal1.Len(); j++ {
					// For example, each fieldVal1.Index(j) is a model object
					id := fieldVal1.Index(j).FieldByName("ID").Interface().(*uuid.UUID)
					set1.Add(id.String())

					oriM[id.String()] = fieldVal1.Index(j).Addr().Interface() // re-wrap a dock
				}

				for j := 0; j < fieldVal2.Len(); j++ {
					id := fieldVal2.Index(j).FieldByName("ID").Interface().(*uuid.UUID)
					if id != nil {
						// ID doesn't exist? ignore, it's a new entry without ID
						set2.Add(id.String())
						newM[id.String()] = fieldVal2.Index(j).Addr().Interface()
					}
				}

				// remove when stuff in the old set that's not in the new set
				setIsGone := set1.Difference(set2)

				for uuid1 := range setIsGone.List {
					modelToDel := oriM[uuid1]

					if tagNField.Tag == qtag.QryTagPeg {
						if err := db.Delete(modelToDel).Error; err != nil {
							return err
						}
						// Similar to directly deleting the model,
						// just deleting it won't work, need to traverse down the chain
						if err := DeleteModelFixManyToManyAndPegAndPegAssoc(db, modelToDel.(mdl.IModel)); err != nil {
							return err
						}
					} else if tagNField.Tag == qtag.QryTagPegAssoc {
						columnName := v1.Type().Field(i).Name
						// assocModel := reflect.Indirect(reflect.ValueOf(modelToDel)).Type().Name()
						// fieldName := v1.Type().Field(i).Name
						// fieldName = fieldName[0 : len(fieldName)-1] // get rid of s
						// tableName := letters.CamelCaseToPascalCase(fieldName)
						if err = db.Model(oldModelObj).Association(columnName).Delete(modelToDel); err != nil {
							return err
						}
					} else if tagNField.Tag == qtag.QryTagPegAssocMany2Many {
						// many to many, here we remove the entry in the actual immediate table
						// because that's actually the link table. Thought we don't delete the
						// Model table itself
						linkTableName := tagNField.Field
						// Get the base type of this field

						inter := v1.Field(i).Interface()
						typ := reflect.TypeOf(inter).Elem() // Get the type of the element of slice
						m2, _ := reflect.New(typ).Interface().(mdl.IModel)

						fieldTableName := mdl.GetTableNameFromIModel(m2)
						fieldIDName := fieldTableName + "_id"

						selfTableName := mdl.GetTableNameFromIModel(oldModelObj)
						selfID := selfTableName + "_id"

						// The following line seems to puke on a many-to-many, I hope I don't need it anywhere
						// else in another many-to-many
						// idToDel := reflect.Indirect(reflect.ValueOf(modelToDel)).Elem().FieldByName("ID").Interface()
						idToDel := reflect.Indirect(reflect.ValueOf(modelToDel)).FieldByName("ID").Interface()

						stmt := fmt.Sprintf("DELETE FROM \"%s\" WHERE \"%s\" = ? AND \"%s\" = ?",
							linkTableName, fieldIDName, selfID)
						err := db.Exec(stmt, idToDel.(*uuid.UUID).String(), oldModelObj.GetID().String()).Error
						if err != nil {
							return err
						}

					}
				}

				setIsNew := set2.Difference(set1)
				for uuid := range setIsNew.List {
					modelToAdd := newM[uuid]
					if tagNField.Tag == qtag.QryTagPeg {
						// Don't need peg, because gorm already does auto-create by default
						// for truly nested data without endpoint
						// err = db.Save(modelToAdd).Error
						// if err != nil {
						// 	return err
						// }

						// Wait does Gormv2 do that? Need to test.
					} else if tagNField.Tag == qtag.QryTagPegAssoc {

						tableToAssoc := mdl.GetTableNameFromIModel(modelToAdd.(mdl.IModel))
						// If the new model doesn't exist, it is an error.
						// If it exists, all we have to do is update the parent table reference of the nested table
						if newModelObj.GetID() == nil {
							return fmt.Errorf("%s does not exists during an update", tableToAssoc)
						}
						var c int64
						if err := db.Table(tableToAssoc).Where(fmt.Sprintf("%s.id = ?", tableToAssoc), newModelObj.GetID()).
							Count(&c).Error; err != nil {
							return err
						}
						if c != 1 {
							return fmt.Errorf("%s does not exists during an update", modelToAdd)
						}
						columnName := mdl.GetTableNameFromIModel(newModelObj) + "_id"
						if err = db.Model(modelToAdd).Where(fmt.Sprintf("%s.id = ?", tableToAssoc), modelToAdd.(mdl.IModel).GetID()).
							Update(columnName, newModelObj.GetID()).Error; err != nil {
							return err
						}
					}
					//else if strings.HasPrefix(tag, "pegassoc-manytomany") {
					// No need either, Gorm already creates it
					// It's the preloading that's the issue.
					//}
				}

				// both exists
				setMayBeEdited := set1.Intersect(set2)
				for uuid := range setMayBeEdited.List {
					oriModelToEdit := oriM[uuid]
					newModelToEdit := newM[uuid]
					if tagNField.Tag == qtag.QryTagPeg {
						if err := UpdateNestedFields(db, oriModelToEdit.(mdl.IModel), newModelToEdit.(mdl.IModel)); err != nil {
							return err
						}
					}
				}

			case reflect.Struct:
				if tagNField.Tag == qtag.QryTagPeg {
					if err := UpdateNestedFields(db, fieldVal1.Addr().Interface().(mdl.IModel), fieldVal2.Addr().Interface().(mdl.IModel)); err != nil {
						return err
					}
				}

				if tagNField.Tag == qtag.QryTagPegAssoc {

					assocModel := fieldVal2.Addr().Interface().(mdl.IModel)
					tableToAssoc := mdl.GetTableNameFromIModel(assocModel)

					// If the new model doesn't exist, it is an error.
					// If it exists, all we have to do is update the parent table reference of the nested table
					if assocModel.GetID() == nil {
						return fmt.Errorf("%s does not exists during an update", tableToAssoc)
					}
					var c int64
					if err := db.Table(tableToAssoc).Where(fmt.Sprintf("%s.id = ?", tableToAssoc), assocModel.GetID()).
						Count(&c).Error; err != nil {
						return err
					}
					if c != 1 {
						return fmt.Errorf("%s does not exists during an update", tableToAssoc)
					}
					columnName := mdl.GetTableNameFromIModel(newModelObj) + "_id"
					if err = db.Table(tableToAssoc).Where(fmt.Sprintf("%s.id", tableToAssoc), assocModel.GetID()).
						Update(columnName, newModelObj.GetID()).Error; err != nil {
						return err
					}
				}

			default:
				// embedded object is considered part of the structure, so no removal
			}
		}
	}

	return nil
}

func GrabFirstLevelAssociatedFieldsForSlice(value interface{}) (map[uuid.UUID]map[string][]uuid.UUID, error) {
	// This is actually a reduce pattern, TODO: refactor this and hte kind like it (GrabAllNestedPeggedStructsForSlice)
	val := reflect.Indirect(reflect.ValueOf(value))

	retval := make(map[uuid.UUID]map[string][]uuid.UUID, 0) // embedded_table_name -> embedded_table_id

	if val.Len() == 0 {
		return retval, nil
	}

	isPtr := false
	if val.Index(0).Kind() == reflect.Ptr {
		isPtr = true
	}

	for j := 0; j < val.Len(); j++ {
		var modelObj mdl.IModel
		if isPtr {
			modelObj = val.Index(j).Interface().(mdl.IModel)
		} else {
			modelObj = val.Index(j).Addr().Interface().(mdl.IModel)
		}

		if err := grabFirstLevelAssociatedFields_core(modelObj, &retval); err != nil {
			return retval, err
		}
	}
	return retval, nil
}

func GrabFirstLevelAssociatedFields(modelObj mdl.IModel) (map[uuid.UUID]map[string][]uuid.UUID, error) {
	// We don't deal with nested, because nested means you're changing
	// associated data, which we don't do

	retval := make(map[uuid.UUID]map[string][]uuid.UUID, 0) // parent_table_id -> embedded_table_name -> embedded_table_ids
	if err := grabFirstLevelAssociatedFields_core(modelObj, &retval); err != nil {
		return retval, err
	}
	return retval, nil
}

// retval: embedded_table_name -> embedded_table_id
func grabFirstLevelAssociatedFields_core(modelObj mdl.IModel, retval *map[uuid.UUID]map[string][]uuid.UUID) error {
	// We don't deal with nested, because nested means you're changing
	// associated data, which we don't do

	// parent_table_id -> embedded_table_name -> embedded_table_ids

	retval2 := *retval

	val := reflect.Indirect(reflect.ValueOf(modelObj))
	for i := 0; i < val.NumField(); i++ {
		tag := val.Type().Field(i).Tag
		if qtag.GetQryTag(tag) == qtag.QryTagPegAssoc {
			// log.Println("seriously???, waht is the tag?", tag, val.Type().Field(i).Name)
			field := val.Field(i)
			switch field.Kind() {
			case reflect.Slice, reflect.Array:
				for j := 0; j < field.Len(); j++ {
					m := field.Index(j).Addr().Interface().(mdl.IModel)
					if m != nil && m.GetID() != nil {
						tableName := mdl.GetTableNameFromIModel(m)

						if retval2[*modelObj.GetID()] == nil {
							retval2[*modelObj.GetID()] = make(map[string][]uuid.UUID)
						}
						if retval2[*modelObj.GetID()][tableName] == nil {
							retval2[*modelObj.GetID()][tableName] = make([]uuid.UUID, 0)
						}
						retval2[*modelObj.GetID()][tableName] = append(retval2[*modelObj.GetID()][tableName], *m.GetID())
					}
				}
			case reflect.Ptr:
				if !field.IsNil() {
					m := field.Interface().(mdl.IModel)
					if m != nil && m.GetID() != nil {
						tableName := mdl.GetTableNameFromIModel(m)

						if retval2[*modelObj.GetID()] == nil {
							retval2[*modelObj.GetID()] = make(map[string][]uuid.UUID)
						}
						if retval2[*modelObj.GetID()][tableName] == nil {
							retval2[*modelObj.GetID()][tableName] = make([]uuid.UUID, 0)
						}
						retval2[*modelObj.GetID()][tableName] = append(retval2[*modelObj.GetID()][tableName], *m.GetID())
					}
				}
			case reflect.Struct:
				m := field.Addr().Interface().(mdl.IModel)
				if m != nil && m.GetID() != nil {
					tableName := mdl.GetTableNameFromIModel(m)

					if retval2[*modelObj.GetID()] == nil {
						retval2[*modelObj.GetID()] = make(map[string][]uuid.UUID)
					}
					if retval2[*modelObj.GetID()][tableName] == nil {
						retval2[*modelObj.GetID()][tableName] = make([]uuid.UUID, 0)
					}
					retval2[*modelObj.GetID()][tableName] = append(retval2[*modelObj.GetID()][tableName], *m.GetID())
				}
			}
		}
	}
	return nil
}

// It could be []any model, the only parameter I can have is value interface{}
func GrabAllNestedPeggedStructsForSlice(value interface{}) (map[int]map[string][]mdl.IModel, error) {
	val := reflect.Indirect(reflect.ValueOf(value))

	tblGrps := make(map[int]map[string][]mdl.IModel, 0)

	if val.Len() == 0 {
		return tblGrps, nil
	}

	isPtr := false
	if val.Index(0).Kind() == reflect.Ptr {
		isPtr = true
	}

	for j := 0; j < val.Len(); j++ {
		var modelObj mdl.IModel
		if isPtr {
			modelObj = val.Index(j).Interface().(mdl.IModel)
		} else {
			modelObj = val.Index(j).Addr().Interface().(mdl.IModel)
		}

		if err := grabAllNestedPeggedStructs_core(modelObj, &tblGrps, 1); err != nil {
			return tblGrps, err
		}
	}
	return tblGrps, nil
}

func GrabAllNestedPeggedStructs(modelObj mdl.IModel) (map[int]map[string][]mdl.IModel, error) {
	// level -> tblname -> models, level help us in that we create the highest-level one first
	tblGrps := make(map[int]map[string][]mdl.IModel, 0)
	return tblGrps, grabAllNestedPeggedStructs_core(modelObj, &tblGrps, 1)
}

func grabAllNestedPeggedStructs_core(modelObj mdl.IModel, tblGrps *map[int]map[string][]mdl.IModel, level int) (err error) {
	val := reflect.Indirect(reflect.ValueOf(modelObj))

	tblGrps2 := *tblGrps

	// In case embedded struct doesn't have a parent ID reference, fill it
	parentTableRef := strings.Split(reflect.TypeOf(modelObj).String(), ".")[1] + "ID"

	// We need this because otherwise nested data has no ID to point back
	if modelObj.GetID() == nil {
		newID := datatype.NewUUID()
		modelObj.SetID(&newID)
	}

	for i := 0; i < val.NumField(); i++ {
		tag := val.Type().Field(i).Tag
		if qtag.GetQryTag(tag) == qtag.QryTagPeg {
			field := val.Field(i)
			switch field.Kind() {
			case reflect.Slice, reflect.Array:
				for j := 0; j < field.Len(); j++ {
					// m := field.Index(j)
					m := field.Index(j).Addr().Interface().(mdl.IModel)
					if m != nil {
						// In case embedded struct doesn't have a parent ID reference, fill it
						reflectParentTableRef := field.Index(j).FieldByName(parentTableRef)
						log.Println("trying to set parent table ref:", parentTableRef, "to:", modelObj.GetID())
						parentID := reflectParentTableRef.Interface().(*uuid.UUID)
						if parentID == nil {
							reflectParentTableRef.Set(reflect.ValueOf(modelObj.GetID()))
						}

						tableName := mdl.GetTableNameFromIModel(m)
						if tblGrps2[level] == nil {
							tblGrps2[level] = make(map[string][]mdl.IModel, 0)
						}
						if tblGrps2[level][tableName] == nil {
							tblGrps2[level][tableName] = make([]mdl.IModel, 0)
						}
						tblGrps2[level][tableName] = append(tblGrps2[level][tableName], m)

						if err := grabAllNestedPeggedStructs_core(m, tblGrps, level+1); err != nil {
							return err
						}
					}
				}
			case reflect.Ptr:
				if !field.IsNil() {
					reflectParentTableRef := field.Elem().FieldByName(parentTableRef)
					parentID := reflectParentTableRef.Interface().(*uuid.UUID)
					if parentID == nil {
						reflectParentTableRef.Set(reflect.ValueOf(modelObj.GetID()))
					}

					m := field.Interface().(mdl.IModel)
					if m != nil {
						tableName := mdl.GetTableNameFromIModel(m)
						if tblGrps2[level] == nil {
							tblGrps2[level] = make(map[string][]mdl.IModel, 0)
						}
						if tblGrps2[level][tableName] == nil {
							tblGrps2[level][tableName] = make([]mdl.IModel, 0)
						}
						tblGrps2[level][tableName] = append(tblGrps2[level][tableName], m)

						if err := grabAllNestedPeggedStructs_core(m, tblGrps, level+1); err != nil {
							return err
						}
					}
				}
			case reflect.Struct:
				m := field.Addr().Interface().(mdl.IModel)
				if m != nil {
					// In case embedded struct doesn't have a parent ID reference, fill it
					reflectParentTableRef := field.FieldByName(parentTableRef)
					parentID := reflectParentTableRef.Interface().(*uuid.UUID)
					if parentID == nil {
						reflectParentTableRef.Set(reflect.ValueOf(modelObj.GetID()))
					}

					tableName := mdl.GetTableNameFromIModel(m)
					if tblGrps2[level] == nil {
						tblGrps2[level] = make(map[string][]mdl.IModel, 0)
					}

					if tblGrps2[level][tableName] == nil {
						tblGrps2[level][tableName] = make([]mdl.IModel, 0)
					}
					tblGrps2[level][tableName] = append(tblGrps2[level][tableName], m)
					if err := grabAllNestedPeggedStructs_core(m, tblGrps, level+1); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
