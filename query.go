package qry

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/t2wu/qry/datatype"
	"github.com/t2wu/qry/mdl"

	"github.com/jinzhu/gorm"
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
			builder:   b,
			processed: false,
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
			builder:   b,
			processed: false,
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
	q.setLogger(db)
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
	q.setLogger(db)
	q.Err = db.First(modelObj).Error

	return q
}

func (q *Query) Count(modelObj mdl.IModel, no *int) IQuery {
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

	var err error
	db, err = q.buildQueryCore(db, modelObj)
	if err != nil {
		q.Err = err
		return q
	}

	db = q.buildQueryOrderOffSetAndLimit(db, modelObj)
	q.setLogger(db)
	q.Err = db.Find(modelObjs).Error

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
	db = buildPreload(db).Model(modelObj)

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

			db = db.Model(mb.modelObj).Where(s, vals...)
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

func (q *Query) Create(modelObj mdl.IModel) IQuery {
	q.Reset() // This shouldn't matter, unless it's a left-over bug
	defer resetWithoutResetError(q)
	db := q.db

	if err := RemoveIDForNonPegOrPeggedFieldsBeforeCreate(db, modelObj); err != nil {
		q.Err = err
		return q
	}

	q.setLogger(db)
	if err := db.Create(modelObj).Error; err != nil {
		PrintFileAndLine(err)
		q.Err = err
		return q
	}

	// For pegassociated, the since we expect association_autoupdate:false
	// need to manually create it
	if err := CreatePeggedAssocFields(db, modelObj); err != nil {
		q.Err = err
		return q
	}

	return q
}

func (q *Query) CreateMany(modelObjs []mdl.IModel) IQuery {
	q.Reset() // This shouldn't matter, unless it's a left-over bug
	defer resetWithoutResetError(q)
	db := q.db

	car := BatchCreateData{}
	car.toProcess = make(map[string][]mdl.IModel)

	// TODO: do a batch create instead
	for _, modelObj := range modelObjs {
		if err := RemoveIDForNonPegOrPeggedFieldsBeforeCreate(db, modelObj); err != nil {
			q.Err = err
			return q
		}

		q.Err = db.Create(modelObj).Error
		if q.Err != nil {
			PrintFileAndLine(q.Err)
			return q
		}

		// if err := gatherModelToCreate(reflect.ValueOf(modelObj).Elem(), &car); err != nil {
		// 	q.Err = err
		// 	return q
		// }

		// For pegassociated, the since we expect association_autoupdate:false
		// need to manually create it
		if err := CreatePeggedAssocFields(db, modelObj); err != nil {
			q.Err = err
			return q
		}
	}

	return q
}

// Delete can be with criteria, or can just delete the mdl directly
func (q *Query) Delete(modelObj mdl.IModel) IQuery {
	db := q.db

	if q.Err != nil {
		return q
	}

	if modelObj.GetID() == nil && q.mainMB == nil && len(q.mbs) == 0 {
		// You could delete every record in the database with Gormv1
		q.Err = errors.New("delete must have a modelID or include at least one PredicateRelationBuilder")
		return q
	}

	if q.mainMB != nil {
		q.mainMB.modelObj = modelObj
	} else {
		db = db.Model(modelObj)
	}

	// Won't work, builtqueryCore has "ORDER BY Clause"
	var err error
	db = db.Unscoped()
	db, err = q.buildQueryCore(db, modelObj)
	if err != nil {
		q.Err = err
		return q
	}

	q.setLogger(db)
	if err := db.Delete(modelObj).Error; err != nil {
		q.Err = err
		return q
	}

	if err := DeleteModelFixManyToManyAndPegAndPegAssoc(db, modelObj); err != nil {
		q.Err = err
		return q
	}

	return q
}

func (q *Query) DeleteMany(modelObjs []mdl.IModel) IQuery {
	q.Reset() // needed only if left-over bug
	defer resetWithoutResetError(q)
	db := q.db

	// Collect all the ids, non can be nil
	ids := make([]*datatype.UUID, len(modelObjs))
	for i, modelObj := range modelObjs {
		ids[i] = modelObj.GetID()
		if modelObj.GetID() == nil {
			q.Err = errors.New("modelObj to delete cannot have an ID of nil")
			return q
		}
	}

	m := reflect.New(reflect.TypeOf(modelObjs[0]).Elem()).Interface().(mdl.IModel)
	// Batch delete, not documented for Gorm v1 but actually works
	q.setLogger(db)
	if q.Err = db.Unscoped().Delete(m, ids).Error; q.Err != nil {
		return q
	}

	for _, modelObj := range modelObjs {
		if err := DeleteModelFixManyToManyAndPegAndPegAssoc(db, modelObj); err != nil {
			q.Err = err
			return q
		}
	}

	return q
}

func (q *Query) Save(modelObj mdl.IModel) IQuery {
	q.saveLck.Lock()
	defer q.saveLck.Unlock()

	defer resetWithoutResetError(q)
	if q.Err != nil {
		return q
	}

	q.Err = q.db.Save(modelObj).Error
	if q.Err != nil {
		PrintFileAndLine(q.Err)
	}
	return q
}

// Update only allow one level of builder
func (q *Query) Update(modelObj mdl.IModel, p *PredicateRelationBuilder) IQuery {
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

	field2Struct, _ := FindFieldNameToStructAndStructFieldNameIfAny(rel) // hacky
	if field2Struct != nil {
		q.Err = fmt.Errorf("dot notation in update")
		PrintFileAndLine(q.Err)
		return q
	}

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

	q.setLogger(db)
	q.Err = db.Update(updateMap).Error

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

func (q *Query) setLogger(db *gorm.DB) {
	_, filepath, line, ok := runtime.Caller(2)
	var source string
	if ok {
		source = fmt.Sprintf("%s:%d", filepath, line)
	}
	db.SetLogger(NewLogger(source))
}

// ------------------

type TableAndArgs struct {
	TblName string // The table the predicate relation applies to, at this level (non-nested)
	Args    []interface{}
}

func buildPreload(tx *gorm.DB) *gorm.DB {
	return tx.Set("gorm:auto_preload", true)
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
