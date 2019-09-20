package response

// UsageResponse is the serialized JSON Response for a RedboxAccount usage
// to be returned by usage API
type UsageResponse struct {
	PrincipalID  string  `json:"PrincipalId"`  // User Principal ID
	AccountID    string  `json:"AccountId"`    // AWS Account ID
	StartDate    int64   `json:"StartDate"`    // Usage start date Epoch Timestamp
	EndDate      int64   `json:"EndDate"`      // Usage ends date Epoch Timestamp
	CostAmount   float64 `json:"CostAmount"`   // Cost Amount for given period
	CostCurrency string  `json:"CostCurrency"` // Cost currency
	TimeToLive   int64   `json:"TimeToLive"`   // ttl attribute
}
