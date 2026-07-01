package attribution

import (
	"github.com/sanoguzhan/krato/internal/metrics"
	"github.com/sanoguzhan/krato/internal/pricing"
)

type Engine struct {
	Collector       metrics.MetricsCollector
	PricingProvider pricing.PricingProvider
}
