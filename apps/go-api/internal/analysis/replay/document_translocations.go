package replay

// document_translocations.go — LES TÉLÉPORTATIONS DU TRANSLOCATEUR, datées ET SITUÉES par
// l'événement du film (schéma 38, décision D4 du PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03).
//
// CE QUE C'EST. Chaque usage du translocateur émet un événement type 117 dans le film
// (filmdec/transloc_events.go — précision 18/18, rappel 8/8 sur 5 films, rapport R1). Ce
// calque le publie sur l'axe de frames : QUI saute, QUAND, et D'OÙ À OÙ — le va-et-vient est
// dans la CHARGE de l'événement (position A = départ, position B = arrivée, lues dans
// l'exécutable et validées à 0,00-0,26 m sur les 18 événements des 5 films, rapport R6 §1).
// Le client en tire l'éclat de fiche, le lien du va-et-vient et la faille dessinée APRÈS le
// premier échange — sans plus jamais passer par le `spent`, qui peut suivre l'usage de
// 16,5 s (mesure des modes de fin, plan §Découvertes), ni par une discontinuité de piste
// (qui n'est lisible qu'à la frame près, et pas du tout quand la vie n'est pas publiée).
//
// CE QUE ÇA NE DONNE PAS. La position de la faille AVANT le premier échange : entre la pose
// et lui, aucun canal ne la porte (R1 §1-3 : la faille activée n'est pas une entité
// répliquée lisible ; R6 §1.4 : la charge du 117 ne porte rien d'autre que {effet, départ,
// arrivée}). APRÈS le premier échange, en revanche, la balise est au point de DÉPART du saut
// — c'est la sémantique du va-et-vient, et c'est ce que `fx/fy/fz` donne. Le RANG du
// translocateur n'intervient pas ici : l'événement EST la mesure, la palette du titre nomme
// le rang ailleurs (`abilityLabels`).
//
// PAS DE BORNES DE CARTE -> PAS DE POSITIONS. Une charge non conforme, une carte hors
// catalogue, une région que le catalogue ne décrit pas : la téléportation se publie avec son
// instant et son slot, SANS le va-et-vient, et `coverage.translocations.positioned` compte
// celles qui l'ont. Jamais une position devinée (règle filmdec/map_bounds.go).

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
	// FX/FY/FZ et TX/TY/TZ sont le DÉPART et l'ARRIVÉE du saut en coordonnées monde (mêmes
	// axes que Point.X/Y/Z), lus dans la charge de l'événement. Z est gratuit (le même champ
	// le porte) et publié pour la cohérence d'étage, comme GrappleLine.AZ.
	//
	// POINTEURS, PAS float32, ET SOLIDAIRES. Une coordonnée de zéro est une valeur, et
	// omitempty l'effacerait (même piège qu'OriginMs) ; surtout, les six sont armés
	// ENSEMBLE ou pas du tout — absents, la téléportation reste vraie mais son va-et-vient
	// n'est pas connu, et un client qui lirait six zéros dessinerait un saut vers l'origine
	// du monde. `coverage.translocations.positioned` dit combien en portent.
	FX *float32 `json:"fx,omitempty"`
	FY *float32 `json:"fy,omitempty"`
	FZ *float32 `json:"fz,omitempty"`
	TX *float32 `json:"tx,omitempty"`
	TY *float32 `json:"ty,omitempty"`
	TZ *float32 `json:"tz,omitempty"`
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
	// Positioned compte, PARMI les publiées, celles qui portent leur va-et-vient. La
	// différence avec Published est la DÉGRADATION HONNÊTE (charge non conforme, carte hors
	// catalogue, région étrangère) : sans ce compteur, un client verrait des téléportations
	// sans saut et ne saurait pas si le film ou la lecture est en cause.
	Positioned int `json:"positioned"`
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
		tr := Translocation{T: int((e.TimestampUS - origin) / step), Slot: e.Slot}
		if e.HasPositions {
			setTranslocationJump(&tr, e.From, e.To)
			cov.Positioned++
		}
		out = append(out, tr)
		cov.Published++
	}
	return out, cov
}

// setTranslocationJump arme les SIX coordonnées d'un coup : elles sont solidaires (cf. le
// contrat de Translocation), et un demi-saut publié serait pire qu'un saut sans position.
func setTranslocationJump(tr *Translocation, from, to [3]float32) {
	tr.FX, tr.FY, tr.FZ = coordPtr(from[0]), coordPtr(from[1]), coordPtr(from[2])
	tr.TX, tr.TY, tr.TZ = coordPtr(to[0]), coordPtr(to[1]), coordPtr(to[2])
}

// coordPtr arrondit au centimètre (coordScale, comme les points de piste) et rend un
// POINTEUR — cf. le piège omitempty documenté sur Translocation.
func coordPtr(v float32) *float32 {
	r := round2(v)
	return &r
}

// logTranslocationCoverage sort la couverture du calque — un journal qui ne dirait que les
// publiées laisserait croire qu'elles sont toutes là, et qu'elles portent toutes leur saut.
func logTranslocationCoverage(cov TranslocationCoverage) {
	slog.Info("rejeu : teleportations du translocateur",
		"evenements", cov.Events, "publiees", cov.Published, "positionnees", cov.Positioned,
		"sansPosition", cov.Published-cov.Positioned,
		"avantOrigine", cov.BeforeOrigin, "sansPiste", cov.Unpublished)
}
