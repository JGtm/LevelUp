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
// Il ne lit AUCUNE base : les identités de carte (noms candidats) ET les faits du match
// (`port.MatchFacts` — lignes de match, scores des deux camps, nom de variante) viennent de
// l'appelant, qui les résout où il veut (registre partagé, flag CLI, fichier). C'est ce qui le
// garde utilisable hors ligne, et c'est la même frontière que `replay.Options.Objectives`.
//
// Il décode le film à DEUX endroits, et pour deux grammaires différentes : `analysis/replay`
// pour les positions et les événements de réplication (sérialisé par le verrou process de
// `filmdec`), `analysis/objectiveevents` pour les enregistrements d'entité d'où sortent la
// courbe de score et les actions d'objectif (cf. matchfacts.go).
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
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
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
	// objectives / objectivesTried : le catalogue versionné d'objectifs de carte, d'où sortent
	// les socles de drapeau (cf. flagspawns.go), chargé au plus une fois par Builder. Le drapeau
	// booléen distingue « pas encore chargé » de « chargement tenté et échoué ».
	objectives *replay.MapObjectivesCatalog
	// pads / padsTried : le catalogue versionne des socles, d'ou sortent les POINTS
	// D'APPARITION (cf. spawnpoints.go). Meme motif que `objectives` — chargement unique par
	// Builder, et un booleen qui distingue « pas encore charge » de « tente et echoue ».
	pads            *replay.MapWeaponPadsCatalog
	padsTried       bool
	objectivesTried bool
	// roles / rolesTried : la table de roles d objectif du titre, d ou sortent les zones du
	// calque d ETAT DES ZONES (cf. zones.go). Meme cache, meme raison que ci-dessus.
	roles      *mappings.ObjectiveRoleSet
	rolesTried bool
	// regulation : la table de reglement du titre (regulation.toml), d ou sort la CIBLE DE
	// VICTOIRE publiee avec la courbe de score. Chargee une fois au NewBuilder, best-effort :
	// table absente ou illisible = aucune cible, jamais un echec (le client a son repli).
	regulation *mappings.RegulationSet
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
	// La table de reglement (cible de victoire) est BEST-EFFORT : un titre sans
	// regulation.toml construit des artefacts sans cible, et le client a son repli.
	regulation, err := mappings.LoadRegulationFromFile(
		filepath.Join(pr.TitleMappingsDir(titleSlug), "regulation.toml"))
	if err != nil {
		slog.Warn("replaybuild: table de reglement illisible — artefacts sans cible de victoire",
			"err", err, "title", titleSlug)
		regulation = nil
	}
	return &Builder{
		repoRoot:   repoRoot,
		titleSlug:  titleSlug,
		catalog:    cat,
		labels:     labels,
		geometry:   geometry,
		structures: map[string][]replay.Surface{},
		regulation: regulation,
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

// BuildBytes décode le film de filmDir et rend l'artefact SÉRIALISÉ — il n'écrit RIEN.
// mapNames sont les identités de carte candidates (cf. ResolveMapEntry). Le décodage est
// sérialisé par le verrou process de filmdec (dans replay.BuildFromFilm) — jamais deux films
// en parallèle dans un même process.
//
// C'EST LA MOITIÉ QUI EXPLOSE, ET C'EST POURQUOI ELLE EST SÉPARABLE (lot BUILDALL,
// 2026-08-26). Le décodage est un amplificateur mémoire (7,9 Go en 2,6 s sur `51101d1d`) ;
// l'écriture, elle, porte le garde anti-régression et publie l'événement de notification.
// Les rendre appelables séparément permet de les mettre dans DEUX PROCESSUS : un enfant borné
// construit, le serveur range (cf. sync/replayartifacts/buildone.go). `BuildMatch` reste la
// composition des deux pour les appelants in-processus.
//
// `facts` est CE QUE LA BASE SAIT DU MATCH et que le film ne dit pas (cf. port.MatchFacts) :
// les lignes de match qui apparient les slots d'entité aux joueurs, les scores des deux camps
// et le nom de variante. CE PAQUET N'OUVRE AUCUNE BASE — c'est l'appelant qui les résout, là
// où il sait le faire. Des faits vides restent un cas nominal (ouvrier distant, CLI unitaire) :
// l'artefact sort sans compteurs de joueur ni actions d'objectif, et la dégradation est
// journalisée.
func (b *Builder) BuildBytes(matchID string, mapNames []string, filmDir string, facts port.MatchFacts) (Built, error) {
	entry, err := b.ResolveMapEntry(mapNames)
	if err != nil {
		return Built{}, err
	}
	stats := readFilmStats(context.Background(), matchID, filmDir, facts)
	// La CIBLE DE VICTOIRE vient de la table de règlement du titre, jamais du film : elle
	// s'ajoute à l'entrée du calque de score, et la garde de publication vit chez lui
	// (`publishableTarget` — une table périmée se tait au lieu de publier une cible fausse).
	// Le DÉNOMINATEUR DE LA GARDE vient de la même table, pour la même raison : le film porte le
	// compteur de tics, pas combien de tics valent un point. Sa garde de
	// publication vit elle aussi chez le calque (`publishableHold`).
	if stats.score != nil {
		stats.score.TargetScore, _ = b.regulation.ScoreTarget(facts.GameVariantName)
		stats.score.HoldTicksPerPoint, _ = b.regulation.HoldTicksPerPoint(facts.GameVariantName)
	}
	// Les SOCLES de drapeau viennent du catalogue de carte, pas du film : ils s'ajoutent aux
	// lectures que le second décodage a déjà faites (cf. flagspawns.go).
	stats.flag.Spawns = b.flagSpawns(matchID, facts.MapID)
	// Les ZONES du mode viennent du même catalogue de carte, dans l'ORDRE OÙ LE SERVICE LES SERT :
	// c'est cet ordre qui donne son sens à `zoneStates[].zoneRef` (cf. zones.go). Aucune zone =
	// aucun balayage de `ti=13`, donc aucun coût sur les modes qui n'en ont pas.
	zones, zoneRoles := b.matchZones(matchID, facts.MapID, facts.GameVariantName)
	// UN SEUL décodage killsource par match : neutralDeaths ET killRefs (cf. kills.go) en
	// dérivent tous les deux, pour ne payer qu'UNE fois le verrou filmdec partagé avec
	// `replay.BuildFromFilm` — au lieu de deux, comme avant la jointure des frags sous
	// effet actif (PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.1).
	ksRes := b.decodeKillSource(matchID, filmDir)
	// Les POINTS D'APPARITION viennent du catalogue des socles, par map_id — ils donnent leur
	// origine aux ramassages non-arme (cf. spawnpoints.go).
	spawnPts, mapState := b.spawnPoints(matchID, facts.MapID, mapNames)
	doc, err := replay.BuildFromFilm(matchID, b.titleSlug, filmDir, replay.Options{
		FrameIntervalMS: b.interval,
		Geometry:        b.geometry,
		Structure:       b.structureFor(entry.Module),
		Labels:          b.labels,
		NeutralDeaths:   b.neutralDeaths(matchID, ksRes),
		Kills:           b.killRefs(matchID, filmDir, ksRes),
		Bots:            botIdentities(ksRes),
		Objectives:      stats.objectives,
		Score:           stats.score,
		Flag:            stats.flag,
		Vip:             stats.vip,
		Skull:           stats.skull,
		Bomb:            stats.bomb,
		Zone: replay.ZoneInput{Zones: zones, Roles: zoneRoles, TeamByXUID: teamByXUID(facts),
			Hill: isHillVariant(facts.GameVariantName)},
		MapQuant:         &entry,
		SpawnPoints:      spawnPts,
		SpawnPointsState: mapState,
	})
	if err != nil {
		return Built{}, fmt.Errorf("décodage du film %s: %w", matchID, err)
	}
	if len(doc.Tracks) == 0 {
		return Built{}, fmt.Errorf("%w (match %s)", ErrNoTracks, matchID)
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		return Built{}, fmt.Errorf("sérialisation artefact %s: %w", matchID, err)
	}
	return Built{Blob: blob, Module: entry.Module, Tracks: len(doc.Tracks)}, nil
}

// Built : un artefact CONSTRUIT mais PAS ENCORE RANGÉ.
//
// POURQUOI CE TYPE EXISTE (lot BUILDALL, 2026-08-26). Le décodage est un amplificateur mémoire ;
// l'écriture, elle, porte le garde anti-régression et publie l'événement qui déclenche la
// notification Discord. Les deux doivent pouvoir vivre dans DEUX PROCESSUS DIFFÉRENTS : un
// enfant borné construit, le serveur range. Sans cette frontière, déléguer le décodage
// emporterait la notification avec lui (cf. sync/replayartifacts/buildone.go).
type Built struct {
	// Blob est le document SÉRIALISÉ — ce qui traverse la frontière de processus.
	Blob []byte
	// Module est le module de carte retenu (traçabilité du choix d'identité).
	Module string
	// Tracks est le nombre de trajectoires du document CONSTRUIT — pas de celui qui finira sur
	// le disque : quand le garde anti-régression conserve l'artefact en place, les deux
	// diffèrent, et c'est l'écriture qui fait foi (cf. StoreArtifact).
	Tracks int
}

// BuildMatch construit l'artefact d'un match et le RANGE à sa place canonique.
//
// C'est `BuildBytes` suivi de l'écriture — la composition que les appelants IN-PROCESSUS
// utilisent (CLI unitaire, enfant de backfill, action admin). Le post-sync, lui, sépare les
// deux moitiés entre son enfant et lui-même.
func (b *Builder) BuildMatch(matchID string, mapNames []string, filmDir string, facts port.MatchFacts) (Outcome, error) {
	built, err := b.BuildBytes(matchID, mapNames, filmDir, facts)
	if err != nil {
		return Outcome{}, err
	}
	outPath := title.NewPathResolver(b.repoRoot).ReplayArtifactPath(b.titleSlug, matchID)
	surDisque, err := writeArtifactBytes(outPath, b.titleSlug, matchID, built.Blob)
	if err != nil {
		return Outcome{}, fmt.Errorf("écriture artefact %s: %w", outPath, err)
	}
	return Outcome{ArtifactPath: outPath, Module: built.Module, Tracks: built.Tracks,
		Bytes: surDisque.bytes}, nil
}

// neutralDeaths rend les entrées d'artefact déjà résolues (type de mort + pictogramme du
// titre) pour les morts que PERSONNE ne revendique, à partir d'un décodage killsource DÉJÀ
// FAIT (cf. decodeKillSource, kills.go) — ce fichier ne décode plus rien lui-même depuis le
// lot F.1 (jointure des frags sous effet actif), qui a besoin du MÊME `*killsource.Result`.
//
// POURQUOI CE DÉCODAGE-CI VIT DANS `replaybuild`, ET PAS DANS `analysis/replay`. La source du
// dégât fatal se lit dans le composant dead-state du film, et ce décodage a UN seul
// propriétaire dans le dépôt (`film/killsource`, avec ses golden et ses ancres Theater).
// `analysis/` est title-agnostic et n'a pas à le connaître ; ce paquet, lui, est la couche
// d'ASSEMBLAGE — il compose déjà les libellés du titre de la même façon. Deux décodeurs du
// même fait divergeraient.
//
// DEUX ACQUISITIONS DU VERROU filmdec, ET C'EST VOULU : `killsource.Decode` (dans
// decodeKillSource) prend et rend le verrou process, puis `replay.BuildFromFilm` le reprend.
// Ce sont deux décodages complets du MÊME film, chacun sérialisé de bout en bout ; c'est
// exactement ce que fait déjà le cycle post-sync (arme du kill puis artefacts). Les enchaîner
// sous un seul verrou exigerait un mutex réentrant, que Go n'a pas — et le contrat qui compte
// (« jamais deux films entrelacés dans un décodage ») est tenu par chacune des deux.
//
// TOUT ÉCHEC EST NON FATAL : un film dont la source de dégât ne se décode pas reste un rejeu
// parfaitement valide, avec des lignes de mort neutres au repère générique. Le refus est
// JOURNALISÉ (dans decodeKillSource), jamais avalé.
func (b *Builder) neutralDeaths(matchID string, res *killsource.Result) []replay.NeutralDeath {
	if res == nil {
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

// botIdentities projette les bots que le film déclare (BOT_METADATA, décodage killsource
// DÉJÀ fait) vers l'assemblage du rejeu. Le suffixe « [bot] » suit la même règle que le
// kill-feed (killsource.botSuffix) : un consommateur ne doit jamais confondre un bot avec
// un joueur. Les bots NON ÉPINGLÉS (slot contredisant l'espace des humains) n'entrent pas —
// leur index est une anomalie déclarée, pas une identité.
func botIdentities(res *killsource.Result) []replay.BotIdentity {
	if res == nil || len(res.Roster.Bots) == 0 {
		return nil
	}
	unpinned := make(map[int]bool, len(res.Roster.UnpinnedBots))
	for _, b := range res.Roster.UnpinnedBots {
		unpinned[b.BotID] = true
	}
	out := make([]replay.BotIdentity, 0, len(res.Roster.Bots))
	for _, b := range res.Roster.Bots {
		if b.Name == "" || unpinned[b.BotID] {
			continue
		}
		out = append(out, replay.BotIdentity{FilmIndex: b.Slot, Name: b.Name + " [bot]"})
	}
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

// ArtifactHasPlayerCounters dit si l'artefact au chemin donné PORTE des compteurs de joueur
// (`scoreTimeline.players` non vide). Faux aussi quand le fichier est absent ou illisible.
//
// CE QU'IL AFFIRME, ET CE QU'IL N'AFFIRME PAS. Il constate une PROPRIÉTÉ DU DOCUMENT, il ne
// devine pas comment il a été cuit. L'implication ne vaut que dans un sens :
//
//	compteurs présents  =>  les lignes de match ont été fournies   (sûr)
//	compteurs absents   =>  les lignes de match manquaient          (FAUX en général)
//
// TROIS FAÇONS LÉGITIMES d'être vide MALGRÉ des faits complets, toutes constatées dans le code :
// (a) le film n'a aucun enregistrement d'entité à lire (cas journalisé, `matchfacts.go:70-73`) ;
// (b) l'appariement slot -> joueur échoue — `SlotIdentityFrom` écarte les triplets ambigus, et
// les `COALESCE(..., 0)` de `replay_facts_repo.go` peuvent rendre plusieurs (0,0,0)
// indistinguables ; (c) aucun compteur ne bouge dans la fenêtre lue (`PlayerScore` vide).
//
// UN APPELANT NE DOIT DONC JAMAIS EN DÉDUIRE « à re-cuire » À LUI SEUL. Le vide est une
// PRÉSOMPTION d'appauvrissement, pas une preuve : c'est pour cela que `replayartifacts.enqueueAll`
// exige EN PLUS de tenir des lignes de match (`len(facts.Players) > 0`), et que le rangement
// d'artefact (`StoreArtifact`) refuse de son côté toute régression. Le pire résidu possible est
// alors UN cycle d'ouvrier gâché — jamais un artefact rétrogradé, jamais une boucle qui converge
// vers rien.
//
// POURQUOI CE SIGNAL PLUTÔT QU'UN AUTRE. Mesuré sur deux témoins le 2026-08-24 : 8 joueurs avec
// faits, 0 sans, sur 7344d24f comme sur 530820e5. Les autres candidats sont pires :
// `coverage.score.teamIdentity` vaut légitimement `unresolved` sur 7 des 34 artefacts du cache
// POURTANT cuits avec faits, et `objectives` est vide de plein droit sur un Slayer.
//
// PAS DE CHAMP DÉDIÉ DANS LE DOCUMENT, ET C'EST DÉLIBÉRÉ : un marqueur `factsApplied` — qui
// LUI porterait l'implication dans les deux sens — forcerait un incrément de
// `replay.SchemaVersion`, donc la re-cuisson de tout le cache, aujourd'hui bloquée par la bombe
// RAM de `NamedEventsFrom` (registre du 2026-08-24). C'est la dette assumée de ce choix.
func ArtifactHasPlayerCounters(path string) bool {
	d, ok := readArtifactDigest(path)
	return ok && d.players > 0
}

// artifactDigest : les seules marques d'un artefact que les gardes ont à lire. Une SEULE forme
// de lecture pour tous — deux structures anonymes concurrentes finiraient par diverger sur le
// nom d'un champ, et le garde deviendrait muet sans que rien ne le signale.
type artifactDigest struct {
	matchID       string
	schemaVersion int
	players       int
	tracks        int
	bytes         int
}

// readArtifactDigest lit un artefact sur disque. ok=false si absent ou illisible : dans le
// doute, l'appelant traite l'artefact comme inexploitable, jamais comme bon.
func readArtifactDigest(path string) (artifactDigest, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return artifactDigest{}, false
	}
	return digestFromBytes(raw)
}

// digestFromBytes lit les marques d'un artefact déjà en mémoire (cas du dépôt d'un ouvrier :
// le blob est là, le relire depuis le disque serait absurde).
func digestFromBytes(raw []byte) (artifactDigest, bool) {
	var head struct {
		MatchID       string            `json:"matchId"`
		SchemaVersion int               `json:"schemaVersion"`
		Tracks        []json.RawMessage `json:"tracks"`
		ScoreTimeline struct {
			Players []json.RawMessage `json:"players"`
		} `json:"scoreTimeline"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return artifactDigest{}, false
	}
	return artifactDigest{
		matchID:       head.MatchID,
		schemaVersion: head.SchemaVersion,
		players:       len(head.ScoreTimeline.Players),
		tracks:        len(head.Tracks),
		bytes:         len(raw),
	}, true
}

// writeArtifact sérialise le document et l'écrit ATOMIQUEMENT (cf.
// writeArtifactBytes, artifact_store.go) ; renvoie la taille en octets. Même
// écriture que le dépôt d'un ouvrier : le service de lecture sert le fichier tel
// quel, il ne doit jamais tomber sur un artefact à moitié écrit.
// La taille rendue est celle de ce qui est FINALEMENT sur le disque, pas celle du document
// qu'on voulait écrire : quand le garde anti-régression conserve l'artefact en place, annoncer
// la taille du candidat ferait croire à une écriture qui n'a pas eu lieu.
func writeArtifact(outPath, titleSlug, matchID string, doc replay.ReplayDocument) (int, error) {
	blob, err := json.Marshal(doc)
	if err != nil {
		return 0, err
	}
	surDisque, err := writeArtifactBytes(outPath, titleSlug, matchID, blob)
	if err != nil {
		return 0, err
	}
	return surDisque.bytes, nil
}
