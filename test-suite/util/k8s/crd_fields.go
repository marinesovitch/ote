// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package k8s

import (
	"fmt"
	"math"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type CRDFields struct {
	U *unstructured.Unstructured
}

func NewCRDFields(u *unstructured.Unstructured) CRDFields {
	return CRDFields{U: u}
}

func (c *CRDFields) HasField(fields ...string) bool {
	_, found, err := unstructured.NestedFieldNoCopy(c.U.Object, fields...)
	return found && err == nil
}

func (c *CRDFields) GetString(fields ...string) string {
	value, found, err := unstructured.NestedString(c.U.Object, fields...)
	if err != nil || !found {
		return fmt.Sprintf("--- incorrect value: cannot get meta string %v ---", fields)
	}
	return value
}

func (c *CRDFields) GetStrings(fields ...string) []string {
	value, found, err := unstructured.NestedStringSlice(c.U.Object, fields...)
	if err != nil || !found {
		return nil
	}
	return value
}

func (c *CRDFields) GetSlice(fields ...string) []interface{} {
	value, found, err := unstructured.NestedSlice(c.U.Object, fields...)
	if err != nil || !found {
		return nil
	}
	return value
}

func (c *CRDFields) GetSliceOfStrings(fields ...string) []string {
	slice := c.GetSlice(fields...)
	if slice == nil {
		return nil
	}

	strings := make([]string, len(slice))
	for i, item := range slice {
		switch itemValue := item.(type) {
		case map[string]interface{}:
			for _, value := range itemValue {
				strings[i] = value.(string)
			}
		}
	}
	return strings
}

func (c *CRDFields) GetInt(fields ...string) int {
	value, found, err := unstructured.NestedInt64(c.U.Object, fields...)
	if err != nil || !found {
		const IncorrectValue = math.MinInt
		return IncorrectValue
	}
	return int(value)
}

func (c *CRDFields) GetOptionalInt(defaultValue int, fields ...string) int {
	value, found, err := unstructured.NestedInt64(c.U.Object, fields...)
	if err != nil || !found {
		return defaultValue
	}
	return int(value)
}

func (c *CRDFields) GetInt64(fields ...string) int64 {
	value, found, err := unstructured.NestedInt64(c.U.Object, fields...)
	if err != nil || !found {
		const IncorrectValue = math.MinInt64
		return IncorrectValue
	}
	return value
}

func (c *CRDFields) GetTimeStamp(fields ...string) int64 {
	value, found, err := unstructured.NestedInt64(c.U.Object, fields...)
	if err != nil || !found {
		const IncorrectValue = math.MinInt64
		return IncorrectValue
	}
	return value
}

func (c *CRDFields) GetStatus() string {
	return c.GetString("status", "status")
}

func (c *CRDFields) AskString(fields ...string) (string, bool, error) {
	return unstructured.NestedString(c.U.Object, fields...)
}

func (c *CRDFields) AskTimeStamp(fields ...string) (int64, bool, error) {
	return unstructured.NestedInt64(c.U.Object, fields...)
}

func (c *CRDFields) AskStatus() (string, bool, error) {
	return c.AskString("status", "status")
}

// our CRD types
type MySQLBackup struct {
	*unstructured.Unstructured
	CRDFields
}

type InnoDBCluster struct {
	*unstructured.Unstructured
	CRDFields
}
