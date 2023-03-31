package qry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/t2wu/qry/datatype"
	"github.com/t2wu/qry/mdl"
)

func TestSave_Works(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	doguuid1 := datatype.NewUUID()
	testModel := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "second",
		Age:       3,
		EmbedDog: SecLevelEmbedDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Color:     "mahogany",
		},
	}
	if err := DB(tx).Create(&testModel).Error(); assert.Nil(t, err) {
		return
	}
	tm := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&tm).Error(); assert.Nil(t, err) {
		return
	}

	// Change the name to something else
	tm.Name = "TestSave_Works"
	tm.EmbedDog.Color = "purple"
	if err := Q(tx).Save(&tm).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	// Find it back to make sure it has been changed
	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, "TestSave_Works", searched.Name)
	assert.Equal(t, "purple", searched.EmbedDog.Color)
}

func TestUpdate_Field_Works(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	testModel := TopLevel{BaseModel: mdl.BaseModel{ID: &u1}, Name: "second", Age: 3}
	if err := DB(tx).Create(&testModel).Error(); assert.Nil(t, err) {
		return
	}

	if err := Q(tx, C("Name =", "second")).Update(&TopLevel{}, C("Age =", 120)).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	check := TopLevel{}
	if err := Q(tx, C("Name =", "second")).First(&check).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, 120, check.Age)
}

func TestUpdate_Field_ForMultipleIModel_Works(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()
	testModel1 := TopLevel{BaseModel: mdl.BaseModel{ID: &u1}, Name: "second", Age: 3}
	testModel2 := TopLevel{BaseModel: mdl.BaseModel{ID: &u2}, Name: "not", Age: 4}
	testModel3 := TopLevel{BaseModel: mdl.BaseModel{ID: &u3}, Name: "second", Age: 4}
	if err := DB(tx).Create(&testModel1).Create(&testModel2).Create(&testModel3).Error(); assert.Nil(t, err) {
		return
	}

	if err := Q(tx, C("Name =", "second")).Update(&TopLevel{}, C("Age =", 120)).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	searched := make([]TopLevel, 0)
	if err := Q(tx, C("Name =", "second")).Find(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	if assert.Len(t, searched, 2) {
		assert.Equal(t, 120, searched[0].Age)
		assert.Equal(t, u1, *searched[0].ID)
		assert.Equal(t, 120, searched[1].Age)
		assert.Equal(t, u3, *searched[0].ID)
	}
}

// Update currently allows only one-level of nested field
func TestUpdate_NestedField_ShouldGiveWarning(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	doguuid1 := datatype.NewUUID()
	testModel := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "second",
		Age:       3,
		EmbedDog: SecLevelEmbedDog{
			BaseModel: mdl.BaseModel{ID: &doguuid1},
			Color:     "mahogany",
		},
	}
	if err := DB(tx).Create(&testModel).Error(); assert.Nil(t, err) {
		return
	}

	// Name:        "Doggie2",
	if err := Q(db, C("Name =", "second")).Update(&TopLevel{}, C("Dogs.Color =", "purple")).Error(); err != nil {
		assert.Equal(t, "dot notation in update", err.Error())
		return
	}

	assert.Fail(t, "should not be here")
}

func TestSave_PegArray_ShouldUpdateData(t *testing.T) {
	doguuid := datatype.NewUUID()

	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()

	newDog := SecLevelArrDog{
		BaseModel:  mdl.BaseModel{ID: &doguuid},
		Name:       "NewBuddy",
		Color:      "red",
		TopLevelID: &u1,
	}

	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "MyTestModel",
		Age:       1,
		// not really testing this but this is required since it's associated and has no id
		EmbedCat: SecLevelEmbedCat{
			BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		},
		Dogs: []SecLevelArrDog{newDog},
	}

	if err := Q(tx).Create(&tm.EmbedCat).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	tm.Dogs[0].Color = "black"

	if err := Q(tx).Save(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, u1, *searched.ID)
	if assert.Equal(t, 1, len(searched.Dogs)) { // should be preloaded (need association with Gormv2)
		assert.Equal(t, doguuid, *searched.Dogs[0].ID)
		assert.Equal(t, "NewBuddy", searched.Dogs[0].Name)
		assert.Equal(t, "black", searched.Dogs[0].Color)
	}
}

func TestSave_PegAssocArray_WhichDidNotPreviouslyExist_ShouldReturnError(t *testing.T) {
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

	u1 := datatype.NewUUID()
	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "MyTestModel",
		Age:       1,
	}

	if err := Q(tx).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	tm.Cats = []SecLevelArrCat{cat}

	err := Q(tx).Save(&tm).Error()
	assert.NotNil(t, err)
}

func TestSave_PegEmbed_ShouldUpdateData(t *testing.T) {
	doguuid := datatype.NewUUID()

	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()

	newDog := SecLevelEmbedDog{
		BaseModel:  mdl.BaseModel{ID: &doguuid},
		Name:       "NewBuddy",
		Color:      "red",
		TopLevelID: &u1,
	}

	tm := TopLevel{BaseModel: mdl.BaseModel{ID: &u1},
		Name: "MyTestModel",
		Age:  1,
		EmbedCat: SecLevelEmbedCat{
			BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		},
		EmbedDog: newDog,
	}

	if err := Q(tx).Create(&tm.EmbedCat).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	tm.EmbedDog.Color = "black"

	if err := Q(tx).Save(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, u1, *searched.ID)
	assert.Equal(t, doguuid, *searched.EmbedDog.ID)
	assert.Equal(t, "NewBuddy", searched.EmbedDog.Name)
	assert.Equal(t, "black", searched.EmbedDog.Color)
}

func TestSave_PegAssocEmbed_ShouldNotUpdateData(t *testing.T) {
	doguuid := datatype.NewUUID()

	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()

	tm := TopLevel{BaseModel: mdl.BaseModel{ID: &u1},
		Name: "MyTestModel",
		Age:  1,
	}

	cat := SecLevelEmbedCat{
		BaseModel:  mdl.BaseModel{ID: &doguuid},
		Name:       "NewBuddy",
		Color:      "red",
		TopLevelID: &u1,
	}

	if err := Q(tx).Create(&tm).Create(&cat).Error(); !assert.Nil(t, err) {
		return
	}

	tm.EmbedCat = cat
	tm.EmbedCat.Color = "black"

	if err := Q(tx).Save(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, u1, *searched.ID)
	assert.Equal(t, doguuid, *searched.EmbedCat.ID)
	assert.Equal(t, "NewBuddy", searched.EmbedCat.Name)
	assert.Equal(t, "red", searched.EmbedCat.Color)
}

func TestSave_PeggedAssocEmbed_WhichDidNotPreviouslyExist_ShouldReturnError(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	catuuid1 := datatype.NewUUID()

	cat := SecLevelEmbedCat{
		BaseModel: mdl.BaseModel{ID: &catuuid1},
		Name:      "Kiddy",
		Color:     "black",
	}

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel1",
	}

	if err := DB(tx).Create(&testModel1).Error(); !assert.Nil(t, err) {
		return
	}

	testModel1.EmbedCat = cat
	err := DB(tx).Save(&testModel1).Error()
	assert.NotNil(t, err)
}

func TestSave_PegPtr_ShouldUpdateData(t *testing.T) {
	doguuid := datatype.NewUUID()

	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()

	newDog := SecLevelPtrDog{
		BaseModel:  mdl.BaseModel{ID: &doguuid},
		Name:       "NewBuddy",
		Color:      "red",
		TopLevelID: &u1,
	}

	tm := TopLevel{BaseModel: mdl.BaseModel{ID: &u1},
		Name: "MyTestModel",
		Age:  1,
		EmbedCat: SecLevelEmbedCat{
			BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		},
		PtrDog: &newDog,
	}

	if err := Q(tx).Create(&tm.EmbedCat).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	tm.PtrDog.Color = "black"

	if err := Q(tx).Save(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, u1, *searched.ID)
	assert.Equal(t, doguuid, *searched.PtrDog.ID)
	assert.Equal(t, "NewBuddy", searched.PtrDog.Name)
	assert.Equal(t, "black", searched.PtrDog.Color)
}

func TestSave_PtrAssocEmbed_ShouldNotUpdateData(t *testing.T) {
	doguuid := datatype.NewUUID()

	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()

	cat := SecLevelPtrCat{
		BaseModel:  mdl.BaseModel{ID: &doguuid},
		Name:       "NewBuddy",
		Color:      "red",
		TopLevelID: &u1,
	}

	tm := TopLevel{BaseModel: mdl.BaseModel{ID: &u1},
		Name: "MyTestModel",
		Age:  1,
		EmbedCat: SecLevelEmbedCat{
			BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		},
	}

	if err := Q(tx).Create(&tm.EmbedCat).Create(&tm).Create(&cat).Error(); !assert.Nil(t, err) {
		return
	}

	tm.PtrCat = &cat
	tm.PtrCat.Color = "black"

	if err := Q(tx).Save(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, u1, *searched.ID)
	assert.Equal(t, doguuid, *searched.PtrCat.ID)
	assert.Equal(t, "NewBuddy", searched.PtrCat.Name)
	assert.Equal(t, "red", searched.PtrCat.Color)
}

func TestSave_PeggedAssocPtr_WhichDidNotPreviouslyExist_ShouldReturnError(t *testing.T) {
	tx := db.Begin()
	defer tx.Rollback()

	u1 := datatype.NewUUID()
	catuuid1 := datatype.NewUUID()

	cat := SecLevelPtrCat{
		BaseModel: mdl.BaseModel{ID: &catuuid1},
		Name:      "Kiddy",
		Color:     "black",
	}

	testModel1 := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "TestModel1",
	}

	if err := DB(tx).Create(&testModel1).Error(); !assert.Nil(t, err) {
		return
	}

	testModel1.PtrCat = &cat
	err := DB(tx).Save(&testModel1).Error()
	assert.NotNil(t, err)
}

func TestSave_ThirdLevelEmbedPeg_ShouldNotUpdateData(t *testing.T) {
	u1 := datatype.NewUUID()
	doguuid := datatype.NewUUID()
	dogtoy := datatype.NewUUID()

	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "MyTestModel",
		Age:       1,
		EmbedCat: SecLevelEmbedCat{
			BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		},
		EmbedDog: SecLevelEmbedDog{
			BaseModel: mdl.BaseModel{ID: &doguuid},
			Name:      "Happy",
			Color:     "white",
			DogToys: []ThirdLevelEmbedDogToy{
				{
					BaseModel: mdl.BaseModel{ID: &dogtoy},
					ToyName:   "bone",
				},
			},
		},
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&tm.EmbedCat).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	tm.EmbedDog.DogToys[0].ToyName = "ring"

	if err := Q(tx).Save(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, "ring", searched.EmbedDog.DogToys[0].ToyName)
}

func TestSave_ThirdLevelPtrPeg_ShouldNotUpdateData(t *testing.T) {
	u1 := datatype.NewUUID()
	doguuid := datatype.NewUUID()
	dogtoy := datatype.NewUUID()

	tm := TopLevel{
		BaseModel: mdl.BaseModel{ID: &u1},
		Name:      "MyTestModel",
		Age:       1,
		EmbedCat: SecLevelEmbedCat{
			BaseModel: mdl.BaseModel{ID: datatype.AddrOfUUID(datatype.NewUUID())},
		},
		PtrDog: &SecLevelPtrDog{
			BaseModel: mdl.BaseModel{ID: &doguuid},
			Name:      "Happy",
			Color:     "white",
			DogToys: []ThirdLevelPtrDogToy{
				{
					BaseModel: mdl.BaseModel{ID: &dogtoy},
					ToyName:   "bone",
				},
			},
		},
	}

	tx := db.Begin()
	defer tx.Rollback()

	if err := Q(tx).Create(&tm.EmbedCat).Create(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	tm.PtrDog.DogToys[0].ToyName = "ring"

	if err := Q(tx).Save(&tm).Error(); !assert.Nil(t, err) {
		return
	}

	searched := TopLevel{}
	if err := Q(tx, C("ID =", u1)).First(&searched).Error(); err != nil {
		assert.Nil(t, err)
		return
	}

	assert.Equal(t, "ring", searched.PtrDog.DogToys[0].ToyName)
}
