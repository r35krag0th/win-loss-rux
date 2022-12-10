package numericsapp

// CounterWidgetResponse is the full response to satisfy the Numerics iOS App request.
type CounterWidgetResponse struct {
	WidgetResponse
	Data *NDataInt `json:"data"`
}
