package metrics

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"sync"
	"time"
)

var (
	col            *collector
	collectorMutex sync.RWMutex
)

func Init(role string, metricsPort uint) {
	collectorMutex.Lock()
	defer collectorMutex.Unlock()

	col = &collector{
		resourceExpiryMetric: prometheus.NewDesc("vault_sidekick_resource_expiry_gauge",
			"vault_sidekick_resource_expiry_gauge",
			[]string{"resource_id"},
			nil,
		),

		resourceTotalMetric: prometheus.NewDesc("vault_sidekick_resource_total_counter",
			"vault_sidekick_resource_total_counter",
			[]string{"resource_id"},
			nil,
		),
		resourceSuccessMetric: prometheus.NewDesc("vault_sidekick_resource_success_counter",
			"vault_sidekick_resource_success_counter",
			[]string{"resource_id"},
			nil,
		),
		resourceErrorsMetric: prometheus.NewDesc("vault_sidekick_resource_error_counter",
			"vault_sidekick_resource_error_counter",
			[]string{"resource_id"},
			nil,
		),

		resourceProcessTotalMetric: prometheus.NewDesc("vault_sidekick_resource_process_total_counter",
			"vault_sidekick_resource_process_total_counter",
			[]string{"resource_id", "stage"},
			nil,
		),
		resourceProcessSuccessMetric: prometheus.NewDesc("vault_sidekick_resource_process_success_counter",
			"vault_sidekick_resource_process_",
			[]string{"resource_id", "stage"},
			nil,
		),
		resourceProcessErrorsMetric: prometheus.NewDesc("vault_sidekick_resource_process_error_counter",
			"vault_sidekick_resource_process_",
			[]string{"resource_id", "stage"},
			nil,
		),

		tokenTotalMetric: prometheus.NewDesc("vault_sidekick_token_total_counter",
			"vault_sidekick_token_total_counter",
			nil,
			nil,
		),
		tokenSuccessMetric: prometheus.NewDesc("vault_sidekick_token_success_counter",
			"vault_sidekick_token_success_counter",
			nil,
			nil,
		),
		tokenErrorsMetric: prometheus.NewDesc("vault_sidekick_token_error_counter",
			"vault_sidekick_token_error_counter",
			nil,
			nil,
		),

		errorsMetric: prometheus.NewDesc("vault_sidekick_error_counter",
			"vault_sidekick_error_counter",
			[]string{"reason"},
			nil,
		),

		resourceExpiry: make(map[string]time.Time),

		resourceTotals:    make(map[string]int64),
		resourceSuccesses: make(map[string]int64),
		resourceErrors:    make(map[string]int64),

		resourceProcessTotals:    make(map[string]map[string]int64),
		resourceProcessSuccesses: make(map[string]map[string]int64),
		resourceProcessErrors:    make(map[string]map[string]int64),

		errors: make(map[string]int),
	}

	prometheus.MustRegister(col)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil))
	}()
}

func ResourceExpiry(resourceID string, expiry time.Time) {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()

	if col == nil {
		return
	}
	col.ResourceExpiry(resourceID, expiry)
}

func ResourceTotal(resourceID string) {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()

	if col == nil {
		return
	}
	col.ResourceTotal(resourceID)
}

func ResourceSuccess(resourceID string) {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()
	if col == nil {
		return
	}
	col.ResourceSuccess(resourceID)
}

func ResourceError(resourceID string) {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()

	if col == nil {
		return
	}
	col.ResourceError(resourceID)
}

func ResourceProcessTotal(resourceID, stage string) {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()

	if col == nil {
		return
	}
	col.ResourceProcessTotal(resourceID, stage)
}

func ResourceProcessSuccess(resourceID, stage string) {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()
	if col == nil {
		return
	}
	col.ResourceProcessSuccess(resourceID, stage)
}

func ResourceProcessError(resourceID, stage string) {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()

	if col == nil {
		return
	}
	col.ResourceProcessError(resourceID, stage)
}

func TokenTotal() {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()

	if col == nil {
		return
	}
	col.TokenTotal()
}

func TokenSuccess() {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()
	if col == nil {
		return
	}
	col.TokenSuccess()
}

func TokenError() {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()

	if col == nil {
		return
	}
	col.TokenError()
}

func Error(reason string) {
	collectorMutex.RLock()
	defer collectorMutex.RUnlock()

	if col == nil {
		return
	}
	col.Error(reason)
}
