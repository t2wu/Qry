package qry

import (
	"reflect"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/t2wu/qry/datatype"
)

// --- Predicate ---

func TestPredicateFromStringAndVal_works(t *testing.T) {
	tests := []struct {
		query string
		value interface{}
		want  *Predicate
	}{
		{
			query: "Age >",
			value: 20,
			want: &Predicate{
				Field: "Age",
				Cond:  PredicateCondGT,
				Value: 20,
			},
		},
		{
			query: "Age >=",
			value: 20,
			want: &Predicate{
				Field: "Age",
				Cond:  PredicateCondGTEQ,
				Value: 20,
			},
		},
		{
			query: "Age <",
			value: 20,
			want: &Predicate{
				Field: "Age",
				Cond:  PredicateCondLT,
				Value: 20,
			},
		},
		{
			query: "Age <=",
			value: 20,
			want: &Predicate{
				Field: "Age",
				Cond:  PredicateCondLTEQ,
				Value: 20,
			},
		},
		{
			query: "Name =",
			value: "Christy",
			want: &Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "Christy",
			},
		},
	}

	for _, test := range tests {
		result, _ := NewPredicateFromStringAndVal(test.query, test.value)
		isTrue := reflect.DeepEqual(test.want, result)
		assert.True(t, isTrue)
	}
}

func TestPredicateFromStringAndVal_INClause_works(t *testing.T) {
	tests := []struct {
		query string
		value interface{}
		want  *Predicate
	}{
		{
			query: "Name IN",
			value: []string{"Christy", "Joseph"},
			want: &Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: []string{"Christy", "Joseph"},
			},
		},
		{
			query: "Name in", // lower case
			value: []string{"Christy", "Joseph"},
			want: &Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: []string{"Christy", "Joseph"},
			},
		},
	}

	for _, test := range tests {
		result, _ := NewPredicateFromStringAndVal(test.query, test.value)
		if v, ok := result.Value.([]string); ok {
			if assert.Equal(t, 2, len(v)) {
				wantv := test.want.Value.([]string)
				assert.Equal(t, wantv[0], v[0])
				assert.Equal(t, wantv[1], v[1])
			}
		} else {
			assert.Fail(t, "value not expected")
		}
	}
}

func TestPredicateFromStringAndVal_whenWrongValue_hasError(t *testing.T) {
	tests := []struct {
		query string
		value interface{}
		want  *Predicate
	}{
		{
			query: "Age > wrong",
			value: 20,
			want: &Predicate{
				Field: "Age",
				Cond:  PredicateCondGT,
				Value: 20,
			}},
	}

	for _, test := range tests {
		_, err := NewPredicateFromStringAndVal(test.query, test.value)
		assert.Error(t, err)
	}
}

func TestBuildQueryStringAndValueForAllTypeOfConditions_Works(t *testing.T) {
	tests := []struct {
		predicate *Predicate
		want      struct {
			s string
			v interface{}
		}
	}{
		{
			predicate: &Predicate{
				Field: "Age",
				Cond:  PredicateCondEQ,
				Value: 20,
			},
			want: struct {
				s string
				v interface{}
			}{s: "\"top_level\".age = ?", v: 20},
		},
		{
			predicate: &Predicate{
				Field: "Age",
				Cond:  PredicateCondLT,
				Value: 20,
			},
			want: struct {
				s string
				v interface{}
			}{s: "\"top_level\".age < ?", v: 20},
		},
		{
			predicate: &Predicate{
				Field: "Age",
				Cond:  PredicateCondLTEQ,
				Value: 20,
			},
			want: struct {
				s string
				v interface{}
			}{s: "\"top_level\".age <= ?", v: 20},
		},
		{
			predicate: &Predicate{
				Field: "Age",
				Cond:  PredicateCondGT,
				Value: 20,
			},
			want: struct {
				s string
				v interface{}
			}{s: "\"top_level\".age > ?", v: 20},
		},
		{
			predicate: &Predicate{
				Field: "Age",
				Cond:  PredicateCondGTEQ,
				Value: 20,
			},
			want: struct {
				s string
				v interface{}
			}{s: "\"top_level\".age >= ?", v: 20},
		},
	}
	for _, test := range tests {
		s, vals, err := test.predicate.BuildQueryStringAndValues(&TopLevel{})
		assert.Nil(t, err)
		assert.Equal(t, test.want.s, s)
		if assert.Equal(t, 1, len(vals)) {
			assert.Equal(t, test.want.v, vals[0])
		}
	}
}

func TestBuildQueryStringAndValue_Escape_Rawtring(t *testing.T) {
	tests := []struct {
		predicate *Predicate
		want      struct {
			s string
		}
	}{
		{
			predicate: &Predicate{
				Field: "Age",
				Cond:  PredicateCondEQ,
				Value: &Escape{Value: "20"},
			},
			want: struct {
				s string
			}{s: "\"top_level\".age = 20"},
		},
	}
	for _, test := range tests {
		s, vals, err := test.predicate.BuildQueryStringAndValues(&TopLevel{})
		assert.Nil(t, err)
		assert.Equal(t, test.want.s, s)
		assert.Equal(t, 0, len(vals))
	}
}

func TestBuildQueryStringAndValueForInClause_Works(t *testing.T) {
	u1 := datatype.NewUUID()
	u2 := datatype.NewUUID()
	u3 := datatype.NewUUID()

	tests := []struct {
		predicate *Predicate
		want      struct {
			s string
			v []uuid.UUID
		}
	}{
		{
			predicate: &Predicate{
				Field: "ID",
				Cond:  PredicateCondIN,
				Value: []uuid.UUID{
					u1,
					u2,
					u3,
				},
			},
			want: struct {
				s string
				v []uuid.UUID
			}{s: "\"top_level\".id IN (?)", v: []uuid.UUID{u1, u2, u3}},
		},
	}
	for _, test := range tests {
		s, vals, err := test.predicate.BuildQueryStringAndValues(&TopLevel{})
		assert.Nil(t, err)
		assert.Equal(t, test.want.s, s)

		if assert.Equal(t, 1, len(vals)) {
			v2, ok := vals[0].([]uuid.UUID)
			if ok {
				assert.Equal(t, test.want.v[0], v2[0])
				assert.Equal(t, test.want.v[1], v2[1])
				assert.Equal(t, test.want.v[2], v2[2])
			} else {
				assert.Fail(t, "wrong type")
			}
		}
	}
}

func TestBuildQueryStringAndValueForBetweenClause_Works(t *testing.T) {
	now := time.Now()
	before := now.Add(-60 * time.Second)

	tests := []struct {
		predicate *Predicate
		want      struct {
			s string
			v []time.Time
		}
	}{
		{
			predicate: &Predicate{
				Field: "CreatedAt",
				Cond:  PredicateCondBETWEEN,
				Value: []time.Time{
					before,
					now,
				},
			},
			want: struct {
				s string
				v []time.Time
			}{s: "\"top_level\".created_at BETWEEN ? AND ?", v: []time.Time{before, now}},
		},
	}

	for _, test := range tests {
		s, vals, err := test.predicate.BuildQueryStringAndValues(&TopLevel{})
		assert.Nil(t, err)
		assert.Equal(t, test.want.s, s)

		if assert.Equal(t, 1, len(vals)) {
			v2, ok := vals[0].([]time.Time)
			if ok {
				assert.Equal(t, test.want.v[0].UnixNano(), v2[0].UnixNano())
				assert.Equal(t, test.want.v[1].UnixNano(), v2[1].UnixNano())
			} else {
				assert.Fail(t, "wrong type")
			}
		}
	}
}

func TestBuildQueryString_ArrInnerStructQuery_Works(t *testing.T) {
	tests := []struct {
		predicate *Predicate
		want      struct {
			s string
			v interface{}
		}
	}{
		{
			predicate: &Predicate{
				Field: "Dogs.Name",
				Cond:  PredicateCondEQ,
				Value: "doggie1",
			},
			want: struct {
				s string
				v interface{}
			}{s: "\"sec_level_arr_dog\".name = ?", v: "doggie1"},
		},
	}
	for _, test := range tests {
		s, vals, err := test.predicate.BuildQueryStringAndValues(&TopLevel{})
		assert.Nil(t, err)
		assert.Equal(t, test.want.s, s)
		if assert.Equal(t, len(vals), 1) {
			assert.Equal(t, test.want.v, vals[0])
		}
	}
}

func TestBuildQueryString_ArrNonExistingInnerStructQuery_ReturnsError(t *testing.T) {
	tests := []struct {
		predicate *Predicate
		want      struct {
			s string
			v interface{}
		}
	}{
		{
			predicate: &Predicate{
				Field: "Bogus.Name",
				Cond:  PredicateCondEQ,
				Value: "doggie1",
			},
			want: struct {
				s string
				v interface{}
			}{s: "\"bogus\".name = ?", v: "doggie1"},
		},
	}
	for _, test := range tests {
		_, _, err := test.predicate.BuildQueryStringAndValues(&TopLevel{})
		assert.Error(t, err)
	}
}

func TestBuildQueryString_ArrLevel2InnerStructQuery_Works(t *testing.T) {
	tests := []struct {
		predicate *Predicate
		want      struct {
			s string
			v interface{}
		}
	}{
		{
			predicate: &Predicate{
				Field: "Dogs.DogToys.ToyName",
				Cond:  PredicateCondEQ,
				Value: "MyToy",
			},
			want: struct {
				s string
				v interface{}
			}{s: "\"third_level_arr_dog_toy\".toy_name = ?", v: "MyToy"},
		},
	}
	for _, test := range tests {
		s, vals, err := test.predicate.BuildQueryStringAndValues(&TopLevel{})
		if assert.Nil(t, err) {
			assert.Equal(t, test.want.s, s)
			if assert.Equal(t, len(vals), 1) {
				assert.Equal(t, test.want.v, vals[0])
			}
		}
	}
}

func TestPredicate_ArrGetDesignatedModel_Works(t *testing.T) {
	tests := []struct {
		predicate *Predicate
	}{
		{
			predicate: &Predicate{
				Field: "Dogs.DogToys.ToyName",
				Cond:  PredicateCondEQ,
				Value: "MyToy",
			},
		},
		{
			predicate: &Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "SomeName",
			},
		},
	}
	test1 := tests[0]
	m, err := test1.predicate.GetDesignatedModel(&TopLevel{})
	if assert.Nil(t, err) {
		_, ok := m.(*ThirdLevelArrDogToy)
		assert.True(t, ok)
	}
	test2 := tests[1]
	m, err = test2.predicate.GetDesignatedModel(&TopLevel{})
	if assert.Nil(t, err) {
		_, ok := m.(*TopLevel)
		assert.True(t, ok)
	}
}

func TestPredicate_GetDesignatedField_Works(t *testing.T) {
	tests := []struct {
		predicate *Predicate
	}{
		{
			predicate: &Predicate{
				Field: "Dogs.DogToys.ToyName",
				Cond:  PredicateCondEQ,
				Value: "MyToy",
			},
		},
		{
			predicate: &Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "SomeName",
			},
		},
	}
	test1 := tests[0]
	s := test1.predicate.GetDesignatedField(&TopLevel{})
	assert.Equal(t, "Dogs.DogToys", s)

	test2 := tests[1]
	s = test2.predicate.GetDesignatedField(&TopLevel{})
	assert.Equal(t, "", s)
}

func Test_GetAllUnqueStructFieldDesignator_whenHavingDot_works(t *testing.T) {
	tests := []struct {
		predicate *Predicate
		wants     []string
	}{
		{
			predicate: &Predicate{
				Field: "A.B.C",
				Cond:  PredicateCondEQ,
				Value: "what",
			},
			wants: []string{"A.B", "A"},
		},
		{
			predicate: &Predicate{
				Field: "A.B",
				Cond:  PredicateCondEQ,
				Value: "what",
			},
			wants: []string{"A"},
		},
	}

	for _, test := range tests {
		m := test.predicate.GetAllUnqueStructFieldDesignator()
		if assert.Equal(t, len(test.wants), len(m)) {
			for _, want := range test.wants {
				found := false
				for designator := range m {
					if want == designator {
						found = true
					}
				}
				assert.True(t, found)
			}
		}
	}
}

func Test_GetAllUnqueStructFieldDesignator_whenOnlyOneLevel_returnEmptyMap(t *testing.T) {
	predicate := &Predicate{
		Field: "A",
		Cond:  PredicateCondEQ,
		Value: "what",
	}

	m := predicate.GetAllUnqueStructFieldDesignator()
	assert.Equal(t, 0, len(m))
}

func Test_Predicate_NestedLevel(t *testing.T) {
	predicate := &Predicate{
		Field: "A.B.C",
		Cond:  PredicateCondEQ,
		Value: "what",
	}

	assert.Equal(t, 3, predicate.GetNestedLevel())

	predicate = &Predicate{
		Field: "A",
		Cond:  PredicateCondEQ,
		Value: "what",
	}

	assert.Equal(t, 1, predicate.GetNestedLevel())
}

// // --- PredicateRelation ---

func TestPredicateRelationStringAndValuesOnePredicte(t *testing.T) {
	tests := []struct {
		pr   *PredicateRelation
		want struct {
			s string
			v int
		}
	}{
		{
			pr: &PredicateRelation{
				PredOrRels: []Criteria{
					&Predicate{
						Field: "Age",
						Cond:  PredicateCondGT,
						Value: 20,
					},
				},
			},
			want: struct {
				s string
				v int
			}{
				s: "\"top_level\".age > ?",
				v: 20,
			},
		},
	}

	for _, test := range tests {
		s, vals, err := test.pr.BuildQueryStringAndValues(&TopLevel{})
		assert.Nil(t, err)
		assert.Equal(t, test.want.s, s)
		if assert.Equal(t, 1, len(vals)) {
			assert.Equal(t, test.want.v, vals[0])
		}
	}
}

func TestPredicateRelationStringAndValuesWithSecondNested(t *testing.T) {
	innerPred := &PredicateRelation{
		PredOrRels: []Criteria{
			&Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "Christy",
			},
			&Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "Jenny",
			},
		},
		Logics: []PredicateLogic{PredicateLogicOR},
	}

	outerPred := &PredicateRelation{
		PredOrRels: []Criteria{
			&Predicate{
				Field: "Age",
				Cond:  PredicateCondGT,
				Value: 20,
			},
			&Predicate{
				Field: "Age",
				Cond:  PredicateCondLT,
				Value: 30,
			},
			innerPred,
		},
		Logics: []PredicateLogic{PredicateLogicAND, PredicateLogicAND},
	}
	s, vals, err := outerPred.BuildQueryStringAndValues(&TopLevel{})
	assert.Nil(t, err)
	assert.Equal(t, "(\"top_level\".age > ?) AND (\"top_level\".age < ?) AND ((\"top_level\".real_name_column = ?) OR (\"top_level\".real_name_column = ?))", s)
	if assert.Equal(t, 4, len(vals)) {
		assert.Equal(t, 20, vals[0].(int), 20)
		assert.Equal(t, 30, vals[1].(int), 30)
		assert.Equal(t, "Christy", vals[2].(string))
		assert.Equal(t, "Jenny", vals[3].(string))

	}
}

func TestPredicateRelationStringAndValuesWithFirstNested(t *testing.T) {
	innerRel := &PredicateRelation{
		PredOrRels: []Criteria{
			&Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "Christy",
			},
			&Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "Jenny",
			},
		},
		Logics: []PredicateLogic{PredicateLogicOR},
	}

	outerPred := &PredicateRelation{
		PredOrRels: []Criteria{
			innerRel,
			&Predicate{
				Field: "Age",
				Cond:  PredicateCondGT,
				Value: 20,
			},
			&Predicate{
				Field: "Age",
				Cond:  PredicateCondLT,
				Value: 30,
			},
		},
		Logics: []PredicateLogic{PredicateLogicAND, PredicateLogicAND},
	}
	s, vals, err := outerPred.BuildQueryStringAndValues(&TopLevel{})
	assert.Nil(t, err)
	assert.Equal(t, "((\"top_level\".real_name_column = ?) OR (\"top_level\".real_name_column = ?)) AND (\"top_level\".age > ?) AND (\"top_level\".age < ?)", s)
	if assert.Equal(t, 4, len(vals)) {
		assert.Equal(t, "Christy", vals[0].(string))
		assert.Equal(t, "Jenny", vals[1].(string))
		assert.Equal(t, 20, vals[2].(int), 20)
		assert.Equal(t, 30, vals[3].(int), 30)
	}
}

func TestBuildQueryString_DifferentLevelOfNesting_ReturnError(t *testing.T) {
	rel := &PredicateRelation{
		PredOrRels: []Criteria{
			&Predicate{
				Field: "Inner.Name",
				Cond:  PredicateCondEQ,
				Value: "Christy",
			},
			&Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "Jenny",
			},
		},
		Logics: []PredicateLogic{PredicateLogicOR},
	}

	_, _, err := rel.BuildQueryStringAndValues(&TopLevel{})
	assert.Error(t, err)
}

func TestPredicateRelation_ArrGetDesignatedModel_Works(t *testing.T) {
	rel := &PredicateRelation{
		PredOrRels: []Criteria{
			&Predicate{
				Field: "Dogs.DogToys.ToyName",
				Cond:  PredicateCondEQ,
				Value: "Something",
			},
		},
		Logics: []PredicateLogic{PredicateLogicAND, PredicateLogicAND},
	}

	m, err := rel.GetDesignatedModel(&TopLevel{})
	if assert.Nil(t, err) {
		_, ok := m.(*ThirdLevelArrDogToy)
		assert.True(t, ok)
	}
}

func TestPredicateRelation_GetDesignatedField_Works(t *testing.T) {
	rel1 := &PredicateRelation{
		PredOrRels: []Criteria{
			&Predicate{
				Field: "Dogs.DogToys.ToyName",
				Cond:  PredicateCondEQ,
				Value: "Something",
			},
		},
		Logics: []PredicateLogic{PredicateLogicAND, PredicateLogicAND},
	}

	rel2 := &PredicateRelation{
		PredOrRels: []Criteria{
			&Predicate{
				Field: "Name",
				Cond:  PredicateCondEQ,
				Value: "Something",
			},
		},
		Logics: []PredicateLogic{PredicateLogicAND, PredicateLogicAND},
	}

	field := rel1.GetDesignatedField(&TopLevel{})
	assert.Equal(t, "Dogs.DogToys", field)
	field = rel2.GetDesignatedField(&TopLevel{})
	assert.Equal(t, "", field)
}

func TestRelation_GetAllUnqueStructFieldDesignator_Works(t *testing.T) {
	rel := &PredicateRelation{
		PredOrRels: []Criteria{
			&Predicate{
				Field: "Dogs.DogToys.ToyToy",
				Cond:  PredicateCondEQ,
				Value: "Something",
			},
			&Predicate{
				Field: "A.B.C",
				Cond:  PredicateCondEQ,
				Value: "Something",
			},
			&PredicateRelation{
				PredOrRels: []Criteria{
					&Predicate{
						Field: "Dogs.DogToys.Ant.Bat.Cat",
						Cond:  PredicateCondEQ,
						Value: "Something",
					},
				},
			},
		},
		Logics: []PredicateLogic{PredicateLogicAND, PredicateLogicAND},
	}

	want := []string{"Dogs", "Dogs.DogToys", "A.B", "A", "Dogs.DogToys.Ant", "Dogs.DogToys.Ant.Bat"}

	m := rel.GetAllUnqueStructFieldDesignator()
	if assert.Equal(t, len(want), len(m)) {
		for _, want := range want {
			found := false
			for designator := range m {
				if want == designator {
					found = true
				}
			}
			assert.True(t, found)
		}
	}

}
