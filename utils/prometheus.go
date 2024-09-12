package utils

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	PrometheusSyncStarted         *prometheus.CounterVec
	PrometheusSyncFinished        *prometheus.CounterVec
	PrometheusBundlesSynced       *prometheus.CounterVec
	PrometheusSyncStepFailedRetry *prometheus.CounterVec

	PrometheusProcessDuration *prometheus.GaugeVec
	PrometheusBundleHeight    *prometheus.GaugeVec
)

func StartPrometheus(port string) {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(":"+port, nil)
		if err != nil {
			logger.Error().Str("err", err.Error()).Msg("prometheus start error")
		}
	}()
	logger.Info().Str("port", port).Msg("Started prometheus")
}

func init() {
	var labelNames = []string{"poolId", "chainId"}

	PrometheusSyncStarted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sync_started",
	}, labelNames)

	PrometheusSyncFinished = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sync_finished",
	}, labelNames)

	PrometheusBundlesSynced = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bundles_synced",
	}, labelNames)

	PrometheusSyncStepFailedRetry = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sync_step_failed_retry",
	}, labelNames)

	PrometheusProcessDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bundle_process_duration",
	}, labelNames)

	PrometheusBundleHeight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bundle_height",
	}, labelNames)
}
