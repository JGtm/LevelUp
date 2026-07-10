package teammates

// Helpers pointeurs pour les tests. Dupliqués de service/testhelpers_test.go
// (K3b : packages de test disjoints).
func strPtr(s string) *string       { return &s }
func intPtr(i int) *int             { return &i }
func float64Ptr(f float64) *float64 { return &f }
