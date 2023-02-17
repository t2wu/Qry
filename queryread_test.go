package qry

import (
	"errors"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/t2wu/qry/datatype"
	"github.com/t2wu/qry/mdl"
	"gorm.io/gorm"
)

// func TestQueryFirst_ByID(t *testing.T) {
// 	tm := TopLevel{}
// 	u1 := datatype.NewUUID()
// 	if err := db.Preload("FavoriteDog.DogToys").Preload("EvilDog.DogToys").Preload(clause.Associations).Where("id = ?", u1).First(&tm).Error; err != nil {
// 		assert.Fail(t, err.Error())
// 		return
// 	}

// 	assert.Len(t, tm.FavoriteDog.DogToys, 1)
// 	assert.Equal(t, "second", tm.Name)
// 	assert.Fail(t, "on purpose")
// }

func TestQueryFirst_ByWrongID_ShouldNotBeFoundAndGiveError(t *testing.T) {
	u1 := datatype.NewUUID()
	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
	}

	tx := db.Begin()
	defer tx.Rollback()
	if err := Q(tx).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	// whether it's a pointer or the struct itself, gorm should handle it just fine right?
	u2 := datatype.NewUUID()
	if err := Q(tx, C("ID =", u2)).First(&tm).Error(); err != nil {
		assert.Error(t, err)
		return
	}
	assert.Fail(t, "should not be found")
}

func TestQueryFirst_ByOneIntField(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 3,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 1,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create([]TopLevel{tm1, tm2}).Error(); !assert.Nil(t, err) {
		return
	}

	tests := []struct {
		query string
		val   int
		want  uuid.UUID
	}{
		{"Age =", 3, u1},
		{"Age =", 1, u2},
	}

	for _, test := range tests {
		tm := TopLevel{}
		if err := Q(tx, C(test.query, test.val)).First(&tm).Error(); err != nil {
			assert.Fail(t, err.Error(), "record not found")
			return
		}
		assert.Equal(t, test.want, *tm.ID)
	}
}

func TestQueryFirst_ByOneStringField(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 3,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "first", Age: 1,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2}).Error(); !assert.Nil(t, err) {
		return
	}

	tests := []struct {
		query string
		val   string
		want  uuid.UUID
	}{
		{"Name =", "same", u1},
		{"Name =", "first", u2},
	}

	for _, test := range tests {
		tm := TopLevel{}
		if err := Q(tx, C(test.query, test.val)).First(&tm).Error(); err != nil {
			assert.Fail(t, err.Error(), "record not found")
			return
		}
		assert.Equal(t, test.want, *tm.ID)
	}
}

func TestQueryFirst_ByBothStringAndIntField(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "first", Age: 3,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "second", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2}).Error(); !assert.Nil(t, err) {
		return
	}

	searched1 := TopLevel{}
	if err := Q(tx, C("Name =", "first").And("Age =", 3)).First(&searched1).Error(); err != nil {
		assert.Fail(t, err.Error(), "record not found")
		return
	}
	assert.Equal(t, u1, *searched1.ID)

	searched2 := TopLevel{}
	if err := Q(tx, C("Name =", "second").And("Age =", 3)).First(&searched2).Error(); err != nil {
		assert.Fail(t, err.Error(), "record not found")
		return
	}
	assert.Equal(t, u2, *searched2.ID)
}

func TestQuery_ByDB_ThenQ_Works(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "first", Age: 3,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "second", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2}).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := DB(tx).Q(C("Name =", "second").And("Age =", 3)).First(&searched).Error(); err != nil {
		assert.Fail(t, err.Error(), "record not found")
	}
	assert.Equal(t, u2, *searched.ID)
}

func TestQueryFirst_ByWrongValue_NotFoundShouldGiveError(t *testing.T) {
	u1 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "first", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(db, C("Name =", "tim")).First(&searched).Error(); err != nil {
		assert.Equal(t, true, errors.Is(err, gorm.ErrRecordNotFound))
		return
	}

	assert.Fail(t, "should not be found")
}

func TestQueryFirst_ByNonExistingFieldName_ShouldGiveAnError(t *testing.T) {
	u1 := datatype.NewUUID()
	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
	}

	tx := db.Begin()
	defer tx.Rollback()
	if err := Q(tx).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}

	if err := Q(db, C("deleteCmdForExample =", "same")).First(&searched).Error(); err != nil {
		assert.NotNil(t, err)
		return
	}

	assert.Fail(t, "should not be here")
}

func TestQueryFirst_ByNonExistingOperator_ShouldGiveAnError(t *testing.T) {
	u1 := datatype.NewUUID()
	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
	}

	tx := db.Begin()
	defer tx.Rollback()
	if err := Q(tx).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}

	if err := Q(db, C("Name WrongOp", "same")).First(&searched).Error(); err != nil {
		assert.Error(t, err)
		return
	}

	assert.Fail(t, "should not be here")
}

func TestQueryFind_ShouldGiveMultiple(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2, tm3}).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)
	if err := Q(tx, C("Name =", "same")).Find(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	if assert.Equal(t, 3, len(searched)) {
		assert.Contains(t, []uuid.UUID{u1, u2, u3}, *searched[0].ID)
		assert.Contains(t, []uuid.UUID{u1, u2, u3}, *searched[1].ID)
		assert.Contains(t, []uuid.UUID{u1, u2, u3}, *searched[2].ID)
	}
}

func TestQueryFindOffset_ShouldBeCorrect(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&tm1).Create(&tm2).Create(&tm3).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)

	if err := Q(tx, C("Name =", "same")).Offset(1).Find(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	if assert.Equal(t, 2, len(searched)) {
		// Contain the two oldest
		assert.Contains(t, []uuid.UUID{u1, u2}, *searched[0].ID)
		assert.Contains(t, []uuid.UUID{u1, u2}, *searched[1].ID)
	}
}

func TestQueryFindLimit_ShouldBeCorrect(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&tm1).Create(&tm2).Create(&tm3).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)
	if err := Q(tx, C("Name =", "same")).Limit(2).Find(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	if assert.Equal(t, 2, len(searched)) {
		// Contain the two oldest
		assert.Contains(t, []uuid.UUID{u2, u3}, *searched[0].ID)
		assert.Contains(t, []uuid.UUID{u2, u3}, *searched[1].ID)
	}
}

func TestQueryFindOrderBy_ShouldBeCorrect(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&tm1).Create(&tm2).Create(&tm3).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)

	if err := Q(tx, C("Name =", "same")).Order("CreatedAt", OrderAsc).Find(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	if assert.Equal(t, 3, len(searched)) {
		assert.Equal(t, u1, *searched[0].ID)
		assert.Equal(t, u2, *searched[1].ID)
		assert.Equal(t, u3, *searched[2].ID)
	}

	searched = make([]TopLevel, 0)
	if err := Q(tx, C("Name =", "same")).Order("CreatedAt", OrderDesc).Find(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	if assert.Equal(t, 3, len(searched)) {
		assert.Equal(t, u3, *searched[0].ID)
		assert.Equal(t, u2, *searched[1].ID)
		assert.Equal(t, u1, *searched[2].ID)
	}
}

func TestQueryFindOrderBy_BogusFieldShouldHaveError(t *testing.T) {
	u1 := datatype.NewUUID()
	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
	}

	tx := db.Begin()
	defer tx.Rollback()
	if err := Q(tx).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)

	// Currently order works not by field.
	if err := Q(db, C("Name =", "same")).Order("Bogus", OrderAsc).Find(&searched).Error(); err != nil {
		assert.Error(t, err)
		return
	}

	assert.Fail(t, "should not be here")
}

func TestQueryFind_WhenGivenSliceAndNotFound_ShouldNotGiveAnError(t *testing.T) {
	searched := make([]TopLevel, 0)

	err := Q(db, C("Name =", "Greg")).Find(&searched).Error()
	assert.Nil(t, err)

	assert.Equal(t, 0, len(searched))
}

func TestQueryFind_WithoutCriteria_ShouldGetAll(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create([]TopLevel{tm1, tm2, tm3}).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)

	if err := Q(tx).Find(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	if assert.Equal(t, 3, len(searched)) {
		assert.Contains(t, []uuid.UUID{u1, u2, u3}, *searched[0].ID)
		assert.Contains(t, []uuid.UUID{u1, u2, u3}, *searched[1].ID)
		assert.Contains(t, []uuid.UUID{u1, u2, u3}, *searched[2].ID)
	}
}

func TestQueryCount_ShouldBeCorrect(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "diff", Age: 2,
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2, tm3}).Error(); !assert.Nil(t, err) {
		return
	}

	var count int64

	if err := Q(tx, C("Name =", "same")).Count(&TopLevel{}, &count).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, int64(2), count)
}

// -------------------

func TestQueryFirst_Nested_Query(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TL1",
		Age:       1,
		EmbedDog: SecLevelEmbedDog{
			Name:  "buddy",
			Color: "black",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "DogToySameName",
				},
			},
		},
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "TL2",
		Age:       2,
		EmbedDog: SecLevelEmbedDog{
			Name:  "bull",
			Color: "gray",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "stick",
				},
			},
		},
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "TL3",
		Age:       3,
		EmbedDog: SecLevelEmbedDog{
			Name:  "happy",
			Color: "white",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "DogToySameName",
				},
			},
		},
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2, tm3}).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}

	err := Q(tx, C("EmbedDog.Name =", "bull")).First(&searched).Error()
	if assert.Nil(t, err) {
		assert.Equal(t, u2, *searched.ID)
	}
}

func TestFirst_NestedQueryWithInnerJoinWithCriteriaOnMainTable_Works(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
		EmbedDog: SecLevelEmbedDog{
			Name:  "buddy",
			Color: "black",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "DogToySameName",
				},
			},
		},
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
		EmbedDog: SecLevelEmbedDog{
			Name:  "bull",
			Color: "gray",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "stick",
				},
			},
		},
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
		EmbedDog: SecLevelEmbedDog{
			Name:  "happy",
			Color: "white",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "DogToySameName",
				},
			},
		},
	}

	unid1 := datatype.NewUUID()
	unid2 := datatype.NewUUID()
	unid3 := datatype.NewUUID()
	un1 := Unnested{
		BaseModel: mdl.BaseModel{ID: &unid1},
		Name:      "unnested1",
		UnnestedInner: UnnestedInner{
			Name: "unnested_same_name1&3",
		},
		TopLevelID: &u1,
	}
	un2 := Unnested{
		BaseModel: mdl.BaseModel{ID: &unid2},
		Name:      "unnested2",
		UnnestedInner: UnnestedInner{
			Name: "unnested_other_name",
		},
		TopLevelID: &u1,
	}
	un3 := Unnested{
		BaseModel: mdl.BaseModel{ID: &unid3},
		Name:      "unnested3",
		UnnestedInner: UnnestedInner{
			Name: "unnested_same_name1&3",
		},
		TopLevelID: &u3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2, tm3}).Error(); !assert.Nil(t, err) {
		return
	}

	if err := Q(tx).Create(&[]Unnested{un1, un2, un3}).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)

	if err := Q(tx, C("EmbedDog.Name =", "buddy")).InnerJoin(&Unnested{}, &TopLevel{}, C("UnnestedInner.Name =", "unnested_same_name1&3")).Find(&searched).Error(); !assert.Nil(t, err) {
		return
	}

	if assert.Len(t, searched, 1) {
		assert.Equal(t, u1, *searched[0].ID)
	}
}

func TestQueryFind_ThirdLevelEmbedNested_Query(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
		EmbedDog: SecLevelEmbedDog{
			Name:  "buddy",
			Color: "black",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "DogToySameName",
				},
			},
		},
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
		EmbedDog: SecLevelEmbedDog{
			Name:  "bull",
			Color: "gray",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "stick",
				},
			},
		},
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
		EmbedDog: SecLevelEmbedDog{
			Name:  "happy",
			Color: "white",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "DogToySameName",
				},
			},
		},
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2, tm3}).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)

	err := Q(tx, C("EmbedDog.DogToys.ToyName =", "DogToySameName")).Find(&searched).Error()
	if assert.Nil(t, err) && assert.Equal(t, 2, len(searched)) {
		assert.Contains(t, []uuid.UUID{u1, u3}, *searched[0].ID)
	}
}

func TestFirst_InnerJoin_Works(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
		EmbedDog: SecLevelEmbedDog{
			Name:  "buddy",
			Color: "black",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "DogToySameName",
				},
			},
		},
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
		EmbedDog: SecLevelEmbedDog{
			Name:  "bull",
			Color: "gray",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "stick",
				},
			},
		},
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
		EmbedDog: SecLevelEmbedDog{
			Name:  "happy",
			Color: "white",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					ToyName: "DogToySameName",
				},
			},
		},
	}

	unid1 := datatype.NewUUID()
	un1 := Unnested{
		BaseModel: mdl.BaseModel{ID: &unid1},
		Name:      "unnested1",
		UnnestedInner: UnnestedInner{
			Name: "unnested_inner1",
		},
		TopLevelID: &u1,
	}
	unid2 := datatype.NewUUID()
	un2 := Unnested{
		BaseModel: mdl.BaseModel{ID: &unid2},
		Name:      "unnested2",
		UnnestedInner: UnnestedInner{
			Name: "unnested_inner2",
		},
		TopLevelID: &u2,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2, tm3}).Error(); !assert.Nil(t, err) {
		return
	}

	if err := Q(tx).Create(&[]Unnested{un1, un2}).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}

	err := Q(tx).InnerJoin(&Unnested{}, &TopLevel{}, C("Name =", "unnested2")).First(&searched).Error()
	assert.Nil(t, err)
	if err == nil {
		assert.Equal(t, u2, *searched.ID)
	}
}

func TestFind_UnnestedQueryWithInnerJoin_Works(t *testing.T) {
	// Unnested, but has reference to top level ID

	// err := db.Model(&TopLevel{}).Joins("inner join unnested on unnested.top_level_id = top_level.id").
	// 	Joins("inner join unnested_inner ON unnested_inner.unnested_id = unnested.id").Where("unnested_inner.name = ?", "UnNestedInnerSameNameWith1&2").
	// 	Order("created_at DESC").
	// 	Find(&tms).Error

	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "same", Age: 1,
	}
	tm2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "same", Age: 2,
	}
	tm3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u3},
		Name:      "same", Age: 3,
	}

	unid1 := datatype.NewUUID()
	unid2 := datatype.NewUUID()
	unid3 := datatype.NewUUID()
	un1 := Unnested{
		BaseModel: mdl.BaseModel{ID: &unid1},
		Name:      "unnested1",
		UnnestedInner: UnnestedInner{
			Name: "unnested_same_name1&3",
		},
		TopLevelID: &u1,
	}
	un2 := Unnested{
		BaseModel: mdl.BaseModel{ID: &unid2},
		Name:      "unnested2",
		UnnestedInner: UnnestedInner{
			Name: "unnested_other_name",
		},
		TopLevelID: &u2,
	}
	un3 := Unnested{
		BaseModel: mdl.BaseModel{ID: &unid3},
		Name:      "unnested3",
		UnnestedInner: UnnestedInner{
			Name: "unnested_same_name1&3",
		},
		TopLevelID: &u3,
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&[]TopLevel{tm1, tm2, tm3}).Error(); !assert.Nil(t, err) {
		return
	}

	if err := Q(tx).Create(&[]Unnested{un1, un2, un3}).Error(); !assert.Nil(t, err) {
		return
	}

	searched := make([]TopLevel, 0)
	if err := Q(tx).InnerJoin(&Unnested{}, &TopLevel{}, C("UnnestedInner.Name =", "unnested_same_name1&3")).
		Find(&searched).Error(); !assert.Nil(t, err) {
		return
	}
	if assert.Len(t, searched, 2) {
		assert.Contains(t, []uuid.UUID{u1, u3}, *searched[0].ID)
		assert.Contains(t, []uuid.UUID{u1, u3}, *searched[1].ID)
	}
}
