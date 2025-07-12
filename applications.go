package main

import (
	"context"
	"fmt"
	"time"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

type AppParameter struct {
	Key   string `json:"key" jsonschema:"required,description=Parameter key"`
	Value string `json:"value" jsonschema:"required,description=Parameter value"`
}

type InstallAppArgs struct {
	Name              string         `json:"name" jsonschema:"required,description=The name of the application instance"`
	Namespace         string         `json:"namespace" jsonschema:"required,description=The namespace to install the application in"`
	ProjectID         int32          `json:"projectId" jsonschema:"required,description=The project ID to install the application in"`
	CatalogAppID      int32          `json:"catalogAppId" jsonschema:"required,description=The catalog application ID to install"`
	ExtraValues       string         `json:"extraValues,omitempty" jsonschema:"description=Base64-encoded YAML extra values for the application (optional)"`
	AutoSync          bool           `json:"autoSync,omitempty" jsonschema:"description=Enable automatic synchronization (default: false)"`
	TaikunLinkEnabled bool           `json:"taikunLinkEnabled,omitempty" jsonschema:"description=Enable Taikun link integration (default: false)"`
	Timeout           int32          `json:"timeout,omitempty" jsonschema:"description=Installation timeout in seconds (optional)"`
	Parameters        []AppParameter `json:"parameters,omitempty" jsonschema:"description=Application parameters as key-value pairs (optional)"`
	WaitForReady      bool           `json:"waitForReady,omitempty" jsonschema:"description=Wait for application to be ready before returning (default: false)"`
	WaitTimeout       int32          `json:"waitTimeout,omitempty" jsonschema:"description=Wait timeout in seconds (default: 600)"`
}

type ListAppsArgs struct {
	ProjectID int32  `json:"projectId" jsonschema:"required,description=The project ID to list applications from"`
	Limit     int32  `json:"limit,omitempty" jsonschema:"description=Maximum number of results to return (optional)"`
	Offset    int32  `json:"offset,omitempty" jsonschema:"description=Number of results to skip (optional)"`
	Search    string `json:"search,omitempty" jsonschema:"description=Search term to filter results (optional)"`
}

type GetAppArgs struct {
	ProjectAppID int32 `json:"projectAppId" jsonschema:"required,description=The project application ID to get details for"`
}

type UpdateSyncAppArgs struct {
	ProjectAppID int32  `json:"projectAppId" jsonschema:"required,description=The project application ID to update/sync"`
	ExtraValues  string `json:"extraValues,omitempty" jsonschema:"description=Base64-encoded YAML extra values (optional - if not provided, will only sync)"`
	Timeout      int32  `json:"timeout,omitempty" jsonschema:"description=Operation timeout in seconds (optional)"`
	SyncOnly     bool   `json:"syncOnly,omitempty" jsonschema:"description=If true, only sync without updating values (default: false)"`
}

type UninstallAppArgs struct {
	ProjectAppID int32 `json:"projectAppId" jsonschema:"required,description=The project application ID to uninstall"`
}

// waitForAppReady waits for an application to reach READY status
func waitForAppReady(client *taikungoclient.Client, projectAppID int32, timeoutSeconds int32) error {
	ctx := context.Background()
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeoutSeconds == 0 {
		timeout = 10 * time.Minute // Default 10 minutes like Terraform provider
	}

	start := time.Now()
	for {
		// Check if we've exceeded the timeout
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for application ID %d to be ready after %v", projectAppID, timeout)
		}

		// Query the application status
		appDetails, response, err := client.Client.ProjectAppsAPI.ProjectappDetails(ctx, projectAppID).Execute()
		if err != nil {
			return fmt.Errorf("error checking application status: %v", err)
		}

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("HTTP error checking application status: %d", response.StatusCode)
		}

		if appDetails != nil {
			status := string(appDetails.GetStatus())
			logger.Printf("Application ID %d status: %s", projectAppID, status)

			// Check if app is ready
			if status == "Ready" {
				return nil // Success!
			}

			// Check for failure states
			if status == "Failed" {
				return fmt.Errorf("application ID %d installation failed - status: %s", projectAppID, status)
			}

			// Continue polling if still installing/pending
			// Valid pending states: "NONE", "NOT_READY", "INSTALLING", "UNINSTALLING"
		}

		// Wait before next poll
		time.Sleep(10 * time.Second)
	}
}

func installApp(client *taikungoclient.Client, args InstallAppArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	createCmd := taikuncore.NewCreateProjectAppCommand()
	createCmd.SetName(args.Name)
	createCmd.SetNamespace(args.Namespace)
	createCmd.SetProjectId(args.ProjectID)
	createCmd.SetCatalogAppId(args.CatalogAppID)
	createCmd.SetAutoSync(args.AutoSync)
	createCmd.SetTaikunLinkEnabled(args.TaikunLinkEnabled)

	if args.ExtraValues != "" {
		createCmd.SetExtraValues(args.ExtraValues)
	}

	if args.Timeout > 0 {
		createCmd.SetTimeout(args.Timeout)
	}

	if len(args.Parameters) > 0 {
		var params []taikuncore.ProjectAppParamsDto
		for _, param := range args.Parameters {
			p := taikuncore.NewProjectAppParamsDto()
			p.SetKey(param.Key)
			p.SetValue(param.Value)
			params = append(params, *p)
		}
		createCmd.SetParameters(params)
	}

	response, httpResponse, err := client.Client.ProjectAppsAPI.ProjectappInstall(ctx).
		CreateProjectAppCommand(*createCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "install application"); errorResp != nil {
		return errorResp, nil
	}

	// Get the application ID by searching for the newly created app
	var projectAppID int32
	if args.WaitForReady {
		// Find the newly created application by searching
		appList, _, err := client.Client.ProjectAppsAPI.ProjectappList(ctx).
			ProjectId(args.ProjectID).
			Search(args.Name).
			Execute()

		if err == nil && appList != nil && len(appList.Data) > 0 {
			// Find the app with matching name and namespace
			for _, app := range appList.Data {
				if app.GetName() == args.Name && app.GetNamespace() == args.Namespace {
					projectAppID = app.GetId()
					logger.Printf("Found application '%s' with ID: %d", args.Name, projectAppID)
					break
				}
			}
		}
	}

	var resultMsg string
	if args.WaitForReady && projectAppID > 0 {
		logger.Printf("Waiting for application '%s' (ID: %d) to be ready...", args.Name, projectAppID)
		waitTimeout := args.WaitTimeout
		if waitTimeout == 0 {
			waitTimeout = 600 // Default 10 minutes
		}

		err := waitForAppReady(client, projectAppID, waitTimeout)
		if err != nil {
			errorResp := ErrorResponse{
				Error: fmt.Sprintf("Application '%s' installation initiated but failed during wait: %v", args.Name, err),
			}
			return createJSONResponse(errorResp), nil
		}
		resultMsg = fmt.Sprintf("Application '%s' (ID: %d) installed successfully and is ready in namespace '%s'", args.Name, projectAppID, args.Namespace)
		logger.Printf("Application '%s' (ID: %d) is now ready", args.Name, projectAppID)
	} else {
		if response != nil && response.GetMessage() != "" {
			resultMsg = fmt.Sprintf("Application '%s' installation initiated successfully. Message: %s", args.Name, response.GetMessage())
		} else {
			resultMsg = fmt.Sprintf("Application '%s' installation initiated successfully in namespace '%s'", args.Name, args.Namespace)
		}
		if projectAppID > 0 {
			resultMsg += fmt.Sprintf(" (ID: %d)", projectAppID)
		}
	}

	type InstallAppResponse struct {
		Message      string `json:"message"`
		Success      bool   `json:"success"`
		ProjectAppID int32  `json:"projectAppId,omitempty"`
		Name         string `json:"name"`
		Namespace    string `json:"namespace"`
		Status       string `json:"status"`
	}

	responseData := InstallAppResponse{
		Message:      resultMsg,
		Success:      true,
		ProjectAppID: projectAppID,
		Name:         args.Name,
		Namespace:    args.Namespace,
		Status:       "initiated",
	}

	if args.WaitForReady {
		responseData.Status = "ready"
	}

	return createJSONResponse(responseData), nil
}

func listApps(client *taikungoclient.Client, args ListAppsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	req := client.Client.ProjectAppsAPI.ProjectappList(ctx).ProjectId(args.ProjectID)

	if args.Limit > 0 {
		req = req.Limit(args.Limit)
	}
	if args.Offset > 0 {
		req = req.Offset(args.Offset)
	}
	if args.Search != "" {
		req = req.Search(args.Search)
	}

	appList, httpResponse, err := req.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "list applications"); errorResp != nil {
		return errorResp, nil
	}

	// Prepare response data
	type AppSummary struct {
		ID         int32  `json:"id"`
		Name       string `json:"name"`
		Namespace  string `json:"namespace"`
		Status     string `json:"status"`
		Version    string `json:"version"`
		CatalogApp string `json:"catalogApp"`
		AutoSync   bool   `json:"autoSync"`
		Created    string `json:"created,omitempty"`
		CreatedBy  string `json:"createdBy,omitempty"`
	}

	var applications []AppSummary
	if appList != nil && len(appList.Data) > 0 {
		for _, app := range appList.Data {
			appSummary := AppSummary{
				ID:         app.GetId(),
				Name:       app.GetName(),
				Namespace:  app.GetNamespace(),
				Status:     string(app.GetStatus()),
				Version:    app.GetVersion(),
				CatalogApp: app.GetCatalogAppName(),
				AutoSync:   app.GetAutoSync(),
			}

			if app.GetCreated() != "" {
				appSummary.Created = app.GetCreated()
			}
			if app.GetCreatedBy() != "" {
				appSummary.CreatedBy = app.GetCreatedBy()
			}

			applications = append(applications, appSummary)
		}
	}

	message := fmt.Sprintf("Found %d applications in project %d", len(applications), args.ProjectID)
	if len(applications) == 0 {
		message = fmt.Sprintf("No applications found in project %d", args.ProjectID)
	}

	listResp := struct {
		Applications []AppSummary `json:"applications"`
		Total        int          `json:"total"`
		ProjectID    int32        `json:"projectId"`
		Message      string       `json:"message"`
	}{
		Applications: applications,
		Total:        len(applications),
		ProjectID:    args.ProjectID,
		Message:      message,
	}

	return createJSONResponse(listResp), nil
}

func getApp(client *taikungoclient.Client, args GetAppArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	appDetails, httpResponse, err := client.Client.ProjectAppsAPI.ProjectappDetails(ctx, args.ProjectAppID).Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "get application details"); errorResp != nil {
		return errorResp, nil
	}

	if appDetails == nil {
		errorResp := ErrorResponse{
			Error: fmt.Sprintf("Application with ID %d not found", args.ProjectAppID),
		}
		return createJSONResponse(errorResp), nil
	}

	// Prepare response data
	type AppParameter struct {
		Key                         string `json:"key"`
		Value                       string `json:"value"`
		IsMandatory                 bool   `json:"isMandatory"`
		IsEditableAfterInstallation bool   `json:"isEditableAfterInstallation"`
	}

	type AppDetails struct {
		ID                int32          `json:"id"`
		Name              string         `json:"name"`
		Namespace         string         `json:"namespace"`
		Status            string         `json:"status"`
		Version           string         `json:"version"`
		ProjectName       string         `json:"projectName"`
		ProjectID         int32          `json:"projectId"`
		CatalogName       string         `json:"catalogName"`
		CatalogID         int32          `json:"catalogId"`
		CatalogAppName    string         `json:"catalogAppName"`
		CatalogAppID      int32          `json:"catalogAppId"`
		PackageID         string         `json:"packageId"`
		AutoSync          bool           `json:"autoSync"`
		HasJsonSchema     bool           `json:"hasJsonSchema"`
		TaikunLinkEnabled bool           `json:"taikunLinkEnabled"`
		TaikunLinkUrl     string         `json:"taikunLinkUrl,omitempty"`
		Logo              string         `json:"logo,omitempty"`
		Parameters        []AppParameter `json:"parameters,omitempty"`
		Values            string         `json:"values,omitempty"`
		HelmResult        string         `json:"helmResult,omitempty"`
		Logs              string         `json:"logs,omitempty"`
		ReleaseNotes      string         `json:"releaseNotes,omitempty"`
	}

	appDetail := AppDetails{
		ID:                args.ProjectAppID,
		Name:              appDetails.GetName(),
		Namespace:         appDetails.GetNamespace(),
		Status:            string(appDetails.GetStatus()),
		Version:           appDetails.GetVersion(),
		ProjectName:       appDetails.GetProjectName(),
		ProjectID:         appDetails.GetProjectId(),
		CatalogName:       appDetails.GetCatalogName(),
		CatalogID:         appDetails.GetCatalogId(),
		CatalogAppName:    appDetails.GetCatalogAppName(),
		CatalogAppID:      appDetails.GetCatalogAppId(),
		PackageID:         appDetails.GetPackageId(),
		AutoSync:          appDetails.GetAutoSync(),
		HasJsonSchema:     appDetails.GetHasJsonSchema(),
		TaikunLinkEnabled: appDetails.GetTaikunLinkEnabled(),
	}

	if appDetails.GetTaikunLinkUrl() != "" {
		appDetail.TaikunLinkUrl = appDetails.GetTaikunLinkUrl()
	}

	if appDetails.GetLogo() != "" {
		appDetail.Logo = appDetails.GetLogo()
	}

	if len(appDetails.ProjectAppParams) > 0 {
		for _, param := range appDetails.ProjectAppParams {
			appDetail.Parameters = append(appDetail.Parameters, AppParameter{
				Key:                         param.GetKey(),
				Value:                       param.GetValue(),
				IsMandatory:                 param.GetIsMandatory(),
				IsEditableAfterInstallation: param.GetIsEditableAfterInstallation(),
			})
		}
	}

	if appDetails.GetValues() != "" {
		appDetail.Values = appDetails.GetValues()
	}

	if appDetails.GetHelmResult() != "" {
		appDetail.HelmResult = appDetails.GetHelmResult()
	}

	if appDetails.GetLogs() != "" {
		appDetail.Logs = appDetails.GetLogs()
	}

	if appDetails.GetReleaseNotes() != "" {
		appDetail.ReleaseNotes = appDetails.GetReleaseNotes()
	}

	return createJSONResponse(appDetail), nil
}

func updateSyncApp(client *taikungoclient.Client, args UpdateSyncAppArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	var resultMsg string

	// First, get the app details to check autosync status
	appDetails, httpResponse, err := client.Client.ProjectAppsAPI.ProjectappDetails(ctx, args.ProjectAppID).Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "get application details"); errorResp != nil {
		return errorResp, nil
	}

	if appDetails == nil {
		errorResp := ErrorResponse{
			Error: fmt.Sprintf("Application with ID %d not found", args.ProjectAppID),
		}
		return createJSONResponse(errorResp), nil
	}

	hasAutoSync := appDetails.GetAutoSync()
	logger.Printf("Application ID %d has autosync enabled: %t", args.ProjectAppID, hasAutoSync)

	if !args.SyncOnly && args.ExtraValues != "" {
		// Update the values
		updateCmd := taikuncore.NewEditProjectAppExtraValuesCommand()
		updateCmd.SetProjectAppId(args.ProjectAppID)
		updateCmd.SetExtraValues(args.ExtraValues)

		if args.Timeout > 0 {
			updateCmd.SetTimeout(args.Timeout)
		}

		_, httpResponse, err := client.Client.ProjectAppsAPI.ProjectappUpdateExtraValues(ctx).
			EditProjectAppExtraValuesCommand(*updateCmd).
			Execute()

		if err != nil {
			return createError(httpResponse, err), nil
		}

		if errorResp := checkResponse(httpResponse, "update application values"); errorResp != nil {
			return errorResp, nil
		}

		if hasAutoSync {
			// App has autosync enabled, no need to manually sync
			resultMsg = fmt.Sprintf("Application ID %d values updated successfully (autosync enabled)", args.ProjectAppID)
		} else {
			// App doesn't have autosync, need to manually sync
			syncCmd := taikuncore.NewSyncProjectAppCommand()
			syncCmd.SetProjectAppId(args.ProjectAppID)

			if args.Timeout > 0 {
				syncCmd.SetTimeout(args.Timeout)
			}

			httpResponse, err := client.Client.ProjectAppsAPI.ProjectappSync(ctx).
				SyncProjectAppCommand(*syncCmd).
				Execute()

			if err != nil {
				return createError(httpResponse, err), nil
			}

			if errorResp := checkResponse(httpResponse, "sync application"); errorResp != nil {
				return errorResp, nil
			}

			resultMsg = fmt.Sprintf("Application ID %d values updated and synced successfully", args.ProjectAppID)
		}
	} else {
		// Only sync (no value update)
		if hasAutoSync {
			resultMsg = fmt.Sprintf("Application ID %d has autosync enabled, no manual sync needed", args.ProjectAppID)
		} else {
			syncCmd := taikuncore.NewSyncProjectAppCommand()
			syncCmd.SetProjectAppId(args.ProjectAppID)

			if args.Timeout > 0 {
				syncCmd.SetTimeout(args.Timeout)
			}

			httpResponse, err := client.Client.ProjectAppsAPI.ProjectappSync(ctx).
				SyncProjectAppCommand(*syncCmd).
				Execute()

			if err != nil {
				return createError(httpResponse, err), nil
			}

			if errorResp := checkResponse(httpResponse, "sync application"); errorResp != nil {
				return errorResp, nil
			}

			resultMsg = fmt.Sprintf("Application ID %d synced successfully", args.ProjectAppID)
		}
	}

	successResp := SuccessResponse{
		Message: resultMsg,
		Success: true,
	}
	return createJSONResponse(successResp), nil
}

func uninstallApp(client *taikungoclient.Client, args UninstallAppArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	uninstallResponse, httpResponse, err := client.Client.ProjectAppsAPI.ProjectappDelete(ctx, args.ProjectAppID).Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "uninstall application"); errorResp != nil {
		return errorResp, nil
	}

	var resultMsg string
	if uninstallResponse != nil && uninstallResponse.GetIsJobSkipped() {
		resultMsg = fmt.Sprintf("Application ID %d uninstall skipped (already uninstalled or not found)", args.ProjectAppID)
	} else {
		resultMsg = fmt.Sprintf("Application ID %d uninstall initiated successfully", args.ProjectAppID)
	}

	successResp := SuccessResponse{
		Message: resultMsg,
		Success: true,
	}
	return createJSONResponse(successResp), nil
}
