package mdl

import (
	"time"

	"github.com/t2wu/qry/datatype"

	"github.com/asaskevich/govalidator"
	"github.com/jinzhu/gorm"
)

// BaseModel is the base class domain mdl which has standard ID
type BaseModel struct {
	// For MySQL
	// ID        *datatype.UUID `gorm:"type:binary(16);primary_key;" json:"id"`

	// For Postgres
	ID        *datatype.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	CreatedAt time.Time      `sql:"index" json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt *time.Time     `sql:"index" json:"deletedAt"`

	// Ownership with the most previledged permission can delete the device and see every field.
	// So there can be an ownership number, say 3, and that maps to a permission type
	// (within the ownership table), say "admin ownership" (int 0), and whoever is a member of ownership
	// 3 thus has the admin priviledge
	// The "guest" of mdl "device" and "guest" of mdl of "scene" is vastly different, because
	// the fields are different, and specific field permission is based on priviledge -> field mapping
	// defined when getting permission()
	// Ownership []int64
}

// GetID Get the ID field of the mdl (useful when using interface)
func (b *BaseModel) GetID() *datatype.UUID {
	return b.ID
}

// SetID Set the ID field of the mdl (useful when using interface)
func (b *BaseModel) SetID(id *datatype.UUID) {
	b.ID = id
}

// GetCreatedAt gets the time stamp the record is created
func (b *BaseModel) GetCreatedAt() *time.Time {
	return &b.CreatedAt
}

// GetUpdatedAt gets the time stamp the record is updated
func (b *BaseModel) GetUpdatedAt() *time.Time {
	return &b.UpdatedAt
}

// GetUpdatedAt gets the time stamp the record is deleted (which we don't use)
func (b *BaseModel) GetDeletedAt() *time.Time {
	return b.DeletedAt
}

// BeforeCreate sets a UUID if no ID is set
// (this is Gorm's hookpoint)
func (b *BaseModel) BeforeCreate(scope *gorm.Scope) error {
	if b.ID == nil {
		uuid := datatype.NewUUID()
		return scope.SetColumn("ID", uuid)
	}

	return nil
}

// Validate validates the mdl
func (b *BaseModel) Validate() error {
	if ok, err := govalidator.ValidateStruct(b); !ok && err != nil {
		return err
	}
	return nil
}

// IModel is the interface for all domain mdl
type IModel interface {
	// Permissions(role UserRole, scope *string) jsontrans.JSONFields

	// The following two avoids having to use reflection to access ID
	GetID() *datatype.UUID
	SetID(id *datatype.UUID)
	GetCreatedAt() *time.Time
	GetUpdatedAt() *time.Time
	// GetDeletedAt() // we don't use this one
}

// ---------------

// IHasTableName we know if there is Gorm's defined custom TableName
type IHasTableName interface {
	TableName() string
}
