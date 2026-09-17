package replay

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// coverage_decoder.go — LE BLOC `coverage.decoder` : SOUS QUELLES REVISIONS CET ARTEFACT A ETE
// CUIT (schema 61, lot 2.6.3 du PLAN_DECODEUR_FILM).
//
// Fichier a part et non une section de `coverage.go` : celui-ci porte la couverture des CALQUES
// (ce qui a ete rattache sur ce qui existait) et frole son plafond de taille. Ce bloc-ci ne
// mesure rien du film — il DATE la cuisson.

// DecoderCoverage dit SOUS QUELLES RÉVISIONS cet artefact a été cuit (schéma 61, lot 2.6.3 du
// PLAN_DECODEUR_FILM, décision V15 (11)).
//
// # POURQUOI LES QUATRE, ET PAS UNE SEULE
//
// Le décodeur est cinq calques et quatre d'entre eux portent leur révision (ADR 0034, D-6). Une
// seule valeur ne dirait pas OÙ le changement a eu lieu — or c'est précisément ce que la
// recuisson sélective par calque (4.4) doit décider : une montée de `source` re-décode tout, une
// montée de `facts` ne concerne que les faits persistés. Les quatre sont donc publiées, dans
// l'ordre du sens unique.
//
// TÉLÉMÉTRIE PURE : aucun rendu n'en dépend, aucune décision de décodage n'en dépend. Le champ
// existe dès M2 et ses valeurs deviennent EXPLOITABLES à M4 (D-7 : « la présence d'un calque se
// lit dans sa révision, jamais dans l'absence d'un champ »).
type DecoderCoverage struct {
	// SourceRev : la porte aux octets (chargement, décompression, découpage, lecteur de bits).
	SourceRev string `json:"sourceRev"`
	// ProfileRev : la table du décodeur (largeurs, bornes, provenances par build et par carte).
	ProfileRev string `json:"profileRev"`
	// GrammarRev : la grammaire de lecture (cadres, ordres de composants, lecteurs).
	GrammarRev string `json:"grammarRev"`
	// FactsRev : la couche des faits — celle qui commande le backlog killsource.
	FactsRev string `json:"factsRev"`
	// Build est la CLÉ DU PROFIL, lue en clair dans la section 2 de `chunk_00` (ADR 0034, D-3).
	//
	// CHAÎNE VIDE ET BLOC PRÉSENT SUR UN BUILD INCONNU (décision V15 (15)). Un film dont
	// `chunk_00` ne porte aucune section d'identification — 5 films du cache, versions majeures
	// 31 et 33 — n'écrit pas de build ; un film dont le build est absent de la table de profil
	// rend `ErrUnknownBuild`. Dans les deux cas la chaîne est vide ET le bloc reste là : sans
	// cette règle, l'absence du bloc porterait DEUX sens — « cuit avant le schéma 61 » et
	// « build inconnu » — c'est-à-dire exactement l'ambiguïté « entre deux versions de schéma »
	// que D-7 interdit.
	Build string `json:"build"`
	// Registry CLASSE l'empreinte du registre ECS du film. Absent quand le registre n'a pas été
	// lu (film sans section d'identification, `chunk_00` tronqué, encore compressé).
	Registry *RegistryCoverage `json:"registry,omitempty"`
}

// RegistryCoverage CLASSE l'empreinte du registre ECS du film — la grammaire de ses composants
// (item 3.2.1 du PLAN_DECODEUR_FILM, volet code partiel ; lot 2.6.3).
//
// # LE DÉFAUT QU'IL FERME
//
// L'empreinte du registre est l'identité de la grammaire des composants : elle est bit-à-bit
// identique d'un film à l'autre DANS UN MÊME BUILD, et elle change d'un build à l'autre. Le
// décodeur n'en connaissait qu'UNE, en constante de paquet (`grammar.KnownRegistryFingerprint`,
// le build de référence) : tout le reste sortait en UNE ligne de journal « registre INCONNU »,
// dédupliquée par processus, et rien n'en restait dans l'artefact. Un film cuit sous une
// grammaire de composants jamais vue était donc indistinguable d'un film nominal.
//
// # DEUX STATUTS ICI, TROIS À TERME — ET LA RAISON EST MESURÉE
//
// Le catalogue `games/halo_infinite/filmprofile` porte une TABLE d'empreintes par clé écrite du
// film, avec le statut `connue` ou `presumee` par ligne. Le décodeur NE PEUT PAS la lire : ce
// serait un import d'un paquet de catalogue depuis un calque, ce que le sens unique interdit
// (ADR 0034, D-1). Le chemin prévu est que la table du PROFIL
// (`film/internal/profile/profile_table.go`) recopie ces empreintes, et que la classification se
// fasse là où l'empreinte est calculée — `grammar.ReadFilmIdentity`, décision D4 (3.2).
//
// CE CHEMIN N'EXISTE PAS ENCORE : mesure du 2026-09-17, `profile_table.go` ne porte AUCUNE
// empreinte de registre, et `TestCatalogueConformeALaTableDuLot21` tient les deux tables égales
// ligne pour ligne. Ce lot livre donc la classification sur la SEULE empreinte que le décodeur
// connaît — `connue` / `inconnue` — et le statut `presumee` naîtra avec la recopie, au volet
// 3.1.1. Le champ `status` est une chaîne et non un booléen précisément pour que ce troisième
// état n'oblige aucun consommateur à changer de forme.
type RegistryCoverage struct {
	// Fingerprint est l'empreinte FNV-1a 64 bits des entrées NOMMÉES, `0x` + 16 chiffres
	// minuscules — la même forme que le catalogue, pour que les deux se joignent sans conversion.
	Fingerprint string `json:"fingerprint"`
	// Status vaut `connue` quand l'empreinte est celle du build de référence, `inconnue` sinon.
	// Un troisième état `presumee` est prévu (cf. l'en-tête du type).
	Status string `json:"status"`
	// Blocks est le nombre de blocs d'archétype du registre (49 ou 50 selon le build).
	Blocks int `json:"blocks"`
	// NamedSlots est le nombre d'entrées NOMMÉES hachées — le dénominateur de l'empreinte. Sans
	// lui, une empreinte différente ne se distingue pas d'un registre tronqué.
	NamedSlots int `json:"namedSlots"`
}

// Les deux statuts que ce lot publie. Le troisième (`presumee`) naîtra avec la recopie des
// empreintes du catalogue dans la table du profil (volet 3.1.1).
const (
	// RegistryStatutConnue : l'empreinte est celle du build de référence du dépôt.
	RegistryStatutConnue = "connue"
	// RegistryStatutInconnue : l'empreinte n'est PAS celle du build de référence. Le film reste
	// décodé — l'empreinte est un signal, pas une porte.
	RegistryStatutInconnue = "inconnue"
)

// ProjectileCoverage est la couverture du calque des projectiles.
type ProjectileCoverage struct {
	// Tracks est le nombre de pistes DÉCODÉES — le dénominateur.
	Tracks int `json:"tracks"`
	// Published est le nombre de trajectoires publiées. L'écart avec Tracks tient aux pistes
	// trop courtes pour se dessiner (moins de deux points de grille) et à celles qui naissent
	// avant l'origine du rejeu.
	Published int `json:"published"`
	// Truncated est le nombre de trajectoires COUPÉES à un pas impossible (cf.
	// projectileMaxStepM). Elle compte aussi les coupures si précoces que la trajectoire n'est
	// plus publiable : sans cela le compteur mentirait par omission.
	Truncated int `json:"truncated"`
}

// slotFor rend le slot du joueur pi à l'instant tUS, et la cause du rejet le cas échéant.
//
// C'EST LA MÊME PORTE QUE `uniqueSlotFor`, mais elle DIT pourquoi elle se ferme. L'ancienne
// version rendait un booléen : « slot introuvable » et « slot ambigu » y étaient
// indiscernables, alors qu'ils désignent deux chantiers différents.
func slotFor(tracks map[uint32]slotTrack, owner map[uint32]int, pi int, tUS uint64) (uint32, rejectReason) {
	var found uint32
	n := 0
	for slot, idx := range owner {
		if idx != pi {
			continue
		}
		if _, d := tracks[slot].at(tUS); d <= shotPosToleranceUS {
			found = slot
			n++
		}
	}
	switch {
	case n == 1:
		return found, reasonAttached
	case n > 1:
		return 0, reasonAmbiguous
	default:
		return 0, reasonNoSlot
	}
}

// countUnpublished mesure, après filtrage, combien d'événements rattachés ont été retirés
// faute de trajectoire publiée. Rendu en tant que catégorie propre : ce n'est pas un échec
// du rattachement, et le confondre avec « slot introuvable » orienterait vers le mauvais
// chantier.
func countUnpublished(before, after int) int {
	if d := before - after; d > 0 {
		return d
	}
	return 0
}

// couvertureDuDecodeur rend le bloc `coverage.decoder` de CETTE cuisson.
//
// LES QUATRE REVISIONS SONT DES CONSTANTES DE COMPILATION : elles ne dependent d aucun film, donc
// elles sont TOUJOURS posees — un document assemble depuis des positions sans film les porte
// aussi. Seuls `build` et `registry` viennent du film, et leur absence a un sens ecrit :
// build vide sur un film sans section d identification (V15 (15)), bloc `registry` absent quand
// le registre n a pas ete lu.
func couvertureDuDecodeur(id *profile.FilmIdentity) *DecoderCoverage {
	cov := &DecoderCoverage{
		SourceRev:  source.Rev,
		ProfileRev: profile.Rev,
		GrammarRev: grammar.Rev,
		FactsRev:   facts.Rev,
	}
	if id == nil {
		return cov
	}
	cov.Build = id.Build
	if id.RegistryFingerprint == 0 {
		return cov
	}
	cov.Registry = &RegistryCoverage{
		Fingerprint: fmt.Sprintf("0x%016x", id.RegistryFingerprint),
		Status:      statutDuRegistre(id.RegistryFingerprint),
		Blocks:      id.RegistryBlocks,
		NamedSlots:  id.RegistryNamedSlots,
	}
	return cov
}

// statutDuRegistre CLASSE une empreinte de registre contre ce que le decodeur connait.
//
// DEUX ETATS ICI, TROIS A TERME : la table du PROFIL ne recopie pas encore les empreintes du
// catalogue (mesure du 2026-09-17), donc `presumee` n existe pas — cf. l en-tete de
// [RegistryCoverage]. Le jour ou elle les portera, cette fonction consultera la table et rendra
// le statut qu elle declare ; aucun consommateur n aura a changer de forme.
func statutDuRegistre(fp uint64) string {
	if fp == grammar.KnownRegistryFingerprint {
		return RegistryStatutConnue
	}
	return RegistryStatutInconnue
}

// identiteDuFilm rend la section 2 de `chunk_00` telle que le PROFIL du contexte l a deja lue,
// ou nil quand ce film n en porte pas.
//
// LE PROFIL LA TIENT DEJA : il est resolu a la construction du contexte (lot 2.1, D1), et sa
// resolution ouvre `chunk_00`, lit le registre et cherche l ancre de la chaine de build. La
// reprendre ici ne relit AUCUN octet — c est la propriete que le lot « resolu une fois » a
// achetee, et la raison pour laquelle ce bloc de couverture ne coute rien a la cuisson.
//
// NIL VEUT DIRE QUELQUE CHOSE, ET C EST ECRIT : le film ne porte AUCUNE section
// d identification — `chunk_00` tronque, encore compresse, ou sans section du tout (5 films du
// cache, majeures 31 et 33). Le bloc `coverage.decoder` reste alors PRESENT, avec un `build`
// VIDE (V15 (15)) et sans bloc `registry`.
//
// UN BUILD INCONNU DU PROFIL N EST PAS UN NIL, ET LA DISTINCTION EST LE POINT DE V15 (15). Le
// film a bien ECRIT sa cle, la table de profil ne la connait pas (`profile.ErrUnknownBuild`) :
// la chaine `build` est alors VIDEE — publier une cle que le profil ne sait pas appliquer
// laisserait croire qu elle a servi — mais tout ce qui a ete LU reste publie, le bloc `registry`
// en particulier. C est meme sur ces films-la qu il vaut le plus : une empreinte de registre
// `inconnue` est exactement ce qui explique un build inconnu.
func identiteDuFilm(fc *grammar.FilmContext) *profile.FilmIdentity {
	if fc == nil {
		return nil
	}
	p := fc.Profile()
	id := p.Identity()
	if id.Build == "" && id.RegistryFingerprint == 0 {
		return nil
	}
	if p.Err() != nil {
		id.Build = ""
	}
	return &id
}
