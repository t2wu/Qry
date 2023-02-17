package qry

import (
	"github.com/t2wu/qry/mdl"
	"gorm.io/gorm"
)

type Order string

const (
	OrderAsc  Order = "ASC"
	OrderDesc Order = "DESC"
)

// IQuery so we can stubb out the DB
type IQuery interface {
	Q(args ...interface{}) IQuery
	Order(field string, order Order) IQuery
	Limit(limit int) IQuery
	Offset(offset int) IQuery
	InnerJoin(modelObj mdl.IModel, foreignObj mdl.IModel, args ...interface{}) IQuery
	BuildQuery(modelObj mdl.IModel) (*gorm.DB, error)
	Take(modelObj mdl.IModel) IQuery
	First(modelObj mdl.IModel) IQuery
	Find(modelObjs interface{}) IQuery
	Count(modelObj mdl.IModel, no *int64) IQuery
	Create(value interface{}) IQuery
	Delete(value interface{}) IQuery
	Save(modelObj mdl.IModel) IQuery
	// Update(modelObjs interface{}, attrs ...interface{}) IQuery
	Update(modelObj mdl.IModel, p *PredicateRelationBuilder) IQuery
	GetDB() *gorm.DB
	Reset() IQuery
	Error() error
}
