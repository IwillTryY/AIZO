package layer3

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// SQLiteMetricsStorage implements MetricsStorage backed by SQLite
type SQLiteMetricsStorage struct {
	db *sql.DB
}

// NewSQLiteMetricsStorage creates a new SQLite-backed metrics storage
func NewSQLiteMetricsStorage(db *sql.DB) *SQLiteMetricsStorage {
	return &SQLiteMetricsStorage{db: db}
}

// Store stores a metric in SQLite
func (s *SQLiteMetricsStorage) Store(ctx context.Context, metric *Metric) error {
	labels, _ := json.Marshal(metric.Labels)
	metadata, _ := json.Marshal(metric.Metadata)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO metrics (name, type, value, timestamp, entity_id, labels, metadata) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		metric.Name, string(metric.Type), metric.Value, metric.Timestamp, metric.EntityID, string(labels), string(metadata),
	)
	return err
}

// Query queries metrics from SQLite
func (s *SQLiteMetricsStorage) Query(ctx context.Context, req *QueryRequest) ([]Metric, error) {
	query := "SELECT name, type, value, timestamp, entity_id, labels, metadata FROM metrics WHERE 1=1"
	args := make([]interface{}, 0)

	if req.EntityID != "" {
		query += " AND entity_id = ?"
		args = append(args, req.EntityID)
	}
	if !req.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, req.StartTime)
	}
	if !req.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, req.EndTime)
	}
	if name, ok := req.Filters["name"]; ok {
		query += " AND name = ?"
		args = append(args, name)
	}

	query += " ORDER BY timestamp DESC"
	if req.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, req.Limit)
	} else {
		query += " LIMIT 1000"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make([]Metric, 0)
	for rows.Next() {
		var m Metric
		var mtype string
		var labelsJSON, metadataJSON sql.NullString
		if err := rows.Scan(&m.Name, &mtype, &m.Value, &m.Timestamp, &m.EntityID, &labelsJSON, &metadataJSON); err != nil {
			continue
		}
		m.Type = MetricType(mtype)
		if labelsJSON.Valid {
			json.Unmarshal([]byte(labelsJSON.String), &m.Labels)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &m.Metadata)
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

// QuerySeries queries a time series from SQLite
func (s *SQLiteMetricsStorage) QuerySeries(ctx context.Context, name, entityID string, start, end time.Time) (*MetricSeries, error) {
	query := "SELECT value, timestamp FROM metrics WHERE name = ? AND entity_id = ? AND timestamp >= ? AND timestamp <= ? ORDER BY timestamp ASC"
	rows, err := s.db.QueryContext(ctx, query, name, entityID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	series := &MetricSeries{
		Name:     name,
		EntityID: entityID,
		Points:   make([]MetricPoint, 0),
	}
	for rows.Next() {
		var p MetricPoint
		if err := rows.Scan(&p.Value, &p.Timestamp); err != nil {
			continue
		}
		series.Points = append(series.Points, p)
	}
	return series, nil
}

// Aggregate performs aggregation on metrics
func (s *SQLiteMetricsStorage) Aggregate(ctx context.Context, req *AggregationRequest) (*AggregationResult, error) {
	var aggFunc string
	switch req.Function {
	case AggFuncSum:
		aggFunc = "SUM(value)"
	case AggFuncAvg:
		aggFunc = "AVG(value)"
	case AggFuncMin:
		aggFunc = "MIN(value)"
	case AggFuncMax:
		aggFunc = "MAX(value)"
	case AggFuncCount:
		aggFunc = "COUNT(value)"
	default:
		aggFunc = "AVG(value)"
	}

	// Group by time interval using strftime
	intervalSec := int(req.Interval.Seconds())
	if intervalSec < 1 {
		intervalSec = 60
	}

	query := "SELECT " + aggFunc + " as agg_value, (CAST(strftime('%s', timestamp) AS INTEGER) / ?) * ? as bucket FROM metrics WHERE name = ?"
	args := []interface{}{intervalSec, intervalSec, req.MetricName}

	if req.EntityID != "" {
		query += " AND entity_id = ?"
		args = append(args, req.EntityID)
	}
	if !req.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, req.StartTime)
	}
	if !req.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, req.EndTime)
	}

	query += " GROUP BY bucket ORDER BY bucket ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]MetricPoint, 0)
	for rows.Next() {
		var value float64
		var bucket int64
		if err := rows.Scan(&value, &bucket); err != nil {
			continue
		}
		points = append(points, MetricPoint{
			Timestamp: time.Unix(bucket, 0),
			Value:     value,
		})
	}

	return &AggregationResult{
		MetricName: req.MetricName,
		Function:   req.Function,
		Interval:   req.Interval,
		Series: []AggregatedMetricSeries{
			{Points: points},
		},
	}, nil
}
