package reads

import (
	"context"

	"influxdb.cluster/platform/models"
	"influxdb.cluster/platform/storage/reads/datatypes"
	"influxdb.cluster/platform/tsdb/cursors"
)

type multiShardCursors interface {
	createCursor(row SeriesRow) cursors.Cursor
	newAggregateCursor(ctx context.Context, agg *datatypes.Aggregate, cursor cursors.Cursor) cursors.Cursor
}

type resultSet struct {
	ctx context.Context
	agg *datatypes.Aggregate
	cur SeriesCursor
	row SeriesRow
	mb  multiShardCursors
}

func NewResultSet(ctx context.Context, req *datatypes.ReadRequest, cur SeriesCursor) ResultSet {
	return &resultSet{
		ctx: ctx,
		agg: req.Aggregate,
		cur: cur,
		mb:  newMultiShardArrayCursors(ctx, req.TimestampRange.Start, req.TimestampRange.End, !req.Descending, req.PointsLimit),
	}
}

func (r *resultSet) Err() error { return nil }

// Close closes the result set. Close is idempotent.
func (r *resultSet) Close() {
	if r == nil {
		return // Nothing to do.
	}
	r.row.Query = nil
	r.cur.Close()
}

// Next returns true if there are more results available.
func (r *resultSet) Next() bool {
	if r == nil {
		return false
	}

	row := r.cur.Next()
	if row == nil {
		return false
	}

	r.row = *row

	return true
}

func (r *resultSet) Cursor() cursors.Cursor {
	cur := r.mb.createCursor(r.row)
	if r.agg != nil {
		cur = r.mb.newAggregateCursor(r.ctx, r.agg, cur)
	}
	return cur
}

func (r *resultSet) Tags() models.Tags {
	return r.row.Tags
}

// Stats returns the stats for the underlying cursors.
// Available after resultset has been scanned.
func (r *resultSet) Stats() cursors.CursorStats { return r.row.Query.Stats() }
