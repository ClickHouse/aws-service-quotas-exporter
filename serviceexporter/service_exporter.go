// Package serviceexporter implements the logic to export the data collected by
// the servicequotas package as Prometheus metrics
package serviceexporter

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	logging "github.com/sirupsen/logrus"
	"github.com/thought-machine/aws-service-quotas-exporter/servicequotas"
)

var log = logging.WithFields(logging.Fields{})

const unknownCheck = "unknown"

// Metric holds usage and limit desc and values
type Metric struct {
	usageDesc   *prometheus.Desc
	limitDesc   *prometheus.Desc
	usage       float64
	limit       float64
	labelValues []string
}

func metricKey(quota servicequotas.QuotaUsage) string {
	return fmt.Sprintf("%s%s", quota.Name, quota.Identifier())
}

// ServiceQuotasExporter AWS service quotas and usage prometheus
// exporter
type ServiceQuotasExporter struct {
	metricsRegion   string
	quotasClient    servicequotas.QuotasInterface
	metrics         map[string]Metric
	metricsLock     *sync.Mutex
	refreshPeriod   int
	waitForMetrics  chan struct{}
	includedAWSTags []string

	lastRefreshSuccess time.Time
	refreshErrors      map[string]float64
}

// NewServiceQuotasExporter creates a new ServiceQuotasExporter
func NewServiceQuotasExporter(region, profile string, refreshPeriod int, includedAWSTags []string, quotasOpts ...servicequotas.QuotasOptions) (*ServiceQuotasExporter, error) {
	if refreshPeriod <= 0 {
		return nil, fmt.Errorf("refresh period must be positive, got %d", refreshPeriod)
	}

	quotasClient, err := servicequotas.NewServiceQuotas(region, profile, quotasOpts...)
	if err != nil {
		return nil, err
	}

	ch := make(chan struct{})
	exporter := &ServiceQuotasExporter{
		metricsRegion:   region,
		quotasClient:    quotasClient,
		metrics:         map[string]Metric{},
		metricsLock:     &sync.Mutex{},
		refreshPeriod:   refreshPeriod,
		waitForMetrics:  ch,
		includedAWSTags: includedAWSTags,
	}
	go exporter.createOrUpdateQuotasAndDescriptions(false)
	go exporter.refreshMetrics()

	return exporter, nil
}

func (e *ServiceQuotasExporter) refreshMetrics() {
	<-e.waitForMetrics

	ticker := time.NewTicker(time.Duration(e.refreshPeriod) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		e.createOrUpdateQuotasAndDescriptions(true)
	}
}

// createOrUpdateQuotasAndDescriptions replaces the exported metrics
// with the latest quotas and usage, so resources created or deleted
// since the last run are added or removed. On error the previously
// exported metrics are kept. update is false only for the first run,
// which unblocks the refresh loop once it finishes.
func (e *ServiceQuotasExporter) createOrUpdateQuotasAndDescriptions(update bool) {
	if !update {
		defer close(e.waitForMetrics)
	}

	quotas, err := e.quotasClient.QuotasAndUsage()
	if err != nil {
		check := unknownCheck
		var checkErr *servicequotas.CheckError
		if errors.As(err, &checkErr) {
			check = checkErr.Check
		}

		e.metricsLock.Lock()
		if e.refreshErrors == nil {
			e.refreshErrors = map[string]float64{}
		}
		e.refreshErrors[check]++
		e.metricsLock.Unlock()

		log.Errorf("Could not retrieve quotas and limits, keeping previous metrics: %s", err)
		return
	}

	metrics := make(map[string]Metric, len(quotas))
	for _, quota := range quotas {
		resourceID := quota.Identifier()

		labels := []string{"resource"}
		labelValues := []string{resourceID}

		for _, tag := range e.includedAWSTags {
			prometheusFormatTag := servicequotas.ToPrometheusNamingFormat(tag)
			labels = append(labels, prometheusFormatTag)
			// Need to set empty label value to keep label name and value count the same
			labelValues = append(labelValues, quota.Tags[prometheusFormatTag])
		}

		usageHelp := fmt.Sprintf("Used amount of %s", quota.Description)
		usageDesc := newDesc(e.metricsRegion, quota.Name, "used_total", usageHelp, labels)

		limitHelp := fmt.Sprintf("Limit of %s", quota.Description)
		limitDesc := newDesc(e.metricsRegion, quota.Name, "limit_total", limitHelp, labels)

		metrics[metricKey(quota)] = Metric{
			usageDesc:   usageDesc,
			limitDesc:   limitDesc,
			usage:       quota.Usage,
			limit:       quota.Quota,
			labelValues: labelValues,
		}
	}

	e.metricsLock.Lock()
	defer e.metricsLock.Unlock()

	e.metrics = metrics
	e.lastRefreshSuccess = time.Now()
	log.Infof("Refreshed metrics for %d quotas", len(metrics))
}

// Describe sends no descriptors, which makes this an unchecked
// collector. The set of exported metrics changes at runtime as AWS
// resources are created and deleted.
func (e *ServiceQuotasExporter) Describe(_ chan<- *prometheus.Desc) {}

// Collect implements the collect function for prometheus collectors
func (e *ServiceQuotasExporter) Collect(ch chan<- prometheus.Metric) {
	e.metricsLock.Lock()
	defer e.metricsLock.Unlock()

	for _, metric := range e.metrics {
		ch <- prometheus.MustNewConstMetric(metric.limitDesc, prometheus.GaugeValue, metric.limit, metric.labelValues...)
		ch <- prometheus.MustNewConstMetric(metric.usageDesc, prometheus.GaugeValue, metric.usage, metric.labelValues...)
	}

	lastRefreshSuccess := 0.0
	if !e.lastRefreshSuccess.IsZero() {
		lastRefreshSuccess = float64(e.lastRefreshSuccess.UnixNano()) / 1e9
	}
	ch <- prometheus.MustNewConstMetric(e.lastRefreshSuccessDesc(), prometheus.GaugeValue, lastRefreshSuccess)

	refreshErrorsDesc := e.refreshErrorsDesc()
	for check, count := range e.refreshErrors {
		ch <- prometheus.MustNewConstMetric(refreshErrorsDesc, prometheus.CounterValue, count, check)
	}
}

func (e *ServiceQuotasExporter) lastRefreshSuccessDesc() *prometheus.Desc {
	return prometheus.NewDesc(
		"aws_service_quotas_exporter_last_refresh_success_timestamp_seconds",
		"Unix timestamp of the last successful refresh of AWS quotas and usage",
		nil,
		prometheus.Labels{"region": e.metricsRegion},
	)
}

func (e *ServiceQuotasExporter) refreshErrorsDesc() *prometheus.Desc {
	return prometheus.NewDesc(
		"aws_service_quotas_exporter_refresh_errors_total",
		"Total number of failed refreshes of AWS quotas and usage, by failing check",
		[]string{"check"},
		prometheus.Labels{"region": e.metricsRegion},
	)
}

func newDesc(region, quotaName, metricName, help string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName("aws", quotaName, metricName),
		help,
		labels,
		prometheus.Labels{"region": region},
	)
}
