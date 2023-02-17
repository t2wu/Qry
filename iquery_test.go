package qry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func Test_IQuery_QueryComformsToIQuery_Works(t *testing.T) {
	var db *gorm.DB
	q := DB(db)
	_, ok := q.(IQuery)
	assert.True(t, ok)
}
