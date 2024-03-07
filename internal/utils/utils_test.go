package utils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MapAny = map[string]any

func TestGetStringAtKey(t *testing.T) {
	data := MapAny{
		"foo":  "bar",
		"baz":  123,
		"bool": true,
	}

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr error
	}{
		{
			name:    "string value",
			key:     "foo",
			want:    "bar",
			wantErr: nil,
		},
		{
			name:    "int value",
			key:     "baz",
			want:    "123",
			wantErr: nil,
		},
		{
			name:    "bool value",
			key:     "bool",
			want:    "true",
			wantErr: ErrInvalidObjectType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := GetStringAtKey(data, tt.key)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, val)
		})
	}
}

func TestGetObjectAtKey(t *testing.T) {
	tests := []struct {
		name    string
		data    MapAny
		key     string
		want    any
		wantErr error
	}{
		{
			name: "valid nested key",
			data: MapAny{
				"foo": MapAny{
					"bar": "baz",
				},
			},
			key:     "foo/bar",
			want:    "baz",
			wantErr: nil,
		},
		{
			name: "valid direct key",
			data: MapAny{
				"a": 123,
			},
			key:     "a",
			want:    123,
			wantErr: nil,
		},
		{
			name: "valid nested slice",
			data: MapAny{
				"foo": MapAny{
					"bar": []any{1, "2", 3},
				},
			},
			key:     "foo/bar/1",
			want:    "2",
			wantErr: nil,
		},
		{
			name:    "key not found",
			data:    MapAny{},
			key:     "invalid",
			want:    nil,
			wantErr: ErrObjectKeyNotFound,
		},
		{
			name: "invalid object type",
			data: MapAny{
				"foo": "bar",
			},
			key:     "foo/baz",
			want:    nil,
			wantErr: ErrInvalidObjectType,
		},
		{
			name: "key not found in object",
			data: MapAny{
				"foo": MapAny{
					"bar": "baz",
				},
			},
			key:     "foo/baz",
			want:    nil,
			wantErr: ErrObjectKeyNotFound,
		},
		{
			name: "unsaitized path",
			data: MapAny{
				"foo": MapAny{
					"bar": "baz",
				},
			},
			key:     "//foo/////bar///",
			want:    "baz",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetObjectAtKey(tt.data, tt.key)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetKeys(t *testing.T) {
	data := MapAny{
		"foo": "bar",
		"baz": 123,
	}

	keys := GetKeys(data)
	assert.ElementsMatch(t, []string{"foo", "baz"}, keys)
}

func TestGetEnvOrDefault(t *testing.T) {
	val := GetEnvOrDefault("FOO", "default")
	assert.Equal(t, "default", val)

	t.Setenv("FOO", "bar")

	val = GetEnvOrDefault("FOO", "default")
	assert.Equal(t, "bar", val)
}

func TestGetRequestData(t *testing.T) {
	tests := []struct {
		name      string
		data      MapAny
		overwrite MapAny
		expected  string
		wantErr   error
	}{
		{
			name:      "valid data",
			data:      MapAny{"name": "test"},
			overwrite: nil,
			expected:  `{"name":"test"}`,
			wantErr:   nil,
		},
		{
			name:      "overwrite data",
			data:      MapAny{"name": "test"},
			overwrite: MapAny{"name": "overwrite"},
			expected:  `{"name":"overwrite"}`,
			wantErr:   nil,
		},
		{
			name:      "empty data",
			data:      nil,
			overwrite: nil,
			expected:  "",
			wantErr:   nil,
		},
		{
			name:      "empty map",
			data:      make(MapAny),
			overwrite: nil,
			expected:  "{}",
			wantErr:   nil,
		},
		{
			name:      "marshal error",
			data:      MapAny{"name": make(chan int)},
			overwrite: nil,
			expected:  "",
			wantErr:   ErrJSONMarshal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetRequestData(tt.data, tt.overwrite)
			if tt.wantErr != nil {
				assert.Error(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseImportPath(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantID   string
		wantPath string
		wantErr  error
	}{
		{
			name:     "valid path",
			id:       "/api/v1/objects/123",
			wantID:   "123",
			wantPath: "/api/v1/objects",
			wantErr:  nil,
		},
		{
			name:     "valid path with trailing slash",
			id:       "/api/v1/objects/123/",
			wantID:   "123",
			wantPath: "/api/v1/objects",
			wantErr:  nil,
		},
		{
			name:    "invalid path",
			id:      "123",
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotN, err := ParseImportPath(tt.id)
			if tt.wantErr != nil {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantID, gotID)
			assert.Equal(t, tt.wantPath, gotN)
		})
	}
}

func TestIntersectMaps(t *testing.T) {
	testCases := []struct {
		name     string
		map1     MapAny
		map2     MapAny
		expected MapAny
	}{
		{
			name:     "both maps empty",
			map1:     MapAny{},
			map2:     MapAny{},
			expected: MapAny{},
		},
		{
			name:     "one map empty",
			map1:     MapAny{"key1": "value1", "key2": "value2"},
			map2:     MapAny{},
			expected: MapAny{},
		},
		{
			name:     "no intersection",
			map1:     MapAny{"key1": "value1", "key2": "value2"},
			map2:     MapAny{"key3": "value3", "key4": "value4"},
			expected: MapAny{},
		},
		{
			name:     "partial intersection",
			map1:     MapAny{"key1": "value1", "key2": "value2"},
			map2:     MapAny{"key2": "value2", "key3": "value3"},
			expected: MapAny{"key2": "value2"},
		},
		{
			name:     "intersection with more types",
			map1:     MapAny{"key1": 2, "key2": true},
			map2:     MapAny{"key1": 1, "key2": false},
			expected: MapAny{"key1": 1, "key2": false},
		},
		{
			name:     "intersection nested",
			map1:     MapAny{"outside": MapAny{"change": "a"}, "out": "x"},
			map2:     MapAny{"outside": MapAny{"change": "b", "add": []string{}}},
			expected: MapAny{"outside": MapAny{"change": "b"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IntersectMaps(tc.map1, tc.map2)

			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("expected %v but got %v", tc.expected, result)
			}
		})
	}
}
