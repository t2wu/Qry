package qtag

import (
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Field           string `qry:"peg"`
	NotField        string `something:"else"`
	FieldWithValue1 string `qry:"pegassoc-many2many:bottle"`

	// technically this can't happen, but for test we should handle multiple co-exisiting tags in the future
	FieldWithValue2 string `qry:"future:pop,pegassoc:bottle"` // future some tag

	// We don't want to grab other tags
	MultiTas1 string `qry:"pegassoc-many2many:bottle" something:"else"`
	MultiTas2 string `something:"else" qry:"pegassoc-many2many:bottle"`
}

func TestIsAnyQryTag(t *testing.T) {
	modelObj := TestStruct{}
	val := reflect.ValueOf(modelObj)
	tag1 := val.Type().Field(0).Tag
	tag2 := val.Type().Field(1).Tag

	assert.True(t, IsAnyQryTag(tag1))
	assert.False(t, IsAnyQryTag(tag2))
}

func TestGetQryTagAndField(t *testing.T) {
	modelObj := TestStruct{}
	val := reflect.ValueOf(modelObj)
	tag := val.Type().Field(2).Tag
	tagNFields := GetQryTagAndField(tag)

	if !assert.Len(t, tagNFields, 1) {
		return
	}
	tagNField := tagNFields[0]

	tt := tag.Get("qry")
	log.Println(TagFieldByPrefix(tt, string(QryTagPegAssocMany2Many)))

	assert.NotNil(t, tagNField)
	assert.Equal(t, QryTagPegAssocMany2Many, tagNField.Tag)
	assert.Equal(t, "bottle", tagNField.Field)
}

func TestFieldWithValue1_ShouldGrabValue(t *testing.T) {
	modelObj := TestStruct{}
	val := reflect.ValueOf(modelObj)
	tag := val.Type().Field(2).Tag
	tagNFields := GetQryTagAndField(tag)

	if !assert.Len(t, tagNFields, 1) {
		return
	}
	tagNField := tagNFields[0]

	tt := tag.Get("qry")
	log.Println(TagFieldByPrefix(tt, string(QryTagPegAssocMany2Many)))

	assert.NotNil(t, tagNField)
	assert.Equal(t, QryTagPegAssocMany2Many, tagNField.Tag)
	assert.Equal(t, "bottle", tagNField.Field)
}

// Future when we need to handle two tags
// func TestFieldWithValue2_ShouldGrabTwoValues(t *testing.T) {
// 	modelObj := TestStruct{}
// 	val := reflect.ValueOf(modelObj)
// 	tag := val.Type().Field(3).Tag
// 	tagNFields := GetQryTagAndField(tag)

// 	if !assert.Len(t, tagNFields, 2) {
// 		return
// 	}

// 	for _, tagNField := range tagNFields {
// 		if tagNField.Tag == QryTagPeg {
// 			assert.Equal(t, "pop", tagNField.Field)
// 		}
// 		if tagNField.Tag == QryTagPegAssoc {
// 			assert.Equal(t, "bottle", tagNField.Field)
// 		}
// 	}
// }

func TestGetQryTagAndField_WhenMultipleTags_ShouldStillWork1(t *testing.T) {
	modelObj := TestStruct{}
	val := reflect.ValueOf(modelObj)
	tag := val.Type().Field(4).Tag
	tagNFields := GetQryTagAndField(tag)

	if !assert.Len(t, tagNFields, 1) {
		return
	}
	tagNField := tagNFields[0]

	tt := tag.Get("qry")
	log.Println(TagFieldByPrefix(tt, string(QryTagPegAssocMany2Many)))

	assert.NotNil(t, tagNField)
	assert.Equal(t, QryTagPegAssocMany2Many, tagNField.Tag)
	assert.Equal(t, "bottle", tagNField.Field)
}

func TestGetQryTagAndField_WhenMultipleTags_ShouldStillWork2(t *testing.T) {
	modelObj := TestStruct{}
	val := reflect.ValueOf(modelObj)
	tag := val.Type().Field(5).Tag
	tagNFields := GetQryTagAndField(tag)

	if !assert.Len(t, tagNFields, 1) {
		return
	}
	tagNField := tagNFields[0]

	tt := tag.Get("qry")
	log.Println(TagFieldByPrefix(tt, string(QryTagPegAssocMany2Many)))

	assert.NotNil(t, tagNField)
	assert.Equal(t, QryTagPegAssocMany2Many, tagNField.Tag)
	assert.Equal(t, "bottle", tagNField.Field)
}
