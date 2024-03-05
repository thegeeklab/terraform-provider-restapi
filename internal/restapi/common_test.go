package restapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStringAtKey(t *testing.T) {
	data := map[string]any{
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
		data    map[string]any
		key     string
		want    any
		wantErr error
	}{
		{
			name: "valid nested key",
			data: map[string]any{
				"foo": map[string]any{
					"bar": "baz",
				},
			},
			key:     "foo/bar",
			want:    "baz",
			wantErr: nil,
		},
		{
			name: "valid direct key",
			data: map[string]any{
				"a": map[string]any{
					"b": 123,
				},
			},
			key:     "a/b",
			want:    123,
			wantErr: nil,
		},
		{
			name:    "key not found",
			data:    map[string]any{},
			key:     "invalid",
			want:    nil,
			wantErr: ErrObjectKeyNotFound,
		},
		{
			name: "invalid object type",
			data: map[string]any{
				"foo": "bar",
			},
			key:     "foo/baz",
			want:    nil,
			wantErr: ErrInvalidObjectType,
		},
		{
			name: "key not found in object",
			data: map[string]any{
				"foo": map[string]any{
					"bar": "baz",
				},
			},
			key:     "foo/baz",
			want:    nil,
			wantErr: ErrObjectKeyNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetObjectAtKey(tt.data, tt.key)
			if tt.want == nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetKeys(t *testing.T) {
	data := map[string]any{
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
		data      map[string]any
		overwrite map[string]any
		expected  string
		wantErr   error
	}{
		{
			name:      "valid data",
			data:      map[string]any{"name": "test"},
			overwrite: nil,
			expected:  `{"name":"test"}`,
			wantErr:   nil,
		},
		{
			name:      "overwrite data",
			data:      map[string]any{"name": "test"},
			overwrite: map[string]any{"name": "overwrite"},
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
			data:      make(map[string]any),
			overwrite: nil,
			expected:  "{}",
			wantErr:   nil,
		},
		{
			name:      "marshal error",
			data:      map[string]any{"name": make(chan int)},
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
