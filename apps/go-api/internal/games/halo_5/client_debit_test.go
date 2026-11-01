package halo_5

// client_debit_test.go — le débit du client Halo 5 : 10 requêtes par seconde quand l'appelant n'en
// donne pas (<= 0), sinon celui qu'il donne, avec une rafale de 1.

import (
	"testing"

	"golang.org/x/time/rate"
)

func TestNewClient_DebitParDefaut(t *testing.T) {
	for nom, c := range map[string]struct {
		rps   int
		limit rate.Limit
	}{
		"zéro : débit par défaut":    {0, 10},
		"négatif : débit par défaut": {-5, 10},
		"débit donné par l'appelant": {3, 3},
	} {
		t.Run(nom, func(t *testing.T) {
			l, ok := NewClient("jeton", c.rps).api.Limiter().(*rate.Limiter)
			if !ok || l == nil {
				t.Fatalf("limiteur du client = %T, attendu un *rate.Limiter", NewClient("jeton", c.rps).api.Limiter())
			}
			if l.Limit() != c.limit || l.Burst() != 1 {
				t.Errorf("Limit() = %v, Burst() = %d, attendu %v et 1", l.Limit(), l.Burst(), c.limit)
			}
		})
	}
}
