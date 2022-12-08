package response

// LeaseAuthResponse is the structured JSON Response for an Lease
// to be returned for APIs
//
//	{
//		"accessKeyId": "AKIAI44QH8DHBEXAMPLE",
//		"secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
//		"sessionKey": "AQoDYXdzEJr...",
//		"consoleUrl": "https://aws.amazon.com/console/",
//	}
type LeaseAuthResponse struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
	ConsoleURL      string `json:"consoleUrl"`
}
