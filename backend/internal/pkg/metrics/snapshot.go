package metrics

import (
	"math"
	"sort"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"
)

// Snapshot is a point-in-time copy of the registry used by in-process
// evaluators. Labels are retained because production rules need bounded,
// low-cardinality filters such as status_class=result.
type Snapshot struct {
	TakenAt  time.Time
	Families map[string]FamilySnapshot
}

type FamilySnapshot struct {
	Type    dto.MetricType
	Samples []Sample
}

type Sample struct {
	Labels    map[string]string
	Value     float64
	Histogram *HistogramSnapshot
}

type HistogramSnapshot struct {
	Count   uint64
	Sum     float64
	Buckets []HistogramBucket
}

type HistogramBucket struct {
	UpperBound      float64
	CumulativeCount uint64
}

// Snapshot captures metric values with an explicit timestamp for deterministic
// window evaluation and tests.
func (r *Registry) Snapshot(at time.Time) Snapshot {
	out := Snapshot{TakenAt: at.UTC(), Families: make(map[string]FamilySnapshot)}
	if r == nil || r.prom == nil {
		return out
	}
	families, err := r.prom.Gather()
	if err != nil {
		return out
	}
	for _, family := range families {
		name := family.GetName()
		if name == "" {
			continue
		}
		snapshot := FamilySnapshot{Type: family.GetType(), Samples: make([]Sample, 0, len(family.GetMetric()))}
		for _, metric := range family.GetMetric() {
			sample := Sample{Labels: metricLabels(metric), Value: metricValue(metric)}
			if histogram := metric.GetHistogram(); histogram != nil {
				sample.Histogram = &HistogramSnapshot{
					Count:   histogram.GetSampleCount(),
					Sum:     histogram.GetSampleSum(),
					Buckets: make([]HistogramBucket, 0, len(histogram.GetBucket())+1),
				}
				for _, bucket := range histogram.GetBucket() {
					sample.Histogram.Buckets = append(sample.Histogram.Buckets, HistogramBucket{
						UpperBound:      bucket.GetUpperBound(),
						CumulativeCount: bucket.GetCumulativeCount(),
					})
				}
				sample.Histogram.Buckets = append(sample.Histogram.Buckets, HistogramBucket{UpperBound: math.Inf(1), CumulativeCount: histogram.GetSampleCount()})
			}
			snapshot.Samples = append(snapshot.Samples, sample)
		}
		out.Families[name] = snapshot
	}
	r.mu.Lock()
	for name := range r.counters {
		if _, ok := out.Families[name]; !ok {
			out.Families[name] = FamilySnapshot{Type: dto.MetricType_COUNTER}
		}
	}
	for name := range r.gauges {
		if _, ok := out.Families[name]; !ok {
			out.Families[name] = FamilySnapshot{Type: dto.MetricType_GAUGE}
		}
	}
	for name := range r.histogram {
		if _, ok := out.Families[name]; !ok {
			out.Families[name] = FamilySnapshot{Type: dto.MetricType_HISTOGRAM}
		}
	}
	r.mu.Unlock()
	return out
}

// SnapshotNow captures the current registry state.
func (r *Registry) SnapshotNow() Snapshot { return r.Snapshot(time.Now().UTC()) }

// AggregateValues provides the legacy label-free view for display-only
// consumers. Alert and SLO evaluation must use Snapshot directly.
func (s Snapshot) AggregateValues() map[string]float64 {
	out := make(map[string]float64, len(s.Families))
	for name, family := range s.Families {
		for _, sample := range family.Samples {
			out[name] += sample.Value
		}
	}
	return out
}

// CounterDelta returns the increase between snapshots for matching series.
// A reset fails closed because the true increase within the window is unknown.
func CounterDelta(previous, current Snapshot, name string, labels map[string]string) (float64, bool) {
	curFamily, ok := current.Families[name]
	if !ok || curFamily.Type != dto.MetricType_COUNTER {
		return 0, false
	}
	prevFamily, ok := previous.Families[name]
	if !ok || prevFamily.Type != dto.MetricType_COUNTER {
		return 0, false
	}
	previousValues := make(map[string]float64, len(prevFamily.Samples))
	for _, sample := range prevFamily.Samples {
		if labelsMatch(sample.Labels, labels) {
			previousValues[labelKey(sample.Labels)] = sample.Value
		}
	}
	total := 0.0
	matched := false
	for _, sample := range curFamily.Samples {
		if !labelsMatch(sample.Labels, labels) {
			continue
		}
		matched = true
		previousValue, exists := previousValues[labelKey(sample.Labels)]
		if !exists {
			total += sample.Value
			continue
		}
		if sample.Value < previousValue {
			return 0, false
		}
		total += sample.Value - previousValue
	}
	return total, matched || len(curFamily.Samples) == 0 || len(labels) > 0
}

// GaugeMax returns the maximum matching gauge value.
func GaugeMax(snapshot Snapshot, name string, labels map[string]string) (float64, bool) {
	family, ok := snapshot.Families[name]
	if !ok || family.Type != dto.MetricType_GAUGE {
		return 0, false
	}
	value := 0.0
	matched := false
	for _, sample := range family.Samples {
		if !labelsMatch(sample.Labels, labels) {
			continue
		}
		if !matched || sample.Value > value {
			value = sample.Value
		}
		matched = true
	}
	return value, matched
}

// HistogramQuantileDelta estimates a quantile from bucket increases in the
// selected window. It fails closed on a histogram reset.
func HistogramQuantileDelta(previous, current Snapshot, name string, labels map[string]string, quantile float64) (float64, uint64, bool) {
	if quantile <= 0 || quantile > 1 {
		return 0, 0, false
	}
	curFamily, ok := current.Families[name]
	if !ok || curFamily.Type != dto.MetricType_HISTOGRAM {
		return 0, 0, false
	}
	prevFamily, ok := previous.Families[name]
	if !ok || prevFamily.Type != dto.MetricType_HISTOGRAM {
		return 0, 0, false
	}
	previousSamples := make(map[string]Sample, len(prevFamily.Samples))
	for _, sample := range prevFamily.Samples {
		if labelsMatch(sample.Labels, labels) {
			previousSamples[labelKey(sample.Labels)] = sample
		}
	}
	bucketDeltas := map[float64]uint64{}
	var count uint64
	matched := false
	for _, sample := range curFamily.Samples {
		if !labelsMatch(sample.Labels, labels) || sample.Histogram == nil {
			continue
		}
		previousSample, exists := previousSamples[labelKey(sample.Labels)]
		if !exists {
			previousSample = Sample{Histogram: &HistogramSnapshot{}}
		}
		if previousSample.Histogram == nil || sample.Histogram.Count < previousSample.Histogram.Count {
			return 0, 0, false
		}
		matched = true
		count += sample.Histogram.Count - previousSample.Histogram.Count
		previousBuckets := make(map[float64]uint64, len(previousSample.Histogram.Buckets))
		for _, bucket := range previousSample.Histogram.Buckets {
			previousBuckets[bucket.UpperBound] = bucket.CumulativeCount
		}
		for _, bucket := range sample.Histogram.Buckets {
			before, exists := previousBuckets[bucket.UpperBound]
			if !exists || bucket.CumulativeCount < before {
				return 0, 0, false
			}
			bucketDeltas[bucket.UpperBound] += bucket.CumulativeCount - before
		}
	}
	if !matched || count == 0 {
		return 0, count, false
	}
	bounds := make([]float64, 0, len(bucketDeltas))
	for bound := range bucketDeltas {
		bounds = append(bounds, bound)
	}
	sort.Float64s(bounds)
	target := uint64(math.Ceil(float64(count) * quantile))
	for _, bound := range bounds {
		if bucketDeltas[bound] >= target {
			return bound, count, true
		}
	}
	return math.Inf(1), count, true
}

// HistogramThresholdDelta returns observations at or below upperBound and the
// total observation count for the selected window.
func HistogramThresholdDelta(previous, current Snapshot, name string, labels map[string]string, upperBound float64) (uint64, uint64, bool) {
	curFamily, ok := current.Families[name]
	if !ok || curFamily.Type != dto.MetricType_HISTOGRAM {
		return 0, 0, false
	}
	prevFamily, ok := previous.Families[name]
	if !ok || prevFamily.Type != dto.MetricType_HISTOGRAM {
		return 0, 0, false
	}
	previousSamples := make(map[string]Sample, len(prevFamily.Samples))
	for _, sample := range prevFamily.Samples {
		if labelsMatch(sample.Labels, labels) {
			previousSamples[labelKey(sample.Labels)] = sample
		}
	}
	var within, total uint64
	matched := false
	for _, sample := range curFamily.Samples {
		if !labelsMatch(sample.Labels, labels) || sample.Histogram == nil {
			continue
		}
		previousSample, exists := previousSamples[labelKey(sample.Labels)]
		if !exists {
			previousSample = Sample{Histogram: &HistogramSnapshot{}}
		}
		if previousSample.Histogram == nil || sample.Histogram.Count < previousSample.Histogram.Count {
			return 0, 0, false
		}
		matched = true
		total += sample.Histogram.Count - previousSample.Histogram.Count
		previousBuckets := make(map[float64]uint64, len(previousSample.Histogram.Buckets))
		for _, bucket := range previousSample.Histogram.Buckets {
			previousBuckets[bucket.UpperBound] = bucket.CumulativeCount
		}
		var selectedCount uint64
		selected := false
		for _, bucket := range sample.Histogram.Buckets {
			if bucket.UpperBound > upperBound {
				continue
			}
			before := previousBuckets[bucket.UpperBound]
			if bucket.CumulativeCount < before {
				return 0, 0, false
			}
			selectedCount = bucket.CumulativeCount - before
			selected = true
		}
		if selected {
			within += selectedCount
		}
	}
	return within, total, matched
}

func metricLabels(metric *dto.Metric) map[string]string {
	labels := make(map[string]string, len(metric.GetLabel()))
	for _, pair := range metric.GetLabel() {
		labels[pair.GetName()] = pair.GetValue()
	}
	return labels
}

func labelsMatch(actual, expected map[string]string) bool {
	for key, value := range expected {
		if actual[key] != value {
			return false
		}
	}
	return true
}

func labelKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(labels[key])
		b.WriteByte('\x00')
	}
	return b.String()
}
