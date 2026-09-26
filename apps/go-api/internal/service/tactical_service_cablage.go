package service

// tactical_service_cablage.go — LES INJECTEURS CHAINABLES DU SERVICE TACTIQUE.
//
// Extrait de tactical_service.go le 2026-09-07 (lot 7C), quand le retour de la table du radar
// l'a repousse au-dela du seuil de 500 lignes. La coupure suit une frontiere nette : LA-BAS ce
// que le service FAIT, ICI ce dont il a besoin pour le faire.
//
// # POURQUOI DES `With*` ET NON DES PARAMETRES DE CONSTRUCTEUR
//
// Aucune de ces dependances ne sert a TOUTES les lectures : le lecteur de sidecars ne sert
// qu'aux lectures d'artefact, la table du radar qu'a l'isolement, les callouts qu'au nommage des
// grappes. Les mettre au constructeur ferait porter cinq dependances a vingt sites de test qui
// n'en ont aucun besoin. L'oubli de cablage, lui, ne peut pas passer inapercu : chaque lecture
// qui depend d'une absente rend un 503 ou un compte a part, ET une ligne de journal nominative.

import (
	"levelup/go-api/internal/port"
)

// WithRasterStore injecte le lecteur de sidecars d'occupation. Chainable.
//
// UN `With*` PLUTOT QU'UN 4e PARAMETRE, et la raison n'est pas la commodite : ce lecteur
// ne sert QU'A la lecture d'occupation, qui est la seule des quatre a ne pas venir de la
// base. Le mettre au constructeur aurait fait porter une dependance de fichier a vingt
// sites de test qui n'en ont aucun besoin. L'oubli de cablage, lui, ne peut pas passer
// inapercu : il rend un 503 ET une ligne ERROR nominative (cf. lectureOccupation).
func (s *TacticalService) WithRasterStore(store port.TacticalRasterStore) *TacticalService {
	s.rasters = store
	return s
}

// WithCalloutsStore injecte le lecteur de zones nommees (nommage des grappes). Chainable.
func (s *TacticalService) WithCalloutsStore(store port.TacticalCalloutsStore) *TacticalService {
	s.callouts = store
	return s
}

// WithRadarRange injecte la table des portees de radar du titre. Chainable.
func (s *TacticalService) WithRadarRange(parVariante map[string]int) *TacticalService {
	s.radar = parVariante
	return s
}

// WithRetentionMois injecte la fenetre de retention des artefacts de rejeu. Chainable.
//
// UNE FONCTION ET NON UNE VALEUR : la fenetre est un REGLAGE, relu a chaque lecture comme
// le cron de purge le relit a chaque tick. Une valeur figee au cablage aurait fait diverger
// ce que la page annonce de ce que la purge applique, jusqu'au prochain redemarrage.
func (s *TacticalService) WithRetentionMois(f func() int) *TacticalService {
	s.retentionMois = f
	return s
}
