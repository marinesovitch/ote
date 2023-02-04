// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package common

import "sort"

type Command int

const (
	Unknown Command = iota
	Start
	Stop
	Deploy
)

type StringSet map[string]struct{}
type StringToStringMap map[string]string

func (s *StringSet) ToSortedSlice() []string {
	keys := make([]string, 0, len(*s))
	for key := range *s {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
