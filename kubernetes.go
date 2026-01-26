package main

import (
	"context"
	"fmt"
	"net/http"

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

type ListKubernetesResourcesArgs struct {
	ProjectID  int32  `json:"projectId" jsonschema:"required,description=The project ID to list resources from"`
	Kind       string `json:"kind" jsonschema:"required,description=The kind of Kubernetes resource (e.g., Pods, Deployments, Services, Namespaces, ConfigMaps, Secrets, Ingress, CronJobs, DaemonSets, Jobs, Nodes, Pvcs, StorageClasses, Sts)"`
	Limit      int32  `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (optional)"`
	Offset     int32  `json:"offset,omitempty" jsonschema:"description=Number of results to skip (optional)"`
	SearchTerm string `json:"searchTerm,omitempty" jsonschema:"description=Search term to filter results (optional)"`
}

type DescribeKubernetesResourceArgs struct {
	ProjectID int32  `json:"projectId" jsonschema:"required,description=The project ID of the resource"`
	Name      string `json:"name" jsonschema:"required,description=The name of the resource"`
	Kind      string `json:"kind" jsonschema:"required,description=The kind of the resource (e.g., Pod, Deployment, Service, etc.)"`
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

func listKubernetesResources(client *taikungoclient.Client, args ListKubernetesResourcesArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	var result interface{}
	var httpResponse *http.Response
	var err error

	switch args.Kind {
	case "Pods":
		cmd := taikuncore.NewPodsSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchPods(ctx).PodsSearchCommand(*cmd).Execute()
	case "Deployments":
		cmd := taikuncore.NewDeploymentSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchDeployments(ctx).DeploymentSearchCommand(*cmd).Execute()
	case "Services":
		cmd := taikuncore.NewServiceSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchServices(ctx).ServiceSearchCommand(*cmd).Execute()
	case "Namespaces":
		cmd := taikuncore.NewNamespaceSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchNamespaces(ctx).NamespaceSearchCommand(*cmd).Execute()
	case "ConfigMaps":
		cmd := taikuncore.NewConfigMapSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchConfigMaps(ctx).ConfigMapSearchCommand(*cmd).Execute()
	case "Secrets":
		cmd := taikuncore.NewSecretSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchSecrets(ctx).SecretSearchCommand(*cmd).Execute()
	case "Ingress":
		cmd := taikuncore.NewIngressSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchIngress(ctx).IngressSearchCommand(*cmd).Execute()
	case "CronJobs":
		cmd := taikuncore.NewCronjobsSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchCronJobs(ctx).CronjobsSearchCommand(*cmd).Execute()
	case "DaemonSets":
		cmd := taikuncore.NewDaemonSetSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchDaemonSets(ctx).DaemonSetSearchCommand(*cmd).Execute()
	case "Jobs":
		cmd := taikuncore.NewJobsSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchJobs(ctx).JobsSearchCommand(*cmd).Execute()
	case "Nodes":
		cmd := taikuncore.NewNodesSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchNodes(ctx).NodesSearchCommand(*cmd).Execute()
	case "Pvcs":
		cmd := taikuncore.NewPvcSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchPvcs(ctx).PvcSearchCommand(*cmd).Execute()
	case "StorageClasses":
		cmd := taikuncore.NewStorageClassesSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchStorageClasses(ctx).StorageClassesSearchCommand(*cmd).Execute()
	case "Sts":
		cmd := taikuncore.NewStsSearchCommand()
		if args.Limit > 0 {
			cmd.SetLimit(args.Limit)
		}
		if args.Offset > 0 {
			cmd.SetOffset(args.Offset)
		}
		if args.SearchTerm != "" {
			cmd.SetSearchTerm(args.SearchTerm)
		}
		result, httpResponse, err = client.Client.SearchAPI.SearchSts(ctx).StsSearchCommand(*cmd).Execute()
	default:
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unsupported resource kind: %s", args.Kind))), nil
	}

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, fmt.Sprintf("list %s", args.Kind)); errorResp != nil {
		return errorResp, nil
	}

	return createJSONResponse(result), nil
}

func describeKubernetesResource(client *taikungoclient.Client, args DescribeKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	kind, err := taikuncore.NewEKubernetesResourceFromValue(args.Kind)
	if err != nil {
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Invalid resource kind: %s", args.Kind))), nil
	}

	describeCmd := taikuncore.NewDescribeKubernetesResourceCommand(args.ProjectID, args.Name, *kind)

	description, httpResponse, err := client.Client.KubernetesAPI.KubernetesDescribeResource(ctx).
		DescribeKubernetesResourceCommand(*describeCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, fmt.Sprintf("describe %s %s", args.Kind, args.Name)); errorResp != nil {
		return errorResp, nil
	}

	return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(description)), nil
}
