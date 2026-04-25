package canonical

import "time"

// TimeBucket représente un point d'une série temporelle.
type TimeBucket struct {
	BucketStart time.Time
	BucketEnd   time.Time
	Value       float64
	Count       int
}

// MetricSeries est le résultat canonique d'une LoadTimeseries.
type MetricSeries struct {
	Metric  FieldKey
	Bucket  Bucket
	Buckets []TimeBucket
	GroupBy []GroupBy
}
