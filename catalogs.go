package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

type CreateCatalogArgs struct {
	Name        string `json:"name" jsonschema:"required,description=The name of the catalog"`
	Description string `json:"description" jsonschema:"required,description=The description of the catalog"`
}

type ListCatalogsArgs struct {
	Limit  int32  `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (optional)"`
	Offset int32  `json:"offset,omitempty" jsonschema:"description=Number of results to skip (optional)"`
	Search string `json:"search,omitempty" jsonschema:"description=Search term to filter results (optional)"`
	ID     int32  `json:"id,omitempty" jsonschema:"description=Get specific catalog by ID (optional)"`
}

type UpdateCatalogArgs struct {
	CatalogID   int32  `json:"catalogId" jsonschema:"required,description=The ID of the catalog to update"`
	Name        string `json:"name" jsonschema:"required,description=The new name of the catalog"`
	Description string `json:"description" jsonschema:"required,description=The new description of the catalog"`
}

type DeleteCatalogArgs struct {
	CatalogID int32 `json:"catalogId" jsonschema:"required,description=The ID of the catalog to delete"`
}

type BindProjectsToCatalogArgs struct {
	CatalogID  int32   `json:"catalogId" jsonschema:"required,description=The ID of the catalog"`
	ProjectIDs []int32 `json:"projectIds" jsonschema:"required,description=Array of project IDs to bind to the catalog"`
}

type UnbindProjectsFromCatalogArgs struct {
	CatalogID  int32   `json:"catalogId" jsonschema:"required,description=The ID of the catalog"`
	ProjectIDs []int32 `json:"projectIds" jsonschema:"required,description=Array of project IDs to unbind from the catalog"`
}

func createCatalog(client *taikungoclient.Client, args CreateCatalogArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	createCmd := taikuncore.NewCreateCatalogCommand()
	createCmd.SetName(args.Name)
	createCmd.SetDescription(args.Description)

	response, err := client.Client.CatalogAPI.CatalogCreate(ctx).
		CreateCatalogCommand(*createCmd).
		Execute()

	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "create catalog"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Catalog '%s' created successfully", args.Name),
		Success: true,
	}
	return createJSONResponse(successResp), nil
}

func listCatalogs(client *taikungoclient.Client, args ListCatalogsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	req := client.Client.CatalogAPI.CatalogList(ctx)

	if args.Limit > 0 {
		req = req.Limit(args.Limit)
	}
	if args.Offset > 0 {
		req = req.Offset(args.Offset)
	}
	if args.Search != "" {
		req = req.Search(args.Search)
	}
	if args.ID != 0 {
		req = req.Id(args.ID)
	}

	catalogList, response, err := req.Execute()
	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "list catalogs"); errorResp != nil {
		return errorResp, nil
	}

	// Prepare response data
	type CatalogSummary struct {
		ID             int32  `json:"id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		IsLocked       bool   `json:"isLocked"`
		IsDefault      bool   `json:"isDefault"`
		OrganizationID int32  `json:"organizationId,omitempty"`
		BoundProjects  []struct {
			ID   int32  `json:"id"`
			Name string `json:"name"`
		} `json:"boundProjects,omitempty"`
		BoundApplications []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"boundApplications,omitempty"`
		PackageIds []string `json:"packageIds,omitempty"`
	}

	var catalogs []CatalogSummary
	if catalogList != nil && len(catalogList.Data) > 0 {
		for _, catalog := range catalogList.Data {
			catalogSummary := CatalogSummary{
				ID:          catalog.GetId(),
				Name:        catalog.GetName(),
				Description: catalog.GetDescription(),
				IsLocked:    catalog.GetIsLocked(),
				IsDefault:   catalog.GetIsDefault(),
			}

			if catalog.GetOrganizationId() != 0 {
				catalogSummary.OrganizationID = catalog.GetOrganizationId()
			}

			if len(catalog.BoundProjects) > 0 {
				for _, project := range catalog.BoundProjects {
					catalogSummary.BoundProjects = append(catalogSummary.BoundProjects, struct {
						ID   int32  `json:"id"`
						Name string `json:"name"`
					}{
						ID:   project.GetId(),
						Name: project.GetName(),
					})
				}
			}

			if len(catalog.BoundApplications) > 0 {
				for _, app := range catalog.BoundApplications {
					catalogSummary.BoundApplications = append(catalogSummary.BoundApplications, struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					}{
						ID:   app.GetPackageId(),
						Name: app.GetName(),
					})
				}
			}

			if len(catalog.PackageIds) > 0 {
				catalogSummary.PackageIds = catalog.PackageIds
			}

			catalogs = append(catalogs, catalogSummary)
		}
	}

	message := fmt.Sprintf("Found %d catalogs", len(catalogs))
	if len(catalogs) == 0 {
		message = "No catalogs found"
	}

	listResp := struct {
		Catalogs []CatalogSummary `json:"catalogs"`
		Total    int              `json:"total"`
		Message  string           `json:"message"`
	}{
		Catalogs: catalogs,
		Total:    len(catalogs),
		Message:  message,
	}

	return createJSONResponse(listResp), nil
}

func updateCatalog(client *taikungoclient.Client, args UpdateCatalogArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	editCmd := taikuncore.NewEditCatalogCommand()
	editCmd.SetId(args.CatalogID)
	editCmd.SetName(args.Name)
	editCmd.SetDescription(args.Description)

	response, err := client.Client.CatalogAPI.CatalogEdit(ctx).
		EditCatalogCommand(*editCmd).
		Execute()

	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "update catalog"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Catalog ID %d updated successfully", args.CatalogID),
		Success: true,
	}
	return createJSONResponse(successResp), nil
}

func deleteCatalog(client *taikungoclient.Client, args DeleteCatalogArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	response, err := client.Client.CatalogAPI.CatalogDelete(ctx, args.CatalogID).Execute()

	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "delete catalog"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Catalog ID %d deleted successfully", args.CatalogID),
		Success: true,
	}
	return createJSONResponse(successResp), nil
}

func bindProjectsToCatalog(client *taikungoclient.Client, args BindProjectsToCatalogArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	response, err := client.Client.CatalogAPI.CatalogAddProject(ctx, args.CatalogID).
		RequestBody(args.ProjectIDs).
		Execute()

	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "bind projects to catalog"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Successfully bound %d projects to catalog ID %d", len(args.ProjectIDs), args.CatalogID),
		Success: true,
	}
	return createJSONResponse(successResp), nil
}

func unbindProjectsFromCatalog(client *taikungoclient.Client, args UnbindProjectsFromCatalogArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	response, err := client.Client.CatalogAPI.CatalogDeleteProject(ctx, args.CatalogID).
		RequestBody(args.ProjectIDs).
		Execute()

	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "unbind projects from catalog"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Successfully unbound %d projects from catalog ID %d", len(args.ProjectIDs), args.CatalogID),
		Success: true,
	}
	return createJSONResponse(successResp), nil
}

func addAppToCatalog(client *taikungoclient.Client, args AddAppToCatalogArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	createCmd := taikuncore.NewCreateCatalogAppCommand()
	createCmd.SetCatalogId(args.CatalogID)
	createCmd.SetRepoName(args.Repository)
	createCmd.SetPackageName(args.PackageName)
	createCmd.SetParameters([]taikuncore.CatalogAppParamsDto{})

	_, response, err := client.Client.CatalogAppAPI.CatalogAppCreate(ctx).
		CreateCatalogAppCommand(*createCmd).
		Execute()

	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "add application to catalog"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Application '%s' from repository '%s' added to catalog ID %d", args.PackageName, args.Repository, args.CatalogID),
		Success: true,
	}

	return createJSONResponse(successResp), nil
}

func addAppToCatalogWithParameters(client *taikungoclient.Client, args AddAppToCatalogWithParametersArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	createCmd := taikuncore.NewCreateCatalogAppCommand()
	createCmd.SetCatalogId(args.CatalogID)
	createCmd.SetRepoName(args.Repository)
	createCmd.SetPackageName(args.PackageName)

	if len(args.Parameters) > 0 {
		params := make([]taikuncore.CatalogAppParamsDto, 0, len(args.Parameters))
		for _, param := range args.Parameters {
			p := taikuncore.NewCatalogAppParamsDto()
			p.SetKey(param.Key)
			p.SetValue(param.Value)
			params = append(params, *p)
		}
		createCmd.SetParameters(params)
	}

	_, response, err := client.Client.CatalogAppAPI.CatalogAppCreate(ctx).
		CreateCatalogAppCommand(*createCmd).
		Execute()

	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "add application to catalog with parameters"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Application '%s' from repository '%s' added to catalog ID %d with %d parameter overrides", args.PackageName, args.Repository, args.CatalogID, len(args.Parameters)),
		Success: true,
	}

	return createJSONResponse(successResp), nil
}

func removeAppFromCatalog(client *taikungoclient.Client, args RemoveAppFromCatalogArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	// Get the catalog apps to find the specific app to delete
	req := client.Client.CatalogAppAPI.CatalogAppList(ctx).CatalogId(args.CatalogID)
	if args.PackageName != "" {
		req = req.Search(args.PackageName)
	}

	catalogAppList, response, err := req.Execute()
	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "list catalog applications"); errorResp != nil {
		return errorResp, nil
	}

	// Find the specific app to delete
	var appToDelete *taikuncore.CatalogAppListDto
	if catalogAppList != nil && len(catalogAppList.Data) > 0 {
		for _, app := range catalogAppList.Data {
			// Check package name (using PackageId field)
			packageMatches := false
			if app.GetName() != "" {
				packageMatches = app.GetName() == args.PackageName
			}

			// Check repository name if provided
			repoMatches := true // Default to true if no repository filter
			if args.Repository != "" {
				repoMatches = false
				if app.RepoName.IsSet() && app.RepoName.Get() != nil {
					repoMatches = *app.RepoName.Get() == args.Repository
				}
			}

			if packageMatches && repoMatches {
				appToDelete = &app
				break
			}
		}
	}

	if appToDelete == nil {
		var errorMsg string
		if args.Repository != "" {
			errorMsg = fmt.Sprintf("Application '%s' from repository '%s' not found in catalog ID %d", args.PackageName, args.Repository, args.CatalogID)
		} else {
			errorMsg = fmt.Sprintf("Application '%s' not found in catalog ID %d", args.PackageName, args.CatalogID)
		}
		errorResp := ErrorResponse{
			Error: errorMsg,
		}
		return createJSONResponse(errorResp), nil
	}

	// Delete the application
	deleteResponse, err := client.Client.CatalogAppAPI.CatalogAppDelete(ctx, *appToDelete.CatalogAppId).Execute()
	if err != nil {
		return createError(deleteResponse, err), nil
	}

	if errorResp := checkResponse(deleteResponse, "remove application from catalog"); errorResp != nil {
		return errorResp, nil
	}

	var successMsg string
	if args.Repository != "" {
		successMsg = fmt.Sprintf("Application '%s' from repository '%s' removed from catalog ID %d", args.PackageName, args.Repository, args.CatalogID)
	} else {
		successMsg = fmt.Sprintf("Application '%s' removed from catalog ID %d", args.PackageName, args.CatalogID)
	}

	successResp := SuccessResponse{
		Message: successMsg,
		Success: true,
	}

	return createJSONResponse(successResp), nil
}

func listCatalogApps(client *taikungoclient.Client, args ListCatalogAppsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	req := client.Client.CatalogAppAPI.CatalogAppList(ctx)

	// Add catalogId filter only if provided
	if args.CatalogID != 0 {
		req = req.CatalogId(args.CatalogID)
	}

	if args.Limit > 0 {
		req = req.Limit(args.Limit)
	}
	if args.Offset > 0 {
		req = req.Offset(args.Offset)
	}
	if args.Search != "" {
		req = req.Search(args.Search)
	}

	catalogAppList, response, err := req.Execute()
	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "list catalog applications"); errorResp != nil {
		return errorResp, nil
	}

	// Prepare response data
	var applications []CatalogAppSummary
	if catalogAppList != nil && len(catalogAppList.Data) > 0 {
		for _, app := range catalogAppList.Data {
			appSummary := CatalogAppSummary{}

			// Handle nullable/pointer fields safely
			if app.CatalogAppId != nil {
				appSummary.ID = *app.CatalogAppId
			}

			if app.GetName() != "" {
				appSummary.Name = app.GetName()
			}

			if app.RepoName.IsSet() && app.RepoName.Get() != nil {
				appSummary.Repository = *app.RepoName.Get()
			}

			if app.CatalogId != nil {
				appSummary.CatalogID = *app.CatalogId
			}

			if app.CatalogName.IsSet() && app.CatalogName.Get() != nil {
				appSummary.CatalogName = *app.CatalogName.Get()
			}

			applications = append(applications, appSummary)
		}
	}

	// Create response
	var message string
	if args.CatalogID != 0 {
		message = fmt.Sprintf("Found %d applications in catalog ID %d", len(applications), args.CatalogID)
		if len(applications) == 0 {
			message = fmt.Sprintf("No applications found in catalog ID %d", args.CatalogID)
		}
	} else {
		message = fmt.Sprintf("Found %d applications across all catalogs", len(applications))
		if len(applications) == 0 {
			message = "No applications found in any catalog"
		}
	}

	listResp := CatalogAppListResponse{
		Applications: applications,
		Total:        len(applications),
		CatalogID:    args.CatalogID,
		Message:      message,
	}

	return createJSONResponse(listResp), nil
}

func getCatalogAppParameters(client *taikungoclient.Client, args GetCatalogAppParamsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	cmd := taikuncore.NewGetCatalogAppValueAutocompleteCommand()
	if args.CatalogAppID == 0 && (args.PackageID == "" || args.Version == "") {
		errorResp := ErrorResponse{
			Error: "Provide catalogAppId or both packageId and version",
		}
		return createJSONResponse(errorResp), nil
	}

	if args.CatalogAppID != 0 {
		cmd.SetCatalogAppId(args.CatalogAppID)
	}

	if (args.PackageID == "" || args.Version == "") && args.CatalogAppID != 0 {
		details, response, err := client.Client.CatalogAppAPI.CatalogAppDetails(ctx, args.CatalogAppID).Execute()
		if err != nil {
			return createError(response, err), nil
		}
		if errorResp := checkResponse(response, "get catalog app details"); errorResp != nil {
			return errorResp, nil
		}

		if details != nil {
			if value, ok := details.GetPackageIdOk(); ok && value != nil {
				args.PackageID = *value
			}
			if value, ok := details.GetVersionOk(); ok && value != nil {
				args.Version = *value
			}
		}
	}

	if args.PackageID == "" {
		errorResp := ErrorResponse{
			Error: "Package ID is required when catalogAppId is not provided",
		}
		return createJSONResponse(errorResp), nil
	}
	cmd.SetPackageId(args.PackageID)

	if args.Version == "" {
		errorResp := ErrorResponse{
			Error: "Version is required when catalogAppId is not provided",
		}
		return createJSONResponse(errorResp), nil
	}
	cmd.SetVersion(args.Version)

	availableParams, response, err := client.Client.PackageAPI.PackageValueAutocomplete(ctx).
		GetCatalogAppValueAutocompleteCommand(*cmd).
		Execute()
	if err != nil {
		return createError(response, err), nil
	}
	if errorResp := checkResponse(response, "get catalog app available parameters"); errorResp != nil {
		return errorResp, nil
	}

	var addedParams []taikuncore.CatalogAppParamsDetailsDto
	if args.CatalogAppID != 0 {
		addedParams, response, err = client.Client.CatalogAppAPI.CatalogAppParamDetails(ctx, args.CatalogAppID).Execute()
		if err != nil {
			return createError(response, err), nil
		}
		if errorResp := checkResponse(response, "get catalog app added parameters"); errorResp != nil {
			return errorResp, nil
		}
	}

	type AvailableParam struct {
		Key          string   `json:"key,omitempty"`
		Value        string   `json:"value,omitempty"`
		Description  string   `json:"description,omitempty"`
		Type         string   `json:"type,omitempty"`
		IsQuestion   *bool    `json:"isQuestion,omitempty"`
		Options      []string `json:"options,omitempty"`
		IsTaikunLink *bool    `json:"isTaikunLink,omitempty"`
	}

	type AddedParam struct {
		ID                          int32  `json:"id,omitempty"`
		CatalogAppName              string `json:"catalogAppName,omitempty"`
		Key                         string `json:"key,omitempty"`
		Value                       string `json:"value,omitempty"`
		IsEditableWhenInstalling    *bool  `json:"isEditableWhenInstalling,omitempty"`
		IsEditableAfterInstallation *bool  `json:"isEditableAfterInstallation,omitempty"`
		IsMandatory                 *bool  `json:"isMandatory,omitempty"`
		HasJsonSchema               *bool  `json:"hasJsonSchema,omitempty"`
		IsTaikunLink                *bool  `json:"isTaikunLink,omitempty"`
	}

	available := make([]AvailableParam, 0, len(availableParams))
	for _, param := range availableParams {
		if args.IsTaikunLink != nil {
			if param.IsTaikunLink == nil || *param.IsTaikunLink != *args.IsTaikunLink {
				continue
			}
		}

		detail := AvailableParam{}
		if value, ok := param.GetKeyOk(); ok && value != nil {
			detail.Key = *value
		}
		if value, ok := param.GetValueOk(); ok && value != nil {
			detail.Value = *value
		}
		if value, ok := param.GetDescriptionOk(); ok && value != nil {
			detail.Description = *value
		}
		if value, ok := param.GetTypeOk(); ok && value != nil {
			detail.Type = string(*value)
		}
		if value, ok := param.GetIsQuestionOk(); ok {
			detail.IsQuestion = value
		}
		if len(param.Options) > 0 {
			detail.Options = param.Options
		}
		if value, ok := param.GetIsTaikunLinkOk(); ok {
			detail.IsTaikunLink = value
		}

		available = append(available, detail)
	}

	added := make([]AddedParam, 0, len(addedParams))
	for _, param := range addedParams {
		if args.IsTaikunLink != nil {
			if value, ok := param.GetIsTaikunLinkOk(); ok {
				if value == nil || *value != *args.IsTaikunLink {
					continue
				}
			} else {
				continue
			}
		}

		detail := AddedParam{}
		if param.Id != nil {
			detail.ID = *param.Id
		}
		if value, ok := param.GetCatalogAppNameOk(); ok && value != nil {
			detail.CatalogAppName = *value
		}
		if value, ok := param.GetKeyOk(); ok && value != nil {
			detail.Key = *value
		}
		if value, ok := param.GetValueOk(); ok && value != nil {
			detail.Value = *value
		}
		if value, ok := param.GetIsEditableWhenInstallingOk(); ok {
			detail.IsEditableWhenInstalling = value
		}
		if value, ok := param.GetIsEditableAfterInstallationOk(); ok {
			detail.IsEditableAfterInstallation = value
		}
		if value, ok := param.GetIsMandatoryOk(); ok {
			detail.IsMandatory = value
		}
		if value, ok := param.GetHasJsonSchemaOk(); ok {
			detail.HasJsonSchema = value
		}
		if value, ok := param.GetIsTaikunLinkOk(); ok {
			detail.IsTaikunLink = value
		}

		added = append(added, detail)
	}

	message := fmt.Sprintf("Found %d available parameters and %d added parameters", len(available), len(added))
	if len(available) == 0 && len(added) == 0 {
		message = "No parameters found"
	}

	listResp := struct {
		CatalogAppID   int32            `json:"catalogAppId,omitempty"`
		PackageID      string           `json:"packageId,omitempty"`
		Version        string           `json:"version,omitempty"`
		Available      []AvailableParam `json:"available"`
		Added          []AddedParam     `json:"added"`
		TotalAvailable int              `json:"totalAvailable"`
		TotalAdded     int              `json:"totalAdded"`
		Message        string           `json:"message"`
	}{
		CatalogAppID:   args.CatalogAppID,
		PackageID:      args.PackageID,
		Version:        args.Version,
		Available:      available,
		Added:          added,
		TotalAvailable: len(available),
		TotalAdded:     len(added),
		Message:        message,
	}

	return createJSONResponse(listResp), nil
}

func updateCatalogAppParameters(client *taikungoclient.Client, args SetCatalogAppDefaultParamsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	updateCmd := taikuncore.NewEditCatalogAppParamCommand()
	updateCmd.SetCatalogAppId(args.CatalogAppID)

	params := make([]taikuncore.CatalogAppParamsDto, 0, len(args.Parameters))
	for _, param := range args.Parameters {
		p := taikuncore.NewCatalogAppParamsDto()
		p.SetKey(param.Key)
		p.SetValue(param.Value)
		params = append(params, *p)
	}
	updateCmd.SetParameters(params)

	response, err := client.Client.CatalogAppAPI.CatalogAppEditParams(ctx).
		EditCatalogAppParamCommand(*updateCmd).
		Execute()

	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "update catalog app parameters"); errorResp != nil {
		return errorResp, nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Catalog app ID %d parameters updated successfully", args.CatalogAppID),
		Success: true,
	}

	return createJSONResponse(successResp), nil
}

func listRepositories(client *taikungoclient.Client, args ListRepositoriesArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	// Get all catalogs first
	catalogReq := client.Client.CatalogAPI.CatalogList(ctx)
	catalogList, response, err := catalogReq.Execute()
	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "list catalogs for repository discovery"); errorResp != nil {
		return errorResp, nil
	}

	// Collect unique repositories from all catalog apps
	repositorySet := make(map[string]bool)

	if catalogList != nil && len(catalogList.Data) > 0 {
		for _, catalog := range catalogList.Data {
			// List apps in each catalog to find repositories
			appReq := client.Client.CatalogAppAPI.CatalogAppList(ctx).CatalogId(catalog.GetId())
			catalogAppList, _, err := appReq.Execute()
			if err != nil {
				// Continue with other catalogs if one fails
				continue
			}

			if catalogAppList != nil && len(catalogAppList.Data) > 0 {
				for _, app := range catalogAppList.Data {
					if app.RepoName.IsSet() && app.RepoName.Get() != nil {
						repoName := *app.RepoName.Get()
						if repoName != "" {
							// Apply search filter if provided
							if args.Search == "" || strings.Contains(strings.ToLower(repoName), strings.ToLower(args.Search)) {
								repositorySet[repoName] = true
							}
						}
					}
				}
			}
		}
	}

	// Convert set to slice
	var repositories []string
	for repo := range repositorySet {
		repositories = append(repositories, repo)
	}

	// Apply pagination
	total := len(repositories)
	start := int(args.Offset)
	end := start + int(args.Limit)

	if args.Limit > 0 && start < total {
		if end > total {
			end = total
		}
		repositories = repositories[start:end]
	} else if start >= total {
		repositories = []string{}
	}

	message := fmt.Sprintf("Found %d unique repositories", total)
	if len(repositories) == 0 {
		message = "No repositories found"
	}

	listResp := struct {
		Repositories []string `json:"repositories"`
		Total        int      `json:"total"`
		Message      string   `json:"message"`
	}{
		Repositories: repositories,
		Total:        total,
		Message:      message,
	}

	return createJSONResponse(listResp), nil
}

func listAvailablePackages(client *taikungoclient.Client, args ListAvailablePackagesArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	// Use the PackageAPI to list all available packages
	req := client.Client.PackageAPI.PackageList(ctx)

	if args.Limit > 0 {
		req = req.Limit(args.Limit)
	}
	if args.Offset > 0 {
		req = req.Offset(args.Offset)
	}
	if args.Search != "" {
		req = req.Search(args.Search)
	}
	// Note: Repository filtering might need to be done client-side if not supported by API

	packageList, response, err := req.Execute()
	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "list available packages"); errorResp != nil {
		return errorResp, nil
	}

	// Prepare response data
	type PackageInfo struct {
		Name        string `json:"name"`
		Repository  string `json:"repository"`
		Version     string `json:"version,omitempty"`
		Description string `json:"description,omitempty"`
		AppVersion  string `json:"appVersion,omitempty"`
		Stars       int64  `json:"stars,omitempty"`
		Deprecated  bool   `json:"deprecated,omitempty"`
	}

	var packages []PackageInfo
	if packageList != nil && len(packageList.Data) > 0 {
		for _, pkg := range packageList.Data {
			packageInfo := PackageInfo{}

			// Handle nullable/pointer fields safely using the IsSet() and Get() methods
			if pkg.Name.IsSet() && pkg.Name.Get() != nil {
				packageInfo.Name = *pkg.Name.Get()
			}

			if pkg.Repository != nil && pkg.Repository.Name.IsSet() && pkg.Repository.Name.Get() != nil {
				packageInfo.Repository = *pkg.Repository.Name.Get()
			}

			if pkg.Version.IsSet() && pkg.Version.Get() != nil {
				packageInfo.Version = *pkg.Version.Get()
			}

			if pkg.Description.IsSet() && pkg.Description.Get() != nil {
				packageInfo.Description = *pkg.Description.Get()
			}

			if pkg.AppVersion.IsSet() && pkg.AppVersion.Get() != nil {
				packageInfo.AppVersion = *pkg.AppVersion.Get()
			}

			if pkg.Stars != nil {
				packageInfo.Stars = *pkg.Stars
			}

			if pkg.Deprecated != nil {
				packageInfo.Deprecated = *pkg.Deprecated
			}

			// Apply repository filter client-side if specified
			if args.Repository != "" && packageInfo.Repository != args.Repository {
				continue
			}

			packages = append(packages, packageInfo)
		}
	}

	total := len(packages)
	message := fmt.Sprintf("Found %d available packages", total)
	if total == 0 {
		message = "No packages found"
	}
	if args.Repository != "" {
		message += fmt.Sprintf(" in repository '%s'", args.Repository)
	}

	listResp := struct {
		Packages []PackageInfo `json:"packages"`
		Total    int           `json:"total"`
		Message  string        `json:"message"`
	}{
		Packages: packages,
		Total:    total,
		Message:  message,
	}

	return createJSONResponse(listResp), nil
}

func listAvailableApps(client *taikungoclient.Client, args ListAvailableAppsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	req := client.Client.PackageAPI.PackageList(ctx)

	if args.Limit > 0 {
		req = req.Limit(args.Limit)
	}
	if args.Offset > 0 {
		req = req.Offset(args.Offset)
	}
	if args.Search != "" {
		req = req.Search(args.Search)
	}

	packageList, response, err := req.Execute()
	if err != nil {
		return createError(response, err), nil
	}

	if errorResp := checkResponse(response, "list available apps"); errorResp != nil {
		return errorResp, nil
	}

	type AvailableAppInfo struct {
		PackageID      string `json:"packageId,omitempty"`
		Name           string `json:"name,omitempty"`
		Repository     string `json:"repository,omitempty"`
		Version        string `json:"version,omitempty"`
		AppVersion     string `json:"appVersion,omitempty"`
		Description    string `json:"description,omitempty"`
		CatalogAppID   int32  `json:"catalogAppId,omitempty"`
		CatalogID      int32  `json:"catalogId,omitempty"`
		InstalledCount int32  `json:"installedInstanceCount,omitempty"`
		IsAdded        *bool  `json:"isAdded,omitempty"`
		Deprecated     *bool  `json:"deprecated,omitempty"`
	}

	var apps []AvailableAppInfo
	if packageList != nil && len(packageList.Data) > 0 {
		for _, pkg := range packageList.Data {
			app := AvailableAppInfo{}

			if value, ok := pkg.GetPackageIdOk(); ok && value != nil {
				app.PackageID = *value
			}
			if value, ok := pkg.GetNameOk(); ok && value != nil {
				app.Name = *value
			}
			if pkg.Repository != nil && pkg.Repository.Name.IsSet() && pkg.Repository.Name.Get() != nil {
				app.Repository = *pkg.Repository.Name.Get()
			}
			if value, ok := pkg.GetVersionOk(); ok && value != nil {
				app.Version = *value
			}
			if value, ok := pkg.GetAppVersionOk(); ok && value != nil {
				app.AppVersion = *value
			}
			if value, ok := pkg.GetDescriptionOk(); ok && value != nil {
				app.Description = *value
			}
			if value, ok := pkg.GetCatalogAppIdOk(); ok && value != nil {
				app.CatalogAppID = *value
			}
			if value, ok := pkg.GetCatalogIdOk(); ok && value != nil {
				app.CatalogID = *value
			}
			if value, ok := pkg.GetInstalledInstanceCountOk(); ok && value != nil {
				app.InstalledCount = *value
			}
			if value, ok := pkg.GetIsAddedOk(); ok {
				app.IsAdded = value
			}
			if pkg.Deprecated != nil {
				app.Deprecated = pkg.Deprecated
			}

			if args.Repository != "" && app.Repository != args.Repository {
				continue
			}

			apps = append(apps, app)
		}
	}

	message := fmt.Sprintf("Found %d available apps", len(apps))
	if len(apps) == 0 {
		message = "No available apps found"
	}
	if args.Repository != "" {
		message += fmt.Sprintf(" in repository '%s'", args.Repository)
	}

	listResp := struct {
		Apps    []AvailableAppInfo `json:"apps"`
		Total   int                `json:"total"`
		Message string             `json:"message"`
	}{
		Apps:    apps,
		Total:   len(apps),
		Message: message,
	}

	return createJSONResponse(listResp), nil
}
