package response

// UsageResponse is the serialized JSON Response for an account usage
// to be returned by usage API
type UsageResponse struct {
	PrincipalID  string  `json:"principalId"`  // User Principal ID
	AccountID    string  `json:"accountId"`    // AWS Account ID
	StartDate    int64   `json:"startDate"`    // Usage start date Epoch Timestamp
	EndDate      int64   `json:"endDate"`      // Usage ends date Epoch Timestamp
	CostAmount   float64 `json:"costAmount"`   // Cost Amount for given period
	CostCurrency string  `json:"costCurrency"` // Cost currency
	TimeToLive   int64   `json:"timeToLive"`   // ttl attribute
}
