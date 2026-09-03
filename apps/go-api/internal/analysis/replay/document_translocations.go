package replay

// document_translocations.go — LES TÉLÉPORTATIONS DU TRANSLOCATEUR, datées par l'événement
// du film (schéma 38, décision D4 du PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03).
//
// CE QUE C'EST. Chaque usage du translocateur émet un événement type 117 dans le film
// (filmdec/transloc_events.go — précision 18/18, rappel 8/8 sur 5 films, rapport R1). Ce
// calque le publie sur l'axe de frames : QUI saute, et QUAND. Le client en tire l'éclat de
// fiche, le lien du va-et-vient (from/to dérivés de la discontinuité de sa piste à cette
// frame) et la faille dessinée APRÈS le premier échange — sans plus jamais passer par le
// `spent`, qui peut suivre l'usage de 16,5 s (mesure des modes de fin, plan §Découvertes).
//
// CE QUE ÇA NE DONNE PAS. La POSITION de la faille : entre la pose et le premier échange,
// aucun canal ne la porte (R1 §1-3 : la faille activée n'est pas une entité répliquée
// lisible). Et le RANG du translocateur n'intervient pas ici : l'événement EST la mesure,
// la palette du titre nomme le rang ailleurs (`abilityLabels`).

import (
	"log/slog"

	"levelup/go-api/internal/analysis/filmdec"
)

// Translocation est UNE téléportation exécutée, sur l'axe de frames du document.
type Translocation struct {
	// T est l'index de frame, sur le même axe que Point.T. L'événement précède la
	// discontinuité de position de 5 à 80 ms : à la grille de 100 ms, l'arrivée est publiée
	// à T ou T+1 (mesure R3 §2 : +0 ou +1 frame sur 18/18).
	T int `json:"t"`
	// Slot désigne la Track du bipède qui se téléporte — une VIE, pas un joueur.
	Slot uint32 `json:"slot"`
}

// TranslocationCoverage dit ce que le calque a vu et ce qu'il a écarté — le patron des
// autres canaux d'événements.
type TranslocationCoverage struct {
	// Events est le nombre de têtes 117 décodées du film.
	Events int `json:"events"`
	// Published est le nombre publié dans le document.
	Published int `json:"published"`
	// BeforeOrigin compte les événements antérieurs à la première frame — écartés.
	BeforeOrigin int `json:"beforeOrigin"`
	// Unpublished compte les événements dont le slot n'a pas de trajectoire publiée —
	// écartés (le client n'aurait aucune piste où poser le saut).
	Unpublished int `json:"unpublished"`
}

// buildTranslocations projette les téléportations lues dans le film sur l'axe de frames.
// Même règle que les autres calques : rien avant l'origine, rien sans piste publiée.
func buildTranslocations(
	evts []filmdec.TranslocatorTeleport, tracks []Track, origin, step uint64,
) ([]Translocation, TranslocationCoverage) {
	cov := TranslocationCoverage{Events: len(evts)}
	if len(evts) == 0 || step == 0 {
		return nil, cov
	}
	published := publishedSlots(tracks)
	var out []Translocation
	for _, e := range evts {
		if e.TimestampUS < origin {
			cov.BeforeOrigin++
			continue
		}
		if !published[e.Slot] {
			cov.Unpublished++
			continue
		}
		out = append(out, Translocation{T: int((e.TimestampUS - origin) / step), Slot: e.Slot})
		cov.Published++
	}
	return out, cov
}

// logTranslocationCoverage sort la couverture du calque — un journal qui ne dirait que les
// publiées laisserait croire qu'elles sont toutes là.
func logTranslocationCoverage(cov TranslocationCoverage) {
	slog.Info("rejeu : teleportations du translocateur",
		"evenements", cov.Events, "publiees", cov.Published,
		"avantOrigine", cov.BeforeOrigin, "sansPiste", cov.Unpublished)
}
