package grammar

// tete_du_payload_test.go — LES DECODEURS D EVENEMENT DE TETE SUR DES PAYLOADS FABRIQUES : la tete
// se lit hors de la structure ([teteDuPayload]), comme la passe des tetes la rangerait.

import "levelup/go-api/internal/games/halo_infinite/film/types"

// decodeVehicleEventDuPayload est [decodeVehicleEvent] sous la tete du payload.
func decodeVehicleEventDuPayload(pay []byte, base uint32, inBand SlotBand) (types.VehicleEvent, bool) {
	return decodeVehicleEvent(pay, base, inBand, teteDuPayload(pay))
}

// decodeBipedPickupDuPayload est [decodeBipedPickup] sous la tete du payload.
func decodeBipedPickupDuPayload(pay []byte, st *types.BipedPickupStats) (types.BipedPickup, bool) {
	return decodeBipedPickup(pay, st, teteDuPayload(pay))
}
