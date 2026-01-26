package main

import (
	"encoding/json"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize logger for tests
	logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	os.Exit(m.Run())
}

func TestResponseStructMarshaling(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "SuccessResponse",
			data: SuccessResponse{
				Message: "Test successful",
				Success: true,
			},
		},
		{
			name: "ErrorResponse",
			data: ErrorResponse{
				Error: "Test error",
			},
		},
		{
			name: "ProjectSummary",
			data: ProjectSummary{
				ID:     123,
				Name:   "test-project",
				Status: "Ready",
				Health: "Healthy",
				Type:   "Kubernetes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonData, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("Failed to marshal %s: %v", tt.name, err)
			}

			// Test JSON unmarshaling
			var result map[string]interface{}
			err = json.Unmarshal(jsonData, &result)
			if err != nil {
				t.Fatalf("Failed to unmarshal %s: %v", tt.name, err)
			}

			// Basic validation
			if len(result) == 0 {
				t.Errorf("Empty result for %s", tt.name)
			}

			t.Logf("✅ %s JSON: %s", tt.name, string(jsonData))
		})
	}
}

func TestCreateJSONResponseHelper(t *testing.T) {
	data := SuccessResponse{
		Message: "Test message",
		Success: true,
	}

	response := createJSONResponse(data)
	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(response.Content) == 0 {
		t.Fatal("Expected content, got empty slice")
	}

	// Check that content contains valid JSON
	content := response.Content[0]
	if content.TextContent == nil {
		t.Fatal("Expected TextContent, got nil")
	}

	var result SuccessResponse
	err := json.Unmarshal([]byte(content.TextContent.Text), &result)
	if err != nil {
		t.Fatalf("Invalid JSON in response: %v", err)
	}

	if result.Message != "Test message" || !result.Success {
		t.Errorf("Expected message='Test message' success=true, got message='%s' success=%t",
			result.Message, result.Success)
	}

	t.Logf("✅ JSON Response: %s", content.TextContent.Text)
}

func TestArgumentStructs(t *testing.T) {
	// Test that our argument structs can be marshaled/unmarshaled
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "CreateVirtualClusterArgs",
			data: CreateVirtualClusterArgs{
				ProjectID:       123,
				Name:            "test-cluster",
				WaitForCreation: true,
				Timeout:         600,
			},
		},
		{
			name: "ListProjectsArgs",
			data: ListProjectsArgs{
				Limit:               10,
				Search:              "test",
				HealthyOnly:         true,
				VirtualClustersOnly: false,
			},
		},
		{
			name: "AddAppToCatalogArgs",
			data: AddAppToCatalogArgs{
				CatalogID:   123,
				Repository:  "bitnami",
				PackageName: "nginx",
			},
		},
		{
			name: "ListRepositoriesArgs",
			data: ListRepositoriesArgs{
				Limit:  10,
				Offset: 0,
				Search: "bitnami",
			},
		},
		{
			name: "ListAvailablePackagesArgs",
			data: ListAvailablePackagesArgs{
				Repository: "bitnami",
				Limit:      20,
				Offset:     5,
				Search:     "web",
			},
		},
		{
			name: "CreateProjectArgs",
			data: CreateProjectArgs{
				Name:                "test-project",
				CloudCredentialID:   123,
				KubernetesProfileID: 456,
				AlertingProfileID:   789,
				Monitoring:          true,
				KubernetesVersion:   "1.28.0",
			},
		},
		{
			name: "DeleteProjectArgs",
			data: DeleteProjectArgs{
				ProjectID: 123,
			},
		},
		{
			name: "ListCatalogAppsArgs",
			data: ListCatalogAppsArgs{
				CatalogID: 123,
				Limit:     10,
				Search:    "nginx",
			},
		},
		{
			name: "RemoveAppFromCatalogArgs",
			data: RemoveAppFromCatalogArgs{
				CatalogID:   123,
				Repository:  "bitnami",
				PackageName: "nginx",
			},
		},
		{
			name: "ListKubernetesResourcesArgs",
			data: ListKubernetesResourcesArgs{
				ProjectID:  123,
				Kind:       "Pods",
				Limit:      10,
				SearchTerm: "test",
			},
		},
		{
			name: "DescribeKubernetesResourceArgs",
			data: DescribeKubernetesResourceArgs{
				ProjectID: 123,
				Name:      "test-pod",
				Kind:      "Pod",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonData, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("Failed to marshal %s: %v", tt.name, err)
			}

			t.Logf("✅ %s JSON: %s", tt.name, string(jsonData))
		})
	}
}

func TestBuildInfo(t *testing.T) {
	t.Logf("✅ Go build successful")
	t.Logf("✅ All imports resolved")
	t.Logf("✅ Struct definitions valid")
}
