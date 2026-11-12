package migration

import "testing"

// canonicalOrderFloor : longueur de canonicalOrder figée le 2026-11-01. Elle ne peut que
// MONTER : la garde de version du démarrage (version_guard.go) compare la version enregistrée
// en base (= longueur de l'ordre canonique au dernier démarrage) à la longueur du binaire ;
// une liste qui raccourcit (squash, retrait d'une étape obsolète) ferait lire « base plus
// récente que le binaire » sur TOUTES les installations et refuserait leur démarrage. Un
// squash exige d'abord une révision de la garde (version explicite et monotone, ADR à écrire),
// pas une baisse de ce plancher. Ajouter une étape : relever le plancher à la nouvelle longueur.
const canonicalOrderFloor = 237

func TestCanonicalOrderNeRaccourcitJamais(t *testing.T) {
	got := len(canonicalOrder)
	if got < canonicalOrderFloor {
		t.Fatalf("canonicalOrder compte %d étapes, plancher %d : une liste qui raccourcit fait refuser le démarrage de toutes les bases existantes (schema_newer) ; réviser la garde de version avant tout squash", got, canonicalOrderFloor)
	}
	if got > canonicalOrderFloor {
		t.Fatalf("canonicalOrder compte %d étapes, plancher %d : relever canonicalOrderFloor à %d dans ce commit", got, canonicalOrderFloor, got)
	}
}
