package title

// registry_tactical.go — LES CHEMINS DE L'ONGLET TACTIQUE.
//
// Extrait de registry.go le 2026-09-06 (constat C8 de la revue de la phase 6). Ce
// fichier-la porte 957 lignes : c'est une dette de seuil GELEE par la baseline, et la
// regle du depot est de ne pas l'accroitre (CLAUDE.md n 5). Le lot invoquait deja cette
// regle pour ne pas grossir `domain/tactical.go` ; il l'appliquait a une couche et pas a
// l'autre.
//
// Le paquet est le meme : `PathResolver` reste UN type, avec UNE racine, et rien de ce qui
// suit ne peut diverger de ses voisins.

import (
	"path/filepath"
	"strings"
)

// SousDossierRasters est le segment de chemin des sidecars de raster, SOUS le dossier des
// artefacts d'un titre.
//
// EXPORTE parce que la purge recurrente (`scheduler.replay_purge_cron`) ne connait que le
// DOSSIER d'artefacts — jamais la racine du depot, donc jamais un PathResolver : elle passe par
// [TacticalRasterFileIn] et [TacticalRasterLegacyFileIn], qui portent ce segment.
const SousDossierRasters = "rasters"

// TacticalRasterDir retourne le dossier des SIDECARS de raster tactique d'un titre — la
// projection « ou chaque joueur a passe son temps » d'un match, calculee UNE FOIS a la
// cuisson (cf. internal/sync/replayartifacts/raster.go).
// Ex: data/cache/replays/halo_infinite/rasters/
//
// # POURQUOI SOUS LE DOSSIER DES ARTEFACTS, ET EN SOUS-DOSSIER
//
// Un raster est un DERIVE de l'artefact et n'a aucun sens sans lui : le ranger a cote
// garde les deux ensemble (une purge de titre, une copie de poste, un diagnostic). Le
// SOUS-dossier, lui, est ce qui rend la cohabitation sure — les deux parcours du dossier
// d'artefacts (service.replayService.AvailableSet et scheduler.purgeReplayArtifacts-
// ForTitle) ne comptent QUE les artefacts de premier niveau et sautent les
// repertoires (`e.IsDir()`). Un sidecar pose a plat aurait donc ete lu comme l'artefact
// d'un match inexistant par le premier, et supprime comme un artefact indatable par le
// second.
func (p *PathResolver) TacticalRasterDir(titleSlug string) string {
	return filepath.Join(p.ReplayArtifactsDir(titleSlug), SousDossierRasters)
}

// TacticalRasterPath retourne le chemin du sidecar de raster tactique d'un match.
// Ex: data/cache/replays/halo_infinite/rasters/000d5950.json.gz
//
// MEME CLE QUE L'ARTEFACT : la forme COURTE (cf. FilmShortMatchID), donc le match_id
// complet et sa forme courte donnent le MEME chemin. C'est ce qui permet au rattrapage
// de nommer un sidecar depuis un simple listing du dossier d'artefacts, et a la lecture
// de le retrouver depuis le match_id complet du registre.
func (p *PathResolver) TacticalRasterPath(titleSlug, matchID string) string {
	return TacticalRasterFileIn(p.ReplayArtifactsDir(titleSlug), FilmShortMatchID(matchID))
}

// TacticalRasterLegacyPath retourne le chemin du fichier tactique d'une version ancienne
// (`rasters/{short8}.json`, JSON brut) d'un match. Aucune lecture ne le sert : seules la purge et la
// reecriture du fichier courant le retirent.
func (p *PathResolver) TacticalRasterLegacyPath(titleSlug, matchID string) string {
	return TacticalRasterLegacyFileIn(p.ReplayArtifactsDir(titleSlug), FilmShortMatchID(matchID))
}

// TacticalRasterFileIn rend le chemin du fichier tactique de la forme courte `short` sous le dossier
// d'artefacts `artifactsDir`. C'est la seule construction de ce chemin : [PathResolver.TacticalRasterPath]
// la porte, et la purge recurrente, qui ne tient que le dossier, l'appelle directement.
func TacticalRasterFileIn(artifactsDir, short string) string {
	return filepath.Join(artifactsDir, SousDossierRasters, short+ReplaySidecarExt)
}

// TacticalRasterLegacyFileIn est le pendant de [TacticalRasterFileIn] pour le fichier d'une version
// ancienne ([ReplaySidecarLegacyExt]).
func TacticalRasterLegacyFileIn(artifactsDir, short string) string {
	return filepath.Join(artifactsDir, SousDossierRasters, short+ReplaySidecarLegacyExt)
}

// TacticalRasterLegacyOf rend, d'un chemin de fichier tactique courant (`{short8}.json.gz`), le chemin
// du fichier d'une version ancienne qui lui est voisin. Un chemin qui n'a pas l'extension courante est
// rendu vide : il n'a pas de voisin ancien.
func TacticalRasterLegacyOf(path string) string {
	stem, ok := strings.CutSuffix(path, ReplaySidecarExt)
	if !ok {
		return ""
	}
	return stem + ReplaySidecarLegacyExt
}

// SpawnModelPath retourne le chemin du fichier des POIDS DU MODELE DE REAPPARITION d'un titre
// (cf. domain.PoidsReapparition) : donnee de REFERENCE versionnee, ecrite hors serveur par
// `levelup spawn-model fit` et relue telle quelle par le service.
// Ex: data/titles/halo_infinite/reference/spawn_model.json
func (p *PathResolver) SpawnModelPath(titleSlug string) string {
	return filepath.Join(p.referenceDir(titleSlug), "spawn_model.json")
}
