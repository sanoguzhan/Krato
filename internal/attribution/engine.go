package attribution
import (
	"context"
	"github.com/sanoguzhan/krato/internal/metrics"
	"github.com/sanoguzhan/krato/internal/pricing"
	"github.com/sanoguzhan/krato/internal/workload"
)

type Engine struct {
	Collector metrics.MetricsCollector
	PricingProvider pricing.PricingProvider
}

