package restapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrInvalidObjectType = errors.New("invalid object type")
	ErrObjectKeyNotFound = errors.New("key not found in object")
	ErrJSONMarshal       = errors.New("can not parse json")
)

// GetStringAtKey uses GetObjectAtKey to verify the resulting object is either
// a JSON string or Number and returns it as a string.
func GetStringAtKey(data map[string]interface{}, path string) (string, error) {
	res, err := GetObjectAtKey(data, path)
	if err != nil {
		return "", err
	}

	switch v := res.(type) {
	case string:
		return v, nil
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v), nil
	default:
		return "", fmt.Errorf("%w: path '%s': '%T'", ErrInvalidObjectType, path, res)
	}
}

// GetObjectAtKey is a handy helper that will dig through a map and find something
// at the defined key. The returned data is not type checked
//
// Example:
// Given:
//
//	{
//		"attrs": {
//			"id": 1234
//		},
//		"config": {
//			"foo": "abc",
//			"bar": "xyz"
//		}
//	}
//
// Result:
// attrs/id => 1234
// config/foo => "abc".
func GetObjectAtKey(data map[string]interface{}, path string) (interface{}, error) {
	hash := data
	parts := strings.Split(path, "/")
	part := ""
	seen := ""

	for len(parts) > 1 {
		part, parts = parts[0], parts[1:]

		// Protect against double slashes by mistake
		if part == "" {
			continue
		}

		// See if this key exists in the hash at this point
		if _, ok := hash[part]; ok {
			seen += "/" + part

			if tmp, ok := hash[part].(map[string]interface{}); ok {
				hash = tmp
			} else if tmp, ok := hash[part].([]interface{}); ok {
				mapString := make(map[string]interface{})

				for key, value := range tmp {
					strKey := fmt.Sprintf("%v", key)
					mapString[strKey] = value
				}

				hash = mapString
			} else {
				return nil, fmt.Errorf("%w: object '%s': not a map, please check the path", ErrInvalidObjectType, seen)
			}
		} else {
			return nil, fmt.Errorf("%w: want key '%s' in data after '%s': available: %s",
				ErrObjectKeyNotFound, part, seen, strings.Join(GetKeys(hash), ","))
		}
	}

	// Containing map of the wanted value found */
	part = parts[0]
	if _, ok := hash[part]; !ok {
		return nil, fmt.Errorf("%w: want key '%s' in map at '%s': available: %s",
			ErrObjectKeyNotFound, part, seen, strings.Join(GetKeys(hash), ","))
	}

	return hash[part], nil
}

// GetKeys is a handy helper to just dump the keys of a map into a slice.
func GetKeys(hash map[string]interface{}) []string {
	keys := make([]string, 0)

	for k := range hash {
		keys = append(keys, k)
	}

	return keys
}

// GetEnvOrDefault is a helper function that returns the value of the
// given environment variable, if one exists, or the default value.
func GetEnvOrDefault(k, defaultvalue string) string {
	v := os.Getenv(k)
	if v == "" {
		return defaultvalue
	}

	return v
}

func GetRequestData(data, overwrite map[string]any) (string, error) {
	var err error

	b := []byte{}

	if overwrite != nil {
		data = overwrite
	}

	if data != nil {
		b, err = json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("%w: %s", ErrJSONMarshal, err.Error())
		}
	}

	return string(b), nil
}
