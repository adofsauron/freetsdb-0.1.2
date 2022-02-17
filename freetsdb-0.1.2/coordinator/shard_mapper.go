package coordinator

import (
	"context"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/freetsdb/freetsdb/query"
	"github.com/freetsdb/freetsdb/services/influxql"
	"github.com/freetsdb/freetsdb/tsdb"
)

// IteratorCreator is an interface that combines mapping fields and creating iterators.
type IteratorCreator interface {
	query.IteratorCreator
	influxql.FieldMapper
	io.Closer
}

// LocalShardMapper implements a ShardMapper for local shards.
type LocalShardMapper struct {
	//MetaClient interface {
	//	ShardGroupsByTimeRange(database, policy string, min, max time.Time) (a []meta.ShardGroupInfo, err error)
	//}
	MetaClient MetaClient

	TSDBStore interface {
		ShardGroup(ids []uint64) tsdb.ShardGroup
		Shards(ids []uint64) []*tsdb.Shard
		CreateShard(database, retentionPolicy string, shardID uint64, enabled bool) error
	}
}

// MapShards maps the sources to the appropriate shards into an IteratorCreator.
func (e *LocalShardMapper) MapShards(sources influxql.Sources, t influxql.TimeRange, opt query.SelectOptions) (query.ShardGroup, error) {
	a := &LocalShardMapping{
		ShardMap:  make(map[Source]tsdb.ShardGroup),
		RemoteICs: make(map[Source][]remoteIteratorCreator),
	}

	tmin := time.Unix(0, t.MinTimeNano())
	tmax := time.Unix(0, t.MaxTimeNano())
	a.MinTime, a.MaxTime = tmin, tmax
	a.LocalNodeID = opt.NodeID
	if err := e.mapShards(a, sources, tmin, tmax); err != nil {
		return nil, err
	}

	return a, nil
}

func (e *LocalShardMapper) mapShards(a *LocalShardMapping, sources influxql.Sources, tmin, tmax time.Time) error {
	for _, s := range sources {
		switch s := s.(type) {
		case *influxql.Measurement:
			source := Source{
				Database:        s.Database,
				RetentionPolicy: s.RetentionPolicy,
			}
			// Retrieve the list of shards for this database. This list of
			// shards is always the same regardless of which measurement we are
			// using.
			if _, ok := a.ShardMap[source]; !ok {
				groups, err := e.MetaClient.ShardGroupsByTimeRange(s.Database, s.RetentionPolicy, tmin, tmax)
				if err != nil {
					return err
				}

				if len(groups) == 0 {
					a.ShardMap[source] = nil
					continue
				}
				a.RemoteICs[source] = make([]remoteIteratorCreator, 0, len(groups[0].Shards)*len(groups))

				shardIDs := make([]uint64, 0, len(groups[0].Shards)*len(groups))
				for _, g := range groups {
					for _, si := range g.Shards {
						var nodeID uint64
						if si.OwnedBy(a.LocalNodeID) {
							nodeID = a.LocalNodeID
						} else if len(si.Owners) > 0 {
							nodeID = si.Owners[rand.Intn(len(si.Owners))].NodeID
						} else {
							// This should not occur but if the shard has no owners then
							// we don't want this to panic by trying to randomly select a node.
							continue
						}
						if nodeID == a.LocalNodeID {
							shardIDs = append(shardIDs, si.ID)

						} else {
							dialer := &NodeDialer{
								MetaClient: e.MetaClient,
								Timeout:    time.Duration(3 * time.Second),
							}
							remoteShardIDs := []uint64{si.ID}
							remoteIC := newRemoteIteratorCreator(dialer, nodeID, remoteShardIDs)
							a.RemoteICs[source] = append(a.RemoteICs[source], remoteIC)

						}
					}
				}
				shards := e.TSDBStore.Shards(shardIDs)
				if len(shards) != len(shardIDs) {
					for _, shardID := range shardIDs {
						//err = e.TSDBStore.CreateShard(database, retentionPolicy, shardID, true)
						e.TSDBStore.CreateShard(s.Database, s.RetentionPolicy, shardID, true)
					}

				}
				a.ShardMap[source] = e.TSDBStore.ShardGroup(shardIDs)
			}
		case *influxql.SubQuery:
			if err := e.mapShards(a, s.Statement.Sources, tmin, tmax); err != nil {
				return err
			}
		}
	}
	return nil
}

// ShardMapper maps data sources to a list of shard information.
type LocalShardMapping struct {
	ShardMap map[Source]tsdb.ShardGroup

	RemoteICs map[Source][]remoteIteratorCreator

	// MinTime is the minimum time that this shard mapper will allow.
	// Any attempt to use a time before this one will automatically result in using
	// this time instead.
	MinTime time.Time

	// MaxTime is the maximum time that this shard mapper will allow.
	// Any attempt to use a time after this one will automatically result in using
	// this time instead.
	MaxTime time.Time

	LocalNodeID uint64
}

func (a *LocalShardMapping) FieldDimensions(m *influxql.Measurement) (fields map[string]influxql.DataType, dimensions map[string]struct{}, err error) {
	source := Source{
		Database:        m.Database,
		RetentionPolicy: m.RetentionPolicy,
	}

	sg := a.ShardMap[source]
	RemoteICs := a.RemoteICs[source]
	if sg == nil && RemoteICs == nil {
		return nil, nil, nil
	}

	fields = make(map[string]influxql.DataType)
	dimensions = make(map[string]struct{})

	var measurements []string
	if m.Regex != nil {
		measurements = sg.MeasurementsByRegex(m.Regex.Val)
	} else {
		measurements = []string{m.Name}
	}

	if sg != nil {
		f, d, err := sg.FieldDimensions(measurements)
		if err != nil {
			return nil, nil, err
		}
		for k, typ := range f {
			fields[k] = typ
		}
		for k := range d {
			dimensions[k] = struct{}{}
		}
	}

	if RemoteICs != nil {
		for _, remoteIC := range RemoteICs {
			f, d, err := remoteIC.FieldDimensions(m)
			if err != nil {
				return nil, nil, err
			}
			for k, typ := range f {
				fields[k] = typ
			}
			for k := range d {
				dimensions[k] = struct{}{}
			}
		}
	}

	return
}

func (a *LocalShardMapping) MapType(m *influxql.Measurement, field string) influxql.DataType {
	source := Source{
		Database:        m.Database,
		RetentionPolicy: m.RetentionPolicy,
	}

	sg := a.ShardMap[source]
	if sg == nil {
		return influxql.Unknown
	}

	var names []string
	if m.Regex != nil {
		names = sg.MeasurementsByRegex(m.Regex.Val)
	} else {
		names = []string{m.Name}
	}

	var typ influxql.DataType
	for _, name := range names {
		if m.SystemIterator != "" {
			name = m.SystemIterator
		}
		t := sg.MapType(name, field)
		if typ.LessThan(t) {
			typ = t
		}
	}

	return typ
}

// ParseTags returns an instance of Tags for a comma-delimited list of key/values.
func ParseTags(s string) query.Tags {
	m := make(map[string]string)
	for _, kv := range strings.Split(s, ",") {
		a := strings.Split(kv, "=")
		m[a[0]] = a[1]
	}
	return query.NewTags(m)
}

func (a *LocalShardMapping) CreateIterator(ctx context.Context, m *influxql.Measurement, opt query.IteratorOptions) (query.Iterator, error) {
	source := Source{
		Database:        m.Database,
		RetentionPolicy: m.RetentionPolicy,
	}

	sg := a.ShardMap[source]
	RemoteICs := a.RemoteICs[source]
	if sg == nil && RemoteICs == nil {
		return nil, nil
	}

	// Override the time constraints if they don't match each other.
	if !a.MinTime.IsZero() && opt.StartTime < a.MinTime.UnixNano() {
		opt.StartTime = a.MinTime.UnixNano()
	}
	if !a.MaxTime.IsZero() && opt.EndTime > a.MaxTime.UnixNano() {
		opt.EndTime = a.MaxTime.UnixNano()
	}

	inputs := []query.Iterator{}
	if m.Regex != nil {
		measurements := sg.MeasurementsByRegex(m.Regex.Val)
		if err := func() error {
			// Create a Measurement for each returned matching measurement value
			// from the regex.
			for _, measurement := range measurements {
				mm := m.Clone()
				mm.Name = measurement // Set the name to this matching regex value.
				input, err := sg.CreateIterator(ctx, mm, opt)
				if err != nil {
					return err
				}
				inputs = append(inputs, input)

				for _, remoteIC := range RemoteICs {
					input, err := remoteIC.CreateIterator(ctx, m, opt)
					if err != nil {
						continue
					}
					inputs = append(inputs, input)
				}
			}
			return nil
		}(); err != nil {
			query.Iterators(inputs).Close()
			return nil, err
		}

	} else {

		input, err := sg.CreateIterator(ctx, m, opt)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)

		for _, remoteIC := range RemoteICs {
			input, err := remoteIC.CreateIterator(ctx, m, opt)
			if err != nil {
				continue
			}
			inputs = append(inputs, input)
		}
	}

	return query.Iterators(inputs).Merge(opt)
}

// remoteIteratorCreator creates iterators for remote shards.
type remoteIteratorCreator struct {
	dialer   *NodeDialer
	nodeID   uint64
	shardIDs []uint64
}

// newRemoteIteratorCreator returns a new instance of remoteIteratorCreator for a remote shard.
func newRemoteIteratorCreator(dialer *NodeDialer, nodeID uint64, shardIDs []uint64) remoteIteratorCreator {
	return remoteIteratorCreator{
		dialer:   dialer,
		nodeID:   nodeID,
		shardIDs: shardIDs,
	}
}

// CreateIterator creates a remote streaming iterator.
func (ic *remoteIteratorCreator) CreateIterator(ctx context.Context, m *influxql.Measurement, opt query.IteratorOptions) (query.Iterator, error) {
	conn, err := ic.dialer.DialNode(ic.nodeID)
	if err != nil {
		return nil, err
	}

	var resp CreateIteratorResponse
	if err := func() error {
		// Write request.
		var req = CreateIteratorRequest{
			ShardIDs:    ic.shardIDs,
			Measurement: *(m.Clone()),
			Opt:         opt,
		}
		if err := EncodeTLV(conn, createIteratorRequestMessage, &req); err != nil {
			return err
		}

		// Read the response.
		if _, err := DecodeTLV(conn, &resp); err != nil {
			return err
		} else if resp.Err != nil {
			return err
		}

		return nil
	}(); err != nil {
		conn.Close()
		return nil, err
	}

	return query.NewReaderIterator(ctx, conn, resp.typ, resp.stats), nil
}

// FieldDimensions returns the unique fields and dimensions across a list of sources.
func (ic *remoteIteratorCreator) FieldDimensions(m *influxql.Measurement) (fields map[string]influxql.DataType, dimensions map[string]struct{}, err error) {
	conn, err := ic.dialer.DialNode(ic.nodeID)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()

	// Write request.
	if err := EncodeTLV(conn, fieldDimensionsRequestMessage, &FieldDimensionsRequest{
		ShardIDs:    ic.shardIDs,
		Measurement: *m,
	}); err != nil {
		return nil, nil, err
	}

	// Read the response.
	var resp FieldDimensionsResponse
	if _, err := DecodeTLV(conn, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Fields, resp.Dimensions, resp.Err
}

// NodeDialer dials connections to a given node.
type NodeDialer struct {
	MetaClient MetaClient
	Timeout    time.Duration
}

// DialNode returns a connection to a node.
func (d *NodeDialer) DialNode(nodeID uint64) (net.Conn, error) {
	ni, err := d.MetaClient.DataNode(nodeID)
	if err != nil {
		return nil, err
	}

	conn, err := net.Dial("tcp", ni.TCPHost)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(d.Timeout))

	// Write the cluster multiplexing header byte
	if _, err := conn.Write([]byte{MuxHeader}); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (a *LocalShardMapping) IteratorCost(m *influxql.Measurement, opt query.IteratorOptions) (query.IteratorCost, error) {
	source := Source{
		Database:        m.Database,
		RetentionPolicy: m.RetentionPolicy,
	}

	sg := a.ShardMap[source]
	if sg == nil {
		return query.IteratorCost{}, nil
	}

	// Override the time constraints if they don't match each other.
	if !a.MinTime.IsZero() && opt.StartTime < a.MinTime.UnixNano() {
		opt.StartTime = a.MinTime.UnixNano()
	}
	if !a.MaxTime.IsZero() && opt.EndTime > a.MaxTime.UnixNano() {
		opt.EndTime = a.MaxTime.UnixNano()
	}

	if m.Regex != nil {
		var costs query.IteratorCost
		measurements := sg.MeasurementsByRegex(m.Regex.Val)
		for _, measurement := range measurements {
			cost, err := sg.IteratorCost(measurement, opt)
			if err != nil {
				return query.IteratorCost{}, err
			}
			costs = costs.Combine(cost)
		}
		return costs, nil
	}
	return sg.IteratorCost(m.Name, opt)
}

// Close clears out the list of mapped shards.
func (a *LocalShardMapping) Close() error {
	a.ShardMap = nil
	return nil
}

// Source contains the database and retention policy source for data.
type Source struct {
	Database        string
	RetentionPolicy string
}
