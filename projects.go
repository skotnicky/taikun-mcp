package main

import (
	"context"
	"fmt"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

type ListProjectsArgs struct {
	Limit               int32  `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (optional)"`
	Offset              int32  `json:"offset,omitempty" jsonschema:"description=Number of results to skip (optional)"`
	Search              string `json:"search,omitempty" jsonschema:"description=Search term to filter results (optional)"`
	HealthyOnly         bool   `json:"healthyOnly,omitempty" jsonschema:"description=Return only healthy projects (default: false)"`
	VirtualClustersOnly bool   `json:"virtualClustersOnly,omitempty" jsonschema:"description=Return only virtual cluster projects (default: false)"`
}

func listProjects(client *taikungoclient.Client, args ListProjectsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	req := client.Client.ProjectsAPI.ProjectsList(ctx)

	if args.Limit > 0 {
		req = req.Limit(args.Limit)
	}
	if args.Offset > 0 {
		req = req.Offset(args.Offset)
	}
	if args.Search != "" {
		req = req.Search(args.Search)
	}
	if args.HealthyOnly {
		req = req.Healthy(true)
	}

	projectList, httpResponse, err := req.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "list projects"); errorResp != nil {
		return errorResp, nil
	}

	if projectList == nil || len(projectList.Data) == 0 {
		listResp := ProjectListResponse{
			Projects: []ProjectSummary{},
			Total:    0,
			Message:  "No projects found",
		}
		return createJSONResponse(listResp), nil
	}

	var filteredProjects []taikuncore.ProjectListDetailDto

	for _, project := range projectList.Data {
		include := true

		// Always filter to Kubernetes projects only
		if !project.GetIsKubernetes() {
			include = false
		}

		// Filter by virtual clusters if requested
		if args.VirtualClustersOnly && !project.GetIsVirtualCluster() {
			include = false
		}

		if include {
			filteredProjects = append(filteredProjects, project)
		}
	}

	// Prepare the response data
	var projects []ProjectSummary
	for _, project := range filteredProjects {
		projectSummary := ProjectSummary{
			ID:                     project.GetId(),
			Name:                   project.GetName(),
			Status:                 string(project.GetStatus()),
			Health:                 string(project.GetHealth()),
			Type:                   getProjectType(project),
			Cloud:                  string(project.GetCloudType()),
			Organization:           project.GetOrganizationName(),
			IsLocked:               project.GetIsLocked(),
			IsVirtualCluster:       project.GetIsVirtualCluster(),
			CreatedAt:              project.GetCreatedAt(),
			ServersCount:           project.GetTotalServersCount(),
			StandaloneVMsCount:     project.GetTotalStandaloneVmsCount(),
			HourlyCost:             project.GetTotalHourlyCost(),
			MonitoringEnabled:      project.GetIsMonitoringEnabled(),
			BackupEnabled:          project.GetIsBackupEnabled(),
			AlertsCount:            project.GetAlertsCount(),
			ReadyForVirtualCluster: isProjectReadyForVirtualCluster(project),
		}

		if project.GetIsVirtualCluster() && project.GetParentProjectId() > 0 {
			projectSummary.ParentProjectID = project.GetParentProjectId()
		}

		if !projectSummary.ReadyForVirtualCluster {
			projectSummary.VirtualClusterReason = getVirtualClusterReadinessReason(project)
		}

		projects = append(projects, projectSummary)
	}

	// Create response
	var filterType string
	var message string
	if args.VirtualClustersOnly {
		filterType = "virtual-clusters"
		message = fmt.Sprintf("Found %d virtual cluster projects", len(projects))
	} else {
		filterType = "kubernetes"
		message = fmt.Sprintf("Found %d Kubernetes projects", len(projects))
	}

	if len(projects) == 0 {
		message = "No projects found matching the specified criteria"
	}

	response := ProjectListResponse{
		Projects:   projects,
		Total:      len(projects),
		FilterType: filterType,
		Message:    message,
	}

	return createJSONResponse(response), nil
}

func getProjectType(project taikuncore.ProjectListDetailDto) string {
	if project.GetIsKubernetes() {
		return "Kubernetes"
	}
	return "Standalone"
}

func isProjectReadyForVirtualCluster(project taikuncore.ProjectListDetailDto) bool {
	return project.GetIsKubernetes() &&
		project.GetStatus() == taikuncore.PROJECTSTATUS_READY &&
		project.GetHealth() == taikuncore.PROJECTHEALTH_HEALTHY &&
		!project.GetIsLocked() &&
		!project.GetIsVirtualCluster()
}

func getVirtualClusterReadinessReason(project taikuncore.ProjectListDetailDto) string {
	if !project.GetIsKubernetes() {
		return "Not a Kubernetes project"
	}
	if project.GetStatus() != taikuncore.PROJECTSTATUS_READY {
		return fmt.Sprintf("Status is %s (must be Ready)", project.GetStatus())
	}
	if project.GetHealth() != taikuncore.PROJECTHEALTH_HEALTHY {
		return fmt.Sprintf("Health is %s (must be Healthy)", project.GetHealth())
	}
	if project.GetIsLocked() {
		return "Project is locked (read-only)"
	}
	if project.GetIsVirtualCluster() {
		return "Virtual clusters cannot host other virtual clusters"
	}
	return "Unknown reason"
}

func createProject(client *taikungoclient.Client, args CreateProjectArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	// Create the project command
	createCmd := taikuncore.NewCreateProjectCommand()
	createCmd.SetName(args.Name)
	createCmd.SetCloudCredentialId(args.CloudCredentialID)
	createCmd.SetIsKubernetes(true) // Always create Kubernetes projects

	// Set optional parameters
	if args.KubernetesProfileID != 0 {
		createCmd.SetKubernetesProfileId(args.KubernetesProfileID)
	}
	if args.AlertingProfileID != 0 {
		createCmd.SetAlertingProfileId(args.AlertingProfileID)
	}
	// Note: BackupCredentialID might not be available in this API
	// if args.BackupCredentialID != 0 {
	//     createCmd.SetBackupCredentialId(args.BackupCredentialID)
	// }
	if args.KubernetesVersion != "" {
		createCmd.SetKubernetesVersion(args.KubernetesVersion)
	}
	
	// Set monitoring
	createCmd.SetIsMonitoringEnabled(args.Monitoring)

	// Execute the API call
	projectResponse, httpResponse, err := client.Client.ProjectsAPI.ProjectsCreate(ctx).
		CreateProjectCommand(*createCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "create project"); errorResp != nil {
		return errorResp, nil
	}

	// Prepare success response with project details
	type ProjectCreationResponse struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		CloudCredentialID int32  `json:"cloudCredentialId"`
		IsKubernetes      bool   `json:"isKubernetes"`
		MonitoringEnabled bool   `json:"monitoringEnabled"`
		Message           string `json:"message"`
		Success           bool   `json:"success"`
	}

	var projectID string
	if projectResponse != nil && projectResponse.Id.IsSet() && projectResponse.Id.Get() != nil {
		projectID = *projectResponse.Id.Get()
	}

	response := ProjectCreationResponse{
		ID:                projectID,
		Name:              args.Name,
		CloudCredentialID: args.CloudCredentialID,
		IsKubernetes:      true,
		MonitoringEnabled: args.Monitoring,
		Message:           fmt.Sprintf("Project '%s' created successfully with ID %s", args.Name, projectID),
		Success:           true,
	}

	return createJSONResponse(response), nil
}

func deleteProject(client *taikungoclient.Client, args DeleteProjectArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	// Create the delete command
	deleteCmd := taikuncore.NewDeleteProjectCommand()
	deleteCmd.SetProjectId(args.ProjectID)

	// Execute the API call to delete the project
	httpResponse, err := client.Client.ProjectsAPI.ProjectsDelete(ctx).
		DeleteProjectCommand(*deleteCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "delete project"); errorResp != nil {
		return errorResp, nil
	}

	// Prepare success response
	successResp := SuccessResponse{
		Message: fmt.Sprintf("Project ID %d deleted successfully", args.ProjectID),
		Success: true,
	}

	return createJSONResponse(successResp), nil
}