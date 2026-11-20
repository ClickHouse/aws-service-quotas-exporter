package serviceexporter

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/aws-service-quotas-exporter/servicequotas"
)

func resourceName(name string) *string {
	return &name
}

type ServiceQuotasMock struct {
	quotas []servicequotas.QuotaUsage
	err    error
}

func (s *ServiceQuotasMock) QuotasAndUsage(_ context.Context) ([]servicequotas.QuotaUsage, error) {
	return s.quotas, s.err
}

func TestUpdateMetrics(t *testing.T) {
	quotasClient := &ServiceQuotasMock{
		quotas: []servicequotas.QuotaUsage{
			{ResourceName: resourceName("i-asdasd1"), Usage: 5, Quota: 10, Tags: map[string]string{"dummy_tag": "dummy-value"}},
			{ResourceName: resourceName("i-asdasd2"), Usage: 2, Quota: 3},
			{ResourceName: resourceName("i-asdasd3"), Usage: 5, Quota: 10},
		},
	}

	exporter := &ServiceQuotasExporter{
		metricsRegion: "eu-west-1",
		quotasClient:  quotasClient,
		metrics: map[string]Metric{
			"i-asdasd1": Metric{usage: 3, limit: 5, labelValues: []string{"before-dummy-value"}},
			"i-asdasd2": Metric{usage: 2, limit: 2},
		},
		metricsLock:     &sync.Mutex{},
		includedAWSTags: []string{"dummy-tag"},
		refreshPeriod:   360,
	}

	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)

	usageDesc := newDesc("eu-west-1", "", "used_total", "Used amount of ", []string{"resource", "dummy_tag"})
	limitDesc := newDesc("eu-west-1", "", "limit_total", "Limit of ", []string{"resource", "dummy_tag"})
	expectedMetrics := map[string]Metric{
		"i-asdasd1": Metric{usage: 5, limit: 10, labelValues: []string{"i-asdasd1", "dummy-value"}, usageDesc: usageDesc, limitDesc: limitDesc},
		"i-asdasd2": Metric{usage: 2, limit: 3, labelValues: []string{"i-asdasd2", ""}, usageDesc: usageDesc, limitDesc: limitDesc},
		"i-asdasd3": Metric{usage: 5, limit: 10, labelValues: []string{"i-asdasd3", ""}, usageDesc: usageDesc, limitDesc: limitDesc},
	}
	exporter.metricsLock.Lock()
	defer exporter.metricsLock.Unlock()
	assert.Equal(t, expectedMetrics, exporter.metrics)
}

func TestCreateQuotasAndDescriptions(t *testing.T) {
	region := "eu-west-1"

	firstQ := servicequotas.QuotaUsage{
		Name:         "Name1",
		ResourceName: resourceName("i-asdasd1"),
		Description:  "desc1",
		Usage:        5,
		Quota:        10,
	}
	secondQ := servicequotas.QuotaUsage{
		Name:         "Name2",
		ResourceName: resourceName("i-asdasd2"),
		Description:  "desc2",
		Usage:        1,
		Quota:        8,
		Tags:         map[string]string{"dummy_tag": "dummy-value", "dummy_tag2": "dummy-value2"},
	}
	quotasClient := &ServiceQuotasMock{
		quotas: []servicequotas.QuotaUsage{firstQ, secondQ},
	}

	ch := make(chan struct{})
	exporter := &ServiceQuotasExporter{
		metricsRegion:   region,
		quotasClient:    quotasClient,
		metrics:         map[string]Metric{},
		metricsLock:     &sync.Mutex{},
		refreshPeriod:   360,
		waitForMetrics:  ch,
		includedAWSTags: []string{"dummy-tag", "dummy-tag2"},
	}

	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), false)

	firstUsageDesc := newDesc(region, firstQ.Name, "used_total", "Used amount of desc1", []string{"resource", "dummy_tag", "dummy_tag2"})
	firstLimitDesc := newDesc(region, firstQ.Name, "limit_total", "Limit of desc1", []string{"resource", "dummy_tag", "dummy_tag2"})
	secondUsageDesc := newDesc(region, secondQ.Name, "used_total", "Used amount of desc2", []string{"resource", "dummy_tag", "dummy_tag2"})
	secondLimitDesc := newDesc(region, secondQ.Name, "limit_total", "Limit of desc2", []string{"resource", "dummy_tag", "dummy_tag2"})
	expectedMetrics := map[string]Metric{
		"Name1i-asdasd1": Metric{
			usageDesc:   firstUsageDesc,
			limitDesc:   firstLimitDesc,
			usage:       5,
			limit:       10,
			labelValues: []string{"i-asdasd1", "", ""},
		},
		"Name2i-asdasd2": Metric{
			usageDesc:   secondUsageDesc,
			limitDesc:   secondLimitDesc,
			usage:       1,
			limit:       8,
			labelValues: []string{"i-asdasd2", "dummy-value", "dummy-value2"},
		},
	}

	exporter.metricsLock.Lock()
	defer exporter.metricsLock.Unlock()
	assert.Equal(t, expectedMetrics, exporter.metrics)
}

func TestCreateQuotasAndDescriptionsRefresh(t *testing.T) {
	quotasClient := &ServiceQuotasMock{
		quotas: []servicequotas.QuotaUsage{
			{ResourceName: resourceName("i-asdasd1"),
				Usage:       5,
				Quota:       10,
				Tags:        map[string]string{"dummy_tag": "dummy-value"},
				Description: "Refresh rebuilds the metric description",
			},
			{ResourceName: resourceName("i-asdasd3"), Usage: 5, Quota: 10},
		},
	}

	desc := newDesc("eu-west-1", "some-quota", "some-metric", "help", []string{})

	ch := make(chan struct{})
	exporter := &ServiceQuotasExporter{
		metricsRegion: "eu-west-1",
		quotasClient:  quotasClient,
		metrics: map[string]Metric{
			"i-asdasd1": Metric{usage: 3, limit: 5, labelValues: []string{"before-dummy-value"}, usageDesc: desc},
		},
		metricsLock:     &sync.Mutex{},
		waitForMetrics:  ch,
		includedAWSTags: []string{"dummy-tag"},
		refreshPeriod:   360,
	}

	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)

	labels := []string{"resource", "dummy_tag"}
	expectedMetrics := map[string]Metric{
		"i-asdasd1": Metric{usage: 5, limit: 10, labelValues: []string{"i-asdasd1", "dummy-value"},
			usageDesc: newDesc("eu-west-1", "", "used_total", "Used amount of Refresh rebuilds the metric description", labels),
			limitDesc: newDesc("eu-west-1", "", "limit_total", "Limit of Refresh rebuilds the metric description", labels)},
		"i-asdasd3": Metric{usage: 5, limit: 10, labelValues: []string{"i-asdasd3", ""},
			usageDesc: newDesc("eu-west-1", "", "used_total", "Used amount of ", labels),
			limitDesc: newDesc("eu-west-1", "", "limit_total", "Limit of ", labels)},
	}

	exporter.metricsLock.Lock()
	defer exporter.metricsLock.Unlock()
	assert.Equal(t, expectedMetrics, exporter.metrics)

	close(ch) // should panic if it was already closed
}

func newTestExporter(quotasClient servicequotas.QuotasInterface) *ServiceQuotasExporter {
	return &ServiceQuotasExporter{
		metricsRegion:  "us-east-1",
		quotasClient:   quotasClient,
		metrics:        map[string]Metric{},
		metricsLock:    &sync.Mutex{},
		refreshPeriod:  360,
		waitForMetrics: make(chan struct{}),
	}
}

func sgQuota(id string, usage float64) servicequotas.QuotaUsage {
	return servicequotas.QuotaUsage{
		Name:         "inbound_rules_per_security_group",
		ResourceName: resourceName(id),
		Description:  "inbound rules per security group",
		Usage:        usage,
		Quota:        250,
	}
}

// exportedResources returns the sorted "metric/resource" pairs exported
// by the collector, ignoring the exporter's own metrics
func exportedResources(t *testing.T, exporter *ServiceQuotasExporter) []string {
	t.Helper()

	registry := prometheus.NewPedanticRegistry()
	require.NoError(t, registry.Register(exporter))
	families, err := registry.Gather()
	require.NoError(t, err)

	resources := []string{}
	for _, family := range families {
		if strings.HasPrefix(family.GetName(), "aws_service_quotas_exporter_") {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "resource" {
					resources = append(resources, family.GetName()+"/"+label.GetValue())
				}
			}
		}
	}
	sort.Strings(resources)
	return resources
}

func TestRefreshAddsResourceCreatedAfterStartup(t *testing.T) {
	quotasClient := &ServiceQuotasMock{quotas: []servicequotas.QuotaUsage{sgQuota("sg-1", 10)}}
	exporter := newTestExporter(quotasClient)

	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), false)
	assert.Equal(t, []string{
		"aws_inbound_rules_per_security_group_limit_total/sg-1",
		"aws_inbound_rules_per_security_group_used_total/sg-1",
	}, exportedResources(t, exporter))

	quotasClient.quotas = []servicequotas.QuotaUsage{sgQuota("sg-1", 10), sgQuota("sg-2", 249)}
	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)
	assert.Equal(t, []string{
		"aws_inbound_rules_per_security_group_limit_total/sg-1",
		"aws_inbound_rules_per_security_group_limit_total/sg-2",
		"aws_inbound_rules_per_security_group_used_total/sg-1",
		"aws_inbound_rules_per_security_group_used_total/sg-2",
	}, exportedResources(t, exporter))
}

func TestRefreshAddsQuotaNameNotSeenAtStartup(t *testing.T) {
	quotasClient := &ServiceQuotasMock{quotas: []servicequotas.QuotaUsage{}}
	exporter := newTestExporter(quotasClient)

	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), false)
	assert.Empty(t, exportedResources(t, exporter))

	quotasClient.quotas = []servicequotas.QuotaUsage{sgQuota("sg-1", 10)}
	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)
	assert.Equal(t, []string{
		"aws_inbound_rules_per_security_group_limit_total/sg-1",
		"aws_inbound_rules_per_security_group_used_total/sg-1",
	}, exportedResources(t, exporter))
}

func TestRefreshRemovesDeletedResource(t *testing.T) {
	quotasClient := &ServiceQuotasMock{quotas: []servicequotas.QuotaUsage{sgQuota("sg-1", 10), sgQuota("sg-2", 20)}}
	exporter := newTestExporter(quotasClient)

	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), false)
	assert.Len(t, exportedResources(t, exporter), 4)

	quotasClient.quotas = []servicequotas.QuotaUsage{sgQuota("sg-1", 10)}
	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)
	assert.Equal(t, []string{
		"aws_inbound_rules_per_security_group_limit_total/sg-1",
		"aws_inbound_rules_per_security_group_used_total/sg-1",
	}, exportedResources(t, exporter))
}

func TestRefreshErrorKeepsPreviousMetrics(t *testing.T) {
	quotasClient := &ServiceQuotasMock{quotas: []servicequotas.QuotaUsage{sgQuota("sg-1", 10)}}
	exporter := newTestExporter(quotasClient)

	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), false)
	lastSuccess := exporter.lastRefreshSuccess
	assert.False(t, lastSuccess.IsZero())

	quotasClient.quotas = nil
	quotasClient.err = &servicequotas.CheckError{Check: "RulesPerSecurityGroupUsageCheck", Err: errors.New("throttled")}
	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)

	assert.Equal(t, []string{
		"aws_inbound_rules_per_security_group_limit_total/sg-1",
		"aws_inbound_rules_per_security_group_used_total/sg-1",
	}, exportedResources(t, exporter))
	assert.Equal(t, lastSuccess, exporter.lastRefreshSuccess)
	assert.Equal(t, map[string]float64{"RulesPerSecurityGroupUsageCheck": 1}, exporter.refreshErrors)
}

func TestRefreshErrorOnFirstRun(t *testing.T) {
	quotasClient := &ServiceQuotasMock{err: errors.New("no credentials")}
	exporter := newTestExporter(quotasClient)

	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), false)

	assert.Empty(t, exportedResources(t, exporter))
	assert.True(t, exporter.lastRefreshSuccess.IsZero())
	assert.Equal(t, map[string]float64{unknownCheck: 1}, exporter.refreshErrors)
	_, open := <-exporter.waitForMetrics
	assert.False(t, open, "first run must unblock the refresh loop even on error")
}

func TestConcurrentRefreshAndCollect(t *testing.T) {
	quotasClient := &ServiceQuotasMock{quotas: []servicequotas.QuotaUsage{sgQuota("sg-1", 10), sgQuota("sg-2", 20)}}
	exporter := newTestExporter(quotasClient)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)
		}
	}()
	for i := 0; i < 50; i++ {
		_ = testutil.CollectAndCount(exporter)
	}
	<-done
}

func TestCollectExportsRefreshStatusMetrics(t *testing.T) {
	quotasClient := &ServiceQuotasMock{err: &servicequotas.CheckError{Check: "list_service_quotas_ec2", Err: errors.New("boom")}}
	exporter := newTestExporter(quotasClient)
	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)

	expected := `
# HELP aws_service_quotas_exporter_last_refresh_success_timestamp_seconds Unix timestamp of the last successful refresh of AWS quotas and usage
# TYPE aws_service_quotas_exporter_last_refresh_success_timestamp_seconds gauge
aws_service_quotas_exporter_last_refresh_success_timestamp_seconds{region="us-east-1"} 0
# HELP aws_service_quotas_exporter_refresh_errors_total Total number of failed refreshes of AWS quotas and usage, by failing check
# TYPE aws_service_quotas_exporter_refresh_errors_total counter
aws_service_quotas_exporter_refresh_errors_total{check="list_service_quotas_ec2",region="us-east-1"} 1
`
	assert.NoError(t, testutil.CollectAndCompare(exporter, strings.NewReader(expected)))

	quotasClient.err = nil
	quotasClient.quotas = []servicequotas.QuotaUsage{}
	exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)

	registry := prometheus.NewPedanticRegistry()
	require.NoError(t, registry.Register(exporter))
	families, err := registry.Gather()
	require.NoError(t, err)
	var lastSuccess float64
	for _, family := range families {
		if family.GetName() == "aws_service_quotas_exporter_last_refresh_success_timestamp_seconds" {
			lastSuccess = family.GetMetric()[0].GetGauge().GetValue()
		}
	}
	assert.InDelta(t, float64(exporter.lastRefreshSuccess.Unix()), lastSuccess, 1)
}

// hangingQuotasClient blocks until its refresh context ends.
type hangingQuotasClient struct {
	err error
}

func (c *hangingQuotasClient) QuotasAndUsage(ctx context.Context) ([]servicequotas.QuotaUsage, error) {
	<-ctx.Done()
	c.err = ctx.Err()
	return nil, c.err
}

func TestRefreshTimeoutKeepsPreviousMetrics(t *testing.T) {
	quotasClient := &hangingQuotasClient{}
	exporter := newTestExporter(quotasClient)
	exporter.refreshTimeout = 50 * time.Millisecond
	previousTime := time.Unix(1234, 0)
	exporter.lastRefreshSuccess = previousTime
	previousMetrics := map[string]Metric{"Name1i-asdasd1": {usage: 1, limit: 2}}
	exporter.metrics = map[string]Metric{"Name1i-asdasd1": previousMetrics["Name1i-asdasd1"]}

	done := make(chan struct{})
	go func() {
		exporter.createOrUpdateQuotasAndDescriptions(context.Background(), true)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("refresh did not time out")
	}

	assert.True(t, errors.Is(quotasClient.err, context.DeadlineExceeded))
	exporter.metricsLock.Lock()
	defer exporter.metricsLock.Unlock()
	assert.Equal(t, previousMetrics, exporter.metrics)
	assert.Equal(t, previousTime, exporter.lastRefreshSuccess)
	assert.Equal(t, map[string]float64{unknownCheck: 1}, exporter.refreshErrors)
}

func TestRefreshLoopStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	exporter := newTestExporter(&ServiceQuotasMock{})
	close(exporter.waitForMetrics)
	done := make(chan struct{})
	go func() {
		exporter.refreshMetrics(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("refresh loop did not stop after context cancellation")
	}
	assert.Nil(t, exporter.refreshErrors)
}

func TestNewServiceQuotasExporterRejectsNegativeTimeout(t *testing.T) {
	exporter, err := NewServiceQuotasExporter(context.Background(), "us-east-1", "", 360, -time.Second, nil)
	assert.Nil(t, exporter)
	assert.EqualError(t, err, "refresh timeout must not be negative, got -1s")
}
