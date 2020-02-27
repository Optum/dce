package main

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}
