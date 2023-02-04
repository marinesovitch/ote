// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package auxi

import (
	"encoding/json"
	"fmt"
)

func FromJsonError(path string, jsonErr error) error {
	err := jsonErr.(*json.SyntaxError)
	return fmt.Errorf("%s: %s (offset %d)", path, err, err.Offset)
}
