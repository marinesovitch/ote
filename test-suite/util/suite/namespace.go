// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package suite

import (
	"bytes"
	"strings"
	"time"

	"github.com/marinesovitch/ote/test-suite/util/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type collectPendingItems struct {
	client       *k8s.Client
	namespace    string
	pendingItems bytes.Buffer
}

func (v *collectPendingItems) run() (string, error) {
	err := v.collectPendingItems()
	if err != nil {
		return "", err
	}

	return v.pendingItems.String(), nil
}

func (v *collectPendingItems) collectPendingItems() error {
	var err error

	configMaps, err := v.client.ListConfigMaps(v.namespace)
	if err != nil {
		return err
	}
	if len(configMaps.Items) > 0 {
		v.addSection("config maps")
		for _, cm := range configMaps.Items {
			v.addItem(&cm.ObjectMeta)
		}
	}

	deployments, err := v.client.ListDeployments(v.namespace)
	if err != nil {
		return err
	}
	if len(deployments.Items) > 0 {
		v.addSection("deployments")
		for _, deployment := range deployments.Items {
			v.addItem(&deployment.ObjectMeta)
		}
	}

	jobs, err := v.client.ListJobs(v.namespace)
	if err != nil {
		return err
	}
	if len(jobs.Items) > 0 {
		v.addSection("jobs")
		for _, job := range jobs.Items {
			v.addItem(&job.ObjectMeta)
		}
	}

	pvcs, err := v.client.ListPersistentVolumeClaims(v.namespace)
	if err != nil {
		return err
	}
	if len(pvcs.Items) > 0 {
		v.addSection("persistent volume claims")
		for _, pvc := range pvcs.Items {
			v.addItem(&pvc.ObjectMeta)
		}
	}

	pods, err := v.client.ListPods(v.namespace)
	if err != nil {
		return err
	}
	if len(pods.Items) > 0 {
		v.addSection("pods")
		for _, pod := range pods.Items {
			v.addItem(&pod.ObjectMeta)
		}
	}

	replicaSets, err := v.client.ListReplicaSets(v.namespace)
	if err != nil {
		return err
	}
	if len(replicaSets.Items) > 0 {
		v.addSection("replicaSets")
		for _, replicaSet := range replicaSets.Items {
			v.addItem(&replicaSet.ObjectMeta)
		}
	}

	secrets, err := v.filterPendingSecrets()
	if err != nil {
		return err
	}
	if len(secrets) > 0 {
		v.addSection("secrets")
		for _, secret := range secrets {
			v.addItem(&secret.ObjectMeta)
		}
	}

	services, err := v.client.ListServices(v.namespace)
	if err != nil {
		return err
	}
	if len(services.Items) > 0 {
		v.addSection("services")
		for _, service := range services.Items {
			v.addItem(&service.ObjectMeta)
		}
	}

	serviceAccounts, err := v.client.ListServiceAccounts(v.namespace)
	if err != nil {
		return err
	}
	if len(serviceAccounts.Items) > 0 {
		v.addSection("serviceAccounts")
		for _, serviceAccount := range serviceAccounts.Items {
			v.addItem(&serviceAccount.ObjectMeta)
		}
	}

	statefulSets, err := v.client.ListStatefulSets(v.namespace)
	if err != nil {
		return err
	}
	if len(statefulSets.Items) > 0 {
		v.addSection("statefulSets")
		for _, statefulSet := range statefulSets.Items {
			v.addItem(&statefulSet.ObjectMeta)
		}
	}

	err = v.verifyCustomResources(k8s.CRDInnoDBCluster)
	if err != nil {
		return err
	}

	err = v.verifyCustomResources(k8s.CRDMySQLBackup)
	if err != nil {
		return err
	}

	nsExists, ns, err := v.client.HasNamespace(v.namespace)
	if err != nil {
		return err
	}
	if nsExists {
		v.addSection("namespace")
		v.addItem(&ns.ObjectMeta)
	}

	return err
}

func (v *collectPendingItems) verifyCustomResources(resource k8s.Kind) error {
	customResources, err := v.client.ListCustomResources(v.namespace, resource)
	if err != nil {
		return err
	}
	if len(customResources.Items) > 0 {
		v.addSection(resource.String())
		for _, item := range customResources.Items {
			v.addCustomResourceItem(&item)
		}
	}
	return nil
}

func (v *collectPendingItems) isPendingSecret(secret *corev1.Secret) bool {
	name := secret.GetName()
	const IgnoredSecretPrefix = "default-token-"
	return !strings.HasPrefix(name, IgnoredSecretPrefix)
}

func (v *collectPendingItems) filterPendingSecrets() ([]*corev1.Secret, error) {
	var pendingSecrets []*corev1.Secret
	secrets, err := v.client.ListSecrets(v.namespace)
	if err != nil {
		return pendingSecrets, err
	}
	for _, secret := range secrets.Items {
		if v.isPendingSecret(&secret) {
			pendingSecrets = append(pendingSecrets, &secret)
		}
	}
	return pendingSecrets, nil
}

func (v *collectPendingItems) addSection(name string) {
	v.pendingItems.WriteString(name + ":\n")
}

func (v *collectPendingItems) addItem(objMeta *metav1.ObjectMeta) {
	v.pendingItems.WriteString(" - " + objMeta.Name + "\n")
}

func (v *collectPendingItems) addCustomResourceItem(item *unstructured.Unstructured) {
	v.pendingItems.WriteString(" - " + item.GetName() + "\n")
}

// ------------------------------------------------------------------------

type wipeNamespace struct {
	client    *k8s.Client
	namespace string
}

func (w *wipeNamespace) run() error {
	// delete order: ic, mbk, po, sts, rs, svc, cm, secret, jobs, deploy, pvc, sa
	var err error
	err = w.deleteCustomResources(k8s.CRDInnoDBCluster)
	if err != nil {
		return err
	}

	pods, err := w.client.ListPods(w.namespace)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		err = stripFinalizers(w.client, w.namespace, k8s.Pod, pod.GetName())
		if err != nil {
			return err
		}
		err = w.client.DeletePod(w.namespace, pod.GetName())
		if err != nil {
			return err
		}
	}

	err = w.deleteCustomResources(k8s.CRDMySQLBackup)
	if err != nil {
		return err
	}

	statefulSets, err := w.client.ListStatefulSets(w.namespace)
	if err != nil {
		return err
	}
	if len(statefulSets.Items) > 0 {
		for _, statefulSet := range statefulSets.Items {
			err = w.client.DeleteStatefulSet(w.namespace, statefulSet.GetName())
			if err != nil {
				return err
			}
		}
	}

	replicaSets, err := w.client.ListReplicaSets(w.namespace)
	if err != nil {
		return err
	}
	if len(replicaSets.Items) > 0 {
		for _, replicaSet := range replicaSets.Items {
			err = w.client.DeleteReplicaSet(w.namespace, replicaSet.GetName())
			if err != nil {
				return err
			}
		}
	}

	services, err := w.client.ListServices(w.namespace)
	if err != nil {
		return err
	}
	if len(services.Items) > 0 {
		for _, service := range services.Items {
			err = w.client.DeleteService(w.namespace, service.GetName())
			if err != nil {
				return err
			}
		}
	}

	configMaps, err := w.client.ListConfigMaps(w.namespace)
	if err != nil {
		return err
	}
	if len(configMaps.Items) > 0 {
		for _, cm := range configMaps.Items {
			err = w.client.DeleteConfigMap(w.namespace, cm.GetName())
			if err != nil {
				return err
			}
		}
	}

	secrets, err := w.client.ListSecrets(w.namespace)
	if err != nil {
		return err
	}
	if len(secrets.Items) > 0 {
		for _, secret := range secrets.Items {
			err = w.client.DeleteSecret(w.namespace, secret.GetName())
			if err != nil {
				return err
			}
		}
	}

	jobs, err := w.client.ListJobs(w.namespace)
	if err != nil {
		return err
	}
	if len(jobs.Items) > 0 {
		for _, job := range jobs.Items {
			err = w.client.DeleteJob(w.namespace, job.GetName())
			if err != nil {
				return err
			}
		}
	}

	deployments, err := w.client.ListDeployments(w.namespace)
	if err != nil {
		return err
	}
	if len(deployments.Items) > 0 {
		for _, deployment := range deployments.Items {
			err = w.client.DeleteDeployment(w.namespace, deployment.GetName())
			if err != nil {
				return err
			}
		}
	}

	pvcs, err := w.client.ListPersistentVolumeClaims(w.namespace)
	if err != nil {
		return err
	}
	if len(pvcs.Items) > 0 {
		for _, pvc := range pvcs.Items {
			err = w.client.DeletePersistentVolumeClaim(w.namespace, pvc.GetName())
			if err != nil {
				return err
			}
		}
	}

	pvs, err := w.client.ListPersistentVolumes(w.namespace)
	if err != nil {
		return err
	}
	if len(pvs.Items) > 0 {
		for _, pv := range pvs.Items {
			err = w.client.DeletePersistentVolume(w.namespace, pv.GetName())
			if err != nil {
				return err
			}
		}
	}

	serviceAccounts, err := w.client.ListServiceAccounts(w.namespace)
	if err != nil {
		return err
	}
	if len(serviceAccounts.Items) > 0 {
		for _, serviceAccount := range serviceAccounts.Items {
			err = w.client.DeleteServiceAccount(w.namespace, serviceAccount.GetName())
			if err != nil {
				return err
			}
		}
	}

	return w.client.DeleteNamespace(w.namespace)
}

func (w *wipeNamespace) deleteCustomResources(kind k8s.Kind) error {
	customResources, err := w.client.ListCustomResources(w.namespace, kind)
	if err != nil {
		return err
	}

	const Timeout time.Duration = 90
	for _, item := range customResources.Items {
		name := item.GetName()

		if kind == k8s.CRDInnoDBCluster {
			stripFinalizers(w.client, w.namespace, kind, name)
		}

		err = w.client.DeleteCustomResource(w.namespace, kind, name, Timeout)
		if err != nil && !k8s.IsNotFoundError(err) {
			return err
		}
	}
	return nil
}
