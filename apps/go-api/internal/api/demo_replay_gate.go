package api

import (
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/wire"
)

// demoReplayGate rend la liste blanche des rejeux figés de la démo, en INTERFACE nil hors
// démo : un pointeur nil typé ferait croire à handlers.ReplayGate qu'elle sert une démo.
func demoReplayGate(reg *wire.ServiceRegistry) handlers.DemoReplayAllowlist {
	if a := reg.DemoReplayAllowlist(); a != nil {
		return a
	}
	return nil
}
