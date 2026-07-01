package pricing

type PricingProvider interface {
	// GetPrice retrieves the price for a given resource type and usage.
	GetPrice(resourceType string, usage int64) (float64, error)
}
