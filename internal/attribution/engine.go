package attribution

import (
	"github.com/sanoguzhan/krato/internal/pricing"
	"github.com/sanoguzhan/krato/internal/source"
)

type Engine struct {
	Collector       source.Source
	PricingProvider pricing.PricingProvider
}
