package main

import (
	"context"
	"fmt"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
)

func listCloudCredentials(client *taikungoclient.Client, args ListCloudCredentialsArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	searchCmd := taikuncore.NewCloudCredentialsSearchCommand()
	if args.Limit > 0 {
		searchCmd.SetLimit(args.Limit)
	}
	if args.Offset > 0 {
		searchCmd.SetOffset(args.Offset)
	}
	if args.Search != "" {
		searchCmd.SetSearchTerm(args.Search)
	}

	searchReq := client.Client.SearchAPI.SearchCloudCredentials(ctx).
		CloudCredentialsSearchCommand(*searchCmd)

	result, httpResponse, err := searchReq.Execute()
	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, "list cloud credentials"); errorResp != nil {
		return errorResp, nil
	}

	var credentials []CloudCredentialSummary
	if result != nil && result.Data != nil {
		for _, cred := range result.Data {
			summary := CloudCredentialSummary{
				ID:               cred.GetId(),
				Name:             cred.GetName(),
				CloudType:        cred.GetCloudType(),
				OrganizationName: cred.GetOrganizationName(),
			}
			credentials = append(credentials, summary)
		}
	}

	total := 0
	if result != nil {
		total = len(credentials) // Search API doesn't seem to return total count in this struct, but we return what we got
	}

	response := CloudCredentialListResponse{
		Credentials: credentials,
		Total:       total,
		Message:     fmt.Sprintf("Found %d cloud credentials", len(credentials)),
	}

	return createJSONResponse(response), nil
}
