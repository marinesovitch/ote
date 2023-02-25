# The golang test suite of MySQL Operator for Kubernetes

## Introduction

The project was meant to become a part of [the MySQL Operator for Kubernetes](https://github.com/mysql/mysql-operator) as a complementary counterpart of [the test suite implemented in Python](https://github.com/mysql/mysql-operator/tree/trunk/tests) which relies on kubectl to call k8s commands. While this project is a test suite written in golang and relies on the [native k8s client](https://github.com/kubernetes/client-go/).

In many parts, especially in test cases, it was ported from the python test suite almost one-to-one. Hence it may look neither very 'golangish' nor especially neat. It was planned to change and refactor it in the future but it ain't gonna happen as it is not actively maintained anymore. The golang-test-suite was rejected and abandoned, ultimately. Anyway, I still think, it was worth publishing it on github as the code I wrote in golang :-)

## Sources

There are two subdirectories in the project:
* [test-suite](test-suite/) - the golang test suite of MySQL Operator for Kubernetes - the actual project
* mysql-operator - the MySQL Operator for Kubernetes as the only dependency (git submodule)

### Test suite structure

The project is divided into the following parts:
* [ote-cli](test-suite/ote.go) (ote stands for Operator Test Environment) \
	It is an auxiliary command-line tool to set up a k8s cluster on a local dev machine or in the CI infrastructure. Its use is optional, one can set up a cluster independently.
* [e2e test suite](test-suite/e2e/) \
	Set of e2e test cases covering various functionalities of the MySQL Operator for Kubernetes.
* [utilities](test-suite/util/) \
	Set of various utilities used by both ote-cli and e2e test suite.
* [templates](test-suite/template/) \
	Templates to generate some YAMLs for deploying operator. Also with help of [k8s kustomization](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/).
* config files
	* [default.cfg](test-suite/default.cfg)
	* [custom.cfg.sample](test-suite/custom.cfg.sample)
	* [config.oci.sample](test-suite/config.oci.sample)\
	More details in section [How to config](#how-to-config).
* temporary files\
	By default, all temporary files (e.g. generated YAMLs) will be stored in the subdirectory `ote/out/` - it may be changed in configuration (option `testsuite.outputDirectory`).

### Dependencies

The only dependency is [the MySQL Operator for Kubernetes](https://github.com/mysql/mysql-operator). And it is expected to be located under the subdirectory [mysql-operator](mysql-operator/). Actually, only [deploy yamls](https://github.com/mysql/mysql-operator/tree/trunk/deploy) are needed.

This project was tested against [version 8.0.31-2.0.7](https://github.com/mysql/mysql-operator/releases/tag/8.0.31-2.0.7).

The dependency was added as [a git submodule](https://git-scm.com/book/en/v2/Git-Tools-Submodules). To initialize it, call the folowing command:
```sh
git submodule update --init
```

Optionally, download the specified version, and unpack it to a mentioned subdirectory (so that the deploy yamls are located under `mysql-operator/deploy/*.yaml`).

## How to build

To build `ote-cli` use the following command:
```sh
go build -v -gcflags='all=-N -l' -o ote
```

## How to config

In brief: `default.cfg < environment variables < custom.cfg < command line`

All settings are listed in [default.cfg](test-suite/default.cfg). They can be overridden with environment variables. In turn, they can be overridden with [custom.cfg](test-suite/custom.cfg.sample). And the highest priority has parameters passed in the command line. More details in the following sections.

### default.cfg

The file [default.cfg](test-suite/default.cfg) contains all settings with default values. Some of them are used only by ote-cli, others by e2e test suite, and some by both. Do not modify this file - it should stay read-only. To override default values use environment variables, custom.cfg, or command-line options.

### environment variables

The following environment variables are supported:
* OPERATOR_TEST_REGISTRY
* OPERATOR_TEST_REPOSITORY
* OPERATOR_TEST_PULL_POLICY
* OPERATOR_TEST_IMAGE_NAME
* OPERATOR_TEST_EE_IMAGE_NAME
* OPERATOR_TEST_VERSION_TAG
* OPERATOR_TEST_PULL_POLICY
* OPERATOR_TEST_ENABLE_ENTERPRISE
* OPERATOR_TEST_ENABLE_OCI
* OPERATOR_TEST_OCI_CONFIG_PATH
* OPERATOR_TEST_OCI_BUCKET
* OPERATOR_TEST_K8S_CLUSTER_NAME

Based on the environment variable name it is easy to find a corresponding setting in [default.cfg](test-suite/default.cfg). If set, they will override default.cfg values.

### custom.cfg

To set custom values, create `custom.cfg` file (it will be ignored by git) in the same directory where [default.cfg](test-suite/default.cfg). Then copy from `default.cfg` the settings which you want to override, and assign the desired values. They will override default.cfg and environment variables.\
See a [custom.cfg.sample](test-suite/custom.cfg.sample) how it may look like. Please note that the mentioned example file is not considered at set up due to extension `.sample`. It is in the project only for demonstration purposes.

### command line

Options set with the command line will override all aforementioned mechanisms (default.cfg, envars, custom.cfg). Below is the list of all command-line options supported by `ote-cli`:

```
marines@ubuntu-ds2:~/ote/test-suite$ ./ote --help
Usage of ./ote:
  -cluster-name string
    	cluster name used for testing (default "ote-mysql")
  -data-dir string
    	directory with e2e data (default "../mysql-operator/tests/data")
  -dbg int
    	debug level (default 1)
  -e2e-dir string
    	directory with e2e tests (default "./e2e")
  -engine string
    	container engine [detect|docker|podman] (default "detect")
  -enterprise
    	run enterprise tests
  -env string
    	environment [detect|k3d|minikube] (default "detect")
  -k3d-registry-cfg string
    	path to k3d registry config yaml or its template (default "./template/k3d-registry-config.yaml")
  -kubecfg string
    	kube config path (if 'detect' it first tries ${KUBECONFIG}, then path ~/.kube/config) (default "detect")
  -minikube-registry-insecure
    	is minikube registry insecure (default true)
  -oci
    	run OCI tests
  -oci-bucket-name string
    	OCI bucket name
  -oci-cfg-path string
    	path to a file with OCI profiles
  -operator-dir string
    	operator directory (default "../mysql-operator/deploy")
  -operator-image string
    	image custom config path (default "mysql-operator")
  -operator-image-ee string
    	enterprise edition image custom config path (default "enterprise-operator")
  -operator-pull-policy string
    	pull policy for operator [Always|IfNotPresent|Never] (default "IfNotPresent")
  -operator-tag string
    	version tag for operator image (default "8.0.31-2.0.7")
  -operator-template string
    	path to operator deploy yaml or its template (default "./template/deploy-operator.yaml")
  -operator-yamls string
    	operator yamls (default "deploy-crds.yaml:deploy-operator.yaml")
  -output-dir string
    	output directory for log and tmp files (default "../out")
  -pull-policy string
    	pull policy [Always|IfNotPresent|Never] (default "IfNotPresent")
  -registry string
    	registry, e.g. registry.localhost:5000
  -repository string
    	repository, e.g. qa (default "mysql")
  -skip-delete
    	skip deleting cluster
  -skip-deploy
    	skip deploying operator
Command [start|stop|deploy]
```

### oci

By default, the [OCI (Oracle Cloud Infrastructure)](https://www.oracle.com/cloud/) tests are skipped.

To run tests against OCI one needs to enable them explicitly with any of the following options:
* set the envar `OPERATOR_TEST_ENABLE_OCI`
* `oci.Enable` field in custom.cfg set to `True`
* add the command-line option `-oci`

Besides, one needs a `config.oci` file. It should be defined according to the structure of [config.oci.sample](test-suite/config.oci.sample). Then it can be passed with any of the following options:
* envar `OPERATOR_TEST_OCI_CONFIG_PATH`
* `oci.configPath` field in custom.cfg
* as argument of the command-line option `-oci-cfg-path`

And the same for the OCI bucket name - it should be defined through any of the alternatives:
* envar `OPERATOR_TEST_OCI_BUCKET`
* `oci.bucketName` field in custom.cfg
* as argument of the command-line option `-oci-bucket-name`

### enterprise

By default, the Enterprise tests are skipped.

To run tests against the enterprise edition one needs to enable them explicitly with any of the following options:
* set the envar `OPERATOR_TEST_ENABLE_ENTERPRISE`
* `enterprise.Enable` field in custom.cfg set to `True`
* add the command-line option `-enterprise`

Besides, the related enterprise images are needed.

## How to run

### ote-cli

Check the list of [command line options](#command-line). The following commands are supported:
* start\
	it starts a (k3d or minikube) cluster and deploys the operator unless the `-skip-deploy` option was applied
* stop\
	it stops the current cluster
* deploy\
	it only deploys the MySQL Operator for Kubernetes

### e2e test suite

Test cases are run the same way as ordinary golang tests. Some cases may run quite a long time hence it is recommended to use the `-timeout 30m` option. The cases are expected to run consecutively hence option `-p 1` should be used.

To run tests go to the [test-suite](test-suite/) directory, then one can execute any command like these below:

To run all tests:
```sh
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/...
```

To run chosen subgroups (coarse-grained), e.g.:
```sh
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/backup/...
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/config/...
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/enterprise/...
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/init_db/...
```

For more fine-grained, e.g.:
```sh
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/config -run=TwoClustersOneNamespace
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/config -run=Cluster1Defaults
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/upgrade -run=UpgradeToNext
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/badspec -run=ClusterSpecAdmissionChecks
```

Even more fine-grained, e.g.:
```sh
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/badspec -run=ClusterSpecAdmissionChecks/NameTooLong
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/badspec -run=ClusterSpecRuntimeChecksCreation/UnsupportedVersionDelete
```

The golang regex can also be used to select a case or group of cases, e.g.:
```sh
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/config -run='Cluster\d.*'
go test -p 1 -timeout 30m -v github.com/marinesovitch/ote/test-suite/e2e/cluster/config -run='Cluster\D.*'
```

### Compatibility

The test suite was tested against the following versions:
* k3d v5.4.4 + kubectl v1.24.4 ([click here to see a sample log](https://github.com/marinesovitch/media/blob/trunk/ote/k3d.log))
* minikube v1.25.2 + kubectl v1.24.4 ([click here to see a sample log](https://github.com/marinesovitch/media/blob/trunk/ote/minikube.log))

As the project is not maintained anymore and considering that everything in the k8s world changes rapidly, the results obtained for another version may differ.
