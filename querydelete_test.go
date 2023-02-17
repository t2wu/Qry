package qry

import (
	"errors"
	"testing"

	"github.com/t2wu/qry/datatype"
	"github.com/t2wu/qry/mdl"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestDelete_PeggedStruct_ShouldBeDeleted(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	doguuid1 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "UnderTest",
		Age:       1,
		EmbedDog: SecLevelEmbedDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Name:      "Buddy",
			Color:     "black",
		},
	}

	if err := DB(tx).Create(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	if err := DB(tx).Delete(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	err := Q(tx, C("ID =", doguuid1)).Find(&SecLevelEmbedDog{}).Error()
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	}
}

func TestDelete_PeggedStructPtr_ShouldBeDeleted(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	doguuid1 := datatype.NewUUID()

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "first",
		Age:       1,
		PtrDog: &SecLevelPtrDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Name:      "Buddy",
			Color:     "black",
		},
	}

	if err := DB(tx).Create(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	if err := DB(tx).Delete(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	err := Q(tx, C("ID =", doguuid1)).First(&SecLevelPtrDog{}).Error()
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	}
}

func TestDelete_PeggedAssocStruct_ShouldLeftIntact(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	catuuid1 := datatype.NewUUID()

	cat := SecLevelEmbedCat{
		BaseModel: mdl.BaseModel{ID: &catuuid1},
		Name:      "Buddy",
		Color:     "black",
	}

	if err := DB(tx).Create(&cat).Error(); !assert.Nil(t, err) {
		return
	}

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "first",
		Age:       1,
		EmbedCat:  cat,
	}

	if err := DB(tx).Create(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	if err := DB(tx).Delete(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	searched := SecLevelEmbedCat{}
	err := Q(tx, C("ID =", catuuid1)).First(&searched).Error()
	assert.Nil(t, err)
	assert.Nil(t, searched.TopLevelID)
}

func TestDelete_PeggedAssocStructPtr_ShouldLeftIntact(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	catuuid1 := datatype.NewUUID()

	cat := SecLevelPtrCat{
		BaseModel: mdl.BaseModel{ID: &catuuid1},
		Name:      "Buddy",
		Color:     "black",
	}

	if err := DB(tx).Create(&cat).Error(); !assert.Nil(t, err) {
		return
	}

	tm1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "first",
		Age:       1,
		PtrCat:    &cat,
	}

	if err := DB(tx).Create(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	if err := DB(tx).Delete(&tm1).Error(); !assert.Nil(t, err) {
		return
	}

	searched := SecLevelPtrCat{}
	err := Q(tx, C("ID =", catuuid1)).First(&searched).Error()
	assert.Nil(t, err)
	assert.Nil(t, searched.TopLevelID)
}

func TestBatchDelete_PeggedArray_ShouldNotBeFound(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	doguuid1 := datatype.NewUUID()
	doguuid2 := datatype.NewUUID()

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel1",
		Dogs: []SecLevelArrDog{
			{
				BaseModel: mdl.BaseModel{ID: &doguuid1},
				Name:      "Buddy",
				Color:     "black",
			},
		},
	}
	testModel2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "TestModel2",
		Dogs: []SecLevelArrDog{
			{
				BaseModel: mdl.BaseModel{ID: &doguuid2},
				Name:      "Happy",
				Color:     "red",
			},
		},
	}

	tms := []TopLevel{testModel1, testModel2}

	err := DB(tx).Create(&tms).Delete(&tms).Error() // used to be CreateMany, DeleteMany
	if assert.Nil(t, err) {
		searched := make([]TopLevel, 0)
		err := Q(tx, C("ID IN", []uuid.UUID{u1, u2})).Find(&searched).Error()
		assert.Nil(t, err)
		assert.Equal(t, 0, len(searched))
		assert.Len(t, searched, 0)

		dogSearched := make([]SecLevelArrDog, 0)
		err = Q(tx, C("ID IN", []uuid.UUID{u1, u2})).Find(&dogSearched).Error()

		assert.Nil(t, err)
		assert.Equal(t, 0, len(dogSearched))
	}
}

func TestBatchDelete_PegAssocArray_ShouldLeaveItIntact(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	catuuid1 := datatype.NewUUID()
	catuuid2 := datatype.NewUUID()

	cat1 := SecLevelArrCat{
		BaseModel: mdl.BaseModel{ID: &catuuid1},
		Name:      "Kiddy1",
		Color:     "black",
	}

	cat2 := SecLevelArrCat{
		BaseModel: mdl.BaseModel{ID: &catuuid2},
		Name:      "Kiddy2",
		Color:     "black",
	}

	err := DB(tx).Create(&[]SecLevelArrCat{cat1, cat2}).Error() // used to be CreateMany
	if !assert.Nil(t, err) {
		return
	}

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel1",
		Cats:      []SecLevelArrCat{cat1},
	}
	testModel2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "TestModel2",
		Cats:      []SecLevelArrCat{cat2},
	}

	tms := []TopLevel{testModel1, testModel2}

	err = DB(tx).Create(&tms).Delete(&tms).Error() // used to be CreateMany, DeleteMany
	if assert.Nil(t, err) {
		searched := make([]TopLevel, 0)
		err := Q(tx, C("ID IN", []uuid.UUID{u1, u2})).Find(&searched).Error()
		assert.Nil(t, err)
		assert.Equal(t, 0, len(searched))
		assert.Len(t, searched, 0)

		catSearched := make([]SecLevelArrCat, 0)
		err = Q(tx, C("ID IN", []uuid.UUID{catuuid1, catuuid2})).Find(&catSearched).Error()

		assert.Nil(t, err)
		if assert.Equal(t, 2, len(catSearched)) {
			assert.Nil(t, catSearched[0].TopLevelID)
			assert.Nil(t, catSearched[1].TopLevelID)
		}
	}
}

func TestDelete_PeggedArray_ShouldRemoveAllNestedFields(t *testing.T) {
	u1 := datatype.NewUUID()
	doguuid := datatype.NewUUID()
	dogtoyuui := datatype.NewUUID()
	tm := TopLevel{BaseModel: mdl.BaseModel{
		ID: &u1},
		Name: "MyTestModel",
		Age:  1,
		Dogs: []SecLevelArrDog{
			{
				BaseModel: mdl.BaseModel{
					ID: &doguuid,
				},
				Name:  "Buddy",
				Color: "black",
				DogToys: []ThirdLevelArrDogToy{
					{
						BaseModel: mdl.BaseModel{ID: &dogtoyuui},
					},
				},
			},
		},
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	// Test delete by itself
	if err := DB(tx).Delete(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	loadedTestModel := TopLevel{}
	err := Q(tx, C("ID =", u1)).First(&loadedTestModel).Error()
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	}

	loadedDogModel := SecLevelArrDog{}
	err = Q(tx, C("ID =", doguuid)).First(&loadedDogModel).Error()
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	}

	dogToy := ThirdLevelArrDogToy{}
	err = Q(tx, C("ID =", dogtoyuui)).First(&dogToy).Error()
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	}
}

func TestDelete_PeggedAssoc_ShouldLeaveItIntact(t *testing.T) {
	catuuid := datatype.NewUUID()
	cat := SecLevelArrCat{
		BaseModel: mdl.BaseModel{ID: &catuuid},
		Name:      "Buddy",
		Color:     "black",
	}

	tx := db.Begin()
	defer tx.Rollback()

	err := Q(tx).Create(&cat).Error()
	if !assert.Nil(t, err) {
		return
	}

	u1 := datatype.NewUUID()
	tm := TopLevel{BaseModel: mdl.BaseModel{ID: &u1},
		Name: "MyTestModel",
		Age:  1,
		Cats: []SecLevelArrCat{cat},
	}

	err = Q(tx).Create(&tm).Delete(&tm).Error()
	if !assert.Nil(t, err) {
		return
	}

	loadedTestModel := TopLevel{}
	err = Q(tx, C("ID =", u1)).First(&loadedTestModel).Error()
	if assert.Error(t, err) {
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	}

	loadedCatModel := SecLevelArrCat{}
	err = Q(tx, C("ID =", catuuid)).First(&loadedCatModel).Error()
	if assert.Nil(t, err) {
		assert.Equal(t, catuuid, *loadedCatModel.GetID())
		assert.Nil(t, loadedCatModel.TopLevelID)
	}
}

func TestDelete_criteria_works(t *testing.T) {
	id1 := datatype.NewUUID()
	id2 := datatype.NewUUID()
	id3 := datatype.NewUUID()
	tm1 := TopLevel{BaseModel: mdl.BaseModel{ID: &id1}, Name: "MyTestModel", Age: 1}
	tm2 := TopLevel{BaseModel: mdl.BaseModel{ID: &id2}, Name: "MyTestModel", Age: 1}
	tm3 := TopLevel{BaseModel: mdl.BaseModel{ID: &id3}, Name: "MyTestModel", Age: 3}

	tx := db.Begin()
	defer tx.Rollback()

	err := Q(tx).Create(&tm1).Create(&tm2).Create(&tm3).Error()
	if !assert.Nil(t, err) {
		return
	}

	tms := make([]TopLevel, 0)
	err = Q(tx, C("Name =", "MyTestModel")).Find(&tms).Error()
	if !assert.Nil(t, err) {
		return
	}

	assert.Equal(t, 3, len(tms), "initial condition should be 3")

	err = Q(tx, C("Age =", 3).And("Name =", "MyTestModel")).Delete(&TopLevel{}).Error()
	if !assert.Nil(t, err) {
		return
	}

	tms = make([]TopLevel, 0)
	if err := Q(tx, C("Name =", "MyTestModel")).Find(&tms).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	if assert.Equal(t, 2, len(tms), "Should still have 2 left after one is deleted") {
		assert.Equal(t, id2, *tms[0].ID)
		assert.Equal(t, id1, *tms[1].ID)
	}

	tms = make([]TopLevel, 0)
	err = Q(tx, C("Age =", 3).And("Name =", "same")).Find(&tms).Error()
	if assert.Nil(t, err) {
		return
	}

	assert.Equal(t, 1, len(tms), "The one in setup() should still be left intact")
}
