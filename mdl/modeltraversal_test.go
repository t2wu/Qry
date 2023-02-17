package mdl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestThirdLevel struct {
	BaseModel
	Name string
}

type TestSecondLevel struct {
	BaseModel
	Name        string
	Arr         []TestThirdLevel `qry:"peg"`
	ArrUnpegged []TestThirdLevel
}

type TestStruct struct {
	BaseModel

	Arr []TestSecondLevel `qry:"peg"`
	// ArrPtr []*TestSecondLevel `qry:"peg"` // Difficult case to handle, I'm not even sure if Gorm handles it.

	Second    TestSecondLevel  `qry:"peg"`
	SecondPtr *TestSecondLevel `qry:"peg"`
}

type TestSecondLevel2 struct {
	BaseModel
	Name        string
	Arr         []TestThirdLevel `qry:"peg"`
	ArrUnpegged []TestThirdLevel
}

type TestStruct2 struct {
	BaseModel

	Arr       []TestSecondLevel2 `qry:"peg"`
	Second    TestSecondLevel2   `qry:"peg"`
	SecondPtr *TestSecondLevel2  `qry:"peg"`
}

func TestModelTraversal(t *testing.T) {
	ret := GetPeggedFieldNumAndType(&TestStruct{})

	assert.Len(t, ret, 3)

	assert.Equal(t, "Arr", ret[0].FieldName)
	assert.Equal(t, "TestSecondLevel", ret[0].TypeName)
	assert.Equal(t, 1, ret[0].FieldNum)
	assert.True(t, ret[0].IsSlice)
	assert.False(t, ret[0].IsStruct)
	assert.False(t, ret[0].IsPtr)

	assert.Equal(t, "TestSecondLevel", ret[0].ObjType.Name())

	assert.Equal(t, "Second", ret[1].FieldName)
	assert.Equal(t, 2, ret[1].FieldNum)
	assert.False(t, ret[1].IsSlice)
	assert.True(t, ret[1].IsStruct)
	assert.False(t, ret[1].IsPtr)
	assert.Equal(t, "TestSecondLevel", ret[1].ObjType.Name())

	assert.Equal(t, "SecondPtr", ret[2].FieldName)
	assert.Equal(t, 3, ret[2].FieldNum)
	assert.False(t, ret[2].IsSlice)
	assert.False(t, ret[2].IsStruct)
	assert.True(t, ret[2].IsPtr)
	assert.Equal(t, "TestSecondLevel", ret[2].ObjType.Name())
}

// TODO
// func TestSetSliceAtFieldNum(t *testing.T) {
// 	modelObj := &TestStruct{}
// 	ts := TestSecondLevel{
// 		Name: "MyName",
// 	}

// 	slice := reflect.New(reflect.SliceOf(reflect.TypeOf(ts)))
// 	slice.Elem().Set(reflect.Append(slice.Elem(), reflect.ValueOf(ts)))
// 	SetSliceAtFieldNum(modelObj, 1, slice.Interface())

// 	log.Printf("modelObj: %+v\n", modelObj)
// 	assert.Fail(t, "0")
// }

func TestAppendToSliceAtFieldNum(t *testing.T) {
	modelObj := &TestStruct{}
	ts := TestSecondLevel{
		Name: "MyName",
	}
	AppendToSliceAtFieldNum(modelObj, 1, &ts)
	if !assert.Len(t, modelObj.Arr, 1) {
		return
	}
	assert.Equal(t, ts.Name, modelObj.Arr[0].Name)
}

func TestSetStructAtFieldNum(t *testing.T) {
	modelObj := &TestStruct{}
	ts := TestSecondLevel{
		Name: "MyName",
	}
	SetStructAtFieldNum(modelObj, 2, &ts)
	assert.Equal(t, ts.Name, modelObj.Second.Name)
}

func TestSetStructPtrAtFieldNum(t *testing.T) {
	modelObj := &TestStruct{}
	ts := TestSecondLevel{
		Name: "MyName",
	}
	SetStructPtrAtFieldNum(modelObj, 3, &ts)
	assert.Equal(t, ts.Name, modelObj.SecondPtr.Name)
}
