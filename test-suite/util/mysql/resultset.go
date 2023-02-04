// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package mysql

import (
	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/common"
)

type Strings []string

type RawRow []interface{}
type RawRows []RawRow

type Record struct {
	Columns []string
	Row     RawRow
}

func (r *Record) ColumnIndex(columnName string) int {
	return auxi.Find(r.Columns, columnName)
}

func (r *Record) ScanString(columnName string) (string, bool) {
	columnIndex := r.ColumnIndex(columnName)
	if columnIndex == auxi.NotFound {
		return "", false
	}

	return *r.Row[columnIndex].(*string), true
}

type Records struct {
	Columns []string
	Rows    RawRows
}

func (r *Records) Empty() bool {
	return len(r.Rows) == 0
}

func (r *Records) Size() int {
	return len(r.Rows)
}

func (r *Records) Get(index int) *Record {
	return &Record{Columns: r.Columns, Row: r.Rows[index]}
}

func (r *Records) ToStrings() []Strings {
	result := make([]Strings, len(r.Rows))
	for i, row := range r.Rows {
		result[i] = make(Strings, len(row))
		for j, item := range row {
			result[i][j] = *item.(*string)
		}
	}
	return result
}

func (r *Records) ToStringsSlice(column int) []string {
	result := make([]string, len(r.Rows))
	for i, row := range r.Rows {
		switch value := row[column].(type) {
		case string:
			result[i] = value
		case *string:
			result[i] = *value
		}
	}
	return result
}

func (r *Records) ToStringSet(column int) common.StringSet {
	result := make(common.StringSet)
	for _, row := range r.Rows {
		key := row[column].(*string)
		if key != nil {
			result[*key] = common.MarkExists
		}
	}
	return result
}

func (r *Records) ToStringToStringMap(keyColumn int, valueColumn int) common.StringToStringMap {
	result := make(common.StringToStringMap)
	for _, row := range r.Rows {
		key := row[keyColumn].(*string)
		if key != nil {
			value := row[valueColumn].(*string)
			result[*key] = *value
		}
	}
	return result
}
