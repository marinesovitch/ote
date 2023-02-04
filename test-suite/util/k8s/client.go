// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/marinesovitch/ote/test-suite/util/auxi"
	"github.com/marinesovitch/ote/test-suite/util/setup"
	"github.com/marinesovitch/ote/test-suite/util/system"
	"sigs.k8s.io/yaml"

	"gopkg.in/ini.v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	cfg       *setup.Configuration
	kubectl   Kubectl
	clientset *kubernetes.Clientset
	dynamic   dynamic.Interface
}

func NewClient(oteCfg *setup.Configuration, kubeCfg *rest.Config) (*Client, error) {
	kubectl := Kubectl{}

	clientset, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		return nil, err
	}

	dynamic, err := dynamic.NewForConfig(kubeCfg)
	if err != nil {
		return nil, err
	}

	return &Client{oteCfg, kubectl, clientset, dynamic}, nil
}

func (c *Client) ListConfigMaps(namespace string) (*corev1.ConfigMapList, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListDeployments(namespace string) (*appsv1.DeploymentList, error) {
	return c.clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListJobs(namespace string) (*batchv1.JobList, error) {
	return c.clientset.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListNamespaces() (*corev1.NamespaceList, error) {
	return c.clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListNodes() (*corev1.NodeList, error) {
	return c.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListPods(namespace string) (*corev1.PodList, error) {
	return c.clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListPodsWithFilter(namespace string, pattern string) (*corev1.PodList, error) {
	allPods, err := c.ListPods(namespace)
	if err != nil {
		return nil, err
	}

	rx, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	items := make([]corev1.Pod, 0)
	for _, pod := range allPods.Items {
		if rx.MatchString(pod.Name) {
			items = append(items, pod)
		}
	}

	filteredPods := corev1.PodList{
		TypeMeta: allPods.TypeMeta,
		ListMeta: allPods.ListMeta,
		Items:    items,
	}

	return &filteredPods, err
}

func (c *Client) ListPersistentVolumes(namespace string) (*corev1.PersistentVolumeList, error) {
	return c.clientset.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListPersistentVolumeClaims(namespace string) (*corev1.PersistentVolumeClaimList, error) {
	return c.clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListReplicaSets(namespace string) (*appsv1.ReplicaSetList, error) {
	return c.clientset.AppsV1().ReplicaSets(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListSecrets(namespace string) (*corev1.SecretList, error) {
	return c.clientset.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListServices(namespace string) (*corev1.ServiceList, error) {
	return c.clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListServiceAccounts(namespace string) (*corev1.ServiceAccountList, error) {
	return c.clientset.CoreV1().ServiceAccounts(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListStatefulSets(namespace string) (*appsv1.StatefulSetList, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListCustomResources(namespace string, resource Kind) (*unstructured.UnstructuredList, error) {
	gvr := schema.GroupVersionResource{
		Group:    OperatorGroup,
		Version:  OperatorVersion,
		Resource: resource.String(),
	}

	return c.dynamic.Resource(gvr).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
}

func (c *Client) ListInnoDBClusters(namespace string) (*unstructured.UnstructuredList, error) {
	return c.ListCustomResources(namespace, CRDInnoDBCluster)
}

func (c *Client) ListMySQLBackups(namespace string) (*unstructured.UnstructuredList, error) {
	return c.ListCustomResources(namespace, CRDMySQLBackup)
}

func (c *Client) ListEvents(namespace string, selector string, sinceResourceVersion string) (*corev1.EventList, error) {
	options := metav1.ListOptions{
		FieldSelector:   selector,
		ResourceVersion: sinceResourceVersion,
	}
	return c.clientset.CoreV1().Events(namespace).List(context.Background(), options)
}

func (c *Client) ListInnoDBClusterEvents(namespace string, icname string, sinceResourceVersion string) (*corev1.EventList, error) {
	selector := "involvedObject.kind=InnoDBCluster,involvedObject.name=" + icname
	return c.ListEvents(namespace, selector, sinceResourceVersion)
}

func (c *Client) HasDeployment(namespace string, name string) (bool, error) {
	deployment, err := c.GetDeployment(namespace, name)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return deployment != nil && deployment.GetName() == name, nil
}

func (c *Client) GetDeployment(namespace string, name string) (*appsv1.Deployment, error) {
	return c.clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) HasNamespace(name string) (bool, *corev1.Namespace, error) {
	if name == "" {
		return false, nil, nil
	}

	ns, err := c.GetNamespace(name)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return ns != nil && ns.GetName() == name, ns, nil
}

func (c *Client) GetNamespace(name string) (*corev1.Namespace, error) {
	return c.clientset.CoreV1().Namespaces().Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) HasPod(namespace string, name string) (bool, error) {
	pod, err := c.GetPod(namespace, name)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return pod != nil && pod.GetName() == name, nil
}

func (c *Client) GetPod(namespace string, name string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) GetPersistentVolume(namespace string, name string) (*corev1.PersistentVolume, error) {
	return c.clientset.CoreV1().PersistentVolumes().Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) GetPersistentVolumeClaim(namespace string, name string) (*corev1.PersistentVolumeClaim, error) {
	return c.clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) HasReplicaSet(namespace string, name string) (bool, error) {
	rs, err := c.GetReplicaSet(namespace, name)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return rs != nil && rs.GetName() == name, nil
}

func (c *Client) GetReplicaSet(namespace string, name string) (*appsv1.ReplicaSet, error) {
	return c.clientset.AppsV1().ReplicaSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) GetService(namespace string, name string) (*corev1.Service, error) {
	return c.clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) GetStatefulSet(namespace string, name string) (*appsv1.StatefulSet, error) {
	return c.clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) getCustomResource(namespace string, name string, resource Kind) (*unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group:    OperatorGroup,
		Version:  OperatorVersion,
		Resource: resource.String(),
	}

	return c.dynamic.Resource(gvr).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *Client) HasInnoDBCluster(namespace string, name string) (bool, error) {
	ic, err := c.GetInnoDBCluster(namespace, name)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return ic.GetName() == name, nil
}

func (c *Client) GetInnoDBCluster(namespace string, name string) (*InnoDBCluster, error) {
	idbc, err := c.getCustomResource(namespace, name, CRDInnoDBCluster)
	return &InnoDBCluster{Unstructured: idbc, CRDFields: NewCRDFields(idbc)}, err
}

func (c *Client) GetMySQLBackup(namespace string, name string) (*MySQLBackup, error) {
	mbk, err := c.getCustomResource(namespace, name, CRDMySQLBackup)
	return &MySQLBackup{Unstructured: mbk, CRDFields: NewCRDFields(mbk)}, err
}

type deleterFunc func(ctx context.Context, namespace string, name string) error

func (c *Client) deleteItem(namespace string, name string, deleter deleterFunc, timeout time.Duration) error {
	// Pass a context with a timeout to tell a blocking function that it
	// should abandon its work after the timeout elapses.
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()
	completed := make(chan error)
	go func() {
		completed <- deleter(ctx, namespace, name)
	}()

	select {
	case err := <-completed:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) DeleteConfigMap(namespace string, name string) error {
	const Timeout = 5
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteDeployment(namespace string, name string) error {
	const Timeout = 30
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteJob(namespace string, name string) error {
	const Timeout = 30
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteNamespace(name string) error {
	const Timeout = 300
	return c.deleteItem(
		"",
		name,
		func(ctx context.Context, _ string, name string) error {
			return c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeletePodWithTimeout(namespace string, name string, timeout time.Duration) error {
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		timeout,
	)
}

func (c *Client) DeletePod(namespace string, name string) error {
	const Timeout = 120
	return c.DeletePodWithTimeout(namespace, name, Timeout)
}

func (c *Client) DeletePersistentVolume(namespace string, name string) error {
	const Timeout = 60
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeletePersistentVolumeClaim(namespace string, name string) error {
	const Timeout = 60
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteReplicaSet(namespace string, name string) error {
	const Timeout = 30
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.AppsV1().ReplicaSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteSecret(namespace string, name string) error {
	const Timeout = 5
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteService(namespace string, name string) error {
	const Timeout = 5
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteServiceAccount(namespace string, name string) error {
	const Timeout = 5
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteStatefulSet(namespace string, name string) error {
	const Timeout = 30
	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		Timeout,
	)
}

func (c *Client) DeleteCustomResource(namespace string, resource Kind, name string, timeout time.Duration) error {
	gvr := schema.GroupVersionResource{
		Group:    OperatorGroup,
		Version:  OperatorVersion,
		Resource: resource.String(),
	}

	return c.deleteItem(
		namespace,
		name,
		func(ctx context.Context, namespace string, name string) error {
			return c.dynamic.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		},
		timeout,
	)
}

func (c *Client) DeleteInnoDBCluster(namespace string, name string) error {
	const Timeout = 200
	return c.DeleteCustomResource(namespace, CRDInnoDBCluster, name, Timeout)
}

func (c *Client) DeleteMySQLBackup(namespace string, name string) error {
	const Timeout = 200
	return c.DeleteCustomResource(namespace, CRDMySQLBackup, name, Timeout)
}

func (c *Client) describe(namespace string, resource Kind, name string) (string, error) {
	return c.kubectl.Describe(namespace, resource, name)
}

func (c *Client) DescribeInnoDBCluster(namespace string, name string) (string, error) {
	return c.describe(namespace, CRDInnoDBCluster, name)
}

func (c *Client) Logs(namespace string, name string, containerId ContainerId) (string, error) {
	return c.kubectl.Logs(namespace, name, containerId)
}

func (c *Client) Cat(namespace string, name string, containerId ContainerId, path string) (string, error) {
	return c.kubectl.ExecuteGetOutput(namespace, name, containerId, "cat", path)
}

func (c *Client) Kill(namespace string, name string, containerId ContainerId, sig int, pid int) (err error) {
	killCmd := fmt.Sprintf("kill -%d %d", sig, pid)
	const MaxTrials = 5
	for i := 0; i < MaxTrials; i++ {
		err = c.Execute(namespace, name, containerId, "/bin/sh", "-c", killCmd)
		if err == nil {
			break
		}
		err = fmt.Errorf("cannot kill container %s on %s/%s: %v", GetContainerName(containerId), namespace, name, err)
		log.Print(err)
		time.Sleep(2 * time.Second)
	}
	return
}

func (c *Client) Execute(namespace string, name string, containerId ContainerId, cmd ...string) error {
	return c.kubectl.Execute(namespace, name, containerId, cmd...)
}

func (c *Client) Apply(namespace string, path string) error {
	return c.kubectl.ApplyInNamespace(namespace, path)
}

func (c *Client) ApplyGetOutput(namespace string, path string) (string, error) {
	return c.kubectl.ApplyGetOutput(namespace, path)
}

// https://erosb.github.io/post/json-patch-vs-merge-patch/
type patchPath struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

type jsonPatchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

type jsonPatchIntValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value int    `json:"value"`
}

type JsonPatchOperation string

const PatchRemove JsonPatchOperation = "remove"
const PatchReplace JsonPatchOperation = "replace"

type JsonPatch struct {
	Operation JsonPatchOperation
	Path      string
	Value     interface{}
}

func prepareJsonPatchRemovePayload(patch JsonPatch) ([]byte, error) {
	payload := []patchPath{{
		Op:   string(patch.Operation),
		Path: patch.Path,
	}}
	return json.Marshal(payload)
}

func prepareJsonPatchReplacePayload(patch JsonPatch) ([]byte, error) {
	switch v := patch.Value.(type) {
	case string:
		payload := []jsonPatchStringValue{{
			Op:    string(patch.Operation),
			Path:  patch.Path,
			Value: v,
		}}
		return json.Marshal(payload)
	case int:
		payload := []jsonPatchIntValue{{
			Op:    string(patch.Operation),
			Path:  patch.Path,
			Value: v,
		}}
		return json.Marshal(payload)
	default:
		return nil, errors.New("unsupported patch value")
	}
}

func (c *Client) prepareJsonPatchPayload(patch JsonPatch) ([]byte, error) {
	switch patch.Operation {
	case PatchRemove:
		return prepareJsonPatchRemovePayload(patch)
	case PatchReplace:
		return prepareJsonPatchReplacePayload(patch)
	default:
		return nil, errors.New("unsupported patch operation: " + string(patch.Operation))
	}
}

func (c *Client) patchCustomResource(namespace string, name string, resource Kind, patchType types.PatchType, payload []byte) error {
	gvr := schema.GroupVersionResource{
		Group:    OperatorGroup,
		Version:  OperatorVersion,
		Resource: resource.String(),
	}
	_, err := c.dynamic.Resource(gvr).Namespace(namespace).
		Patch(context.Background(), name, patchType, payload, metav1.PatchOptions{})
	return err
}

func (c *Client) JSONPatchInnoDBCluster(namespace string, name string, patch JsonPatch) error {
	// https://dwmkerr.com/patching-kubernetes-resources-in-golang/
	// https://erosb.github.io/post/json-patch-vs-merge-patch/
	payload, err := c.prepareJsonPatchPayload(patch)
	if err != nil {
		return err
	}

	return c.patchCustomResource(namespace, name, CRDInnoDBCluster, types.JSONPatchType, payload)
}

func (c *Client) MergePatchInnoDBCluster(namespace string, name string, patch []byte) error {
	return c.patchCustomResource(namespace, name, CRDInnoDBCluster, types.MergePatchType, patch)
}

func (c *Client) MergePatchInnoDBClusterFromFile(namespace string, name string, patchYamlPath string) error {
	patchYaml, err := os.ReadFile(patchYamlPath)
	if err != nil {
		return err
	}

	patch, err := yaml.YAMLToJSON(patchYaml)
	if err != nil {
		return err
	}

	return c.patchCustomResource(namespace, name, CRDInnoDBCluster, types.MergePatchType, patch)
}

func (c *Client) PatchPod(namespace string, name string, patch JsonPatch) error {
	payload, err := c.prepareJsonPatchPayload(patch)
	if err != nil {
		return err
	}

	_, err = c.clientset.CoreV1().Pods(namespace).Patch(context.Background(), name, types.JSONPatchType, payload, metav1.PatchOptions{})
	return err
}

func (c *Client) WaitOnPodGone(namespace string, name string, sinceResourceVersion string, timeout time.Duration) error {
	if podExists, err := c.HasPod(namespace, name); !podExists || err != nil {
		return err
	}

	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(context.Background(), metav1.ListOptions{ResourceVersion: sinceResourceVersion})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	podGone := make(chan bool)
	go func() {
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Deleted:
				pod := event.Object.(*corev1.Pod)
				if pod.GetNamespace() == namespace && pod.GetName() == name {
					podGone <- true
					return
				}
			}
		}
	}()

	select {
	case <-podGone:
		return nil
	case <-time.After(timeout * time.Second):
		return fmt.Errorf("timeout waiting for pod %s/%s gone", namespace, name)
	}
}

func (c *Client) WaitOnPod(namespace string, name string, sinceResourceVersion string, status corev1.PodPhase, timeout time.Duration) error {
	watcher, err := c.clientset.CoreV1().Pods(namespace).Watch(context.Background(), metav1.ListOptions{ResourceVersion: sinceResourceVersion})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	podOk := make(chan bool)
	go func() {
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				pod := event.Object.(*corev1.Pod)
				if pod.GetNamespace() == namespace && pod.GetName() == name && pod.Status.Phase == status {
					podOk <- true
					return
				}
			}
		}
	}()

	select {
	case <-podOk:
		return nil
	case <-time.After(timeout * time.Second):
		return fmt.Errorf("timeout waiting for pod %s/%s", namespace, name)
	}
}

func getInnoDBClusterGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    OperatorGroup,
		Version:  OperatorVersion,
		Resource: OperatorInnoDBClusters,
	}
}

func (c *Client) WaitOnInnoDBClusterGone(namespace string, name string, sinceResourceVersion string, timeout time.Duration) error {
	if icExists, err := c.HasInnoDBCluster(namespace, name); !icExists || err != nil {
		return err
	}

	gvr := getInnoDBClusterGVR()
	watcher, err := c.dynamic.Resource(gvr).Namespace(namespace).Watch(context.Background(), metav1.ListOptions{ResourceVersion: sinceResourceVersion})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	icGone := make(chan bool)
	go func() {
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Deleted:
				ic := event.Object.(*unstructured.Unstructured)
				if ic.GetNamespace() == namespace && ic.GetName() == name {
					icGone <- true
					return
				}
			}
		}
	}()

	select {
	case <-icGone:
		return nil
	case <-time.After(timeout * time.Second):
		return fmt.Errorf("timeout waiting for ic %s/%s gone", namespace, name)
	}
}

func verifyInnoDBClusterState(ic *unstructured.Unstructured, expectedStatus []string, expectedNumOnline int64) bool {
	icStatus, found, err := unstructured.NestedString(ic.Object, "status", "cluster", "status")
	statusOk := err == nil && found && auxi.Contains(expectedStatus, icStatus)
	if !statusOk {
		return false
	}

	if expectedNumOnline == -1 {
		return true
	}

	icNumOnline, found, err := unstructured.NestedInt64(ic.Object, "status", "cluster", "onlineInstances")
	return err == nil && found && icNumOnline >= expectedNumOnline
}

type WaitOnInnoDBClusterParams struct {
	Namespace            string
	Name                 string
	ExpectedStatus       []string
	ExpectedNumOnline    int64
	SinceResourceVersion string
	Timeout              time.Duration
}

func (c *Client) WaitOnInnoDBCluster(params WaitOnInnoDBClusterParams) error {
	gvr := getInnoDBClusterGVR()
	watcher, err := c.dynamic.Resource(gvr).Namespace(params.Namespace).Watch(context.Background(), metav1.ListOptions{ResourceVersion: params.SinceResourceVersion})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	icOk := make(chan bool)
	go func() {
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				ic := event.Object.(*unstructured.Unstructured)
				if ic.GetNamespace() == params.Namespace && ic.GetName() == params.Name &&
					verifyInnoDBClusterState(ic, params.ExpectedStatus, params.ExpectedNumOnline) {
					icOk <- true
					return
				}
			}
		}
	}()

	select {
	case <-icOk:
		return nil
	case <-time.After(params.Timeout * time.Second):
		return fmt.Errorf("timeout waiting for ic %s/%s", params.Namespace, params.Name)
	}
}

func (c *Client) CreateNamespace(name string) error {
	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err := c.clientset.CoreV1().Namespaces().Create(context.Background(), nsSpec, metav1.CreateOptions{})
	return err
}

func (c *Client) CreateUserSecrets(namespace string, name string, rootUser string, rootHost string, rootPass string) error {
	data := setup.GenerateUserSecretsData{
		Name:         name,
		RootUser:     auxi.B64encode(rootUser),
		RootHost:     auxi.B64encode(rootHost),
		RootPassword: auxi.B64encode(rootPass),
	}
	userSecretsYamlPath, err := setup.GenerateUserSecrets(c.cfg, &data)
	if err != nil {
		return err
	}
	return c.Apply(namespace, userSecretsYamlPath)
}

func adjustKeyFilePath(cfgPath string, cfgKeyFilePath string) (string, error) {
	cfgKeyFilePath = system.PathExpand(cfgKeyFilePath)
	if filepath.IsAbs(cfgKeyFilePath) {
		return cfgKeyFilePath, nil
	}

	// kubectl doesn't like relative paths
	cfgDir := filepath.Dir(cfgPath)
	keyFilePath := filepath.Join(cfgDir, cfgKeyFilePath)

	if !system.DoesFileExist(keyFilePath) {
		return keyFilePath, fmt.Errorf("key file %s doesn't exist", keyFilePath)
	}

	return keyFilePath, nil
}

func (c *Client) CreateApikeySecret(namespace string, name string, cfgPath string, profileName string) error {
	var err error

	if !filepath.IsAbs(cfgPath) {
		cfgPath, err = filepath.Abs(cfgPath)
		if err != nil {
			return err
		}
	}

	err = system.VerifyFileExist(cfgPath)
	if err != nil {
		return err
	}

	iniCfg, err := ini.Load(cfgPath)
	if err != nil {
		return err
	}

	profile, err := iniCfg.GetSection(profileName)
	if err != nil {
		return err
	}

	const KeyFileOption = "key_file"
	args := []string{"create", "secret", "generic", name, "-n", namespace}

	for _, key := range profile.Keys() {
		optionName := key.Name()
		optionValue := key.Value()
		if optionName != KeyFileOption {
			args = append(args, fmt.Sprintf("--from-literal=%s=%s", optionName, optionValue))
		} else {
			privateKeyPath, err := adjustKeyFilePath(cfgPath, optionValue)
			if err != nil {
				return err
			}
			args = append(args, "--from-file=privatekey="+privateKeyPath)
		}
	}

	return c.kubectl.Run(args...)
}
