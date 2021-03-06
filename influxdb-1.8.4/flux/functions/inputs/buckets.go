package inputs

import (
	"fmt"

	"influxdb.cluster/services/flux"
	"influxdb.cluster/services/flux/execute"
	"influxdb.cluster/services/flux/functions/inputs"
	"influxdb.cluster/services/flux/memory"
	"influxdb.cluster/services/flux/plan"
	"influxdb.cluster/services/flux/values"
	"influxdb.cluster/services/influxql"
	"influxdb.cluster/services/meta"
	"github.com/pkg/errors"
)

func init() {
	execute.RegisterSource(inputs.BucketsKind, createBucketsSource)
}

type BucketsDecoder struct {
	deps  BucketDependencies
	alloc *memory.Allocator
	user  meta.User
}

func (bd *BucketsDecoder) Connect() error {
	return nil
}

func (bd *BucketsDecoder) Fetch() (bool, error) {
	return false, nil
}

func (bd *BucketsDecoder) Decode() (flux.Table, error) {
	kb := execute.NewGroupKeyBuilder(nil)
	kb.AddKeyValue("organizationID", values.NewString("freetsdb"))
	gk, err := kb.Build()
	if err != nil {
		return nil, err
	}

	b := execute.NewColListTableBuilder(gk, bd.alloc)

	b.AddCol(flux.ColMeta{
		Label: "name",
		Type:  flux.TString,
	})
	b.AddCol(flux.ColMeta{
		Label: "id",
		Type:  flux.TString,
	})
	b.AddCol(flux.ColMeta{
		Label: "organization",
		Type:  flux.TString,
	})
	b.AddCol(flux.ColMeta{
		Label: "organizationID",
		Type:  flux.TString,
	})
	b.AddCol(flux.ColMeta{
		Label: "retentionPolicy",
		Type:  flux.TString,
	})
	b.AddCol(flux.ColMeta{
		Label: "retentionPeriod",
		Type:  flux.TInt,
	})

	var hasAccess func(db string) bool
	if bd.user == nil {
		hasAccess = func(db string) bool {
			return true
		}
	} else {
		hasAccess = func(db string) bool {
			return bd.deps.Authorizer.AuthorizeDatabase(bd.user, influxql.ReadPrivilege, db) == nil ||
				bd.deps.Authorizer.AuthorizeDatabase(bd.user, influxql.WritePrivilege, db) == nil
		}
	}

	dbis, _ := bd.deps.MetaClient.Databases()
	for _, bucket := range dbis {
		if hasAccess(bucket.Name) {
			rp := bucket.RetentionPolicy(bucket.DefaultRetentionPolicy)
			b.AppendString(0, bucket.Name)
			b.AppendString(1, "")
			b.AppendString(2, "freetsdb")
			b.AppendString(3, "")
			b.AppendString(4, rp.Name)
			b.AppendInt(5, rp.Duration.Nanoseconds())
		}
	}

	return b.Table()
}

func createBucketsSource(prSpec plan.ProcedureSpec, dsid execute.DatasetID, a execute.Administration) (execute.Source, error) {
	_, ok := prSpec.(*inputs.BucketsProcedureSpec)
	if !ok {
		return nil, fmt.Errorf("invalid spec type %T", prSpec)
	}

	// the dependencies used for FromKind are adequate for what we need here
	// so there's no need to inject custom dependencies for buckets()
	deps := a.Dependencies()[inputs.BucketsKind].(BucketDependencies)

	var user meta.User
	if deps.AuthEnabled {
		user = meta.UserFromContext(a.Context())
		if user == nil {
			return nil, errors.New("createBucketsSource: no user")
		}
	}

	bd := &BucketsDecoder{deps: deps, alloc: a.Allocator(), user: user}

	return inputs.CreateSourceFromDecoder(bd, dsid, a)

}

type MetaClient interface {
	Databases() ([]meta.DatabaseInfo, error)
	Database(name string) *meta.DatabaseInfo
}

type BucketDependencies struct {
	MetaClient  MetaClient
	Authorizer  Authorizer
	AuthEnabled bool
}

func (d BucketDependencies) Validate() error {
	if d.MetaClient == nil {
		return errors.New("validate BucketDependencies: missing MetaClient")
	}
	if d.AuthEnabled && d.Authorizer == nil {
		return errors.New("validate BucketDependencies: missing Authorizer")
	}
	return nil
}

func InjectBucketDependencies(depsMap execute.Dependencies, deps BucketDependencies) error {
	if err := deps.Validate(); err != nil {
		return err
	}
	depsMap[inputs.BucketsKind] = deps
	return nil
}
