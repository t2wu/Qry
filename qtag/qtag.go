package qtag

import (
	"reflect"
	"strings"
)

type QryTag string

const (
	QryTagNone              QryTag = ""
	QryTagPeg               QryTag = "peg"
	QryTagPegAssoc          QryTag = "pegassoc"
	QryTagPegAssocMany2Many QryTag = "pegassoc-many2many"
	QryTagPegIgnore         QryTag = "-" // peg-ignore
)

type TagNField struct {
	Tag   QryTag
	Field string
}

// TagStrings is the tag this library handles, betterrest is deprecated
var TagStrings = []string{"qry", "betterrest"}

// GetQryTag
func GetQryTag(tag reflect.StructTag) QryTag {
	t := tag.Get("qry")
	if t == "" {
		t = tag.Get("betterrest")
	}

	if TagValueHasPrefix(t, string(QryTagPegAssocMany2Many)) {
		return QryTagPegAssocMany2Many
	} else if TagValueHasPrefix(t, string(QryTagPegAssoc)) {
		return QryTagPegAssoc
	} else if TagValueHasPrefix(t, string(QryTagPeg)) {
		return QryTagPeg
	}

	return QryTagNone // shouldn't be here
}

// GetQryTagAndField. So far we only have options that are exclusive of each other
// but they may not be in the future, so we return an array
func GetQryTagAndField(tag reflect.StructTag) []TagNField {
	t := tag.Get("qry")
	if t == "" {
		t = tag.Get("betterrest")
	}

	retval := make([]TagNField, 0)

	// Reverse order since pegassoc-many2-many can match pegassoc and peg as well
	if field := TagFieldByPrefix(t, string(QryTagPegAssocMany2Many)); field != "" {
		retval = append(retval, TagNField{
			Tag:   QryTagPegAssocMany2Many,
			Field: strings.Split(field, ":")[1],
		})
	} else if field := TagFieldByPrefix(t, string(QryTagPegAssoc)); field != "" {
		retval = append(retval, TagNField{
			Tag: QryTagPegAssoc,
		})
	} else if field := TagFieldByPrefix(t, string(QryTagPeg)); field != "" {
		retval = append(retval, TagNField{
			Tag: QryTagPeg,
		})
	}

	return retval
}

func IsTag(tag reflect.StructTag, targets ...QryTag) bool {
	t := GetQryTag(tag)
	for _, target := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// IsAnyQryTag checks if there is any tag under qry or betterrest
func IsAnyQryTag(tag reflect.StructTag) bool {
	return IsTag(tag, QryTagPeg, QryTagPegAssoc, QryTagPegAssocMany2Many)
}
