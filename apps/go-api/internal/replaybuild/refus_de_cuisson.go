package replaybuild

// refus_de_cuisson.go — LES TROIS REFUS DE LA CUISSON, ET LA POLITIQUE DE CLÉ INCONNUE
// (lot 3.1.1, D-4 d'ADR 0034, décision utilisateur V20 (2) du 2026-09-17).
//
// # POURQUOI UN FICHIER, ET POURQUOI IL EMPORTE LES DEUX SENTINELLES EXISTANTES
//
// `replaybuild.go` est au PLAFOND GELÉ du ratchet de taille (577 lignes,
// `archlint/film_file_size_test.go`), et la règle du dépôt est qu'une dette gelée ne s'accroît
// pas : le remède que le ratchet lui-même prescrit est la scission par déplacement pur. Les
// deux sentinelles y sont donc DÉPLACÉES telles quelles — mêmes noms, mêmes valeurs, mêmes
// commentaires — et rejoignent la troisième, que ce lot ajoute.
//
// Les trois disent la même chose sous trois causes : CE FILM NE SE CUIT PAS, et ce n'est pas une
// panne. Les rassembler rend lisible ce que les producteurs classent en « écarté » plutôt qu'en
// « échec » (`filmproc.CodeSkipped`, `sync/replayartifacts`).

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// ErrMapNotInCatalog : aucune identité de carte candidate ne résout dans le catalogue de
// bornes de déquantification. ÉCHEC VOULU, compté À PART par les backfills : construire
// avec les bornes d'une autre carte donnerait des coordonnées fausses d'un facteur
// d'échelle arbitraire (cf. cmd/mapquant-build). Cas nominal : cartes Forge, dont le
// canevas n'est pas la carte.
var ErrMapNotInCatalog = errors.New("replaybuild: carte hors catalogue de bornes")

// ErrNoTracks : le décodage n'a produit aucune trajectoire — l'artefact n'est PAS écrit
// (un document vide se servirait comme un rejeu « propre » alors que le film est vide ou
// illisible).
var ErrNoTracks = errors.New("replaybuild: aucune trajectoire décodée — artefact non écrit")

// ErrUnknownFilmKey : la clé que le film ÉCRIT (format, build, version majeure) est absente de
// la table de profil du décodeur. Le film est MIS DE CÔTÉ : aucun artefact n'est cuit, aucun
// octet n'est rendu.
//
// MÊME FAMILLE QU'`ErrMapNotInCatalog` : le film est là et lisible, mais le dépôt n'a pas encore
// la ligne de table qui dit COMMENT le lire (`docs/RUNBOOK_FILM_PROFILES.md`). Le cuire au
// profil du build le plus proche produirait un document plausible et faux — le seul défaut que
// D-4 déclare inacceptable. Les producteurs le classent donc en ÉCARTÉ
// (`filmproc.CodeSkipped`), jamais en échec de traitement : le match revient de lui-même au
// rattrapage le jour où la ligne est écrite.
var ErrUnknownFilmKey = errors.New("replaybuild: clé du film absente de la table de profil — film mis de côté")

// ecarterSiCleInconnue applique la politique : elle rend [ErrUnknownFilmKey] enveloppée avec la
// clé refusée, ou nil.
//
// C'EST L'ORCHESTRATEUR QUI JOURNALISE ET QUI COMPTE, jamais la couche : D-4 le dit mot pour
// mot. Le VERDICT, lui, vient de la porte partagée `replay.CleDuFilm` — la même que
// `sync/killcollector` interroge. Deux gardes recopiées divergeraient au premier build ajouté,
// et le parc porterait des artefacts sans faits killsource (ou l'inverse) sans que rien ne le
// dise.
//
// FONCTION DE PAQUET ET NON MÉTHODE : elle ne lit rien du `Builder`. `BuildBytes` l'appelle
// juste après le chargement du film et AVANT la première lecture — un film écarté ne doit ni
// faire lire le fil des morts, ni faire décoder la killsource, ni faire cuire quoi que ce soit.
func ecarterSiCleInconnue(ctx context.Context, matchID string, film *decfilm.Film) error {
	cle := replay.CleDuFilm(film)
	if !cle.Refusee() {
		return nil
	}
	replay.PublierCleInconnue(cle)
	slog.WarnContext(ctx, "cuisson: film ECARTE — la clé écrite dans le film est absente de la "+
		"table de profil ; aucun artefact n'est cuit (ajouter la ligne : "+
		"docs/RUNBOOK_FILM_PROFILES.md)",
		"match_id", matchID, "cle", cle.Ecrite, "format", cle.Format, "build", cle.Build,
		"majeure", cle.Majeure, "err", cle.Err)
	return fmt.Errorf("%w (match %s, clé %s) : %w", ErrUnknownFilmKey, matchID, cle.Ecrite, cle.Err)
}
