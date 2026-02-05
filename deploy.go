package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

var (
	projectServerAddLocks   = map[int32]*sync.Mutex{}
	projectServerAddLocksMu sync.Mutex
)

func getProjectServerAddLock(projectId int32) *sync.Mutex {
	projectServerAddLocksMu.Lock()
	defer projectServerAddLocksMu.Unlock()

	lock, ok := projectServerAddLocks[projectId]
	if !ok {
		lock = &sync.Mutex{}
		projectServerAddLocks[projectId] = lock
	}
	return lock
}

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
	lock := getProjectServerAddLock(args.ProjectId)
	lock.Lock()
	defer lock.Unlock()

	ctx := context.Background()

	serverDto := taikuncore.NewServerForCreateDto()
	serverDto.SetName(args.Name)

	role, err := taikuncore.NewCloudRoleFromValue(args.Role)
	if err != nil {
		return createJSONResponse(ErrorResponse{
			Error: fmt.Sprintf("Invalid role: %v", err),
		}), nil
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

	type AddServerResponse struct {
		Message  string          `json:"message"`
		Success  bool            `json:"success"`
		Verified bool            `json:"verified"`
		Expected int32           `json:"expected"`
		Found    int32           `json:"found"`
		Servers  []ServerSummary `json:"servers,omitempty"`
	}

	verifyTimeout := args.VerifyTimeoutSeconds
	if verifyTimeout <= 0 {
		verifyTimeout = 300
	}
	verifyDeadline := time.Now().Add(time.Duration(verifyTimeout) * time.Second)
	var matched []ServerSummary
	for {
		serversResp, listHTTPResponse, listErr := client.Client.ServersAPI.ServersDetails(ctx, args.ProjectId).Execute()
		if listErr != nil {
			return createError(listHTTPResponse, listErr), nil
		}
		if listHTTPResponse == nil {
			return createJSONResponse(ErrorResponse{
				Error: "Failed to verify server creation: no response received",
			}), nil
		}
		if listHTTPResponse.StatusCode < 200 || listHTTPResponse.StatusCode >= 300 {
			return createError(listHTTPResponse, fmt.Errorf("failed to verify server creation")), nil
		}

		matched = matched[:0]
		if serversResp != nil {
			for _, server := range serversResp.Data {
				serverName := server.GetName()
				if args.Name != "" {
					if count == 1 && serverName != args.Name {
						continue
					}
					if count > 1 && !strings.HasPrefix(serverName, args.Name) {
						continue
					}
				}
				if args.Role != "" && string(server.GetRole()) != args.Role {
					continue
				}
				if args.Flavor != "" && server.GetFlavor() != args.Flavor {
					continue
				}

				matched = append(matched, ServerSummary{
					ID:        server.GetId(),
					Name:      serverName,
					Role:      string(server.GetRole()),
					Status:    server.GetStatus(),
					IPAddress: server.GetIpAddress(),
					Flavor:    server.GetFlavor(),
				})
			}
		}

		if int32(len(matched)) >= count {
			return createJSONResponse(AddServerResponse{
				Message:  fmt.Sprintf("Successfully added %d server(s) of type %s with flavor %s to project %d", count, args.Role, args.Flavor, args.ProjectId),
				Success:  true,
				Verified: true,
				Expected: count,
				Found:    int32(len(matched)),
				Servers:  matched,
			}), nil
		}

		if time.Now().After(verifyDeadline) {
			return createJSONResponse(AddServerResponse{
				Message:  fmt.Sprintf("Server creation request accepted but not verified within timeout (expected %d)", count),
				Success:  false,
				Verified: false,
				Expected: count,
				Found:    int32(len(matched)),
				Servers:  matched,
			}), nil
		}

		time.Sleep(5 * time.Second)
	}
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
