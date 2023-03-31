package qry

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/t2wu/qry/mdl"
	"github.com/t2wu/qry/qrylogger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// -----------------------------
type QueryType int

const (
	QueryTypeFirst QueryType = iota
	QueryTypeFind
)

// It would be Q(db, C(...), C(...)...).First() or Q(db).First() with empty PredicateRelationBuilder
// Use multiple C() when working on inner fields (one C() per struct field)
func Q(db *gorm.DB, args ...interface{}) IQuery {
	q := &Query{db: db, saveLck: &sync.Mutex{}}
	return q.Q(args...)
}

// Instead of Q() directly, we can use DB().Q()
// This is so it's easier to stubb out when testing
func DB(db *gorm.DB) IQuery {
	return Q(db) // no argument. That way mainMB would never be null
}

// Q is the query struct
// Q(db).By("Name IN", []strings{name1, name2}, "Age >=", 18).Find(&mdl).Error
// This is a wrapper over Gorm's.
// Query by field name, and prevent SQL injection by making sure that fields are part of the
// mdl
type Query struct {
	db *gorm.DB // Gorm db object can be a transaction

	// args  []interface{}
	Err error

	// custom order to Gorm instead of "created_at DESC"
	orderField *string
	order      *Order

	limit  *int // custom limit
	offset *int // custom offset

	// This is the temporary fix, what should probably happen is that each call to Query should
	// create a new Query intance with the state mantained
	saveLck *sync.Mutex

	mainMB *ModelAndBuilder  // the builder on the main mdl (including the nested one)
	mbs    []ModelAndBuilder // the builder for non-nested mdl, each one is a separate non-nested mdl
}

// Q takes in PredicateRelationBuilder here.
func (q *Query) Q(args ...interface{}) IQuery {
	// q.Reset() // always reset with Q() // do i need to? what if order first?

	// Returns a new IQuery, because we don't really want to keep state here
	// It is expected that after q = qry.DB(db),
	// q.Q() be re-entrant and many can call at the same time.
	// So have to return a new IQuery

	q2 := &Query{db: q.db, saveLck: &sync.Mutex{}}

	mb := ModelAndBuilder{}
	for _, arg := range args {
		b, ok := arg.(*PredicateRelationBuilder)
		if !ok {
			q2.Err = fmt.Errorf("incorrect arguments for Q()")
			PrintFileAndLine(q2.Err)
			return q2
		}

		// Leave mdl empty because it is not going to be filled until
		// Find() or First()
		binfo := BuilderInfo{
			builder: b,
			// processed: false,
		}
		mb.builderInfos = append(mb.builderInfos, binfo)
	}

	q2.mainMB = &mb

	return q2
}

func (q *Query) Order(field string, order Order) IQuery {
	// func (q *Query) Order(order string) IQuery {
	if q.order != nil {
		log.Println("warning: query order already set")
	}

	if strings.Contains(field, ".") {
		q.Err = fmt.Errorf("dot notation in field not supported")
		PrintFileAndLine(q.Err)
		return q
	}

	q.orderField = &field
	q.order = &order
	return q
}

func (q *Query) Limit(limit int) IQuery {
	if q.limit != nil {
		log.Println("warning: query limit already set")
	}
	q.limit = &limit
	return q
}

func (q *Query) Offset(offset int) IQuery {
	if q.offset != nil {
		log.Println("warning: query offset already set")
	}
	q.offset = &offset
	return q
}

// args can be multiple C(), each C() works on one-level of modelObj
// The args are to select the query of modelObj designated, it could work
// on nested level inside the modelObj
// assuming first is top-level, if given.
func (q *Query) InnerJoin(modelObj mdl.IModel, foreignObj mdl.IModel, args ...interface{}) IQuery {
	if q.Err != nil {
		return q
	}

	// Need to build the "On" clause
	// modelObj.foreignObjID = foreignObj.ID plus addition condition if any
	var ok bool
	var b *PredicateRelationBuilder

	typeName := mdl.GetModelTypeNameFromIModel(foreignObj)
	tbl := mdl.GetTableNameFromIModel(foreignObj)
	esc := &Escape{Value: fmt.Sprintf("\"%s\".id", tbl)}

	// Prepare for PredicateRelationBuilder which will be use to generate inner join statement
	// between the modelobj at hand and foreignObj (when joining the immediate table, the forignObj is
	// the modelObj within Find() and First())
	if len(args) > 0 {
		b, ok = args[0].(*PredicateRelationBuilder)
		if !ok {
			q.Err = fmt.Errorf("incorrect arguments for Q()")
			PrintFileAndLine(q.Err)
			return q
		}

		// Check if the designator is about inner field or the outer-most level field
		rel, err := b.GetPredicateRelation()
		if err != nil {
			q.Err = err
			return q
		}
		field2Struct, _ := FindFieldNameToStructAndStructFieldNameIfAny(rel) // hacky
		if field2Struct == nil {                                             // outer-level field
			args[0] = b.And(typeName+"ID = ", esc)
		} else {
			// No other criteria, it is just a join by itself
			args = append(args, C(typeName+"ID = ", esc))
			// mb := ModelAndBuilder{ModelObj: modelObj, Builder: b}
			// q.mbs = append(q.mbs, mb)
		}
	} else { // No PredicateRelationBuilder given, build one from scratch
		args = append(args, C(typeName+"ID = ", esc))
		// mb := ModelAndBuilder{ModelObj: modelObj, Builder: b}
		// q.mbs = append(q.mbs, mb)
	}

	mb := ModelAndBuilder{}
	mb.modelObj = modelObj

	for i := 0; i < len(args); i++ {
		b, ok := args[i].(*PredicateRelationBuilder)
		if !ok {
			q.Err = fmt.Errorf("incorrect arguments for Q()")
			PrintFileAndLine(q.Err)
			return q
		}
		binfo := BuilderInfo{
			builder: b,
			// processed: false,
		}
		mb.builderInfos = append(mb.builderInfos, binfo)
	}

	q.mbs = append(q.mbs, mb)

	return q
}

func (q *Query) Take(modelObj mdl.IModel) IQuery {
	defer resetWithoutResetError(q)

	db := q.db

	if q.Err != nil {
		return q
	}

	if q.mainMB != nil {
		q.mainMB.modelObj = modelObj
	} else {
		db = db.Model(modelObj)
	}

	var err error
	db, err = q.buildQueryCore(db, modelObj)
	if err != nil {
		q.Err = err
		return q
	}

	db = q.buildQueryOrderOffSetAndLimit(db, modelObj)

	names := mdl.GetEmbeddedTablePaths(modelObj)
	db = db.Preload(clause.Associations)
	for _, name := range names {
		db = db.Preload(name)
	}

	q.Err = db.Take(modelObj).Error

	return q
}

func (q *Query) First(modelObj mdl.IModel) IQuery {
	defer resetWithoutResetError(q)

	db := q.db
	if q.Err != nil {
		return q
	}

	if q.mainMB != nil {
		q.mainMB.modelObj = modelObj
	} else {
		db = q.db.Model(modelObj)
	}

	var err error
	db, err = q.buildQueryCore(db, modelObj)
	if err != nil {
		q.Err = err
		return q
	}

	db = q.buildQueryOrderOffSetAndLimit(db, modelObj)

	names := mdl.GetEmbeddedTablePaths(modelObj)
	db = db.Preload(clause.Associations)
	for _, name := range names {
		db = db.Preload(name)
	}

	q.Err = db.First(modelObj).Error

	return q
}

func (q *Query) Count(modelObj mdl.IModel, no *int64) IQuery {
	defer resetWithoutResetError(q)

	db := q.db
	if q.Err != nil {
		return q
	}

	if q.mainMB != nil {
		q.mainMB.modelObj = modelObj
	} else {
		db = db.Model(modelObj)
	}

	var err error
	db, err = q.buildQueryCore(db, modelObj)
	if err != nil {
		q.Err = err
		return q
	}

	db = q.buildQueryOrderOffSetAndLimit(db, modelObj)
	q.Err = db.Count(no).Error
	if q.Err != nil {
		PrintFileAndLine(q.Err)
	}

	return q
}

func (q *Query) Find(modelObjs interface{}) IQuery {
	defer resetWithoutResetError(q)

	db := q.db

	if q.Err != nil {
		return q
	}

	typ := reflect.TypeOf(modelObjs)
loop:
	for {
		switch typ.Kind() {
		case reflect.Slice:
			typ = typ.Elem()
		case reflect.Ptr:
			typ = typ.Elem()
		default:
			break loop
		}
	}

	modelObj := reflect.New(typ).Interface().(mdl.IModel)

	if q.mainMB != nil {
		q.mainMB.modelObj = modelObj
	} else {
		db = db.Model(modelObj)
	}
	// db = db.Model(modelObj)

	var err error
	db, err = q.buildQueryCore(db, modelObj)
	if err != nil {
		q.Err = err
		return q
	}

	db = q.buildQueryOrderOffSetAndLimit(db, modelObj)

	names := mdl.GetEmbeddedTablePaths(modelObj)
	db = db.Preload(clause.Associations)
	for _, name := range names {
		db = db.Preload(name)
	}

	q.Err = db.Find(modelObjs).Error
	// q.Err = db.Find(modelObjs).Error

	// In Gorm v1 if nothing is found, there is an error
	// Whereas in v1 there is no error unlike take, first, last.
	// Here we maintain v1's behavior: error when nothing is found
	// We do that by checking slice length or modelObj ID
	if q.Err == nil {
		// could be a slice, a pointer to struct
		// If find() is given a slice and not found, no warning
		// If find() is given an element, should have warning
		typ := reflect.TypeOf(modelObjs)
		switch typ.Kind() {
		// case reflect.Slice:
		// 	if reflect.ValueOf(modelObjs).Len() == 0 {
		// 		q.Err = gorm.ErrRecordNotFound
		// 	}
		case reflect.Ptr:
			// if id is nil, then it's not found
			if obj, ok := modelObjs.(mdl.IModel); ok {
				if id := obj.GetID(); id == nil {
					q.Err = gorm.ErrRecordNotFound
				}
			}

			// Is it a pointer to a struct?
			// ele := reflect.ValueOf(modelObjs).Elem()
			// if obj, ok := modelObjs.(*mdl.IModel); ok {
			// 	if id := (*obj).(mdl.IModel).GetID(); id == nil {
			// 		q.Err = gorm.ErrRecordNotFound
			// 	}
			// }

			// It could be a pointer to slice
			// sl := reflect.ValueOf(modelObjs).Elem()
			// if sl.Kind() == reflect.Slice {
			// 	if sl.Len() == 0 {
			// 		q.Err = gorm.ErrRecordNotFound
			// 	}
			// }

			// Should already handled all cases

		default:
			break
		}
	}

	return q
}

// This is a passover for building query, we're just building the where clause
func (q *Query) BuildQuery(modelObj mdl.IModel) (*gorm.DB, error) {
	defer resetWithoutResetError(q)

	db := q.db

	if q.Err != nil {
		return db, q.Err
	}

	if q.mainMB != nil {
		q.mainMB.modelObj = modelObj
	} else {
		db = db.Model(modelObj)
	}

	return q.buildQueryCore(db, modelObj)
}

func (q *Query) buildQueryCore(db *gorm.DB, modelObj mdl.IModel) (*gorm.DB, error) {
	var err error
	db = db.Model(modelObj)

	if q.mainMB != nil {

		// handles main modelObj
		q.mainMB.SortBuilderInfosByLevel() // now sorted, so our join statement can join in correct order

		// // First-level queries that have no explicit join table
		// for _, buildInfo := range q.mainMB.builderInfos {
		// 	rel, err := buildInfo.builder.GetPredicateRelation()
		// 	if err != nil {
		// 		return db, err
		// 	}

		// 	if !DesignatorContainsDot(rel) { // where clause
		// 		s, vals, err := rel.BuildQueryStringAndValues(q.mainMB.modelObj)
		// 		if err != nil {
		// 			return db, err
		// 		}
		// 		log.Println("s:", s)
		// 		log.Printf("vals: %+v\n", vals)
		// 		db = db.Where(s, vals...)
		// 	}
		// }

		db, err = q.buildQueryCoreInnerJoin(db, q.mainMB)
		if err != nil {
			return db, err
		}
	}

	// Other non-nested tables
	// where we need table joins for sure and no where clause
	// But join statements foreign keys ha salready been made
	for _, mb := range q.mbs { // Now we work on mb.modelObj
		mb.SortBuilderInfosByLevel()

		for _, buildInfo := range mb.builderInfos { // each of this is on one-level (outer or nested)
			rel, err := buildInfo.builder.GetPredicateRelation()
			if err != nil {
				return db, err
			}

			if !DesignatorContainsDot(rel) {
				// first level, but since this is the other non-nested table
				// we use a join, and the foriegn key join is already set up
				// when we call query.Join
				s, vals, err := rel.BuildQueryStringAndValues(mb.modelObj)
				if err != nil {
					return db, err
				}

				tblName := mdl.GetTableNameFromIModel(mb.modelObj)
				db = db.Joins(fmt.Sprintf("INNER JOIN \"%s\" ON %s", tblName, s), vals...)
			}
		}

		db, err = q.buildQueryCoreInnerJoin(db, &mb)
		if err != nil {
			return db, err
		}
	}

	return db, nil
}

// []string are join table names
func (q *Query) buildQueryCoreInnerJoin(db *gorm.DB, mb *ModelAndBuilder) (*gorm.DB, error) {
	// There may not be any builder for the level of join
	// for example, when querying for 3rd level field, 2nd level also
	// needs to join with the first level
	designators, err := mb.GetAllPotentialJoinStructDesignators()
	if err != nil {
		return db, err
	}

	for _, designator := range designators { // this only loops tables which has joins
		found := false
		for _, buildInfo := range mb.builderInfos {
			rel, err := buildInfo.builder.GetPredicateRelation()
			if err != nil {
				return db, err
			}

			designatedField := rel.GetDesignatedField(mb.modelObj)
			if designator == designatedField { // OK, with this level we have search criteria to go along with it
				found = true
				s, vals, err := rel.BuildQueryStringAndValues(mb.modelObj)
				if err != nil {
					return db, err
				}

				// If it's one-level nested, we can join, but
				innerModel, err := rel.GetDesignatedModel(mb.modelObj)
				if err != nil {
					return db, err
				}
				tblName := mdl.GetTableNameFromIModel(innerModel)
				// get the outer table name
				outerTableName, err := GetOuterTableName(mb.modelObj, designatedField)
				if err != nil {
					return db, err
				}

				// log.Println("*******join:", fmt.Sprintf("INNER JOIN \"%s\" ON \"%s\".%s_id = \"%s\".id AND (%s)", tblName, tblName,
				// outerTableName, outerTableName, s))
				db = db.Joins(fmt.Sprintf("INNER JOIN \"%s\" ON \"%s\".%s_id = \"%s\".id AND (%s)", tblName, tblName,
					outerTableName, outerTableName, s), vals...)
			}
		}
		if !found { // no search critiria, just pure join statement
			toks := strings.Split(designator, ".") // A.B.C then we're concerened about joinnig B & C, A has been done
			// field := toks[len(toks)-1]

			upperTableName := ""
			if len(toks) == 1 {
				upperTableName = mdl.GetTableNameFromIModel(mb.modelObj)
			} else {
				designatorForUpperModel := strings.Join(toks[:len(toks)-1], ".")
				upperTableName, err = mdl.GetModelTableNameInModelIfValid(mb.modelObj, designatorForUpperModel)
				if err != nil {
					return db, err
				}
			}

			currTableName, err := mdl.GetModelTableNameInModelIfValid(mb.modelObj, designator)
			if err != nil {
				return db, err
			}

			// log.Println("*******join:", fmt.Sprintf("INNER JOIN \"%s\" ON \"%s\".%s_id = \"%s\".id",
			// currTableName, currTableName,
			// upperTableName, upperTableName))

			db = db.Joins(fmt.Sprintf("INNER JOIN \"%s\" ON \"%s\".%s_id = \"%s\".id",
				currTableName, currTableName,
				upperTableName, upperTableName))
		}
	}

	// There are still first-level queries that have no explicit join table
	for _, buildInfo := range mb.builderInfos {
		rel, err := buildInfo.builder.GetPredicateRelation()
		if err != nil {
			return db, err
		}

		if !DesignatorContainsDot(rel) { // where clause
			s, vals, err := rel.BuildQueryStringAndValues(mb.modelObj)
			if err != nil {
				return db, err
			}

			// db = db.Model(mb.modelObj).Where(s, vals...)
			db = db.Where(s, vals...)
		}
	}

	return db, nil
}

func (q *Query) buildQueryOrderOffSetAndLimit(db *gorm.DB, modelObj mdl.IModel) *gorm.DB {
	order := ""
	tableName := mdl.GetTableNameFromIModel(modelObj)
	if q.orderField != nil && q.order != nil {
		col, err := mdl.FieldNameToColumn(modelObj, *q.orderField)
		if err != nil {
			q.Err = err
		}

		order = fmt.Sprintf("\"%s\".%s %s", tableName, col, *q.order)
	} else {
		order = fmt.Sprintf("\"%s\".created_at DESC", tableName) // descending by default
	}

	db = db.Order(order)

	if q.offset != nil {
		db = db.Offset(*q.offset)
	}

	if q.limit != nil {
		db = db.Limit(*q.limit)
	}
	return db
}

// Create could be a mdl.IModel or *[]mdl.IModel
func (q *Query) Create(value interface{}) IQuery {
	q.Reset() // This shouldn't matter, unless it's a left-over bug
	defer resetWithoutResetError(q)
	db := q.db

	tableName := "nada"

	var err error
	var nested map[int]map[string][]mdl.IModel            // level->tablename->models
	var associations map[uuid.UUID]map[string][]uuid.UUID // parent_table_id -> embedded_table_name -> embedded_table_ids

	// Single modelObj
	if modelObj, ok := value.(mdl.IModel); ok {
		tableName = mdl.GetTableNameFromIModel(modelObj)
		if err := CheckPeggedFieldsHasNoExistingID(db, modelObj); err != nil {
			q.Err = err
			return q
		}

		nested, err = GrabAllNestedPeggedStructs(modelObj) // map[level int]map[tableName string][]mdl.IModel
		if err != nil {
			q.Err = err
			return q
		}

		associations, err = GrabFirstLevelAssociatedFields(modelObj)
		if err != nil {
			q.Err = err
			return q
		}
	} else {
		val := reflect.Indirect(reflect.ValueOf(value))
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			if val.Len() == 0 {
				return q
			}

			isPtr := false
			if val.Index(0).Kind() == reflect.Ptr {
				isPtr = true
			}

			if isPtr {
				modelObj = val.Index(0).Interface().(mdl.IModel)
			} else {
				modelObj = val.Index(0).Addr().Interface().(mdl.IModel)
			}
			tableName = mdl.GetTableNameFromIModel(modelObj)

			for j := 0; j < val.Len(); j++ {
				var nestedModel mdl.IModel
				if isPtr {
					nestedModel = val.Index(j).Interface().(mdl.IModel)
				} else {
					nestedModel = val.Index(j).Addr().Interface().(mdl.IModel)
				}

				// I wish this doesn't need extra db query..takes too much DB time
				if err := CheckPeggedFieldsHasNoExistingID(db, nestedModel); err != nil {
					q.Err = err
					return q
				}
			}

			associations, err = GrabFirstLevelAssociatedFieldsForSlice(value)
			if err != nil {
				q.Err = err
				return q
			}

			nested, err = GrabAllNestedPeggedStructsForSlice(value)
			if err != nil {
				q.Err = err
				return q
			}
		default:
			q.Err = fmt.Errorf("wrong type, should be mdl.IModel or a slice of mdl.IModel")
		}
	}

	// Skip all association, we create manually so we can differentitate between
	// pegg and pegassociate
	if err := db.Table(tableName).Omit(clause.Associations).Create(value).Error; err != nil {
		// if err := db.Table(tableName).Create(value).Error; err != nil {
		PrintFileAndLine(err)
		q.Err = err
		return q
	}

	// Now the model is created, create all pegged, higher level first
	for lvl := 0; lvl < len(nested)+1; lvl++ {
		for tableName, marr := range nested[lvl] {
			typ := reflect.TypeOf(reflect.ValueOf(marr[0]).Interface())
			// Gorm needs an array with the actual type, not other interface type
			arr := reflect.MakeSlice(reflect.SliceOf(typ), len(marr), len(marr))
			for i, item := range marr {
				arr.Index(i).Set(reflect.ValueOf(item))
			}
			arrPtr := arr.Interface()

			q.Err = q.db.Table(tableName).Omit(clause.Associations).Create(arrPtr).Error
			if q.Err != nil {
				PrintFileAndLine(q.Err)
			}
		}
	}

	// Now for pegassociated, create the associations
	// var associations map[uuid.UUID]map[string][]uuid.UUID  // parent_table_id -> embedded_table_name -> embedded_table_ids
	for parentTableID, tableName2IDs := range associations {
		for embeddedTableName, ids := range tableName2IDs {
			// If the associated is not found, error
			var c int64
			if q.Err = q.db.Table(embeddedTableName).Where(fmt.Sprintf("%s.id IN (?)", embeddedTableName), ids).Count(&c).Error; q.Err != nil {
				return q
			}

			if c != int64(len(ids)) {
				q.Err = fmt.Errorf("pegassoc record %s not found", embeddedTableName)
				return q
			}

			if q.Err = q.db.Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s.id IN (?)", embeddedTableName, tableName+"_id", embeddedTableName), parentTableID, ids).Error; q.Err != nil {
				return q
			}
		}
	}

	// For pegassociated, the since we expect association_autoupdate:false
	// need to manually create it
	// if err := CreatePeggedAssocFields(db, modelObj); err != nil {
	// 	q.Err = err
	// 	return q
	// }

	return q
}

// Delete can be with criteria, or can just delete the mdl directly
func (q *Query) Delete(value interface{}) IQuery {
	db := q.db

	if q.Err != nil {
		return q
	}

	// Single modelObj
	tableName := ""
	if modelObj, ok := value.(mdl.IModel); ok {
		if modelObj.GetID() == nil && q.mainMB == nil && len(q.mbs) == 0 {
			// You could delete every record in the database with Gormv1
			q.Err = errors.New("delete must have a modelID or include at least one PredicateRelationBuilder")
			return q
		}

		tableName = mdl.GetTableNameFromIModel(modelObj)

		// build query core is only allowed whe nvalue is IModel
		// so it's like some qeury and then Delete(&models.TopLevel{})
		if q.mainMB != nil {
			q.mainMB.modelObj = modelObj
		}
		// else {
		// 	db = db.Model(modelObj)
		// }

		// Won't work, builtqueryCore has "ORDER BY Clause"
		var err error
		db = db.Unscoped()
		db, err = q.buildQueryCore(db, modelObj)
		if err != nil {
			q.Err = err
			return q
		}

		// Rely on OnDelete foreign key cascade
		// if err := DeleteModelFixManyToManyAndPegAndPegAssoc(db, modelObj); err != nil {
		// 	q.Err = err
		// 	return q
		// }
	} else {
		val := reflect.Indirect(reflect.ValueOf(value))
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			if val.Len() == 0 {
				return q
			}

			isPtr := false
			if val.Index(0).Kind() == reflect.Ptr {
				isPtr = true
			}

			if isPtr {
				tableName = mdl.GetTableNameFromIModel(val.Index(0).Interface().(mdl.IModel))
			} else {
				tableName = mdl.GetTableNameFromIModel(val.Index(0).Addr().Interface().(mdl.IModel))
			}
		}
	}

	if err := db.Table(tableName).Delete(value).Error; err != nil {
		q.Err = err
		return q
	}

	return q
}

// Save is to update
func (q *Query) Save(modelObj mdl.IModel) IQuery {
	q.saveLck.Lock()
	defer q.saveLck.Unlock()

	defer resetWithoutResetError(q)
	if q.Err != nil {
		return q
	}

	// I need load the old model to compare to the new model.
	// Why? If the parent model has an array of nested model, and the parent model decided to remove one of the nested model,
	// then the newer version of the nested model would not have the nested model, and I need to pull it up to compare to know.
	// For pegged nested model, it means to delete it. For pegassoc, it means we need to remove the association.
	// For many-to-many, we need to remove the entry from the join table.

	oriModel := reflect.New(reflect.TypeOf(modelObj).Elem()).Interface().(mdl.IModel)
	names := mdl.GetEmbeddedTablePaths(oriModel)
	db2 := q.db

	db2 = db2.Preload(clause.Associations)
	for _, name := range names {
		db2 = db2.Preload(name)
	}

	if q.Err = db2.Where("id = ?", modelObj.GetID()).Take(&oriModel).Error; q.Err != nil {
		return q
	}

	// Nested model can be removed or created, so diff here
	// If removed, delete it, if pegassoc changed, add or remove link
	// If pegged, leave it untouched, we simply call save on each one later
	if q.Err = UpdateNestedFields(q.db, oriModel, modelObj); q.Err != nil {
		return q
	}

	// Instead of calling q.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(modelObj)
	// I want to update only the pegged only, so we save them ourselves.
	// TODO: save many of the same table at a time

	nested, err := GrabAllNestedPeggedStructs(modelObj) // map[level int]map[tableName string][]mdl.IModel
	if err != nil {
		q.Err = err
		return q
	}

	// Since pegged, higher level first
	for lvl := 0; lvl < len(nested)+1; lvl++ {
		for tableName, marr := range nested[lvl] {
			typ := reflect.TypeOf(reflect.ValueOf(marr[0]).Interface())
			// Gorm needs an array with the actual type, not other interface type
			arr := reflect.MakeSlice(reflect.SliceOf(typ), len(marr), len(marr))
			for i, item := range marr {
				arr.Index(i).Set(reflect.ValueOf(item))
			}
			arrPtr := arr.Interface()

			q.Err = q.db.Table(tableName).Save(arrPtr).Error
			if q.Err != nil {
				PrintFileAndLine(q.Err)
			}
		}
	}

	// Save the parent model
	// For embedded field calling Gorm Save() does the following:
	// INSERT INTO "sec_level_ptr_dog" ("id","created_at","updated_at","deleted_at","name","color","top_level_id") VALUES
	// ('c39853ec-21aa-46f0-ac38-e617b3196405','2023-03-30 10:51:26.484','2023-03-30 10:51:26.484',NULL,'NewBuddy',
	// 'black','c39863aa-e984-4c19-8920-7043c5ff76e8') ON CONFLICT ("id") DO UPDATE SET "top_level_id"="excluded"."top_level_id"
	// Pretty useless since we do not do full model save

	q.Err = q.db.Save(modelObj).Error
	if q.Err != nil {
		PrintFileAndLine(q.Err)
	}
	return q
}

// Update only allow the top-level update and top-level query.
// Even update the top level and querying the inner one is difficult.
// To update with a join:
// https://stackoverflow.com/questions/7869592/how-to-do-an-update-join-in-postgresql
// prob need to follow Nate Smith's solution.
// Very difficult, and some alias may be necessary
func (q *Query) Update(modelObj mdl.IModel, p *PredicateRelationBuilder) IQuery {
	// Gorm update has a full-association mode
	// db.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&user)
	// but we only want those that are pegged

	defer resetWithoutResetError(q)

	if q.Err != nil {
		return q
	}

	if q.mainMB != nil {
		q.mainMB.modelObj = modelObj
	}

	db := q.db

	// Won't work, builtqueryCore has "ORDER BY Clause"
	var err error
	db, err = q.buildQueryCore(db, modelObj)
	if err != nil {
		q.Err = err
		return q
	}

	updateMap := make(map[string]interface{})
	rel, err := p.GetPredicateRelation()
	if err != nil {
		q.Err = err
		return q
	}

	if DesignatorContainsDot(rel) { // hacky
		q.Err = fmt.Errorf("dot notation in update")
		PrintFileAndLine(q.Err)
		return q
	}
	// field2Struct, _ := FindFieldNameToStructAndStructFieldNameIfAny(rel) // hacky
	// if field2Struct != nil {
	// 	q.Err = fmt.Errorf("dot notation in update")
	// 	PrintFileAndLine(q.Err)
	// 	return q
	// }

	qstr, values, err := rel.BuildQueryStringAndValues(modelObj)
	if err != nil {
		q.Err = err
		return q
	}

	toks := strings.Split(qstr, " = ?")

	for i, tok := range toks[:len(toks)-1] { // last tok is anempty str
		s := strings.Split(tok, ".")[1] // strip away the table name
		updateMap[s] = values[i]
	}

	q.Err = db.Updates(updateMap).Error

	return q
}

func (q *Query) GetDB() *gorm.DB {
	return q.db
}

func (q *Query) Reset() IQuery {
	q.Err = nil
	resetWithoutResetError(q)
	return q
}

func (q *Query) Error() error {
	resetWithoutResetError(q)
	err := q.Err
	q.Err = nil
	return err
}

// ------------------
func SetQryLoggerWithConfig(db *gorm.DB, config logger.Config) {
	loggerc := qrylogger.New(log.New(os.Stderr, "", log.LstdFlags), config)
	db.Config.Logger = loggerc
}

func SetQryLogger(db *gorm.DB) {
	config := logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: false,
		Colorful:                  true,
	}

	loggerc := qrylogger.New(log.New(os.Stderr, "", log.LstdFlags), config)
	db.Config.Logger = loggerc
}

// ------------------

type TableAndArgs struct {
	TblName string // The table the predicate relation applies to, at this level (non-nested)
	Args    []interface{}
}

// hacky...
func FindFieldNameToStructAndStructFieldNameIfAny(rel *PredicateRelation) (*string, *string) {
	for _, pr := range rel.PredOrRels {
		if p, ok := pr.(*Predicate); ok {
			if strings.Contains(p.Field, ".") {
				toks := strings.Split(p.Field, ".")
				name := toks[len(toks)-2] // next to alst
				return &name, &toks[len(toks)-1]
			}
		}
		if rel2, ok := pr.(*PredicateRelation); ok {
			return FindFieldNameToStructAndStructFieldNameIfAny(rel2)
		}
	}
	return nil, nil
}

func DesignatorContainsDot(rel *PredicateRelation) bool {
	_, structFieldName := FindFieldNameToStructAndStructFieldNameIfAny(rel)
	return structFieldName != nil
}

func GetOuterTableName(modelObj mdl.IModel, fieldNameDesignator string) (string, error) {
	outerTableName := ""
	if strings.Contains(fieldNameDesignator, ".") {
		toks := strings.Split(fieldNameDesignator, ".")
		outerFieldNameToStruct := strings.Join(toks[:len(toks)-1], ".")
		typ2, err := mdl.GetModelFieldTypeInModelIfValid(modelObj, outerFieldNameToStruct)
		if err != nil {
			return "", err
		}
		outerTableName = mdl.GetTableNameFromType(typ2)
	} else {
		outerTableName = mdl.GetTableNameFromIModel(modelObj)
	}
	return outerTableName, nil
}

// --------------
func resetWithoutResetError(q *Query) {
	q.order = nil
	q.orderField = nil
	q.limit = nil
	q.offset = nil

	q.mbs = make([]ModelAndBuilder, 0)
	q.mainMB = nil
}
