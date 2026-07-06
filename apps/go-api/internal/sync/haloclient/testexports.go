package haloclient

import (
	"context"

	"golang.org/x/time/rate"
)

// testexports.go — accesseurs exportés des internes rate-limiting de HaloAPIClient,
// pour les tests du câblage POOLED qui vivent dans le package sync (pooled_client
// enveloppe HaloAPIClient). Depuis l'extraction du client (K3e), ces tests ne
// peuvent plus toucher `limiter`/`rateWait` non-exportés — réservé aux tests.

// LimiterForTest expose le rate.Limiter interne (vérif du câblage fallback/lease).
func (c *HaloAPIClient) LimiterForTest() *rate.Limiter { return c.limiter }

// RateWaitForTest exerce le rate-limiting interne.
func (c *HaloAPIClient) RateWaitForTest(ctx context.Context) { c.rateWait(ctx) }
