package detect

import (
	"go/types"
	"testing"
)

func TestOptionMetadata_IsDefault(t *testing.T) {
	tests := []struct {
		name     string
		metadata OptionMetadata
		want     bool
	}{
		{
			name: "default option set",
			metadata: OptionMetadata{
				IsDefault: true,
			},
			want: true,
		},
		{
			name: "default option not set",
			metadata: OptionMetadata{
				IsDefault: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metadata.IsDefault; got != tt.want {
				t.Errorf("OptionMetadata.IsDefault = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptionMetadata_HasTypedInterface(t *testing.T) {
	tests := []struct {
		name     string
		metadata OptionMetadata
		want     bool
	}{
		{
			name: "typed interface set",
			metadata: OptionMetadata{
				TypedInterface: types.NewInterfaceType(nil, nil),
			},
			want: true,
		},
		{
			name: "typed interface not set",
			metadata: OptionMetadata{
				TypedInterface: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.TypedInterface != nil
			if got != tt.want {
				t.Errorf("OptionMetadata.TypedInterface != nil = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptionMetadata_HasName(t *testing.T) {
	tests := []struct {
		name     string
		metadata OptionMetadata
		want     bool
	}{
		{
			name: "name set",
			metadata: OptionMetadata{
				Name: "testName",
			},
			want: true,
		},
		{
			name: "name empty",
			metadata: OptionMetadata{
				Name: "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := len(tt.metadata.Name) > 0
			if got != tt.want {
				t.Errorf("len(OptionMetadata.Name) > 0 = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptionMetadata_WithoutConstructor(t *testing.T) {
	tests := []struct {
		name     string
		metadata OptionMetadata
		want     bool
	}{
		{
			name: "without constructor set",
			metadata: OptionMetadata{
				WithoutConstructor: true,
			},
			want: true,
		},
		{
			name: "without constructor not set",
			metadata: OptionMetadata{
				WithoutConstructor: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metadata.WithoutConstructor; got != tt.want {
				t.Errorf("OptionMetadata.WithoutConstructor = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptionMetadata_MultipleOptions(t *testing.T) {
	// Test that OptionMetadata can hold multiple options simultaneously
	metadata := OptionMetadata{
		IsDefault:      false,
		TypedInterface: types.NewInterfaceType(nil, nil),
		Name:           "customName",
	}

	if metadata.IsDefault {
		t.Error("IsDefault should be false")
	}
	if metadata.TypedInterface == nil {
		t.Error("TypedInterface should be set")
	}
	if metadata.Name != "customName" {
		t.Errorf("Name = %q, want %q", metadata.Name, "customName")
	}
	if metadata.WithoutConstructor {
		t.Error("WithoutConstructor should be false")
	}
}
