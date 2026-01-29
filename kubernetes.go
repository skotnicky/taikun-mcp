package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/itera-io/taikungoclient"
	taikuncore "github.com/itera-io/taikungoclient/client"
	mcp_golang "github.com/metoro-io/mcp-golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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

type DeleteKubernetesResourceArgs struct {
	ProjectID int32  `json:"projectId" jsonschema:"required,description=The project ID of the resource"`
	Kind      string `json:"kind" jsonschema:"required,description=The kind of the resource (e.g., Pod, Deployment, Service)"`
	Name      string `json:"name" jsonschema:"required,description=The name of the resource to delete"`
	Namespace string `json:"namespace,omitempty" jsonschema:"description=The namespace of the resource (optional, defaults to 'default')"`
}

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
	Namespace string `json:"namespace,omitempty" jsonschema:"description=The namespace of the resource (optional, defaults to 'default')"`
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

func getKubernetesClientset(client *taikungoclient.Client, projectID int32) (*kubernetes.Clientset, error) {
	ctx := context.Background()
	kubeconfig, httpResponse, err := client.Client.KubernetesAPI.KubernetesKubeConfig(ctx, projectID).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %v", err)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get kubeconfig: HTTP %d", httpResponse.StatusCode)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig.GetData()))
	if err != nil {
		return nil, fmt.Errorf("failed to create REST config: %v", err)
	}

	// Set a timeout for the kubernetes client testing
	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	return clientset, nil
}

func listKubernetesResources(client *taikungoclient.Client, args ListKubernetesResourcesArgs) (*mcp_golang.ToolResponse, error) {
	clientset, err := getKubernetesClientset(client, args.ProjectID)
	if err != nil {
		errorResp := ErrorResponse{
			Error:   fmt.Sprintf("Failed to initialize Kubernetes client: %v", err),
			Details: "Make sure the project has a valid kubeconfig and cluster is accessible",
		}
		return createJSONResponse(errorResp), nil
	}

	ctx := context.Background()
	var result interface{}

	switch args.Kind {
	case "Pods":
		pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}

		var podSummaries []PodSummary
		for _, pod := range pods.Items {
			var restarts int32
			for _, containerStatus := range pod.Status.ContainerStatuses {
				restarts += containerStatus.RestartCount
			}
			for _, containerStatus := range pod.Status.InitContainerStatuses {
				restarts += containerStatus.RestartCount
			}
			for _, containerStatus := range pod.Status.EphemeralContainerStatuses {
				restarts += containerStatus.RestartCount
			}

			startTime := ""
			if pod.Status.StartTime != nil {
				startTime = pod.Status.StartTime.Format(time.RFC3339)
			}

			podSummaries = append(podSummaries, PodSummary{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    string(pod.Status.Phase),
				PodIP:     pod.Status.PodIP,
				StartTime: startTime,
				Restarts:  restarts,
			})
		}
		result = podSummaries
	case "Deployments":
		deployments, err := clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []DeploymentSummary
		for _, d := range deployments.Items {
			summaries = append(summaries, DeploymentSummary{
				Name:              d.Name,
				Namespace:         d.Namespace,
				Replicas:          *d.Spec.Replicas,
				ReadyReplicas:     d.Status.ReadyReplicas,
				UpdatedReplicas:   d.Status.UpdatedReplicas,
				AvailableReplicas: d.Status.AvailableReplicas,
				Age:               formatAge(d.CreationTimestamp),
			})
		}
		result = summaries
	case "Services":
		services, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []ServiceSummary
		for _, s := range services.Items {
			var ports []string
			for _, p := range s.Spec.Ports {
				ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
			}
			summaries = append(summaries, ServiceSummary{
				Name:      s.Name,
				Namespace: s.Namespace,
				Type:      string(s.Spec.Type),
				ClusterIP: s.Spec.ClusterIP,
				Ports:     ports,
				Age:       formatAge(s.CreationTimestamp),
			})
		}
		result = summaries
	case "Namespaces":
		namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []NamespaceSummary
		for _, ns := range namespaces.Items {
			summaries = append(summaries, NamespaceSummary{
				Name:   ns.Name,
				Status: string(ns.Status.Phase),
				Age:    formatAge(ns.CreationTimestamp),
			})
		}
		result = summaries
	case "ConfigMaps":
		configMaps, err := clientset.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []ConfigMapSummary
		for _, cm := range configMaps.Items {
			summaries = append(summaries, ConfigMapSummary{
				Name:      cm.Name,
				Namespace: cm.Namespace,
				DataCount: len(cm.Data),
				Age:       formatAge(cm.CreationTimestamp),
			})
		}
		result = summaries
	case "Secrets":
		secrets, err := clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []SecretSummary
		for _, s := range secrets.Items {
			summaries = append(summaries, SecretSummary{
				Name:      s.Name,
				Namespace: s.Namespace,
				Type:      string(s.Type),
				DataCount: len(s.Data),
				Age:       formatAge(s.CreationTimestamp),
			})
		}
		result = summaries
	case "Ingress":
		ingresses, err := clientset.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []IngressSummary
		for _, ing := range ingresses.Items {
			var hosts []string
			for _, rule := range ing.Spec.Rules {
				hosts = append(hosts, rule.Host)
			}
			address := ""
			if len(ing.Status.LoadBalancer.Ingress) > 0 {
				address = ing.Status.LoadBalancer.Ingress[0].IP
			}
			summaries = append(summaries, IngressSummary{
				Name:      ing.Name,
				Namespace: ing.Namespace,
				Hosts:     hosts,
				Address:   address,
				Age:       formatAge(ing.CreationTimestamp),
			})
		}
		result = summaries
	case "CronJobs":
		cronJobs, err := clientset.BatchV1().CronJobs("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []CronJobSummary
		for _, cj := range cronJobs.Items {
			suspend := false
			if cj.Spec.Suspend != nil {
				suspend = *cj.Spec.Suspend
			}
			lastSchedule := ""
			if cj.Status.LastScheduleTime != nil {
				lastSchedule = formatAge(*cj.Status.LastScheduleTime)
			}
			summaries = append(summaries, CronJobSummary{
				Name:             cj.Name,
				Namespace:        cj.Namespace,
				Schedule:         cj.Spec.Schedule,
				Suspend:          suspend,
				Active:           len(cj.Status.Active),
				LastScheduleTime: lastSchedule,
				Age:              formatAge(cj.CreationTimestamp),
			})
		}
		result = summaries
	case "DaemonSets":
		daemonSets, err := clientset.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []DaemonSetSummary
		for _, ds := range daemonSets.Items {
			summaries = append(summaries, DaemonSetSummary{
				Name:             ds.Name,
				Namespace:        ds.Namespace,
				DesiredScheduled: ds.Status.DesiredNumberScheduled,
				CurrentScheduled: ds.Status.CurrentNumberScheduled,
				NumberReady:      ds.Status.NumberReady,
				NumberAvailable:  ds.Status.NumberAvailable,
				Age:              formatAge(ds.CreationTimestamp),
			})
		}
		result = summaries
	case "Jobs":
		jobs, err := clientset.BatchV1().Jobs("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []JobSummary
		for _, j := range jobs.Items {
			completions := fmt.Sprintf("%d/%d", j.Status.Succeeded, *j.Spec.Completions)
			if j.Spec.Completions == nil {
				completions = fmt.Sprintf("%d/<nil>", j.Status.Succeeded)
			}
			summaries = append(summaries, JobSummary{
				Name:        j.Name,
				Namespace:   j.Namespace,
				Completions: completions,
				Succeeded:   j.Status.Succeeded,
				Age:         formatAge(j.CreationTimestamp),
			})
		}
		result = summaries
	case "Nodes":
		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []NodeSummary
		for _, n := range nodes.Items {
			status := "Unknown"
			for _, cond := range n.Status.Conditions {
				if cond.Type == "Ready" {
					if cond.Status == "True" {
						status = "Ready"
					} else {
						status = "NotReady"
					}
					break
				}
			}

			// Simple role detection
			roles := "<none>"
			if _, ok := n.Labels["node-role.kubernetes.io/control-plane"]; ok {
				roles = "control-plane"
			} else if _, ok := n.Labels["node-role.kubernetes.io/master"]; ok {
				roles = "control-plane"
			} else if _, ok := n.Labels["node-role.kubernetes.io/worker"]; ok {
				roles = "worker"
			}

			summaries = append(summaries, NodeSummary{
				Name:             n.Name,
				Status:           status,
				Roles:            roles,
				Version:          n.Status.NodeInfo.KubeletVersion,
				OSImage:          n.Status.NodeInfo.OSImage,
				KernelVersion:    n.Status.NodeInfo.KernelVersion,
				ContainerRuntime: n.Status.NodeInfo.ContainerRuntimeVersion,
				Age:              formatAge(n.CreationTimestamp),
			})
		}
		result = summaries
	case "Pvcs":
		pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []PvcSummary
		for _, pvc := range pvcs.Items {
			capacity := ""
			if q, ok := pvc.Status.Capacity["storage"]; ok {
				capacity = q.String()
			}
			storageClass := ""
			if pvc.Spec.StorageClassName != nil {
				storageClass = *pvc.Spec.StorageClassName
			}
			modes := ""
			for _, m := range pvc.Spec.AccessModes {
				if modes != "" {
					modes += ","
				}
				modes += string(m)
			}
			summaries = append(summaries, PvcSummary{
				Name:         pvc.Name,
				Namespace:    pvc.Namespace,
				Status:       string(pvc.Status.Phase),
				Volume:       pvc.Spec.VolumeName,
				Capacity:     capacity,
				AccessModes:  modes,
				StorageClass: storageClass,
				Age:          formatAge(pvc.CreationTimestamp),
			})
		}
		result = summaries
	case "StorageClasses":
		storageClasses, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []StorageClassSummary
		for _, sc := range storageClasses.Items {
			reclaimPolicy := ""
			if sc.ReclaimPolicy != nil {
				reclaimPolicy = string(*sc.ReclaimPolicy)
			}
			summaries = append(summaries, StorageClassSummary{
				Name:          sc.Name,
				Provisioner:   sc.Provisioner,
				ReclaimPolicy: reclaimPolicy,
				Age:           formatAge(sc.CreationTimestamp),
			})
		}
		result = summaries
	case "Sts":
		statefulSets, err := clientset.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		var summaries []StatefulSetSummary
		for _, sts := range statefulSets.Items {
			summaries = append(summaries, StatefulSetSummary{
				Name:          sts.Name,
				Namespace:     sts.Namespace,
				Replicas:      *sts.Spec.Replicas,
				ReadyReplicas: sts.Status.ReadyReplicas,
				Age:           formatAge(sts.CreationTimestamp),
			})
		}
		result = summaries
	default:
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unsupported resource kind for project-scoped listing: %s. Please use one of: Pods, Deployments, Services, Namespaces, ConfigMaps, Secrets, Ingress, CronJobs, DaemonSets, Jobs, Nodes, Pvcs, StorageClasses, Sts.", args.Kind))), nil
	}

	return createJSONResponse(result), nil
}

func describeKubernetesResource(client *taikungoclient.Client, args DescribeKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
	clientset, err := getKubernetesClientset(client, args.ProjectID)
	if err != nil {
		errorResp := ErrorResponse{
			Error:   fmt.Sprintf("Failed to initialize Kubernetes client: %v", err),
			Details: "Make sure the project has a valid kubeconfig and cluster is accessible",
		}
		return createJSONResponse(errorResp), nil
	}

	ctx := context.Background()
	var result string
	namespace := args.Namespace
	if namespace == "" {
		namespace = "default"
	}

	switch args.Kind {
	case "Pod":
		resource, err := clientset.CoreV1().Pods(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "Deployment":
		resource, err := clientset.AppsV1().Deployments(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "Service":
		resource, err := clientset.CoreV1().Services(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "ConfigMap":
		resource, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "Secret":
		resource, err := clientset.CoreV1().Secrets(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "Ingress":
		resource, err := clientset.NetworkingV1().Ingresses(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "CronJob":
		resource, err := clientset.BatchV1().CronJobs(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "DaemonSet":
		resource, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "Job":
		resource, err := clientset.BatchV1().Jobs(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "Pvc":
		resource, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "StatefulSet":
		resource, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "Namespace":
		resource, err := clientset.CoreV1().Namespaces().Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	case "Node":
		resource, err := clientset.CoreV1().Nodes().Get(ctx, args.Name, metav1.GetOptions{})
		if err != nil {
			return createError(nil, err), nil
		}
		yamlData, _ := json.MarshalIndent(resource, "", "  ")
		result = string(yamlData)
	default:
		// Fallback to original describe API if kind not handled yet by clientset
		return originalDescribeKubernetesResource(client, args)
	}

	return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(result)), nil
}

func originalDescribeKubernetesResource(client *taikungoclient.Client, args DescribeKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
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

func deleteKubernetesResource(client *taikungoclient.Client, args DeleteKubernetesResourceArgs) (*mcp_golang.ToolResponse, error) {
	clientset, err := getKubernetesClientset(client, args.ProjectID)
	if err != nil {
		errorResp := ErrorResponse{
			Error:   fmt.Sprintf("Failed to initialize Kubernetes client: %v", err),
			Details: "Make sure the project has a valid kubeconfig and cluster is accessible",
		}
		return createJSONResponse(errorResp), nil
	}

	ctx := context.Background()
	namespace := args.Namespace
	if namespace == "" {
		namespace = "default"
	}

	switch args.Kind {
	case "Pod":
		err = clientset.CoreV1().Pods(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "Deployment":
		err = clientset.AppsV1().Deployments(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "Service":
		err = clientset.CoreV1().Services(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "ConfigMap":
		err = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "Secret":
		err = clientset.CoreV1().Secrets(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "Ingress":
		err = clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "CronJob":
		err = clientset.BatchV1().CronJobs(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "DaemonSet":
		err = clientset.AppsV1().DaemonSets(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "Job":
		err = clientset.BatchV1().Jobs(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "Pvc":
		err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "StatefulSet":
		err = clientset.AppsV1().StatefulSets(namespace).Delete(ctx, args.Name, metav1.DeleteOptions{})
	case "Namespace":
		err = clientset.CoreV1().Namespaces().Delete(ctx, args.Name, metav1.DeleteOptions{})
	default:
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unsupported resource kind for deletion: %s", args.Kind))), nil
	}

	if err != nil {
		return createError(nil, err), nil
	}

	successResp := SuccessResponse{
		Message: fmt.Sprintf("%s '%s' deleted successfully from namespace '%s'", args.Kind, args.Name, namespace),
		Success: true,
	}

	return createJSONResponse(successResp), nil
}
