// cmd/replay-build — génère l'artefact de rejeu 2D d'un match à partir des SEULS CHUNKS
// DU FILM (décodage offline des positions bipeds, zéro entrée Cheat Engine), plus un fond
// de carte optionnel (props Forge). Écrit data/cache/replays/{title}/{matchId}.json
// (résolu par PathResolver).
//
// L'ASSEMBLAGE (catalogue de bornes + libellés + structure + écriture) vit dans
// `internal/replaybuild`, partagé avec `levelup backfill-replay`, l'action admin et
// l'étape post-sync — ce binaire n'est que son enveloppe CLI unitaire.
//
// Usage:
//
//	replay-build --map <nom de carte> [--title slug] [--interval MS] [--geometry DIR] [--facts FICHIER.json] [--cpuprofile FICHIER] [--memprofile FICHIER] <matchId> [filmDir]
//
// PIÈGE : le paquet flag arrête l'analyse au premier argument positionnel — les options
// doivent précéder <matchId>.
//
// MESURE (PLAN_CUISSON_PERF §6) : `LEVELUP_LOG_LEVEL=debug` fait apparaître la durée de chaque
// balayage (le binaire installe un handler slog, cf. logging.InstallCLILevel) ; `--cpuprofile`
// écrit un profil exploitable par `go tool pprof`. Les deux sont inertes par défaut.
//
// --map est OBLIGATOIRE : les bornes de déquantification sont propres à la carte (AABB de
// son BSP). Elles sont lues dans le catalogue versionné
// data/titles/{slug}/reference/map_quant_bounds.json (cmd/mapquant-build). Une carte
// absente du catalogue fait ÉCHOUER le build — un artefact aux bornes d'une autre carte
// serait faux d'un facteur d'échelle arbitraire, sans que rien ne le signale.
//
// Exemple (depuis apps/go-api, cache film dans le main tree) :
//
//	CGO_ENABLED=0 LEVELUP_REPO_ROOT=<repo> go run ./cmd/replay-build --map Cliffhanger 000d5950
package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
)

// outilNom : le nom que ce binaire porte dans le journal des protections (verrou, sentinelle,
// priorite). Il apparait tel quel dans le message de refus lu par le prochain operateur.
const outilNom = "replay-build"

func main() {
	titleFlag := flag.String("title", title.DefaultSlug, "slug du titre")
	interval := flag.Int("interval", 0, "pas de temps du rejeu, en ms (0 = défaut)")
	geomDir := flag.String("geometry", "",
		"répertoire des CSV de props Forge (défaut : PathResolver.MapGeometryDir du titre)")
	mapName := flag.String("map", "", "nom de carte du match (obligatoire : porte les bornes de déquantification)")
	memGiB := flag.Int("mem-gib", filmproc.DefaultLimitGiB,
		"plafond memoire souple de la cuisson, en gibioctets (0 = desarme, echappatoire de l'operateur)")
	factsPath := flag.String("facts", "",
		"fichier JSON des faits du match (lignes de match, scores des deux camps, nom de variante) ; "+
			"sans lui : artefact sans compteurs de joueur ni actions d'objectif")
	cpuProfile := flag.String("cpuprofile", "",
		"fichier de profil CPU de la cuisson (vide = aucun profil) ; lecture : go tool pprof")
	memProfile := flag.String("memprofile", "",
		"fichier de profil de tas, écrit APRÈS la cuisson (vide = aucun profil)")
	flag.Parse()
	debut := time.Now()
	args := flag.Args()
	if len(args) < 1 || *mapName == "" {
		slog.Error("usage: replay-build --map <carte> [--title slug] [--interval MS] [--geometry DIR] [--facts FICHIER.json] [--cpuprofile F] [--memprofile F] <matchId> [filmDir] (les options précèdent le matchId)")
		os.Exit(2)
	}
	matchID := args[0]

	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		slog.Error("racine repo", "err", err)
		os.Exit(1)
	}
	// LE JOURNAL EST INSTALLÉ ICI, ET IL NE L'ÉTAIT PAS DU TOUT : sans handler, ce binaire
	// gardait le défaut de la bibliothèque — les lignes Debug (dont la durée de chaque balayage,
	// cf. analysis/replay/observe.go) étaient donc PERDUES, et rien n'atterrissait dans logs/.
	defer logging.InstallCLILevel(repoRoot, logging.ConsoleLevelFromEnv())()

	builder, err := replaybuild.NewBuilder(repoRoot, *titleFlag)
	if err != nil {
		slog.Error("préparation du builder", "err", err, "title", *titleFlag)
		os.Exit(1)
	}
	builder.WithFrameInterval(*interval)
	if *geomDir != "" {
		builder.WithGeometryDir(*geomDir)
	}

	// La disposition du cache film n'est déclarée que dans filmcache (garde-rail).
	cacheRoot := title.NewPathResolver(repoRoot).CacheRootDir()
	filmDir := filmcache.ChunkDir(cacheRoot, title.FilmShortMatchID(matchID))
	if len(args) >= 2 {
		filmDir = args[1]
	}

	// LE PROFIL SE FERME AUSSI QUAND LA SENTINELLE TRANCHE. La variable est declaree AVANT
	// l'armement pour que le rappel memoire puisse la lire, et elle vaut un no-op tant que le
	// profil n'a pas demarre : l'ordre d'armement (verrou, priorite, sentinelle) reste intact,
	// et le profil demarre APRES, comme avant.
	arreterProfilCPU := func() {}
	defer armerProtections(cacheRoot, matchID, *memGiB, func() { arreterProfilCPU() })()

	arreterProfilCPU = demarrerProfilCPU(*cpuProfile)
	out, err := builder.BuildMatch(matchID, []string{*mapName}, filmDir, loadFacts(*factsPath, matchID))
	// LE PROFIL SE FERME AVANT TOUT `os.Exit` : un fichier de profil non fermé est illisible, et
	// `os.Exit` ne joue aucun `defer`.
	arreterProfilCPU()
	if err != nil {
		slog.Error("construction de l'artefact", "err", err, "filmDir", filmDir, "match", matchID)
		os.Exit(1)
	}
	ecrireProfilTas(*memProfile)
	slog.Info("artefact rejeu écrit",
		"path", out.ArtifactPath, "tracks", out.Tracks, "module", out.Module,
		"bytes", out.Bytes, "match", matchID, "title", *titleFlag)
	slog.Info("cuisson terminee", "match", matchID, "duration", time.Since(debut))
}

// armerProtections arme LES TROIS PROTECTIONS et rend leur relâche, à `defer` chez l'appelant.
//
// LEÇON DU 2026-08-31. Ce binaire n'en avait AUCUNE : il était le seul point d'entrée de
// décodage du dépôt à décoder un film sans plafond mémoire, sans priorité basse et sans
// exclusion mutuelle. Sa justification au ratchet — « CLI unitaire : un film par invocation » —
// est vraie DANS le processus, et ne dit rien du nombre d'invocations : une boucle de shell, et
// pire, deux boucles en parallèle dont une en arrière-plan, ont saturé la machine de travail de
// l'utilisateur. Le corpus connaît un film à 3,3 Go (`1b1e380f`, tué par une surveillance
// EXTERNE le 2026-08-18) ; quatre sinistres mémoire ont suivi, le dernier le 2026-08-31.
// Voir `internal/filmproc/solo.go` pour le récit complet.
//
// Ordre d'armement : le VERROU d'abord (un refus doit coûter zéro décodage), puis la priorité,
// puis la sentinelle. Aucune base n'est ouverte ici, donc la sentinelle a le droit de tuer
// (cf. l'en-tête de `filmproc`). `--mem-gib 0` la désarme : c'est l'échappatoire documentée de
// l'opérateur qui sait ce qu'il fait, et elle est explicite au lieu d'être le défaut.
//
// La relâche suit l'ordre INVERSE, celui que les `defer` tenaient avant l'extraction : la
// sentinelle se désarme et publie son pic, PUIS le verrou tombe.
//
// `arreterProfil` est la fermeture du profil CPU, jouée AVANT `os.Exit` quand la sentinelle
// tranche : `os.Exit` ne joue aucun `defer`, et un fichier de profil non fermé est illisible —
// c'est précisément sur un film qui explose qu'on veut pouvoir le lire.
func armerProtections(cacheRoot, matchID string, memGiB int, arreterProfil func()) func() {
	lock, err := filmproc.AcquireSolo(cacheRoot, outilNom, matchID)
	if err != nil {
		slog.Error("décodage refusé", "err", err)
		os.Exit(filmproc.CodePreparation)
	}
	filmproc.LowerOwnPriority(outilNom)
	g := filmproc.Arm(outilNom, memGiB, func(peak uint64) {
		slog.Error("plafond memoire depasse — cuisson abandonnee",
			"pic_octets", peak, "pic_gio", float64(peak)/(1<<30), "match", matchID)
		arreterProfil()
		lock.Release()
		os.Exit(filmproc.CodeMemory)
	})
	return func() {
		g.Disarm()
		// UNE SEULE LECTURE POUR LES DEUX CHAMPS : depuis que `Peak()` re-echantillonne
		// l'empreinte a l'appel (correctif C4 de la revue R1), deux appels dans la meme
		// instruction rendent deux mesures differentes et `gio` cesse d'etre la conversion
		// d'`octets` (defaut N2 de la revue R2).
		pic := g.Peak()
		slog.Info("pic memoire de la cuisson", "octets", pic, "gio", float64(pic)/(1<<30))
		lock.Release()
	}
}

// demarrerProfilCPU ouvre le profil CPU et rend sa fermeture. Un chemin vide rend un no-op :
// mesurer est un geste d'opérateur, jamais le défaut.
//
// UN ÉCHEC N'ARRÊTE PAS LA CUISSON, mais il n'est jamais avalé : l'opérateur qui a demandé un
// profil doit savoir qu'il n'en aura pas, sans perdre pour autant le décodage qu'il a lancé.
func demarrerProfilCPU(path string) func() {
	if path == "" {
		return func() {}
	}
	f, err := os.Create(path) //nolint:gosec // chemin fourni par l'operateur du CLI
	if err != nil {
		slog.Warn("profil CPU impossible — cuisson sans profil", "err", err, "path", path)
		return func() {}
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		slog.Warn("profil CPU impossible — cuisson sans profil", "err", err, "path", path)
		_ = f.Close()
		return func() {}
	}
	return func() {
		pprof.StopCPUProfile()
		if err := f.Close(); err != nil {
			slog.Warn("fermeture du profil CPU", "err", err, "path", path)
			return
		}
		slog.Info("profil CPU écrit", "path", path)
	}
}

// ecrireProfilTas écrit un profil de tas APRÈS la cuisson. Chemin vide = rien.
//
// LE `runtime.GC()` EST OBLIGATOIRE et il n'est pas cosmétique : sans lui, le profil compte des
// objets déjà inatteignables et fait passer pour vivant ce que le décodeur vient de lâcher.
func ecrireProfilTas(path string) {
	if path == "" {
		return
	}
	f, err := os.Create(path) //nolint:gosec // chemin fourni par l'operateur du CLI
	if err != nil {
		slog.Warn("profil de tas impossible", "err", err, "path", path)
		return
	}
	defer func() { _ = f.Close() }()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		slog.Warn("profil de tas non écrit", "err", err, "path", path)
		return
	}
	slog.Info("profil de tas écrit", "path", path)
}

// loadFacts lit les faits du match dans un fichier JSON.
//
// POURQUOI UN FICHIER ET PAS LA BASE. Ce binaire est HORS LIGNE par construction : il n'ouvre
// aucune DuckDB, et c'est ce qui le rend utilisable sur une machine qui n'a que des chunks de
// film (et qui le laisse compiler sans CGO). Les faits que le film ne dit pas — les lignes de
// match, les scores des deux camps, le nom de variante — arrivent donc par la même porte que
// tout le reste : l'appelant. `levelup backfill-replay`, l'action admin et le fil de l'eau, eux,
// les lisent en base.
//
// Forme attendue (les champs absents dégradent, ils ne font jamais échouer) :
//
//	{"gameVariantName":"CTF:Arena","teamScores":[3,0],
//	 "mapId":"e859cf75-9b8a-429a-91be-2376681c8537",
//	 "players":[{"xuid":"2533274...","kills":12,"deaths":7,"assists":3,"teamId":0}]}
//
// `mapId` est l'asset UGC de la carte : la clé du catalogue d'objectifs, d'où sortent les socles
// de drapeau (sans lui, la vie des drapeaux se publie sans équipe propriétaire ni état `home`).
//
// Un chemin vide rend des faits vides, sans bruit : c'est le mode nominal du binaire. Un fichier
// ILLISIBLE, lui, est journalisé — demander des faits et n'en avoir aucun n'est pas la même chose
// que ne pas en demander.
func loadFacts(path, matchID string) port.MatchFacts {
	if path == "" {
		return port.MatchFacts{}
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur du CLI
	if err != nil {
		slog.Warn("faits du match illisibles — artefact sans compteurs de joueur ni actions d'objectif",
			"err", err, "path", path, "match", matchID)
		return port.MatchFacts{}
	}
	var facts port.MatchFacts
	if err := json.Unmarshal(raw, &facts); err != nil {
		slog.Warn("faits du match invalides — artefact sans compteurs de joueur ni actions d'objectif",
			"err", err, "path", path, "match", matchID)
		return port.MatchFacts{}
	}
	slog.Info("faits du match charges", "path", path, "match", matchID,
		"joueurs", len(facts.Players), "variante", facts.GameVariantName)
	return facts
}
