// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package auxi

import (
	b64 "encoding/base64"
	"reflect"
	"sort"

	"github.com/marinesovitch/ote/test-suite/util/common"
)

func B64encode(s string) string {
	return b64.StdEncoding.EncodeToString([]byte(s))
}

const NotFound = -1

func Contains(values []string, expected string) bool {
	index := Find(values, expected)
	return index != NotFound
}

func Find(values []string, expected string) int {
	for i, value := range values {
		if value == expected {
			return i
		}
	}
	return NotFound
}

func SliceToSet(src []string) common.StringSet {
	dest := make(common.StringSet)
	for _, elem := range src {
		dest[elem] = common.MarkExists
	}
	return dest
}

func AddToSet(src []string, dest common.StringSet) common.StringSet {
	for _, elem := range src {
		dest[elem] = common.MarkExists
	}
	return dest
}

func AreStringSlicesEqual(lhs []string, rhs []string) bool {
	sort.Strings(lhs)
	sort.Strings(rhs)
	return reflect.DeepEqual(lhs, rhs)
}

func AreStringSetsEqual(lhs common.StringSet, rhs common.StringSet) bool {
	return reflect.DeepEqual(lhs, rhs)
}
