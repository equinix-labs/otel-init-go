package otelhelpers

// SimpleCarrier is an abstraction for handling traceparent propagation
// that needs a type that implements the propagation.TextMapCarrier().
// This is the simplest possible implementation that is a little fragile
// but since we're not doing anything else with it, it's fine for this.
type SimpleCarrier map[string]string

// Get implements the otel interface for propagation.
func (otp SimpleCarrier) Get(key string) string {
	if v, ok := otp[key]; ok {
		return v
	}
	return ""
}

// Set implements the otel interface for propagation.
func (otp SimpleCarrier) Set(key, value string) {
	otp[key] = value
}

// Keys implements the otel interface for propagation.
func (otp SimpleCarrier) Keys() []string {
	out := []string{}
	for k := range otp {
		out = append(out, k)
	}
	return out
}

// Clear implements the otel interface for propagation.
func (otp SimpleCarrier) Clear() {
	for k := range otp {
		delete(otp, k)
	}
}
