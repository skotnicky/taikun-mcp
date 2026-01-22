package main

import (
	"context"
	"fmt"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

type DeployKubernetesResourcesArgs struct {
	ProjectID int32  `json:"projectId" jsonschema:"required,description=The project ID to deploy the resources to"`
	YAML      string `json:"yaml" jsonschema:"required,description=The Kubernetes resources in YAML format"`
}

type CreateKubeConfigArgs struct {
	Name                   string `json:"name,omitempty" jsonschema:"description=The name of the kubeconfig (optional)"`
	ProjectID              int32  `json:"projectId" jsonschema:"required,description=The project ID to create the kubeconfig for"`
	IsAccessibleForAll     bool   `json:"isAccessibleForAll,omitempty" jsonschema:"description=Whether the kubeconfig is accessible for all (default: false)"`
	IsAccessibleForManager bool   `json:"isAccessibleForManager,omitempty" jsonschema:"description=Whether the kubeconfig is accessible for managers (default: false)"`
	KubeConfigRoleId       int32  `json:"kubeConfigRoleId,omitempty" jsonschema:"description=The role ID for the kubeconfig (optional)"`
	UserId                 string `json:"userId,omitempty" jsonschema:"description=The user ID for the kubeconfig (optional)"`
	Namespace              string `json:"namespace,omitempty" jsonschema:"description=The namespace for the kubeconfig (optional)"`
	TTL                    int32  `json:"ttl,omitempty" jsonschema:"description=The TTL for the kubeconfig in minutes (optional)"`
}

type GetKubeConfigArgs struct {
	ProjectID int32 `json:"projectId" jsonschema:"required,description=The project ID to get the kubeconfig for"`
}

type ListKubeConfigRolesArgs struct{}

func deployKubernetesResources(client *taikungoclient.Client, args DeployKubernetesResourcesArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	createCmd := taikuncore.NewCreateKubernetesResourceCommand(args.ProjectID, *taikuncore.NewNullableString(&args.YAML))

	httpResponse, err := client.Client.KubernetesAPI.KubernetesCreateResource(ctx).
		CreateKubernetesResourceCommand(*createCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "deploy kubernetes resources"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: "Kubernetes resources deployed successfully",
		Success: true,
	}

	return createJSONResponse(successResp), nil
}

func createKubeConfig(client *taikungoclient.Client, args CreateKubeConfigArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	createCmd := taikuncore.NewCreateKubeConfigCommand()
	createCmd.SetProjectId(args.ProjectID)

	if args.Name != "" {
		createCmd.SetName(args.Name)
	}
	createCmd.SetIsAccessibleForAll(args.IsAccessibleForAll)
	createCmd.SetIsAccessibleForManager(args.IsAccessibleForManager)

	if args.KubeConfigRoleId != 0 {
		createCmd.SetKubeConfigRoleId(args.KubeConfigRoleId)
	}
	if args.UserId != "" {
		createCmd.SetUserId(args.UserId)
	}
	if args.Namespace != "" {
		createCmd.SetNamespace(args.Namespace)
	}
	if args.TTL != 0 {
		createCmd.SetTtl(args.TTL)
	}

	_, httpResponse, err := client.Client.KubeConfigAPI.KubeconfigCreate(ctx).
		CreateKubeConfigCommand(*createCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "create kubeconfig"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Kubeconfig created successfully for project %d", args.ProjectID),
		Success: true,
	}

	return createJSONResponse(successResp), nil
}

func getKubeConfig(client *taikungoclient.Client, args GetKubeConfigArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	kubeconfig, httpResponse, err := client.Client.KubernetesAPI.KubernetesKubeConfig(ctx, args.ProjectID).Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "get kubeconfig"); errorResp != nil {
		return errorResp, nil
	}

	if kubeconfig == nil {
		errorResp := ErrorResponse{
			Error: fmt.Sprintf("Kubeconfig for project %d not found", args.ProjectID),
		}
		return createJSONResponse(errorResp), nil
	}

	type KubeConfigResponseData struct {
		KubeConfig string `json:"kubeConfig"`
		Success    bool   `json:"success"`
	}

	resp := KubeConfigResponseData{
		KubeConfig: kubeconfig.GetData(),
		Success:    true,
	}

	return createJSONResponse(resp), nil
}

func listKubeConfigRoles(client *taikungoclient.Client, _ ListKubeConfigRolesArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	roles, httpResponse, err := client.Client.KubeConfigRoleAPI.KubeconfigroleList(ctx).Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "list kubeconfig roles"); errorResp != nil {
		return errorResp, nil
	}

	type RoleSummary struct {
		ID   int32  `json:"id"`
		Name string `json:"name"`
	}

	var roleSummaries []RoleSummary
	if roles != nil {
		for _, role := range roles.Data {
			roleSummaries = append(roleSummaries, RoleSummary{
				ID:   role.GetId(),
				Name: role.GetName(),
			})
		}
	}

	return createJSONResponse(roleSummaries), nil
}
