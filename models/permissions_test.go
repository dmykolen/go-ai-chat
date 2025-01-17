package models

import (
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestLoadPermissionsFromBytes(t *testing.T) {
	t.Log(os.Getwd())

	// Create a temporary YAML file for testing
	tempFile, err := os.CreateTemp("", "permissions-*.yaml")
	assert.NoError(t, err, "Failed to create temp file")
	defer os.Remove(tempFile.Name())

	// Write test data to the temporary file
	testData := PermissionsConfig{
		Routes: []RoutePermission{
			{
				Path:               "/test",
				Methods:            []string{"GET", "POST"},
				RequiredPermission: "test_permission",
			},
		},
		Templates: []TemplatePermission{
			{
				Template: "test_template",
				Elements: []ElementPermission{
					{
						ID:                 "element1",
						RequiredPermission: "element_permission",
					},
				},
			},
		},
	}

	data, err := yaml.Marshal(&testData)
	assert.NoError(t, err, "Failed to marshal test data")

	_, err = tempFile.Write(data)
	assert.NoError(t, err, "Failed to write to temp file")
	tempFile.Close()

	// Load the permissions config from the temporary file
	pc, err := LoadPermissionsFromBytes(lo.Must(os.ReadFile(tempFile.Name())))
	assert.NoError(t, err, "Failed to load permissions from bytes")

	t.Log(pc.String())

	// Verify the loaded data
	assert.Len(t, pc.Routes, 1, "Expected 1 route")
	assert.Equal(t, "/test", pc.Routes[0].Path, "Expected route path '/test'")
	assert.Len(t, pc.Routes[0].Methods, 2, "Expected 2 methods")
	assert.Equal(t, "test_permission", pc.Routes[0].RequiredPermission, "Expected required permission 'test_permission'")

	assert.Len(t, pc.Templates, 1, "Expected 1 template")
	assert.Equal(t, "test_template", pc.Templates[0].Template, "Expected template 'test_template'")
	assert.Len(t, pc.Templates[0].Elements, 1, "Expected 1 element")
	assert.Equal(t, "element1", pc.Templates[0].Elements[0].ID, "Expected element ID 'element1'")
	assert.Equal(t, "element_permission", pc.Templates[0].Elements[0].RequiredPermission, "Expected element required permission 'element_permission'")
}
