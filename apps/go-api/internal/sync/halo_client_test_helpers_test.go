package sync

import "strings"

// contains est un helper de test (sous-chaîne). Dupliqué de haloclient
// (halo_client_film.go) après extraction du client (K3e) : engine_mock_test.go
// l'utilise et le sous-package de test n'est plus partagé.
func contains(s, sub string) bool { return strings.Contains(s, sub) }

// isNotFoundErr est un helper de test dupliqué de haloclient (halo_client_film.go)
// après extraction du client (K3e). sync_coverage_extra_test.go l'utilise.
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "HTTP 404") || contains(s, "HTTP 410") || contains(s, "ressource absente")
}
