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

import "path/filepath"

// SousDossierRasters est le segment de chemin des sidecars de raster, SOUS le dossier des
// artefacts d'un titre.
//
// EXPORTE parce qu'il a un second consommateur : la purge recurrente
// (`scheduler.replay_purge_cron`) supprime le sidecar d'un artefact qu'elle efface, et
// elle ne connait que le DOSSIER d'artefacts — jamais la racine du depot, donc jamais un
// PathResolver. Recopier « rasters » chez elle aurait fait deux definitions du meme
// segment, dont l'une aurait cesse de trouver l'autre au premier renommage.
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
// ForTitle) ne comptent QUE les fichiers `.json` de premier niveau et sautent les
// repertoires (`e.IsDir()`). Un sidecar pose a plat aurait donc ete lu comme l'artefact
// d'un match inexistant par le premier, et supprime comme un artefact indatable par le
// second.
func (p *PathResolver) TacticalRasterDir(titleSlug string) string {
	return filepath.Join(p.ReplayArtifactsDir(titleSlug), SousDossierRasters)
}

// TacticalRasterPath retourne le chemin du sidecar de raster tactique d'un match.
// Ex: data/cache/replays/halo_infinite/rasters/000d5950.json
//
// MEME CLE QUE L'ARTEFACT : la forme COURTE (cf. FilmShortMatchID), donc le match_id
// complet et sa forme courte donnent le MEME chemin. C'est ce qui permet au rattrapage
// de nommer un sidecar depuis un simple listing du dossier d'artefacts, et a la lecture
// de le retrouver depuis le match_id complet du registre.
func (p *PathResolver) TacticalRasterPath(titleSlug, matchID string) string {
	return filepath.Join(p.TacticalRasterDir(titleSlug), FilmShortMatchID(matchID)+".json")
}
