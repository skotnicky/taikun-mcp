package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

type PodSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	PodIP     string `json:"podIP"`
	StartTime string `json:"startTime,omitempty"`
	Restarts  int32  `json:"restarts"`
}

type DeploymentSummary struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	Replicas          int32  `json:"replicas"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	UpdatedReplicas   int32  `json:"updatedReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
	Age               string `json:"age"`
}

type ServiceSummary struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Type      string   `json:"type"`
	ClusterIP string   `json:"clusterIP"`
	Ports     []string `json:"ports"`
	Age       string   `json:"age"`
}

type NamespaceSummary struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Age    string `json:"age"`
}

type ConfigMapSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	DataCount int    `json:"dataCount"`
	Age       string `json:"age"`
}

type SecretSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	DataCount int    `json:"dataCount"`
	Age       string `json:"age"`
}

type IngressSummary struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Hosts     []string `json:"hosts"`
	Address   string   `json:"address"`
	Age       string   `json:"age"`
}

type CronJobSummary struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	Schedule         string `json:"schedule"`
	Suspend          bool   `json:"suspend"`
	Active           int    `json:"active"`
	LastScheduleTime string `json:"lastScheduleTime,omitempty"`
	Age              string `json:"age"`
}

type DaemonSetSummary struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	DesiredScheduled int32  `json:"desiredScheduled"`
	CurrentScheduled int32  `json:"currentScheduled"`
	NumberReady      int32  `json:"numberReady"`
	NumberAvailable  int32  `json:"numberAvailable"`
	Age              string `json:"age"`
}

type JobSummary struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Completions string `json:"completions"`
	Succeeded   int32  `json:"succeeded"`
	Age         string `json:"age"`
}

type NodeSummary struct {
	Name             string `json:"name"`
	Status           string `json:"status"`
	Roles            string `json:"roles"`
	Version          string `json:"version"`
	OSImage          string `json:"osImage"`
	KernelVersion    string `json:"kernelVersion"`
	ContainerRuntime string `json:"containerRuntime"`
	Age              string `json:"age"`
}

type PvcSummary struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Volume       string `json:"volume"`
	Capacity     string `json:"capacity"`
	AccessModes  string `json:"accessModes"`
	StorageClass string `json:"storageClass"`
	Age          string `json:"age"`
}

type StorageClassSummary struct {
	Name          string `json:"name"`
	Provisioner   string `json:"provisioner"`
	ReclaimPolicy string `json:"reclaimPolicy"`
	Age           string `json:"age"`
}

type StatefulSetSummary struct {
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Replicas      int32  `json:"replicas"`
	ReadyReplicas int32  `json:"readyReplicas"`
	Age           string `json:"age"`
}

type cursorPaginatedResponse[T any] struct {
	Data       []T     `json:"data"`
	Limit      int32   `json:"limit"`
	HasMore    bool    `json:"hasMore"`
	TotalCount int64   `json:"totalCount"`
	NextCursor *string `json:"nextCursor"`
}

type podListItem struct {
	State        string `json:"state"`
	Name         string `json:"name"`
	Ready        string `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	CreatedAt    string `json:"createdAt"`
	Namespace    string `json:"namespace"`
	Node         string `json:"node"`
	IP           string `json:"ip"`
}

type deploymentListItem struct {
	State     string   `json:"state"`
	Name      string   `json:"name"`
	Ready     string   `json:"ready"`
	CreatedAt string   `json:"createdAt"`
	Namespace string   `json:"namespace"`
	Images    []string `json:"images"`
}

type serviceListItem struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Type       string `json:"type"`
	ClusterIP  string `json:"clusterIp"`
	ExternalIP string `json:"externalIp"`
	CreatedAt  string `json:"createdAt"`
}

type nodeListItem struct {
	State   string `json:"state"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Version string `json:"version"`
	IP      string `json:"ip"`
}

type configMapListItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CreatedAt string `json:"createdAt"`
}

type secretListItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	CreatedAt string `json:"createdAt"`
}

type ingressListItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Target       string `json:"target"`
	Default      string `json:"default"`
	IngressClass string `json:"ingressClass"`
	CreatedAt    string `json:"createdAt"`
}

type daemonSetListItem struct {
	Status    string `json:"status"`
	Name      string `json:"name"`
	Desired   int32  `json:"desired"`
	Current   int32  `json:"current"`
	Ready     int32  `json:"ready"`
	Available string `json:"available"`
	CreatedAt string `json:"createdAt"`
	Namespace string `json:"namespace"`
	Image     string `json:"image"`
}

type pvcListItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Volume       string `json:"volume"`
	Capacity     string `json:"capacity"`
	AccessModes  string `json:"accessModes"`
	StorageClass string `json:"storageClass"`
	CreatedAt    string `json:"createdAt"`
}

type statefulSetListItem struct {
	State     string   `json:"state"`
	Name      string   `json:"name"`
	Ready     string   `json:"ready"`
	CreatedAt string   `json:"createdAt"`
	Namespace string   `json:"namespace"`
	Images    []string `json:"images"`
}

func formatAge(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "Unknown"
	}
	duration := time.Since(timestamp.Time)
	duration = duration.Round(time.Second)

	if duration.Hours() > 24 {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}

	return duration.String()
}

func formatAgeFromString(createdAt string) string {
	if createdAt == "" {
		return "Unknown"
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, createdAt); err == nil {
			return formatAge(metav1.NewTime(parsed))
		}
	}
	return createdAt
}

func parseReadyCounts(ready string) (int32, int32) {
	parts := strings.Split(ready, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	readyCount, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0
	}
	totalCount, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0
	}
	return int32(readyCount), int32(totalCount)
}

func parseInt32(value string) int32 {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return int32(parsed)
}

func isLikelyKubernetesYaml(payload string) bool {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "---") {
		return true
	}
	if strings.Contains(trimmed, "apiVersion:") || strings.Contains(trimmed, "kind:") {
		return true
	}
	return false
}

func tryDecodeBase64Yaml(payload string) (string, bool) {
	if strings.TrimSpace(payload) == "" {
		return "", false
	}
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, payload)
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", false
	}
	if !utf8.Valid(decoded) {
		return "", false
	}
	text := strings.TrimSpace(string(decoded))
	if !isLikelyKubernetesYaml(text) {
		return "", false
	}
	return text, true
}

func normalizeYamlInput(payload string) (string, error) {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return "", errors.New("yaml input is empty")
	}
	if decoded, ok := tryDecodeBase64Yaml(trimmed); ok {
		return decoded, nil
	}
	return trimmed, nil
}

func normalizeYamlOutput(payload string) string {
	if decoded, ok := tryDecodeBase64Yaml(payload); ok {
		return strings.ReplaceAll(decoded, "\r\n", "\n")
	}
	return strings.ReplaceAll(payload, "\r\n", "\n")
}

func validateKubernetesYaml(payload string) error {
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(payload), 4096)
	for {
		var raw map[string]interface{}
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("invalid YAML: %w", err)
		}
		if len(raw) == 0 {
			continue
		}
		jsonData, err := json.Marshal(raw)
		if err != nil {
			return fmt.Errorf("failed to serialize YAML: %w", err)
		}
		if _, _, err := scheme.Codecs.UniversalDeserializer().Decode(jsonData, nil, nil); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}
	}
	return nil
}

func splitKubernetesYaml(payload string) ([]string, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(payload), 4096)
	var docs []string
	for {
		var raw map[string]interface{}
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
		if len(raw) == 0 {
			continue
		}
		jsonData, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize YAML: %w", err)
		}
		docs = append(docs, string(jsonData))
	}
	if len(docs) == 0 {
		return nil, errors.New("yaml input contains no resources")
	}
	return docs, nil
}

type DeleteKubernetesResourceArgs struct {
	ProjectID int32  `json:"projectId" jsonschema:"required,description=The project ID of the resource"`
	Kind      string `json:"kind" jsonschema:"required,description=The kind of the resource (e.g., Pod, Deployment, Service)"`
	Name      string `json:"name" jsonschema:"required,description=The name of the resource to delete"`
	Namespace string `json:"namespace,omitempty" jsonschema:"description=The namespace of the resource (optional, defaults to 'default')"`
}

type DeployKubernetesResourcesArgs struct {
	ProjectID int32  `json:"projectId" jsonschema:"required,description=The project ID to deploy the resources to"`
	YAML      string `json:"yaml" jsonschema:"required,description=The Kubernetes resources in YAML format (raw or base64-encoded)"`
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
	Namespace string `json:"namespace,omitempty" jsonschema:"description=The namespace of the resource (optional, defaults to 'default')"`
}

type PatchKubernetesResourceArgs struct {
	ProjectID int32  `json:"projectId" jsonschema:"required,description=The project ID of the resource"`
	Name      string `json:"name" jsonschema:"required,description=The name of the resource to patch"`
	Yaml      string `json:"yaml" jsonschema:"required,description=The YAML patch to apply to the resource (raw or base64-encoded)"`
	Namespace string `json:"namespace,omitempty" jsonschema:"description=The namespace of the resource (optional, defaults to 'default')"`
}

type ListKubeConfigRolesArgs struct{}

func deployKubernetesResources(client *taikungoclient.Client, args DeployKubernetesResourcesArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	normalizedYaml, err := normalizeYamlInput(args.YAML)
	if err != nil {
		return createJSONResponse(ErrorResponse{Error: err.Error()}), nil
	}
	docs, err := splitKubernetesYaml(normalizedYaml)
	if err != nil {
		return createJSONResponse(ErrorResponse{Error: err.Error()}), nil
	}
	for _, doc := range docs {
		if err := validateKubernetesYaml(doc); err != nil {
			return createJSONResponse(ErrorResponse{Error: err.Error()}), nil
		}
		encodedYaml := base64.StdEncoding.EncodeToString([]byte(doc))
		createCmd := taikuncore.NewCreateKubernetesResourceCommand(args.ProjectID, *taikuncore.NewNullableString(&encodedYaml))
		httpResponse, err := client.Client.KubernetesAPI.KubernetesCreateResource(ctx).
			CreateKubernetesResourceCommand(*createCmd).
			Execute()
		if err != nil {
			return createError(httpResponse, err), nil
		}
		if errorResp := checkResponse(httpResponse, "deploy kubernetes resources"); errorResp != nil {
			return errorResp, nil
		}
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Kubernetes resources deployed successfully (%d resource(s))", len(docs)),
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

	listRequest := client.Client.KubeConfigAPI.KubeconfigList(ctx).
		ProjectId(args.ProjectID).
		Limit(100)
	listResp, listHTTPResponse, err := listRequest.Execute()
	if err != nil {
		return createError(listHTTPResponse, err), nil
	}

	if errorResp := checkResponse(listHTTPResponse, "list kubeconfigs"); errorResp != nil {
		return errorResp, nil
	}

	var kubeconfigId int32
	var fallbackId int32
	if listResp != nil {
		for _, item := range listResp.Data {
			if item.ProjectId != args.ProjectID || !item.CanDownload {
				continue
			}
			if item.KubeConfigRoleName == "cluster-admin" || item.KubeConfigRoleName == "admin" {
				kubeconfigId = item.Id
				break
			}
			if fallbackId == 0 {
				fallbackId = item.Id
			}
		}
	}

	if kubeconfigId == 0 {
		kubeconfigId = fallbackId
	}

	if kubeconfigId == 0 {
		errorResp := ErrorResponse{
			Error: fmt.Sprintf("No downloadable kubeconfig found for project %d", args.ProjectID),
		}
		return createJSONResponse(errorResp), nil
	}

	downloadCmd := taikuncore.NewDownloadKubeConfigCommand()
	downloadCmd.SetId(kubeconfigId)
	downloadCmd.SetProjectId(args.ProjectID)
	kubeconfig, downloadHTTPResponse, err := client.Client.KubeConfigAPI.KubeconfigDownload(ctx).
		DownloadKubeConfigCommand(*downloadCmd).
		Execute()
	if err != nil {
		return createError(downloadHTTPResponse, err), nil
	}

	if errorResp := checkResponse(downloadHTTPResponse, "download kubeconfig"); errorResp != nil {
		return errorResp, nil
	}

	if kubeconfig == "" {
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
		KubeConfig: strings.ReplaceAll(kubeconfig, "\r\n", "\n"),
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

func fetchKubernetesListPage[T any](ctx context.Context, client *taikungoclient.Client, projectID int32, resource string, limit int32, cursor string, searchTerm string) (cursorPaginatedResponse[T], *http.Response, error) {
	var result cursorPaginatedResponse[T]

	if client == nil || client.Client == nil {
		return result, nil, fmt.Errorf("Cloudera Cloud Factory client is not initialized")
	}

	cfg := client.Client.GetConfig()
	if cfg == nil || cfg.HTTPClient == nil {
		return result, nil, fmt.Errorf("Cloudera Cloud Factory client config is not available")
	}

	baseURL := fmt.Sprintf("%s://%s", cfg.Scheme, cfg.Host)
	endpoint := fmt.Sprintf("%s/api/v1/kubernetes/list/%d/%s", baseURL, projectID, resource)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return result, nil, err
	}

	query := req.URL.Query()
	if limit > 0 {
		query.Set("Limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		query.Set("Cursor", cursor)
	}
	if searchTerm != "" {
		query.Set("SearchTerm", searchTerm)
	}
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Accept", "application/json")

	response, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return result, response, err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return result, response, fmt.Errorf("request failed with status %d", response.StatusCode)
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return result, response, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, response, err
	}

	return result, response, nil
}

func fetchKubernetesListItems[T any](ctx context.Context, client *taikungoclient.Client, projectID int32, resource string, limit int32, offset int32, searchTerm string) ([]T, *http.Response, error) {
	var allItems []T
	var cursor string
	var lastResponse *http.Response
	perPage := limit
	if perPage <= 0 {
		perPage = 50
	}

	remainingOffset := offset
	remainingLimit := limit

	for {
		page, response, err := fetchKubernetesListPage[T](ctx, client, projectID, resource, perPage, cursor, searchTerm)
		lastResponse = response
		if err != nil {
			return nil, response, err
		}

		items := page.Data
		if remainingOffset > 0 {
			if int32(len(items)) <= remainingOffset {
				remainingOffset -= int32(len(items))
				items = nil
			} else {
				items = items[remainingOffset:]
				remainingOffset = 0
			}
		}

		if remainingLimit > 0 {
			needed := remainingLimit - int32(len(allItems))
			if needed <= 0 {
				break
			}
			if int32(len(items)) > needed {
				items = items[:needed]
			}
		}

		allItems = append(allItems, items...)

		if remainingLimit > 0 && int32(len(allItems)) >= remainingLimit {
			break
		}

		if !page.HasMore || page.NextCursor == nil || *page.NextCursor == "" {
			break
		}

		cursor = *page.NextCursor
	}

	return allItems, lastResponse, nil
}

func listKubernetesError(kind string, response *http.Response, err error) *mcp_golang.ToolResponse {
	if response != nil && (response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices) {
		return createError(response, err)
	}
	errorResp := ErrorResponse{
		Error: fmt.Sprintf("Failed to list %s: %v", kind, err),
	}
	return createJSONResponse(errorResp)
}

func listKubernetesResources(client *taikungoclient.Client, args ListKubernetesResourcesArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()
	var result interface{}

	switch args.Kind {
	case "Pods":
		pods, response, err := fetchKubernetesListItems[podListItem](ctx, client, args.ProjectID, "pods", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("Pods", response, err), nil
		}
		summaries := make([]PodSummary, 0, len(pods))
		for _, pod := range pods {
			summaries = append(summaries, PodSummary{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    pod.State,
				PodIP:     pod.IP,
				StartTime: pod.CreatedAt,
				Restarts:  pod.RestartCount,
			})
		}
		result = summaries
	case "Deployments":
		deployments, response, err := fetchKubernetesListItems[deploymentListItem](ctx, client, args.ProjectID, "deployments", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("Deployments", response, err), nil
		}
		summaries := make([]DeploymentSummary, 0, len(deployments))
		for _, deployment := range deployments {
			readyCount, totalCount := parseReadyCounts(deployment.Ready)
			summaries = append(summaries, DeploymentSummary{
				Name:              deployment.Name,
				Namespace:         deployment.Namespace,
				Replicas:          totalCount,
				ReadyReplicas:     readyCount,
				UpdatedReplicas:   0,
				AvailableReplicas: 0,
				Age:               formatAgeFromString(deployment.CreatedAt),
			})
		}
		result = summaries
	case "Services":
		services, response, err := fetchKubernetesListItems[serviceListItem](ctx, client, args.ProjectID, "service", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("Services", response, err), nil
		}
		summaries := make([]ServiceSummary, 0, len(services))
		for _, service := range services {
			summaries = append(summaries, ServiceSummary{
				Name:      service.Name,
				Namespace: service.Namespace,
				Type:      service.Type,
				ClusterIP: service.ClusterIP,
				Ports:     nil,
				Age:       formatAgeFromString(service.CreatedAt),
			})
		}
		result = summaries
	case "Namespaces":
		namespaces, response, err := client.Client.KubernetesAPI.KubernetesNamespaceList(ctx, args.ProjectID).Execute()
		if err != nil {
			return createError(response, err), nil
		}
		var summaries []NamespaceSummary
		for _, name := range namespaces {
			if args.SearchTerm != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(args.SearchTerm)) {
				continue
			}
			summaries = append(summaries, NamespaceSummary{
				Name:   name,
				Status: "",
				Age:    "",
			})
		}
		if args.Offset > 0 || args.Limit > 0 {
			start := int(args.Offset)
			if start > len(summaries) {
				start = len(summaries)
			}
			end := len(summaries)
			if args.Limit > 0 && start+int(args.Limit) < end {
				end = start + int(args.Limit)
			}
			summaries = summaries[start:end]
		}
		result = summaries
	case "ConfigMaps":
		configMaps, response, err := fetchKubernetesListItems[configMapListItem](ctx, client, args.ProjectID, "configmap", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("ConfigMaps", response, err), nil
		}
		summaries := make([]ConfigMapSummary, 0, len(configMaps))
		for _, cm := range configMaps {
			summaries = append(summaries, ConfigMapSummary{
				Name:      cm.Name,
				Namespace: cm.Namespace,
				DataCount: 0,
				Age:       formatAgeFromString(cm.CreatedAt),
			})
		}
		result = summaries
	case "Secrets":
		secrets, response, err := fetchKubernetesListItems[secretListItem](ctx, client, args.ProjectID, "secret", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("Secrets", response, err), nil
		}
		summaries := make([]SecretSummary, 0, len(secrets))
		for _, secret := range secrets {
			summaries = append(summaries, SecretSummary{
				Name:      secret.Name,
				Namespace: secret.Namespace,
				Type:      secret.Type,
				DataCount: 0,
				Age:       formatAgeFromString(secret.CreatedAt),
			})
		}
		result = summaries
	case "Ingress":
		ingresses, response, err := fetchKubernetesListItems[ingressListItem](ctx, client, args.ProjectID, "ingress", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("Ingress", response, err), nil
		}
		summaries := make([]IngressSummary, 0, len(ingresses))
		for _, ingress := range ingresses {
			var hosts []string
			if ingress.Target != "" {
				hosts = []string{ingress.Target}
			}
			summaries = append(summaries, IngressSummary{
				Name:      ingress.Name,
				Namespace: ingress.Namespace,
				Hosts:     hosts,
				Address:   "",
				Age:       formatAgeFromString(ingress.CreatedAt),
			})
		}
		result = summaries
	case "CronJobs":
		return createJSONResponse(ErrorResponse{
			Error: "CronJobs listing is not available through the Cloudera Cloud Factory Kubernetes list API",
		}), nil
	case "DaemonSets":
		daemonSets, response, err := fetchKubernetesListItems[daemonSetListItem](ctx, client, args.ProjectID, "daemonset", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("DaemonSets", response, err), nil
		}
		summaries := make([]DaemonSetSummary, 0, len(daemonSets))
		for _, ds := range daemonSets {
			summaries = append(summaries, DaemonSetSummary{
				Name:             ds.Name,
				Namespace:        ds.Namespace,
				DesiredScheduled: ds.Desired,
				CurrentScheduled: ds.Current,
				NumberReady:      ds.Ready,
				NumberAvailable:  parseInt32(ds.Available),
				Age:              formatAgeFromString(ds.CreatedAt),
			})
		}
		result = summaries
	case "Jobs":
		return createJSONResponse(ErrorResponse{
			Error: "Jobs listing is not available through the Cloudera Cloud Factory Kubernetes list API",
		}), nil
	case "Nodes":
		nodes, response, err := fetchKubernetesListItems[nodeListItem](ctx, client, args.ProjectID, "nodes", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("Nodes", response, err), nil
		}
		summaries := make([]NodeSummary, 0, len(nodes))
		for _, node := range nodes {
			summaries = append(summaries, NodeSummary{
				Name:             node.Name,
				Status:           node.State,
				Roles:            node.Role,
				Version:          node.Version,
				OSImage:          "",
				KernelVersion:    "",
				ContainerRuntime: "",
				Age:              "",
			})
		}
		result = summaries
	case "Pvcs":
		pvcs, response, err := fetchKubernetesListItems[pvcListItem](ctx, client, args.ProjectID, "pvc", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("Pvcs", response, err), nil
		}
		summaries := make([]PvcSummary, 0, len(pvcs))
		for _, pvc := range pvcs {
			summaries = append(summaries, PvcSummary{
				Name:         pvc.Name,
				Namespace:    pvc.Namespace,
				Status:       pvc.Status,
				Volume:       pvc.Volume,
				Capacity:     pvc.Capacity,
				AccessModes:  pvc.AccessModes,
				StorageClass: pvc.StorageClass,
				Age:          formatAgeFromString(pvc.CreatedAt),
			})
		}
		result = summaries
	case "StorageClasses":
		return createJSONResponse(ErrorResponse{
			Error: "StorageClasses listing is not available through the Cloudera Cloud Factory Kubernetes list API",
		}), nil
	case "Sts":
		statefulSets, response, err := fetchKubernetesListItems[statefulSetListItem](ctx, client, args.ProjectID, "sts", args.Limit, args.Offset, args.SearchTerm)
		if err != nil {
			return listKubernetesError("Sts", response, err), nil
		}
		summaries := make([]StatefulSetSummary, 0, len(statefulSets))
		for _, sts := range statefulSets {
			readyCount, totalCount := parseReadyCounts(sts.Ready)
			summaries = append(summaries, StatefulSetSummary{
				Name:          sts.Name,
				Namespace:     sts.Namespace,
				Replicas:      totalCount,
				ReadyReplicas: readyCount,
				Age:           formatAgeFromString(sts.CreatedAt),
			})
		}
		result = summaries
	default:
		return createJSONResponse(ErrorResponse{
			Error: fmt.Sprintf("Unsupported resource kind: %s", args.Kind),
		}), nil
	}

	return createJSONResponse(result), nil
}

func describeKubernetesResource(client *taikungoclient.Client, args DescribeKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	kind, err := taikuncore.NewEKubernetesResourceFromValue(args.Kind)
	if err != nil {
		return createJSONResponse(ErrorResponse{Error: fmt.Sprintf("Invalid resource kind: %s", args.Kind)}), nil
	}

	describeCmd := taikuncore.NewDescribeKubernetesResourceCommand(args.ProjectID, args.Name, *kind)
	if args.Namespace != "" {
		describeCmd.SetNamespace(args.Namespace)
	}

	description, httpResponse, err := client.Client.KubernetesAPI.KubernetesDescribeResource(ctx).
		DescribeKubernetesResourceCommand(*describeCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, fmt.Sprintf("describe %s %s", args.Kind, args.Name)); errorResp != nil {
		return errorResp, nil
	}

	type DescribeResponse struct {
		YAML    string `json:"yaml"`
		Success bool   `json:"success"`
	}
	resp := DescribeResponse{
		YAML:    normalizeYamlOutput(description),
		Success: true,
	}
	return createJSONResponse(resp), nil
}

func deleteKubernetesResource(client *taikungoclient.Client, args DeleteKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	kind, err := taikuncore.NewEKubernetesResourceFromValue(args.Kind)
	if err != nil {
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Invalid resource kind: %s", args.Kind))), nil
	}

	// Create the action request with name and namespace
	actionRequest := taikuncore.NewKubernetesActionRequest(args.Name)
	if args.Namespace != "" {
		actionRequest.SetNamespace(args.Namespace)
	}

	// Create the delete command
	deleteCmd := taikuncore.NewDeleteKubernetesResourceCommand(args.ProjectID, *kind, []taikuncore.KubernetesActionRequest{*actionRequest})

	_, httpResponse, err := client.Client.KubernetesAPI.KubernetesDeleteResource(ctx).
		DeleteKubernetesResourceCommand(*deleteCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, fmt.Sprintf("delete %s %s", args.Kind, args.Name)); errorResp != nil {
		return errorResp, nil
	}

	namespace := args.Namespace
	if namespace == "" {
		namespace = "default"
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("%s '%s' deleted successfully from namespace '%s'", args.Kind, args.Name, namespace),
		Success: true,
	}

	return createJSONResponse(successResp), nil
}

func patchKubernetesResource(client *taikungoclient.Client, args PatchKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
	ctx := context.Background()

	normalizedYaml, err := normalizeYamlInput(args.Yaml)
	if err != nil {
		return createJSONResponse(ErrorResponse{Error: err.Error()}), nil
	}
	if err := validateKubernetesYaml(normalizedYaml); err != nil {
		return createJSONResponse(ErrorResponse{Error: err.Error()}), nil
	}

	encodedYaml := base64.StdEncoding.EncodeToString([]byte(normalizedYaml))
	patchCmd := taikuncore.NewPatchKubernetesResourceCommand(args.ProjectID, encodedYaml, args.Name)
	if args.Namespace != "" {
		patchCmd.SetNamespace(args.Namespace)
	}

	httpResponse, err := client.Client.KubernetesAPI.KubernetesPatchResource(ctx).
		PatchKubernetesResourceCommand(*patchCmd).
		Execute()

	if err != nil {
		return createError(httpResponse, err), nil
	}

	if errorResp := checkResponse(httpResponse, fmt.Sprintf("patch resource %s", args.Name)); errorResp != nil {
		return errorResp, nil
	}

	namespace := args.Namespace
	if namespace == "" {
		namespace = "default"
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("Resource '%s' patched successfully in namespace '%s'", args.Name, namespace),
		Success: true,
	}

	return createJSONResponse(successResp), nil
}
