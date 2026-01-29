package main

import (
	"context"
	"fmt"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

func bindFlavorsToProject(client *taikungoclient.Client, args BindFlavorsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	command := taikuncore.NewBindFlavorToProjectCommand()
	command.SetProjectId(args.ProjectId)
	command.SetFlavors(args.Flavors)

	request := client.Client.FlavorsAPI.FlavorsBindToProject(ctx).
		BindFlavorToProjectCommand(*command)

	httpResponse, err := request.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "bind flavors to project"); errorResp != nil {
		return errorResp, nil
	}

	return createJSONResponse(map[string]string{
		"message": fmt.Sprintf("Successfully bound %d flavors to project %d", len(args.Flavors), args.ProjectId),
	}), nil
}

func addServerToProject(client *taikungoclient.Client, args AddServerArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	serverDto := taikuncore.NewServerForCreateDto()
	serverDto.SetName(args.Name)

	role, err := taikuncore.NewCloudRoleFromValue(args.Role)
	if err != nil {
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Invalid role: %v", err))), nil
	}
	serverDto.SetRole(*role)

	serverDto.SetProjectId(args.ProjectId)
	serverDto.SetFlavor(args.Flavor)

	if args.DiskSize > 0 {
		serverDto.SetDiskSize(args.DiskSize * 1024 * 1024 * 1024)
	}

	count := args.Count
	if count <= 0 {
		count = 1
	}
	serverDto.SetCount(count)

	request := client.Client.ServersAPI.ServersCreate(ctx).
		ServerForCreateDto(*serverDto)

	_, httpResponse, err := request.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "add server to project"); errorResp != nil {
		return errorResp, nil
	}

	return createJSONResponse(map[string]string{
		"message": fmt.Sprintf("Successfully added %d server(s) of type %s with flavor %s to project %d", count, args.Role, args.Flavor, args.ProjectId),
	}), nil
}

func commitProject(client *taikungoclient.Client, args CommitProjectArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	command := taikuncore.NewProjectDeploymentCommitCommand()
	command.SetProjectId(args.ProjectId)

	request := client.Client.ProjectDeploymentAPI.ProjectDeploymentCommit(ctx).
		ProjectDeploymentCommitCommand(*command)

	httpResponse, err := request.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "commit project"); errorResp != nil {
		return errorResp, nil
	}

	return createJSONResponse(map[string]string{
		"message": fmt.Sprintf("Successfully committed project %d deployment. Note: Deploying a full Kubernetes cluster typically takes 10 to 30 minutes.", args.ProjectId),
	}), nil
}

func getProjectDetails(client *taikungoclient.Client, args GetProjectDetailsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	// Using ProjectsList because it contains status and health info
	request := client.Client.ProjectsAPI.ProjectsList(ctx).
		Id(args.ProjectId)

	result, httpResponse, err := request.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "get project details"); errorResp != nil {
		return errorResp, nil
	}

	if len(result.Data) == 0 {
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Project with ID %d not found", args.ProjectId))), nil
	}

	project := result.Data[0]
	response := ProjectStatusResponse{
		ID:        project.GetId(),
		Name:      project.GetName(),
		Status:    string(project.GetStatus()),
		Health:    string(project.GetHealth()),
		CloudType: string(project.GetCloudType()),
	}

	return createJSONResponse(response), nil
}

func listFlavors(client *taikungoclient.Client, args ListFlavorsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	request := client.Client.CloudCredentialAPI.CloudcredentialsAllFlavors(ctx, args.CloudCredentialId)
	if args.Limit > 0 {
		request = request.Limit(args.Limit)
	}
	if args.Offset > 0 {
		request = request.Offset(args.Offset)
	}
	if args.Search != "" {
		request = request.Search(args.Search)
	}

	result, httpResponse, err := request.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "list flavors"); errorResp != nil {
		return errorResp, nil
	}

	var flavors []FlavorSummary
	if result != nil && result.Data != nil {
		for _, f := range result.Data {
			flavors = append(flavors, FlavorSummary{
				Name: f.GetName(),
				CPU:  f.GetCpu(),
				RAM:  f.GetRam(),
			})
		}
	}

	response := FlavorListResponse{
		Flavors: flavors,
		Total:   int32(len(flavors)),
		Message: fmt.Sprintf("Found %d flavors", len(flavors)),
	}

	return createJSONResponse(response), nil
}

func listServers(client *taikungoclient.Client, args ListServersArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	request := client.Client.ServersAPI.ServersDetails(ctx, args.ProjectId)

	result, httpResponse, err := request.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "list servers"); errorResp != nil {
		return errorResp, nil
	}

	var servers []ServerSummary
	if result != nil {
		for _, s := range result.Data {
			servers = append(servers, ServerSummary{
				ID:        s.GetId(),
				Name:      s.GetName(),
				Role:      string(s.GetRole()),
				Status:    s.GetStatus(),
				IPAddress: s.GetIpAddress(),
				Flavor:    s.GetFlavor(),
			})
		}
	}

	response := ServerListResponse{
		Servers: servers,
		Total:   int32(len(servers)),
		Message: fmt.Sprintf("Found %d servers", len(servers)),
	}

	return createJSONResponse(response), nil
}

func deleteServersFromProject(client *taikungoclient.Client, args DeleteServersArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	command := taikuncore.NewProjectDeploymentDeleteServersCommand()
	command.SetProjectId(args.ProjectId)
	command.SetServerIds(args.ServerIds)
	command.SetForceDeleteVClusters(args.ForceDeleteVClusters)
	command.SetDeleteAutoscalingServers(args.DeleteAutoscalingServers)

	request := client.Client.ProjectDeploymentAPI.ProjectDeploymentDelete(ctx).
		ProjectDeploymentDeleteServersCommand(*command)

	httpResponse, err := request.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "delete servers from project"); errorResp != nil {
		return errorResp, nil
	}

	return createJSONResponse(map[string]interface{}{
		"message":   fmt.Sprintf("Successfully deleted %d server(s) from project %d", len(args.ServerIds), args.ProjectId),
		"serverIds": args.ServerIds,
		"success":   true,
	}), nil
}
