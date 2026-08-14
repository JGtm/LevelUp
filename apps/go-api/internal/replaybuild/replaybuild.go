// Package replaybuild — construction d'un artefact de rejeu 2D depuis les chunks du
// cache film, HORS LIGNE.
//
// C'est LA brique partagée par les producteurs d'artefacts : `cmd/replay-build`
// (unitaire), `levelup backfill-replay` (masse), l'action admin replay-build (job) et
// l'étape post-sync locale. Avant elle, l'assemblage (catalogue de bornes + libellés du
// titre + structure de carte + écriture de l'artefact) vivait dans `cmd/replay-build`
// seul ; à la troisième copie la règle du dépôt impose de centraliser — deux assembleurs
// qui divergent produiraient deux artefacts différents pour le même film sans que rien ne
// le signale.
//
// Ce paquet ne décode rien lui-même : le décodage vit dans `analysis/replay` (qui
// serialise par le verrou process de `filmdec`). Il ne lit AUCUNE base : les identités de
// carte (noms candidats) viennent de l'appelant, qui les résout où il veut (registre
// partagé, flag CLI).
package replaybuild

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	halo "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
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

// Builder porte les entrées chargées UNE fois (catalogue de bornes, libellés du titre,
// props Forge) pour construire N artefacts. Construire un Builder par passe, pas par
// match : le catalogue et les labels sont les mêmes pour tous les films du titre.
type Builder struct {
	repoRoot  string
	titleSlug string
	catalog   *filmdec.MapQuantCatalog
	labels    replay.LabelCatalog
	geometry  []replay.MapObject
	// interval : pas de temps du rejeu en ms ; 0 = défaut de replay.Options.
	interval int
	// structures : cache par module du fond structurel (une carte revient N fois dans une
	// passe de masse ; recharger le fichier à chaque match serait du pur gaspillage).
	structures map[string][]replay.Surface
}

// Outcome décrit un artefact construit.
type Outcome struct {
	ArtifactPath string
	Module       string
	Tracks       int
	Bytes        int
}

// NewBuilder charge le catalogue de bornes et les libellés du titre. Les deux sont
// FATALS : sans bornes aucune carte ne se résout, et un rejeu sans libellés est
// indistinguable à l'écran d'un rejeu aux armes inconnues (même règle que
// cmd/replay-build). Les props Forge, eux, sont optionnels (journalisé).
func NewBuilder(repoRoot, titleSlug string) (*Builder, error) {
	pr := title.NewPathResolver(repoRoot)
	cat, err := filmdec.LoadMapQuantCatalog(pr.MapQuantBoundsPath(titleSlug))
	if err != nil {
		return nil, fmt.Errorf("catalogue de bornes du titre %s: %w", titleSlug, err)
	}
	labels, err := replaylabels.Load(repoRoot, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("catalogue de libellés du titre %s: %w", titleSlug, err)
	}
	geomDir := pr.MapGeometryDir(titleSlug)
	geometry, skipped, err := replay.LoadGeometry(geomDir)
	if err != nil {
		slog.Warn("replaybuild: géométrie de carte indisponible — artefacts sans props",
			"err", err, "dir", geomDir)
		geometry = nil
	} else if skipped > 0 {
		slog.Debug("replaybuild: props sans emprise ignorés", "sansEmprise", skipped, "dir", geomDir)
	}
	return &Builder{
		repoRoot:   repoRoot,
		titleSlug:  titleSlug,
		catalog:    cat,
		labels:     labels,
		geometry:   geometry,
		structures: map[string][]replay.Surface{},
	}, nil
}

// WithFrameInterval règle le pas de temps du rejeu en ms (0 = défaut). Chaînable.
func (b *Builder) WithFrameInterval(ms int) *Builder {
	b.interval = ms
	return b
}

// WithGeometryDir recharge les props Forge depuis un répertoire explicite (CLI --geometry).
// Chaînable ; un répertoire illisible dégrade en « sans props », journalisé.
func (b *Builder) WithGeometryDir(dir string) *Builder {
	objs, skipped, err := replay.LoadGeometry(dir)
	if err != nil {
		slog.Warn("replaybuild: géométrie de carte indisponible — artefacts sans props",
			"err", err, "dir", dir)
		b.geometry = nil
		return b
	}
	slog.Info("replaybuild: géométrie de carte chargée", "objets", len(objs), "sansEmprise", skipped, "dir", dir)
	b.geometry = objs
	return b
}

// ResolveMapEntry résout la première identité de carte candidate qui existe au catalogue
// de bornes. Les candidats s'essaient DANS L'ORDRE (du plus fiable au moins fiable, cf.
// ReplayMapRepo) ; aucun ne résout → ErrMapNotInCatalog.
func (b *Builder) ResolveMapEntry(mapNames []string) (filmdec.MapQuantEntry, error) {
	for _, name := range mapNames {
		if name == "" {
			continue
		}
		if entry, err := b.catalog.Lookup(name); err == nil {
			return entry, nil
		}
	}
	return filmdec.MapQuantEntry{}, fmt.Errorf("%w (candidats: %v)", ErrMapNotInCatalog, mapNames)
}

// BuildMatch décode le film de filmDir et écrit l'artefact de rejeu du match. mapNames
// sont les identités de carte candidates (cf. ResolveMapEntry). Le décodage est sérialisé
// par le verrou process de filmdec (dans replay.BuildFromFilm) — jamais deux films en
// parallèle dans un même process.
func (b *Builder) BuildMatch(matchID string, mapNames []string, filmDir string) (Outcome, error) {
	entry, err := b.ResolveMapEntry(mapNames)
	if err != nil {
		return Outcome{}, err
	}
	worldRange := entry.Range()
	doc, err := replay.BuildFromFilm(matchID, b.titleSlug, filmDir, replay.Options{
		FrameIntervalMS: b.interval,
		Geometry:        b.geometry,
		Structure:       b.structureFor(entry.Module),
		Labels:          b.labels,
		NeutralDeaths:   b.neutralDeaths(matchID, filmDir),
		WorldRange:      &worldRange,
	})
	if err != nil {
		return Outcome{}, fmt.Errorf("décodage du film %s: %w", matchID, err)
	}
	if len(doc.Tracks) == 0 {
		return Outcome{}, fmt.Errorf("%w (match %s)", ErrNoTracks, matchID)
	}
	outPath := title.NewPathResolver(b.repoRoot).ReplayArtifactPath(b.titleSlug, matchID)
	size, err := writeArtifact(outPath, doc)
	if err != nil {
		return Outcome{}, fmt.Errorf("écriture artefact %s: %w", outPath, err)
	}
	return Outcome{ArtifactPath: outPath, Module: entry.Module, Tracks: len(doc.Tracks), Bytes: size}, nil
}

// neutralDeaths décode les morts que PERSONNE ne revendique et rend les entrées d'artefact
// déjà résolues (type de mort + pictogramme du titre).
//
// POURQUOI CE DÉCODAGE-CI VIT ICI, ET PAS DANS `analysis/replay`. La source du dégât fatal se
// lit dans le composant dead-state du film, et ce décodage a UN seul propriétaire dans le dépôt
// (`film/killsource`, avec ses golden et ses ancres Theater). `analysis/` est title-agnostic et
// n'a pas à le connaître ; ce paquet, lui, est la couche d'ASSEMBLAGE — il compose déjà les
// libellés du titre de la même façon. Deux décodeurs du même fait divergeraient.
//
// DEUX ACQUISITIONS DU VERROU filmdec, ET C'EST VOULU : `killsource.Decode` prend et rend le
// verrou process, puis `replay.BuildFromFilm` le reprend. Ce sont deux décodages complets du
// MÊME film, chacun sérialisé de bout en bout ; c'est exactement ce que fait déjà le cycle
// post-sync (arme du kill puis artefacts). Les enchaîner sous un seul verrou exigerait un mutex
// réentrant, que Go n'a pas — et le contrat qui compte (« jamais deux films entrelacés dans un
// décodage ») est tenu par chacune des deux.
//
// TOUT ÉCHEC EST NON FATAL : un film dont la source de dégât ne se décode pas reste un rejeu
// parfaitement valide, avec des lignes de mort neutres au repère générique. Le refus est
// JOURNALISÉ, jamais avalé.
func (b *Builder) neutralDeaths(matchID, filmDir string) []replay.NeutralDeath {
	src, err := killsource.DirChunks(filmDir)
	if err != nil {
		slog.Debug("replaybuild: chunks illisibles pour la source de dégât — morts neutres sans type",
			"err", err, "match_id", matchID)
		return nil
	}
	res, err := killsource.Decode(context.Background(), matchID, src, nil)
	if err != nil {
		slog.Info("replaybuild: source de dégât non décodée — morts neutres sans type",
			"err", err, "match_id", matchID)
		return nil
	}
	// LA MÊME PORTE QUE POUR LES KILLS : ces lignes sont nommées par la bijection indice ->
	// joueur. Sans marge (BTB) ou en alerte de santé, le décodage reste juste EN AGRÉGAT et
	// faux ligne par ligne — et une ligne est précisément ce qu'on publierait ici.
	if !res.LineByLinePublishable() && len(res.UnclaimedDeaths) > 0 {
		slog.Info("replaybuild: attribution ligne par ligne refusée — morts neutres sans type",
			"match_id", matchID, "candidates", len(res.UnclaimedDeaths))
		return nil
	}
	adapter := halo.NewAssetURLAdapter()
	out := make([]replay.NeutralDeath, 0, len(res.UnclaimedDeaths))
	for _, d := range res.UnclaimedDeaths {
		if d.VictimXUID == 0 {
			// Le xuid est la SEULE clé de jointure avec les pistes. Sans lui, l'entrée ne
			// rencontrerait aucune ligne — et un « 0 » sérialisé pourrait en rencontrer une
			// qui ne lui appartient pas.
			continue
		}
		kind, img, ok := adapter.NeutralDeathIcon(d.Source.Tag)
		if !ok {
			continue // nature non établie : le fil garde son repère neutre
		}
		out = append(out, replay.NeutralDeath{
			XUID: strconv.FormatUint(d.VictimXUID, 10), FeedMs: d.TimeMS,
			Kind: kind, Img: img, Tinted: true,
		})
	}
	if len(out) == 0 {
		return nil
	}
	slog.Info("replaybuild: morts sans revendication typées", "match_id", matchID,
		"publiees", len(out), "orphelines", res.Stats.Unclaimed.Population)
	return out
}

// structureFor charge (et met en cache) le fond structurel d'un module. Son absence n'est
// PAS fatale : toutes les cartes n'ont pas de fichier figé, un rejeu sans fond reste
// lisible (même règle que cmd/replay-build).
func (b *Builder) structureFor(module string) []replay.Surface {
	if s, ok := b.structures[module]; ok {
		return s
	}
	path := title.NewPathResolver(b.repoRoot).MapStructurePath(b.titleSlug, module)
	ms, err := replay.LoadMapStructure(path)
	if err != nil {
		slog.Debug("replaybuild: structure de carte indisponible — artefact sans fond structurel",
			"err", err, "path", path, "module", module)
		b.structures[module] = nil
		return nil
	}
	b.structures[module] = ms.Surfaces
	return ms.Surfaces
}

// ArtifactUpToDate dit si l'artefact au chemin donné existe ET porte la version de schéma
// courante. C'est LA clé de reprise des backfills (cf. replay.SchemaVersion) : un artefact
// d'une version antérieure se lit « à re-cuire », jamais « à jour ». Un fichier illisible
// est traité comme périmé (il sera réécrit), pas comme une erreur.
func ArtifactUpToDate(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var head struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return false
	}
	return head.SchemaVersion == replay.SchemaVersion
}

// writeArtifact sérialise le document et l'écrit (crée les répertoires parents) ; renvoie
// la taille en octets.
func writeArtifact(outPath string, doc replay.ReplayDocument) (int, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return 0, err
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		return 0, err
	}
	return len(blob), os.WriteFile(outPath, blob, 0o644)
}
