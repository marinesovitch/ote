package k8s

type Kind int

const OperatorGroup = "mysql.oracle.com"
const OperatorVersion = "v2"
const OperatorInnoDBClusters = "innodbclusters"

const (
	ConfigMap Kind = iota
	CRDInnoDBCluster
	CRDMySQLBackup
	Deploy
	Job
	PersistentVolumeClaim
	PersistentVolume
	Pod
	ReplicaSet
	Secret
	ServiceAccount
	Service
	StatefulSet
)

func (r Kind) String() string {
	switch r {
	case ConfigMap:
		return "configmaps"
	case CRDInnoDBCluster:
		return "innodbclusters"
	case CRDMySQLBackup:
		return "mysqlbackups"
	case Deploy:
		return "deployments"
	case Job:
		return "job"
	case PersistentVolumeClaim:
		return "persistentvolumeclaims"
	case Pod:
		return "pods"
	case ReplicaSet:
		return "replicasets"
	case Secret:
		return "secret"
	case ServiceAccount:
		return "serviceaccounts"
	case Service:
		return "services"
	case StatefulSet:
		return "statefulsets"
	default:
		return "shouldn't happen - unknown"
	}
}

const MBKStatusCompleted string = "Completed"
