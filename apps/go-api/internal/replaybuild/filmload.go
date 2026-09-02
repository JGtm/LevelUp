package replaybuild

// filmload.go — LE FILM EST OUVERT, DECOMPRESSE ET LU UNE SEULE FOIS PAR CUISSON.
//
// # CE QUE CE FICHIER FERME
//
// Avant le lot 1 de PLAN_CUISSON_PERF (2026-09-02), une cuisson relisait et redecompressait le
// film ENTIER une trentaine de fois : chaque `ScanFilm*` de `BuildFromFilm` ouvrait le
// repertoire de chunks pour son propre compte, et le chunk highlight etait parse trois fois (les
// actions d'objectif, les frags sous effet, le fil des morts du document). Le decodage pesait
// ~94 % du temps de cuisson (mesure 0.8, §1 de MESURES_CUISSON_PERF.md).
//
// Ici, trois lectures, une fois chacune :
//
//	le MANIFESTE   ouvrirManifeste  — l'index des chunks (type, debut) ; il sert au statborg
//	               (`objectiveevents.StatRecordsCtx`) ET aux metadonnees du film ;
//	les CHUNKS     chargerFilm      — decompresses et decoupes en paquets par `filmsource` ;
//	les MORTS      lireMorts        — le chunk highlight, parse UNE fois pour les deux
//	               consommateurs de cet etage.
//
// # POURQUOI LA TRADUCTION DU MANIFESTE VIT ICI
//
// `filmsource` est un paquet FEUILLE : il n'importe rien du depot (garde-rail
// `archlint/filmsource_leaf_test.go`), donc il ne connait ni `filmcache` ni le format du
// manifeste. C'est cette couche d'ASSEMBLAGE qui sait ou vit le cache du titre — la meme
// frontiere que pour les faits de match. La traduction tient en une boucle, et elle est le prix
// a payer pour que `filmsource` reste importable par `filmdec`, `killsource` et
// `objectiveevents` sans cycle.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// ouvrirManifeste ouvre le manifeste du film et JOURNALISE ce qu'il en est.
//
// nil = pas de manifeste exploitable, et les deux causes gardent leur niveau d'origine : un
// manifeste PRESENT mais illisible est un Warn (quelque chose est casse), un film ABSENT du
// cache est un Info (le cache est local et partiel, c'est le cas nominal). Le document sort
// alors sans courbe de score ET sans couverture de score, ce qui dit « rien n'a ete lu » plutot
// que « rien n'existait ».
func ouvrirManifeste(ctx context.Context, matchID, filmDir string) *filmcache.Source {
	src, found, err := filmcache.OpenChunkDir(filmDir)
	switch {
	case err != nil:
		slog.WarnContext(ctx, "replaybuild: manifeste de film illisible — rejeu sans courbe de score",
			"err", err, "match_id", matchID, "filmDir", filmDir)
		return nil
	case !found:
		slog.InfoContext(ctx, "replaybuild: film sans manifeste au cache — rejeu sans courbe de score",
			"match_id", matchID, "filmDir", filmDir)
		return nil
	}
	return src
}

// chargerFilm decompresse les chunks du film et decoupe leurs paquets, UNE fois.
//
// nil N'EST PAS FATAL ICI, et le refus arrive au bon endroit : chaque balayage de
// `BuildFromFilm` rend alors son erreur « aucun chunk de donnees dans le film » a sa place, avec
// son propre journal, exactement comme un repertoire vide le faisait avant. Echouer ici priverait
// la cuisson des etapes qui ne dependent pas du film (catalogues, killsource) et changerait
// l'ordre des etapes observees.
func chargerFilm(ctx context.Context, matchID, filmDir string, src *filmcache.Source) *filmsource.Film {
	film, err := filmsource.LoadDir(filmDir, metaDuManifeste(src))
	if err != nil {
		slog.WarnContext(ctx, "replaybuild: chunks du film illisibles — aucun balayage ne lira ce film",
			"err", err, "match_id", matchID, "filmDir", filmDir)
		return nil
	}
	return film
}

// metaDuManifeste traduit l'index du manifeste dans la forme de `filmsource` (cf. l'en-tete).
// Manifeste absent : nil — `filmsource.LoadDir` synthetise alors les numeros de chunk depuis les
// noms de fichiers, ce qui suffit a tous les balayages (seul le start_ms par chunk, que
// l'armement de la bombe demande, vient du manifeste, et il passe par un autre chemin).
func metaDuManifeste(src *filmcache.Source) []filmsource.ChunkMeta {
	if src == nil {
		return nil
	}
	chunks := src.Chunks()
	out := make([]filmsource.ChunkMeta, 0, len(chunks))
	for _, c := range chunks {
		out = append(out, filmsource.ChunkMeta{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS,
		})
	}
	return out
}

// filmDeaths porte L'UNIQUE lecture du fil des morts de cet etage, et SON ERREUR.
//
// L'ERREUR VOYAGE AVEC LA LISTE, et il le faut : les deux consommateurs (`identifiedEvents` et
// `killRefs`) ne journalisent pas la meme chose ni au meme niveau quand le fil manque — l'un
// perd les actions d'objectif, l'autre les frags sous effet actif. Rendre une liste vide sans
// l'erreur aurait fusionne « fil illisible » et « film sans mort », qui ne sont pas le meme fait.
type filmDeaths struct {
	list []replay.Death
	err  error
}

// lireMorts lit le fil des morts du film charge. Le chunk highlight est deja decompresse : ce
// qui reste est le parse, fait une fois pour les deux consommateurs.
func lireMorts(film *filmsource.Film) filmDeaths {
	list, err := replay.ScanDeaths(film)
	return filmDeaths{list: list, err: err}
}
