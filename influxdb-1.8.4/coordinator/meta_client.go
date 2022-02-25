package coordinator

import (
	"time"

	"influxdb.cluster/services/influxql"
	"influxdb.cluster/services/meta"
)

// MetaClient is an interface for accessing meta data.
type MetaClient interface {
	CreateContinuousQuery(database, name, query string) error
	CreateDatabase(name string) (*meta.DatabaseInfo, error)
	CreateDatabaseWithRetentionPolicy(name string, rpi *meta.RetentionPolicyInfo) (*meta.DatabaseInfo, error)
	CreateRetentionPolicy(database string, rpi *meta.RetentionPolicyInfo) (*meta.RetentionPolicyInfo, error)
	CreateSubscription(database, rp, name, mode string, destinations []string) error
	CreateUser(name, password string, admin bool) (*meta.UserInfo, error)
	Database(name string) *meta.DatabaseInfo
	Databases() ([]meta.DatabaseInfo, error)
	DataNode(id uint64) (*meta.NodeInfo, error)
	DataNodes() ([]meta.NodeInfo, error)
	DeleteDataNode(id uint64) error
	MetaNodes() ([]meta.NodeInfo, error)
	DeleteMetaNode(id uint64) error
	DropShard(id uint64) error
	DropContinuousQuery(database, name string) error
	DropDatabase(name string) error
	DropRetentionPolicy(database, name string) error
	DropSubscription(database, rp, name string) error
	DropUser(name string) error
	RetentionPolicy(database, name string) (rpi *meta.RetentionPolicyInfo, err error)
	SetAdminPrivilege(username string, admin bool) error
	SetPrivilege(username, database string, p influxql.Privilege) error
	ShardsByTimeRange(sources influxql.Sources, tmin, tmax time.Time) (a []meta.ShardInfo, err error)
	SetDefaultRetentionPolicy(database, name string) error
	ShardGroupsByTimeRange(database, policy string, min, max time.Time) (a []meta.ShardGroupInfo, err error)
	TruncateShardGroups(t time.Time) error
	UpdateRetentionPolicy(database, name string, rpu *meta.RetentionPolicyUpdate) error
	UpdateUser(name, password string) error
	UserPrivilege(username, database string) (*influxql.Privilege, error)
	UserPrivileges(username string) (map[string]influxql.Privilege, error)
	Users() []meta.UserInfo
}
