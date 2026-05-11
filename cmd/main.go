// Package main implements the aws-service-quotas-exporter entrypoint
package main

import (
	"fmt"
	"net/http"

	"github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	logging "github.com/sirupsen/logrus"
	"github.com/thought-machine/aws-service-quotas-exporter/serviceexporter"
	"github.com/thought-machine/aws-service-quotas-exporter/servicequotas"
)

var log = logging.WithFields(logging.Fields{})

var opts struct {
	Port           int      `long:"port" short:"p" default:"9090" description:"Port on which to serve."`
	Region         string   `long:"region" short:"r" env:"AWS_REGION" required:"true" description:"AWS region name"`
	Profile        string   `long:"profile" short:"f" env:"AWS_PROFILE" default:"" description:"Named AWS profile to be used"`
	RefreshPeriod  int      `long:"refresh-period" default:"360" description:"Refresh period in seconds"`
	IncludeAWSTags        []string `long:"include-aws-tag" description:"The aws resource tags to include as labels for returned metrics"`
	EnableVpcEndpoints    bool     `long:"enable-vpc-endpoints" description:"Enable VPC endpoint quota monitoring"`
	EnableVpcsPerRegion   bool     `long:"enable-vpcs-per-region" description:"Enable VPCs per region quota monitoring"`
	EnableEIPsPerRegion   bool     `long:"enable-eips-per-region" description:"Enable Elastic IPs per region quota monitoring"`
	EnableNLBsPerRegion   bool     `long:"enable-nlbs-per-region" description:"Enable Network Load Balancers per region quota monitoring"`
}

func main() {
	flags.Parse(&opts)
	quotasOpts := servicequotas.QuotasOptions{
		EnableVpcEndpointChecks:  opts.EnableVpcEndpoints,
		EnableVpcsPerRegionCheck: opts.EnableVpcsPerRegion,
		EnableEIPsPerRegionCheck: opts.EnableEIPsPerRegion,
		EnableNLBsPerRegionCheck: opts.EnableNLBsPerRegion,
	}
	quotasExporter, err := serviceexporter.NewServiceQuotasExporter(opts.Region, opts.Profile, opts.RefreshPeriod, opts.IncludeAWSTags, quotasOpts)
	if err != nil {
		log.Fatalf("Failed to create exporter: %s", err)
	}

	prometheus.Register(quotasExporter)

	log.Infof("Serving on port: %d", opts.Port)
	log.Infof("Serving Prometheus metrics on /metrics")
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", opts.Port), nil))
}
