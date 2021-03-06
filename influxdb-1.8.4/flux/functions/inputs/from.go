package inputs

import (
	"strings"

	"influxdb.cluster/services/flux/execute"
	"influxdb.cluster/services/flux/functions/inputs"
	"influxdb.cluster/services/flux/plan"
	"influxdb.cluster/services/meta"
	"influxdb.cluster/services/influxql"
	"influxdb.cluster/platform/query/functions/inputs/storage"
	"github.com/pkg/errors"
)

func init() {
	execute.RegisterSource(inputs.FromKind, createFromSource)
}

func createFromSource(prSpec plan.ProcedureSpec, dsid execute.DatasetID, a execute.Administration) (execute.Source, error) {
	spec := prSpec.(*inputs.FromProcedureSpec)
	var w execute.Window
	bounds := a.StreamContext().Bounds()
	if bounds == nil {
		return nil, errors.New("nil bounds passed to from")
	}

	if spec.WindowSet {
		w = execute.Window{
			Every:  execute.Duration(spec.Window.Every),
			Period: execute.Duration(spec.Window.Period),
			Round:  execute.Duration(spec.Window.Round),
			Start:  bounds.Start,
		}
	} else {
		duration := execute.Duration(bounds.Stop) - execute.Duration(bounds.Start)
		w = execute.Window{
			Every:  duration,
			Period: duration,
			Start:  bounds.Start,
		}
	}
	currentTime := w.Start + execute.Time(w.Period)

	deps := a.Dependencies()[inputs.FromKind].(Dependencies)

	var db, rp string
	if i := strings.IndexByte(spec.Bucket, '/'); i == -1 {
		db = spec.Bucket
	} else {
		rp = spec.Bucket[i+1:]
		db = spec.Bucket[:i]
	}

	// validate and resolve db/rp
	di := deps.MetaClient.Database(db)
	if di == nil {
		return nil, errors.New("no database")
	}

	if deps.AuthEnabled {
		user := meta.UserFromContext(a.Context())
		if user == nil {
			return nil, errors.New("createFromSource: no user")
		}
		if err := deps.Authorizer.AuthorizeDatabase(user, influxql.ReadPrivilege, db); err != nil {
			return nil, err
		}
	}

	if rp == "" {
		rp = di.DefaultRetentionPolicy
	}

	if rpi := di.RetentionPolicy(rp); rpi == nil {
		return nil, errors.New("invalid retention policy")
	}

	return storage.NewSource(
		dsid,
		deps.Reader,
		storage.ReadSpec{
			Database:        db,
			RetentionPolicy: rp,
			Predicate:       spec.Filter,
			PointsLimit:     spec.PointsLimit,
			SeriesLimit:     spec.SeriesLimit,
			SeriesOffset:    spec.SeriesOffset,
			Descending:      spec.Descending,
			OrderByTime:     spec.OrderByTime,
			GroupMode:       storage.ToGroupMode(spec.GroupMode),
			GroupKeys:       spec.GroupKeys,
			AggregateMethod: spec.AggregateMethod,
		},
		*bounds,
		w,
		currentTime,
	), nil
}

type Authorizer interface {
	AuthorizeDatabase(u meta.User, priv influxql.Privilege, database string) error
}

type Dependencies struct {
	Reader      storage.Reader
	MetaClient  MetaClient
	Authorizer  Authorizer
	AuthEnabled bool
}

func (d Dependencies) Validate() error {
	if d.Reader == nil {
		return errors.New("missing reader dependency")
	}
	if d.AuthEnabled && d.Authorizer == nil {
		return errors.New("validate Dependencies: missing Authorizer")
	}
	return nil
}

func InjectFromDependencies(depsMap execute.Dependencies, deps Dependencies) error {
	if err := deps.Validate(); err != nil {
		return err
	}
	depsMap[inputs.FromKind] = deps
	return nil
}
