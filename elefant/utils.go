package elefant

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
)

func scanNullUUID(source interface{}) (uuid.UUID, bool, error) {
	isValid := source != nil
	if !isValid {
		return uuid.UUID{}, false, nil
	}
	switch value := source.(type) {
	case []byte:
		{
			result, err := uuid.ParseBytes(value)
			if err != nil {
				return uuid.UUID{}, false, fmt.Errorf(
					`failed to parse UUID from DB-value "%v": "%v"`, value, err)
			}
			return result, true, nil
		}
	}
	return uuid.UUID{}, false, fmt.Errorf(
		`failed to use DB-type "%v" to read UUID`, reflect.TypeOf(source))
}
