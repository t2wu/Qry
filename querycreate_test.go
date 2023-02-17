package qry

import (
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/t2wu/qry/datatype"
	"github.com/t2wu/qry/mdl"

	"github.com/stretchr/testify/assert"
)

func TestCreate_PeggedArray(t *testing.T) {
	u1 := datatype.NewUUID()
	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "MyTestModel",
		Age:       3,
		Dogs: []SecLevelArrDog{
			{
				Name:  "Buddy",
				Color: "black",
			},
		},
	}
	tx := db.Begin()
	defer tx.Rollback()

	err := Q(tx).Create(&tm).Error()
	if !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, u1, *searched.ID)
	if assert.Equal(t, 1, len(searched.Dogs)) {
		assert.Equal(t, "Buddy", searched.Dogs[0].Name)
		assert.Equal(t, "black", searched.Dogs[0].Color)
	}
}

func TestCreate_PegAssocArray_ShouldAssociateCorrectly(t *testing.T) {
	// First create a cat, and while creating TopLevel, associate it with the cat
	// Then, when you load it, you should see the cat
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
	tm := TopLevel{BaseModel: mdl.BaseModel{
		ID: &u1},
		Name: "MyTestModel",
		Age:  1,
		Cats: []SecLevelArrCat{cat},
	}

	err = Q(tx).Create(&tm).Error()
	if !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, u1, *searched.ID)
	if assert.Equal(t, 1, len(searched.Cats)) { // should be associated
		assert.Equal(t, catuuid, *searched.Cats[0].ID)
		assert.Equal(t, "Buddy", searched.Cats[0].Name)
		assert.Equal(t, "black", searched.Cats[0].Color)
	}
}

func TestCreate_PeggedStruct(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	doguuid1 := datatype.NewUUID()

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel1",
		EmbedDog: SecLevelEmbedDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Name:      "Buddy",
			Color:     "black",
		},
	}

	err := DB(tx).Create(&testModel1).Error()
	if !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	err = Q(tx, C("ID =", u1)).First(&searched).Error()
	if !assert.Nil(t, err) {
		return
	}

	assert.Equal(t, u1, *searched.ID)
	assert.Equal(t, doguuid1, *searched.EmbedDog.GetID())
}

func TestCreate_PeggedStructPtr(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	doguuid1 := datatype.NewUUID()

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel1",
		PtrDog: &SecLevelPtrDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Name:      "Buddy",
			Color:     "black",
		},
	}

	err := DB(tx).Create(&testModel1).Error()
	if !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	err = Q(tx, C("ID =", u1)).First(&searched).Error()
	if !assert.Nil(t, err) {
		return
	}

	assert.Equal(t, u1, *searched.ID)
	assert.Equal(t, doguuid1, *searched.PtrDog.GetID())
}

func TestCreate_PeggedAssocStruct(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	catuuid1 := datatype.NewUUID()
	catuuid2 := datatype.NewUUID()

	cat := SecLevelEmbedCat{
		BaseModel: mdl.BaseModel{ID: &catuuid1},
		Name:      "Kiddy",
		Color:     "black",
	}

	// Unrelated at shouldn't be affected
	unrelatedCat := SecLevelEmbedCat{
		BaseModel: mdl.BaseModel{ID: &catuuid2},
		Name:      "Kiddy",
		Color:     "black",
	}

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel1",
		EmbedCat:  cat,
	}

	if err := DB(tx).Create(&unrelatedCat).Error(); !assert.Nil(t, err) {
		return
	}

	err := DB(tx).Create(&cat).Create(&testModel1).Error()
	if !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	err = Q(tx, C("ID =", u1)).First(&searched).Error()
	if !assert.Nil(t, err) {
		return
	}

	assert.Equal(t, u1, *searched.ID)
	assert.Equal(t, catuuid1, *searched.EmbedCat.GetID())

	// Unrelated at shouldn't be affected
	othercat := SecLevelEmbedCat{}
	err = Q(tx, C("ID =", catuuid2)).First(&othercat).Error()
	if !assert.Nil(t, err) {
		return
	}
	assert.Equal(t, catuuid2, *othercat.GetID())
	assert.Nil(t, othercat.TopLevelID)
}

func TestCreate_PeggedAssocStructPtr(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	catuuid1 := datatype.NewUUID()
	catuuid2 := datatype.NewUUID()

	relatedCat := SecLevelPtrCat{
		BaseModel: mdl.BaseModel{ID: &catuuid1},
		Name:      "Kiddy",
		Color:     "black",
	}

	// Unrelated at shouldn't be affected
	unrelatedCat := SecLevelPtrCat{
		BaseModel: mdl.BaseModel{ID: &catuuid2},
		Name:      "Kiddy",
		Color:     "black",
	}

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel1",
		PtrCat:    &relatedCat,
	}

	err := DB(tx).Create(&relatedCat).Create(&unrelatedCat).Create(&testModel1).Error()
	if !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	err = Q(tx, C("ID =", u1)).First(&searched).Error()
	if !assert.Nil(t, err) {
		return
	}

	assert.Equal(t, u1, *searched.ID)
	if assert.NotNil(t, searched.PtrCat) {
		assert.Equal(t, catuuid1, *searched.PtrCat.GetID())
	}

	// Unrelated at shouldn't be affected
	othercat := SecLevelPtrCat{}
	err = Q(tx, C("ID =", catuuid2)).First(&othercat).Error()
	if !assert.Nil(t, err) {
		return
	}
	assert.Equal(t, catuuid2, *othercat.GetID())
	assert.Nil(t, othercat.TopLevelID)
}

func TestCreate_PeggedArray_WithExistingID_ShouldGiveAnError(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	doguuid := datatype.NewUUID()
	tm1 := TopLevel{BaseModel: mdl.BaseModel{
		ID: &u1},
		Name: "MyTestModel",
		Age:  1,
		Dogs: []SecLevelArrDog{
			{
				BaseModel: mdl.BaseModel{ID: &doguuid},
				Name:      "Buddy",
				Color:     "black",
			},
		},
	}

	err := Q(tx).Create(&tm1).Error()
	if !assert.Nil(t, err) {
		return
	}

	tm2 := TopLevel{BaseModel: mdl.BaseModel{
		ID: &u1},
		Name: "MyTestModel",
		Age:  1,
		Dogs: []SecLevelArrDog{
			{
				BaseModel: mdl.BaseModel{ID: &doguuid},
				Name:      "Buddy",
				Color:     "black",
			},
		},
	}

	err = Q(tx).Create(&tm2).Error()
	assert.Error(t, err)
}

func TestCreate_PeggedStruct_WithExistingID_ShouldGiveAnError(t *testing.T) {
	doguuid1 := datatype.NewUUID()

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		Name:      "TestModel1",
		EmbedDog: SecLevelEmbedDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Name:      "Buddy",
			Color:     "black",
		},
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&testModel1).Error(); !assert.Nil(t, err) {
		return
	}

	// Same doguuid1, and that should give an error
	testModel2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		Name:      "TestModel2",
		EmbedDog: SecLevelEmbedDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Name:      "Buddy",
			Color:     "black",
		},
	}

	err := Q(tx).Create(&testModel2).Error()
	assert.Error(t, err)
}

func TestCreate_PeggedStructPtr_WithExistingID_ShouldGiveAnError(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	doguuid1 := datatype.NewUUID()

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		Name:      "TestModel1",
		PtrDog: &SecLevelPtrDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Name:      "Buddy",
			Color:     "black",
		},
	}

	err := DB(tx).Create(&testModel1).Error()
	if !assert.Nil(t, err) {
		return
	}

	testModel2 := TopLevel{
		BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		Name:      "TestModel2",
		PtrDog: &SecLevelPtrDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Name:      "Buddy",
			Color:     "black",
		},
	}

	err = DB(tx).Create(&testModel2).Error()
	assert.Error(t, err)
}

func TestBatchCreate_PeggedArray(t *testing.T) {
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

	searched := make([]TopLevel, 0)

	err := DB(tx).Create(&tms).Error() // used to be CreateMany

	if assert.Nil(t, err) {
		err := Q(tx, C("ID IN", []uuid.UUID{u1, u2})).Find(&searched).Error()
		if assert.Nil(t, err) {
			assert.Len(t, searched, 2)
			// From Gorm, created time is thes same and we can't differentiate between the two
			if assert.Len(t, searched[0].Dogs, 1) {
				assert.Contains(t, []string{"Happy", "Buddy"}, searched[0].Dogs[0].Name)
			}
			if assert.Len(t, searched[1].Dogs, 1) {
				assert.Contains(t, []string{"Happy", "Buddy"}, searched[1].Dogs[0].Name)
			}
		}
	}
}

func TestBatchCreate_PeggedArray_WithExistingID_ShouldGiveError(t *testing.T) {
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

	testModels := []TopLevel{testModel1, testModel2}

	err := DB(tx).Create(&testModels).Error() // used to be CreateMany
	assert.Nil(t, err)

	testModel3 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel3",
		Dogs: []SecLevelArrDog{
			{
				BaseModel: mdl.BaseModel{ID: &doguuid1},
				Name:      "Buddy",
				Color:     "black",
			},
		},
	}
	testModel4 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u2},
		Name:      "TestModel4",
		Dogs: []SecLevelArrDog{
			{
				BaseModel: mdl.BaseModel{ID: &doguuid2},
				Name:      "Happy",
				Color:     "red",
			},
		},
	}

	testModels = []TopLevel{testModel3, testModel4}
	err = DB(tx).Create(&testModels).Error()
	assert.Error(t, err)
}

func TestBatchCreate_PeggAssociateArray_shouldAssociateCorrectly(t *testing.T) {
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

	searched := make([]TopLevel, 0)

	err = DB(tx).Create(tms).Error()
	if assert.Nil(t, err) {
		err := Q(tx, C("ID IN", []uuid.UUID{u1, u2})).Find(&searched).Error()
		if assert.Nil(t, err) {
			assert.Len(t, searched, 2)
			// Gorm created at the same time now time stamp is the same so has to use assert.Contains
			if assert.Len(t, searched[0].Cats, 1) {
				assert.Contains(t, []string{"Kiddy1", "Kiddy2"}, searched[0].Cats[0].Name)
			}
			if assert.Len(t, searched[1].Cats, 1) {
				assert.Contains(t, []string{"Kiddy1", "Kiddy2"}, searched[1].Cats[0].Name)
			}
		}
	}
}
