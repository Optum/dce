package main

var standardHeaders = map[string][]string{
	"Access-Control-Allow-Origin": []string{"*"},
	"Content-Type":                []string{"application/json"},
}

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptr64(i int64) *int64 {
	ptrI := i
	return &ptrI
}
