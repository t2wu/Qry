package qry

import (
	"log"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	HOST     = "127.0.0.1"
	PORT     = "5432"
	USERNAME = "postgres"
	PASSWORD = "12345678"
	DBNAME   = "localdb"
)

var db *gorm.DB

var topLevel TopLevel
var unnested1 Unnested
var unnested2 Unnested
var unnested3 Unnested

// Package level setup
func TestMain(m *testing.M) {

	dsn := "host=" + HOST + " port=" + PORT + " user=" + USERNAME +
		" dbname=" + DBNAME + " password=" + PASSWORD + " sslmode=disable"

	var err error

	logx := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  true,
	})

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logx,
	})
	if err != nil {
		panic("failed to connect database:" + err.Error())
	}

	// ignore error now that AutoMigrate can't be chained!
	if err := db.AutoMigrate(&TopLevel{}, &SecLevelArrDog{}, &ThirdLevelArrDogToy{},
		&SecLevelEmbedDog{}, &ThirdLevelEmbedDogToy{}, &SecLevelPtrDog{}, &ThirdLevelPtrDogToy{},
		&SecLevelArrCat{}, &SecLevelEmbedCat{}, &SecLevelPtrCat{}, &Unnested{}, &UnnestedInner{}); err != nil {
		panic("failed to create tables:" + err.Error())
	}

	// uuidTopLevel := datatype.AddrOfUUID(uuid.FromStringOrNil("40857A84-57F2-4EBD-BFC7-D0B54AE5BC54"))
	// uuidSecLevelArrDog1 := datatype.AddrOfUUID(uuid.FromStringOrNil("2935AC43-8992-4804-874A-282BF99E96EF"))
	// uuidThirdLevelDogToy := datatype.AddrOfUUID(uuid.FromStringOrNil("93478A1C-96DF-4208-A4E0-F10D0E14D45B"))
	// uuidSecLevelEmbedDog1 := datatype.AddrOfUUID(uuid.FromStringOrNil("C9E18E39-7FF6-4B33-9F40-6057734D7BF9"))
	// uuidSecLevelPtrDog1 := datatype.AddrOfUUID(uuid.FromStringOrNil("0D4B40FD-0AF0-413D-BB56-3F720628B101"))
	// uuidSecLevelArrCat1 := datatype.AddrOfUUID(uuid.FromStringOrNil("D947252E-51F4-4180-8233-71AFA8A0583A"))
	// uuidSecLevelEmbedCat1 := datatype.AddrOfUUID(uuid.FromStringOrNil("BDAD5D7C-EF8E-46AF-8113-7A7545508D0F"))
	// uuidSecLevelPtrCat1 := datatype.AddrOfUUID(uuid.FromStringOrNil("9B82BFBE-E63B-48BC-9A4F-95A6A6B21289"))

	// topLevel = TopLevel{
	// 	BaseModel: mdl.BaseModel{ID: uuidTopLevel},
	// 	Name:      "TopLevel",
	// 	Age:       20,
	// 	Dogs: []SecLevelArrDog{
	// 		{
	// 			BaseModel: mdl.BaseModel{ID: uuidSecLevelArrDog1},
	// 			Name:      "SecondLevelArrDog1",
	// 			Color:     "orange",
	// 			DogToys: []ThirdLevelArrDogToy{
	// 				{
	// 					BaseModel: mdl.BaseModel{ID: uuidThirdLevelDogToy},
	// 					ToyName:   "DogToy5",
	// 				},
	// 			},
	// 		},
	// 	},
	// 	EmbedDog: SecLevelEmbedDog{
	// 		BaseModel: mdl.BaseModel{ID: uuidSecLevelEmbedDog1},
	// 		Name:      "FavoriteDog",
	// 		Color:     "teal",
	// 	},
	// 	PtrDog: &SecLevelPtrDog{
	// 		BaseModel: mdl.BaseModel{ID: uuidSecLevelPtrDog1},
	// 		Name:      "EvilDog",
	// 		Color:     "magenta",
	// 	},

	// 	// cat

	// 	Cats: []SecLevelArrCat{
	// 		{
	// 			BaseModel: mdl.BaseModel{ID: uuidSecLevelArrCat1},
	// 			Name:      "SecondLevelArrCat1",
	// 			Color:     "red",
	// 		},
	// 	},
	// 	EmbedCat: SecLevelEmbedCat{
	// 		BaseModel: mdl.BaseModel{ID: uuidSecLevelEmbedCat1},
	// 		Name:      "SecondLevelArrCat1",
	// 		Color:     "red",
	// 	},
	// 	PtrCat: &SecLevelPtrCat{
	// 		BaseModel: mdl.BaseModel{ID: uuidSecLevelPtrCat1},
	// 		Name:      "SecondLevelPtrCat1",
	// 		Color:     "blue",
	// 	},
	// }

	// uunidUnnested1 := "7192f73d-e56f-4a33-a7fb-eb9d605bc731"
	// uuidUnestedInner1 := "2174e7ce-708d-4b46-a1b4-59c41304b46"
	// unnested1 = Unnested{
	// 	BaseModel: mdl.BaseModel{
	// 		ID: datatype.AddrOfUUID(uuid.FromStringOrNil(uunidUnnested1)),
	// 	},
	// 	UnnestedInner: UnnestedInner{
	// 		BaseModel: mdl.BaseModel{
	// 			ID: datatype.AddrOfUUID(uuid.FromStringOrNil(uuidUnestedInner1)),
	// 		},
	// 		Name: "UnNestedInnerSameNameWith1&2",
	// 		// UnnestedID: datatype.AddrOfUUID(uuid.FromStringOrNil(uunidUnnested1)),
	// 	},
	// 	Name:       "unnested1",
	// 	TopLevelID: uuidTopLevel,
	// }

	// uunidUnnested2 := "6cdb2b20-b6c6-4f8f-9c2f-632888887865"
	// uuidUnestedInner2 := "3441e7ce-708d-4b46-a1b4-59c41300cd48"
	// unnested2 = Unnested{
	// 	BaseModel: mdl.BaseModel{
	// 		ID: datatype.AddrOfUUID(uuid.FromStringOrNil(uunidUnnested2)),
	// 	},
	// 	UnnestedInner: UnnestedInner{
	// 		BaseModel: mdl.BaseModel{
	// 			ID: datatype.AddrOfUUID(uuid.FromStringOrNil(uuidUnestedInner2)),
	// 		},
	// 		Name: "UnNestedInnerSameNameWith1&2",
	// 		// UnnestedID: datatype.AddrOfUUID(uuid.FromStringOrNil(unnesteduuid2)),
	// 	},
	// 	Name:       "unnested2",
	// 	TopLevelID: uuidTopLevel,
	// }

	// uunidUnnested3 := "e2bf6b2a-127c-491b-b6a2-49d88d217425"
	// unnested3 = Unnested{
	// 	BaseModel: mdl.BaseModel{
	// 		ID: datatype.AddrOfUUID(uuid.FromStringOrNil(uunidUnnested3)),
	// 	},
	// 	Name:       "unnested3",
	// 	TopLevelID: uuidTopLevel,
	// }

	// log.Println(tm1, tm2, tm3, tm4, tm5, unnested1, unnested2, dogToy1, dogToy2)
	// db.LogMode(false)
	// db.Config.Logger = logger.Default

	// if err := db.Create(&topLevel).Error; err != nil {
	// 	panic("something wrong with populating the db:" + err.Error())
	// }
	// if err := db.Create(&unnested1).Create(&unnested2).Create(&unnested3).Error; err != nil {
	// 	panic("something wrong with populating the db:" + err.Error())
	// }

	// db.LogMode(true)
	// db.Config.Logger = logger.Default

	exitVal := m.Run()

	// db.Config.Logger = logger.Discard

	// Teardown
	// (Gorm v2 no longer allow me to do .Delete().Delete(), why not?)
	// if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&DogToy{}).Error; err != nil {
	// 	panic("something wrong with removing data from the db:" + err.Error())
	// }

	// if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Dog{}).Error; err != nil {
	// 	panic("something wrong with removing data from the db:" + err.Error())
	// }

	// if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&UnNestedInner{}).Error; err != nil {
	// 	panic("something wrong with removing data from the db:" + err.Error())
	// }

	// if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&UnNested{}).Error; err != nil {
	// 	panic("something wrong with removing data from the db:" + err.Error())
	// }

	// if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&TestModel{}).Error; err != nil {
	// 	panic("something wrong with removing data from the db:" + err.Error())
	// }

	os.Exit(exitVal)
}
