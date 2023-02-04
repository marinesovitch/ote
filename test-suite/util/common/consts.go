// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package common

const OperatorNamespace = "mysql-operator"

const RootUser = "root"
const RootPassword = "sakila"

const AdminUser = "admin"
const AdminPassword = "secret"

const DefaultHost = "%"

const SchemaSeparator = "://"

var DefaultMysqlAccounts = []string{"mysql.infoschema@localhost", "mysql.session@localhost", "mysql.sys@localhost"}

var MarkExists = struct{}{}

const OciProfileBackup = "BACKUP"
const OciProfileRestore = "RESTORE"
const OciProfileDelete = "DELETE"

const EnvMinikube = "minikube"
const EnvK3d = "k3d"

const AnyResourceVersion = ""
