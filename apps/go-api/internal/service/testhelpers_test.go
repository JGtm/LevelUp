package service

// Test helpers shared across service test files.

func strPtr(s string) *string       { return &s }
func intPtr(i int) *int             { return &i }
func float64Ptr(f float64) *float64 { return &f }
