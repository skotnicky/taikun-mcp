package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/itera-io/taikungoclient"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// Build-time variables (set by GoReleaser)
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "unknown"
)

var (
	logger       *log.Logger
	logFilePath  = "/tmp/taikun_mcp_server.log"
	taikunClient *taikungoclient.Client
)

// Response structs for JSON formatting
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

type SuccessResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

type ProjectSummary struct {
	ID                     int32   `json:"id"`
	Name                   string  `json:"name"`
	Status                 string  `json:"status"`
	Health                 string  `json:"health"`
	Type                   string  `json:"type"`
	Cloud                  string  `json:"cloud"`
	Organization           string  `json:"organization"`
	IsLocked               bool    `json:"isLocked"`
	IsVirtualCluster       bool    `json:"isVirtualCluster"`
	ParentProjectID        int32   `json:"parentProjectId,omitempty"`
	CreatedAt              string  `json:"createdAt"`
	ServersCount           int32   `json:"serversCount"`
	StandaloneVMsCount     int32   `json:"standaloneVmsCount"`
	HourlyCost             float64 `json:"hourlyCost"`
	MonitoringEnabled      bool    `json:"monitoringEnabled"`
	BackupEnabled          bool    `json:"backupEnabled"`
	AlertsCount            int32   `json:"alertsCount"`
	ReadyForVirtualCluster bool    `json:"readyForVirtualCluster"`
	VirtualClusterReason   string  `json:"virtualClusterReason,omitempty"`
}

type ProjectListResponse struct {
	Projects   []ProjectSummary `json:"projects"`
	Total      int              `json:"total"`
	FilterType string           `json:"filterType"`
	Message    string           `json:"message"`
}

type VirtualClusterSummary struct {
	ID                 int32  `json:"id"`
	Name               string `json:"name"`
	Status             string `json:"status"`
	Health             string `json:"health"`
	KubernetesVersion  string `json:"kubernetesVersion"`
	CreatedAt          string `json:"createdAt"`
	CreatedBy          string `json:"createdBy"`
	ExpiresAt          string `json:"expiresAt,omitempty"`
	DeleteOnExpiration bool   `json:"deleteOnExpiration"`
	Organization       string `json:"organization"`
	IsLocked           bool   `json:"isLocked"`
	HasKubeconfig      bool   `json:"hasKubeconfig"`
}

type VirtualClusterListResponse struct {
	VirtualClusters []VirtualClusterSummary `json:"virtualClusters"`
	Total           int                     `json:"total"`
	ParentProjectID int32                   `json:"parentProjectId"`
	Message         string                  `json:"message"`
}

type CatalogSummary struct {
	ID            int32  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ProjectsCount int    `json:"projectsCount"`
}

type CatalogListResponse struct {
	Catalogs []CatalogSummary `json:"catalogs"`
	Total    int              `json:"total"`
	Message  string           `json:"message"`
}

type ApplicationSummary struct {
	ID           int32  `json:"id"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	CatalogAppID int32  `json:"catalogAppId"`
	ProjectID    int32  `json:"projectId"`
}

type ApplicationListResponse struct {
	Applications []ApplicationSummary `json:"applications"`
	Total        int                  `json:"total"`
	ProjectID    int32                `json:"projectId"`
	Message      string               `json:"message"`
}

type AddAppToCatalogArgs struct {
	CatalogID   int32  `json:"catalogId" jsonschema:"required,description=The catalog ID to add the application to"`
	Repository  string `json:"repository" jsonschema:"required,description=Repository name (3-30 chars, lowercase/numeric)"`
	PackageName string `json:"packageName" jsonschema:"required,description=Package name (3-30 chars, lowercase/numeric)"`
}

type ListRepositoriesArgs struct {
	Limit  int32  `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (optional)"`
	Offset int32  `json:"offset,omitempty" jsonschema:"description=Number of results to skip (optional)"`
	Search string `json:"search,omitempty" jsonschema:"description=Search term to filter results (optional)"`
}

type ListAvailablePackagesArgs struct {
	Repository string `json:"repository,omitempty" jsonschema:"description=Repository name to filter packages (optional)"`
	Limit      int32  `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (optional)"`
	Offset     int32  `json:"offset,omitempty" jsonschema:"description=Number of results to skip (optional)"`
	Search     string `json:"search,omitempty" jsonschema:"description=Search term to filter results (optional)"`
}

type CreateProjectArgs struct {
	Name                string `json:"name" jsonschema:"required,description=Project name (3-30 characters, alphanumeric with hyphens)"`
	CloudCredentialID   int32  `json:"cloudCredentialId" jsonschema:"required,description=ID of the cloud credential to use for this project"`
	KubernetesProfileID int32  `json:"kubernetesProfileId,omitempty" jsonschema:"description=ID of the Kubernetes profile to use (optional)"`
	AlertingProfileID   int32  `json:"alertingProfileId,omitempty" jsonschema:"description=ID of the alerting profile to use (optional)"`
	Monitoring          bool   `json:"monitoring,omitempty" jsonschema:"description=Enable monitoring for this project (default: false)"`
	KubernetesVersion   string `json:"kubernetesVersion,omitempty" jsonschema:"description=Kubernetes version to install (optional)"`
}

type DeleteProjectArgs struct {
	ProjectID int32 `json:"projectId" jsonschema:"required,description=ID of the project to delete"`
}

type RemoveAppFromCatalogArgs struct {
	CatalogID   int32  `json:"catalogId" jsonschema:"required,description=The catalog ID to remove the application from"`
	Repository  string `json:"repository,omitempty" jsonschema:"description=Repository name (optional - if not provided, will search by package name only)"`
	PackageName string `json:"packageName" jsonschema:"required,description=Package name"`
}

type ListCatalogAppsArgs struct {
	CatalogID int32  `json:"catalogId,omitempty" jsonschema:"description=The catalog ID to list applications from (optional - if not provided, lists from all catalogs)"`
	Limit     int32  `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (optional)"`
	Offset    int32  `json:"offset,omitempty" jsonschema:"description=Number of results to skip (optional)"`
	Search    string `json:"search,omitempty" jsonschema:"description=Search term to filter results (optional)"`
}

type CatalogAppSummary struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	Repository  string `json:"repository"`
	CatalogID   int32  `json:"catalogId"`
	CatalogName string `json:"catalogName"`
}

type CatalogAppListResponse struct {
	Applications []CatalogAppSummary `json:"applications"`
	Total        int                 `json:"total"`
	CatalogID    int32               `json:"catalogId"`
	Message      string              `json:"message"`
}

type CloudCredentialSummary struct {
	ID               int32  `json:"id"`
	Name             string `json:"name"`
	CloudType        string `json:"cloudType"`
	OrganizationName string `json:"organizationName"`
}

type CloudCredentialListResponse struct {
	Credentials []CloudCredentialSummary `json:"credentials"`
	Total       int                      `json:"total"`
	Message     string                   `json:"message"`
}

type ListCloudCredentialsArgs struct {
	Limit   int32  `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (optional)"`
	Offset  int32  `json:"offset,omitempty" jsonschema:"description=Number of results to skip (optional)"`
	Search  string `json:"search,omitempty" jsonschema:"description=Search term to filter results (optional)"`
	IsAdmin bool   `json:"isAdmin,omitempty" jsonschema:"description=Whether to list as admin (optional)"`
}

// createJSONResponse creates a JSON response using NewTextContent
func createJSONResponse(data interface{}) *mcp_golang.ToolResponse {
	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Printf("Error marshaling JSON: %v", err)
		errorResp := ErrorResponse{Error: "Failed to serialize response data"}
		jsonData, _ = json.Marshal(errorResp)
	}
	return mcp_golang.NewToolResponse(
		mcp_golang.NewTextContent(string(jsonData)),
	)
}

// createError creates a formatted error response for MCP tools
func createError(response *http.Response, err error) *mcp_golang.ToolResponse {
	// Use taikungoclient's CreateError for detailed error messages
	taikunErr := taikungoclient.CreateError(response, err)

	var errorResp ErrorResponse
	if taikunErr != nil {
		errorResp.Error = taikunErr.Error()
	} else {
		errorResp.Error = "Unknown error occurred"
	}

	logger.Printf("Error occurred: %s", errorResp.Error)
	return createJSONResponse(errorResp)
}

// checkResponse validates HTTP response status codes
func checkResponse(response *http.Response, operation string) *mcp_golang.ToolResponse {
	if response == nil {
		errorMsg := fmt.Sprintf("No response received for %s", operation)
		logger.Printf("Error: %s", errorMsg)
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent(errorMsg),
		)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		errorMsg := fmt.Sprintf("Failed to %s. HTTP Status: %d", operation, response.StatusCode)
		logger.Printf("Error: %s", errorMsg)
		return mcp_golang.NewToolResponse(
			mcp_golang.NewTextContent(errorMsg),
		)
	}

	return nil
}

func initLogger() {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		os.Exit(1)
	}
	logger = log.New(logFile, "[taikun-mcp] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Logger initialized")
}

func createTaikunClient() *taikungoclient.Client {
	apiHost := os.Getenv("TAIKUN_API_HOST")
	if apiHost == "" {
		apiHost = "api.taikun.cloud"
	}
	logger.Printf("Using API host: %s", apiHost)

	authMode := os.Getenv("TAIKUN_AUTH_MODE")

	// Check for access key/secret key authentication
	accessKey := os.Getenv("TAIKUN_ACCESS_KEY")
	secretKey := os.Getenv("TAIKUN_SECRET_KEY")

	if accessKey != "" && secretKey != "" {
		if authMode == "" {
			authMode = "token"
		}
		logger.Printf("Using access key/secret key authentication with mode: %s", authMode)
		return taikungoclient.NewClientFromCredentials("", "", accessKey, secretKey, authMode, apiHost)
	}

	// Check for email/password (standard taikungoclient env vars)
	email := os.Getenv("TAIKUN_EMAIL")
	password := os.Getenv("TAIKUN_PASSWORD")

	if email != "" && password != "" {
		logger.Printf("Using email/password authentication for user: %s", email)
		return taikungoclient.NewClientFromCredentials(email, password, "", "", "", apiHost)
	}

	logger.Fatal("No valid authentication credentials found. Please set either:\n" +
		"  - TAIKUN_ACCESS_KEY + TAIKUN_SECRET_KEY + TAIKUN_AUTH_MODE (optional, defaults to 'token')\n" +
		"  - TAIKUN_EMAIL + TAIKUN_PASSWORD")
	return nil
}

func main() {
	// Handle version command
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("Taikun MCP Server %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built: %s\n", date)
		fmt.Printf("  by: %s\n", builtBy)
		return
	}

	initLogger()
	logger.Printf("Starting Taikun MCP server v%s", version)

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	logger.Println("MCP server created")

	// Initialize the Taikun client once
	taikunClient = createTaikunClient()
	logger.Println("Taikun client initialized")

	logger.Println("Starting tool registration...")

	// --- MCP Tool Registrations ---

	err := server.RegisterTool("create-virtual-cluster", "Create a new virtual cluster (a project in Taikun) with optional wait for completion", func(args CreateVirtualClusterArgs) (*mcp_golang.ToolResponse, error) {
		return createVirtualCluster(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register create-virtual-cluster tool: %v", err)
	}
	logger.Println("Registered create-virtual-cluster tool")

	err = server.RegisterTool("delete-virtual-cluster", "Delete a virtual cluster (a project in Taikun)", func(args DeleteVirtualClusterArgs) (*mcp_golang.ToolResponse, error) {
		return deleteVirtualCluster(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register delete-virtual-cluster tool: %v", err)
	}
	logger.Println("Registered delete-virtual-cluster tool")

	err = server.RegisterTool("list-virtual-clusters", "List virtual clusters in a parent project (projects in Taikun)", func(args ListVirtualClustersArgs) (*mcp_golang.ToolResponse, error) {
		return listVirtualClusters(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-virtual-clusters tool: %v", err)
	}
	logger.Println("Registered list-virtual-clusters tool")

	err = server.RegisterTool("create-catalog", "Create a new catalog", func(args CreateCatalogArgs) (*mcp_golang.ToolResponse, error) {
		return createCatalog(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register create-catalog tool: %v", err)
	}
	logger.Println("Registered create-catalog tool")

	err = server.RegisterTool("list-catalogs", "List catalogs with optional filtering", func(args ListCatalogsArgs) (*mcp_golang.ToolResponse, error) {
		return listCatalogs(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-catalogs tool: %v", err)
	}
	logger.Println("Registered list-catalogs tool")

	err = server.RegisterTool("update-catalog", "Update catalog name and description", func(args UpdateCatalogArgs) (*mcp_golang.ToolResponse, error) {
		return updateCatalog(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register update-catalog tool: %v", err)
	}
	logger.Println("Registered update-catalog tool")

	err = server.RegisterTool("delete-catalog", "Delete a catalog", func(args DeleteCatalogArgs) (*mcp_golang.ToolResponse, error) {
		return deleteCatalog(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register delete-catalog tool: %v", err)
	}
	logger.Println("Registered delete-catalog tool")

	err = server.RegisterTool("bind-projects-to-catalog", "Bind projects to a catalog", func(args BindProjectsToCatalogArgs) (*mcp_golang.ToolResponse, error) {
		return bindProjectsToCatalog(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register bind-projects-to-catalog tool: %v", err)
	}
	logger.Println("Registered bind-projects-to-catalog tool")

	err = server.RegisterTool("unbind-projects-from-catalog", "Unbind projects from a catalog", func(args UnbindProjectsFromCatalogArgs) (*mcp_golang.ToolResponse, error) {
		return unbindProjectsFromCatalog(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register unbind-projects-from-catalog tool: %v", err)
	}
	logger.Println("Registered unbind-projects-from-catalog tool")

	err = server.RegisterTool("add-app-to-catalog", "Add an application to a catalog", func(args AddAppToCatalogArgs) (*mcp_golang.ToolResponse, error) {
		return addAppToCatalog(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register add-app-to-catalog tool: %v", err)
	}
	logger.Println("Registered add-app-to-catalog tool")

	err = server.RegisterTool("remove-app-from-catalog", "Remove an application from a catalog", func(args RemoveAppFromCatalogArgs) (*mcp_golang.ToolResponse, error) {
		return removeAppFromCatalog(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register remove-app-from-catalog tool: %v", err)
	}
	logger.Println("Registered remove-app-from-catalog tool")

	err = server.RegisterTool("list-catalog-apps", "List applications in a specific catalog or all catalogs", func(args ListCatalogAppsArgs) (*mcp_golang.ToolResponse, error) {
		return listCatalogApps(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-catalog-apps tool: %v", err)
	}
	logger.Println("Registered list-catalog-apps tool")

	err = server.RegisterTool("list-repositories", "List available repositories by discovering them from existing catalog applications", func(args ListRepositoriesArgs) (*mcp_golang.ToolResponse, error) {
		return listRepositories(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-repositories tool: %v", err)
	}
	logger.Println("Registered list-repositories tool")

	err = server.RegisterTool("list-available-packages", "List all available packages from the package repository", func(args ListAvailablePackagesArgs) (*mcp_golang.ToolResponse, error) {
		return listAvailablePackages(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-available-packages tool: %v", err)
	}
	logger.Println("Registered list-available-packages tool")

	err = server.RegisterTool("install-app", "Install a new application instance", func(args InstallAppArgs) (*mcp_golang.ToolResponse, error) {
		return installApp(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register install-app tool: %v", err)
	}
	logger.Println("Registered install-app tool")

	err = server.RegisterTool("list-apps", "List application instances in a project", func(args ListAppsArgs) (*mcp_golang.ToolResponse, error) {
		return listApps(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-apps tool: %v", err)
	}
	logger.Println("Registered list-apps tool")

	err = server.RegisterTool("get-app", "Get detailed application instance information", func(args GetAppArgs) (*mcp_golang.ToolResponse, error) {
		return getApp(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register get-app tool: %v", err)
	}
	logger.Println("Registered get-app tool")

	err = server.RegisterTool("update-sync-app", "Update application values and sync", func(args UpdateSyncAppArgs) (*mcp_golang.ToolResponse, error) {
		return updateSyncApp(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register update-sync-app tool: %v", err)
	}
	logger.Println("Registered update-sync-app tool")

	err = server.RegisterTool("uninstall-app", "Uninstall an application instance", func(args UninstallAppArgs) (*mcp_golang.ToolResponse, error) {
		return uninstallApp(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register uninstall-app tool: %v", err)
	}
	logger.Println("Registered uninstall-app tool")

	err = server.RegisterTool("list-projects", "List Kubernetes projects with optional virtual cluster filtering", func(args ListProjectsArgs) (*mcp_golang.ToolResponse, error) {
		return listProjects(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-projects tool: %v", err)
	}
	logger.Println("Registered list-projects tool")

	err = server.RegisterTool("create-project", "Create a new Kubernetes project in Taikun", func(args CreateProjectArgs) (*mcp_golang.ToolResponse, error) {
		return createProject(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register create-project tool: %v", err)
	}
	logger.Println("Registered create-project tool")

	err = server.RegisterTool("delete-project", "Delete a project in Taikun", func(args DeleteProjectArgs) (*mcp_golang.ToolResponse, error) {
		return deleteProject(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register delete-project tool: %v", err)
	}
	logger.Println("Registered delete-project tool")

	err = server.RegisterTool("deploy-kubernetes-resources", "Deploy Kubernetes resources via YAML in a project", func(args DeployKubernetesResourcesArgs) (*mcp_golang.ToolResponse, error) {
		return deployKubernetesResources(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register deploy-kubernetes-resources tool: %v", err)
	}
	logger.Println("Registered deploy-kubernetes-resources tool")

	err = server.RegisterTool("create-kubeconfig", "Create a new kubeconfig for a project", func(args CreateKubeConfigArgs) (*mcp_golang.ToolResponse, error) {
		return createKubeConfig(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register create-kubeconfig tool: %v", err)
	}
	logger.Println("Registered create-kubeconfig tool")

	err = server.RegisterTool("get-kubeconfig", "Retrieve the kubeconfig content for a project", func(args GetKubeConfigArgs) (*mcp_golang.ToolResponse, error) {
		return getKubeConfig(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register get-kubeconfig tool: %v", err)
	}
	logger.Println("Registered get-kubeconfig tool")

	err = server.RegisterTool("list-kubeconfig-roles", "List available roles for kubeconfigs", func(args ListKubeConfigRolesArgs) (*mcp_golang.ToolResponse, error) {
		return listKubeConfigRoles(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-kubeconfig-roles tool: %v", err)
	}
	logger.Println("Registered list-kubeconfig-roles tool")

	err = server.RegisterTool("list-kubernetes-resources", "List specialized Kubernetes resources in a project", func(args ListKubernetesResourcesArgs) (*mcp_golang.ToolResponse, error) {
		return listKubernetesResources(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-kubernetes-resources tool: %v", err)
	}
	logger.Println("Registered list-kubernetes-resources tool")

	err = server.RegisterTool("describe-kubernetes-resource", "Describe a specialized Kubernetes resource in a project", func(args DescribeKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
		return describeKubernetesResource(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register describe-kubernetes-resource tool: %v", err)
	}
	logger.Println("Registered describe-kubernetes-resource tool")

	err = server.RegisterTool("delete-kubernetes-resource", "Delete a Kubernetes resource", func(args DeleteKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
		return deleteKubernetesResource(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register delete-kubernetes-resource tool: %v", err)
	}
	logger.Println("Registered delete-kubernetes-resource tool")

	err = server.RegisterTool("list-cloud-credentials", "List cloud credentials", func(args ListCloudCredentialsArgs) (*mcp_golang.ToolResponse, error) {
		return listCloudCredentials(taikunClient, args)
	})
	if err != nil {
		logger.Fatalf("Failed to register list-cloud-credentials tool: %v", err)
	}
	logger.Println("Registered list-cloud-credentials tool")

	logger.Println("All tools registered successfully. Starting MCP server...")
	logger.Println("About to call server.Serve()...")
	err = server.Serve()
	logger.Printf("server.Serve() returned with error: %v", err)
	if err != nil {
		logger.Fatalf("Server error: %v", err)
	}

	done := make(chan struct{})
	<-done
}
