package restapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	ErrInvalidObjectType = errors.New("invalid object type")
	ErrObjectKeyNotFound = errors.New("key not found in object")
	ErrJSONMarshal       = errors.New("can not parse json")
)

// GetStringAtKey returns the string value at the given slash-delimited path
// in the provided map[string]any data. It will attempt to convert
// values of other types like numbers to strings. Returns error if path not
// found or value is not a string or convertible type.
func GetStringAtKey(data map[string]any, path string) (string, error) {
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

// GetObjectAtKey recursively traverses a map[string]any to find the value
// at the given slash-delimited path. It can handle maps and slices in the path.
//
// Example inputs:
//
//	data = {
//	  "foo": {
//	    "bar": [1, 2, 3]
//	  }
//	}
//
// path = "foo/bar/0"
//
// Returns 1.
func GetObjectAtKey(data map[string]any, path string) (any, error) {
	var (
		seen, part string
		i          int
	)

	parts := strings.Split(SanitizePath(path), "/")

	for i, part = range parts {
		// Protect against double slashes by mistake
		if part == "" || i == len(parts)-1 {
			continue
		}

		// See if this key exists in the data at this point
		if _, ok := data[part]; !ok {
			return nil, fmt.Errorf("%w: want key '%s' in data after '%s': available: %s",
				ErrObjectKeyNotFound, part, seen, strings.Join(GetKeys(data), ","))
		}

		seen += "/" + part

		if tmp, ok := data[part].(map[string]any); ok {
			data = tmp
		} else if tmp, ok := data[part].([]any); ok {
			mapString := make(map[string]any)

			for key, value := range tmp {
				strKey := fmt.Sprintf("%v", key)
				mapString[strKey] = value
			}

			data = mapString
		} else {
			return nil, fmt.Errorf("%w: object '%s': not a map, please check the path", ErrInvalidObjectType, seen)
		}
	}

	// Containing map of the wanted value found */
	if _, ok := data[part]; !ok {
		return nil, fmt.Errorf("%w: want key '%s' in map at '%s': available: %s",
			ErrObjectKeyNotFound, part, seen, strings.Join(GetKeys(data), ","))
	}

	return data[part], nil
}

// GetKeys returns a slice containing all the keys in the given hash map.
func GetKeys(hash map[string]any) []string {
	keys := make([]string, 0)

	for k := range hash {
		keys = append(keys, k)
	}

	return keys
}

// GetEnvOrDefault returns the value of the environment variable k if set,
// or defaultvalue if not set.
func GetEnvOrDefault(k, defaultvalue string) string {
	v := os.Getenv(k)
	if v == "" {
		return defaultvalue
	}

	return v
}

// GetRequestData marshals the provided data map into a JSON string.
// If overwrite is provided, it will override data.
// Returns the JSON string and any error from json.Marshal.
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

// SanitizePath removes duplicate slashes and trailing slash from the provided path.
func SanitizePath(path string) string {
	// Replace multiple slashes with single slash
	path = regexp.MustCompile(`/{2,}`).ReplaceAllString(path, "/")
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimPrefix(path, "/")

	return path
}
