package qry

import (
	uuid "github.com/satori/go.uuid"
	"github.com/t2wu/qry/mdl"
)

// TopLevel is the model which are associated with all the model
type TopLevel struct {
	mdl.BaseModel

	Name string `gorm:"column:real_name_column" json:"name"`
	Age  int    `json:"age"`

	Dogs []SecLevelArrDog `qry:"peg" gorm:"constraint:OnDelete:CASCADE;" json:"dogs" `

	// Any field with pegassoc should have association_autoupdate:false AND
	// foreign key constraint for cat should have SET NULL on delete and update
	// Cats []Cat `qry:"pegassoc" gorm:"constraint:OnDelete:SET NULL;" gorm:"association_autoupdate:false;" json:"cats"`
	Cats []SecLevelArrCat `qry:"pegassoc" gorm:"constraint:OnDelete:SET NULL;" json:"cats"`

	// Gorm cannot differentiate between Dogs and FavoriteDog
	// so if you set FavoriteDog and try to load Dogs it will also appear there
	EmbedDog SecLevelEmbedDog `qry:"peg" gorm:"constraint:OnDelete:CASCADE;" json:"favoriteDog"`
	EmbedCat SecLevelEmbedCat `qry:"pegassoc" gorm:"constraint:OnDelete:SET NULL;association_autoupdate:false;" json:"favoriteCat"`

	PtrDog *SecLevelPtrDog `qry:"peg" gorm:"constraint:OnDelete:CASCADE;" json:"evilDog"`
	PtrCat *SecLevelPtrCat `qry:"pegassoc" gorm:"constraint:OnDelete:SET NULL;association_autoupdate:false;" json:"evilCat"`
}

// --- dog ---
// Dog are supposed to be pegged
// Peg creation needs to be done by the qry framework (or Gorm framework)
// Peg update also needs to be done by qry framework (or Gorm framework)
// Peg delete should be done with SQL foreign key, UPDATE CASCADE, DELETE CASCADE

type SecLevelArrDog struct {
	mdl.BaseModel

	Name    string                `json:"name"`
	Color   string                `json:"color"`
	DogToys []ThirdLevelArrDogToy `qry:"peg" gorm:"constraint:OnDelete:CASCADE;" json:"dogToy"`

	TopLevelID *uuid.UUID `gorm:"type:uuid;index;not null;" json:"_"`
}

type ThirdLevelArrDogToy struct {
	mdl.BaseModel

	ToyName string `json:"toyName"`

	SecLevelArrDogID *uuid.UUID `gorm:"type:uuid;index;not null;" json:"_"`
}

type SecLevelEmbedDog struct {
	mdl.BaseModel

	Name    string                  `json:"name"`
	Color   string                  `json:"color"`
	DogToys []ThirdLevelEmbedDogToy `qry:"peg" gorm:"constraint:OnDelete:CASCADE;" json:"dogToy"`

	TopLevelID *uuid.UUID `gorm:"type:uuid;index;not null;" json:"_"`
}

type ThirdLevelEmbedDogToy struct {
	mdl.BaseModel

	ToyName string `json:"toyName"`

	SecLevelEmbedDogID *uuid.UUID `gorm:"type:uuid;index;not null;" json:"_"`
}

type SecLevelPtrDog struct {
	mdl.BaseModel

	Name    string                `json:"name"`
	Color   string                `json:"color"`
	DogToys []ThirdLevelPtrDogToy `qry:"peg" gorm:"constraint:OnDelete:CASCADE;" json:"dogToy"`

	TopLevelID *uuid.UUID `gorm:"type:uuid;index;not null;" json:"_"`
}

type ThirdLevelPtrDogToy struct {
	mdl.BaseModel

	ToyName string `json:"toyName"`

	SecLevelPtrDogID *uuid.UUID `gorm:"type:uuid;index;not null;" json:"_"`
}

// --- cat ---
// Cat are supposed to be pegassociated
// Pegassoc creation needs to be done by the qry framework (or Gorm framework)
// Pegassoc update also needs to be done by qry framework (or Gorm framework). The correct behavior is to update
// the association only, not any embedded fields
// Peg delete should remove the association only. UPDATE CASCADE, DELETE NULL

type SecLevelArrCat struct {
	mdl.BaseModel

	Name  string `json:"name"`
	Color string `json:"color"`

	TopLevelID *uuid.UUID `gorm:"type:uuid;index;" json:"_"`
}

type SecLevelEmbedCat struct {
	mdl.BaseModel

	Name  string `json:"name"`
	Color string `json:"color"`

	TopLevelID *uuid.UUID `gorm:"type:uuid;index;" json:"_"`
}

type SecLevelPtrCat struct {
	mdl.BaseModel

	Name  string `json:"name"`
	Color string `json:"color"`

	TopLevelID *uuid.UUID `gorm:"type:uuid;index;" json:"_"`
}

// Test for joins
//

type Unnested struct {
	mdl.BaseModel

	Name          string        `json:"name"`
	UnnestedInner UnnestedInner `qry:"peg" gorm:"constraint:OnDelete:CASCADE;" json:"unnestedInner"`

	TopLevelID *uuid.UUID `gorm:"type:uuid;index;not null;" json:"-"`
}

type UnnestedInner struct {
	mdl.BaseModel

	Name string `json:"name"`

	UnnestedID *uuid.UUID `gorm:"type:uuid;index;not null;" json:"-"`
}
