// Package decfilm EST LA PORTE DU DECODEUR, ET RIEN D AUTRE : il RE-EXPORTE ce que les paquets
// hors du decodeur compilent, et il ne decode pas une ligne.
//
// # POURQUOI IL EXISTE (lot 2.5.e-b, decisions V15 (6) et (7))
//
// ADR 0034 principe 11 : « la compilation seule garantit la frontiere ». Les cinq couches
// (`source`, `profile`, `grammar`, `facts`, `replay`) passent sous `film/internal/` au commit
// suivant ; a cet instant, TOUT consommateur exterieur qui importait une couche en direct ne
// compile plus — c est le compilateur, et non un ratchet, qui prouve la frontiere. La facade
// est ce qui rend ce geste faisable : elle porte, sous un seul nom de paquet, la surface que
// ces consommateurs citaient.
//
// # CE QU ELLE PESE, ET POURQUOI C EST LE CHIFFRE QUI COMPTE
//
// MESURE DU 2026-09-16, par `grep` des qualifieurs hors de `film/`, paquet par paquet et
// consommateur par consommateur (le detail est consigne au §4 du
// PLAN_DECODEUR_FILM_2026-09-13) :
//
//	source       8    profile      7    grammar     46    weaponscan   5
//	positions    2    weaponv3     4    killsource  36    objectives  43
//	fallback    11    facts        1
//	                                          TOTAL :  163 symboles
//
// La note de preparation §2.4 en comptait 127 le 2026-09-17, et le plan 177 a la re-mesure du
// 2.5 ; les deux chiffres portaient sur une arborescence d avant les lots 2.4 et 2.5 (`grammar`
// pesait 78 symboles, il en pese 46 — la porte aux octets et la couche `profile` ont absorbe le
// reste). LE CHIFFRE EST LE POINT : une facade de 163 symboles n est PAS une frontiere, c est un
// ALIAS. V15 (7) l assume pour ce lot — il faut que le compilateur puisse prouver le lieu AVANT
// qu on discute de la surface — et renvoie la REDUCTION a M4, sur la mesure consignee.
//
// # LES TROIS FORMES DE RE-EXPORT, ET LEURS RAISONS
//
//	type X = pkg.X      un ALIAS est le MEME type pour le compilateur et pour `reflect` : un
//	                    appelant peut declarer une variable, un champ, un parametre.
//	const / var X       la valeur elle-meme. Les `var` sont des sentinelles d erreur et des
//	                    tables lues, jamais ecrites.
//	func X(...) { ... } un RENVOI D UNE LIGNE. La signature NOMME les types de la couche
//	                    (`grammar.Lecteur`, `types.StatRecord`) : ceux-la restent
//	                    inaccessibles a l appelant, qui ne peut que faire CIRCULER la valeur —
//	                    et c est exactement la frontiere qu on veut. Un type qu un appelant doit
//	                    NOMMER a son alias ci-dessus ; les autres n en ont pas, volontairement.
//
// UNE SEULE EXCEPTION, et elle est mesuree : [BuildBipedTracks] est re-exportee comme VALEUR de
// fonction, parce que sa signature nomme un type NON EXPORTE de `grammar`
// (`map[uint32][]hitPosSample`). Un renvoi ecrit exigerait d exporter ce type, c est-a-dire un
// changement de contenu dans un lot de deplacements. Consigne au §4.
//
// # CE QUE CETTE FACADE N EST PAS
//
// Elle n a AUCUNE logique : pas de valeur par defaut, pas de garde, pas de log, pas de
// conversion. Un renvoi qui se mettrait a decider serait du decodage pose hors des couches —
// exactement ce que l ADR ferme. Le ratchet des couches la classe `horsCoucheFilm` : elle ne
// lit aucun octet et ne publie rien.
//
// `film/replay` N EST PAS ICI, et c est une decision ecrite : c est la couche de PUBLICATION,
// 239 de ses symboles sont cites hors du decodeur (`sync/replayartifacts`, `sync/killcollector`,
// `service/*`, `api/*`, `replaybuild`, `ops`), et le document de rejeu EST le contrat public.
// Elle reste EXPORTEE. `film/types`, `film/revision`, `film/filmcache` et les trois catalogues
// de libelles (`damagetag`, `killicon`, `medalname`) restent exportes pour la meme raison de
// nature : ils declarent des formes ou nomment des choses, ils ne decodent pas.
package decfilm

import (
	"context"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/domain/playerposition"
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/positions"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponscan"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ---- facts ----
const Rev = facts.Rev

// ---- fallback ----
type Compteur = fallback.Compteur
type Declenchement = fallback.Declenchement

func Lire(n fallback.Nom) (fallback.Repli, bool) { return fallback.Lire(n) }

type Nom = fallback.Nom

const NomGardeEquipementNegatifAZero = fallback.NomGardeEquipementNegatifAZero
const NomGestePremiereVieDuSlot = fallback.NomGestePremiereVieDuSlot

func NouveauCompteur() *fallback.Compteur { return fallback.NouveauCompteur() }

type Site = fallback.Site

func Table() []fallback.Repli                       { return fallback.Table() }
func Texte(rapport []fallback.Declenchement) string { return fallback.Texte(rapport) }
func VerifierRegistre() []string                    { return fallback.VerifierRegistre() }

// ---- grammar ----
type BipedCreation = grammar.BipedCreation
type BipedPosition = grammar.BipedPosition

// PRIVE(hitPosSample) : BuildBipedTracks
var BuildBipedTracks = grammar.BuildBipedTracks

func CountFilmChunks(dir string) int { return grammar.CountFilmChunks(dir) }

const CoverageWarnRatio = grammar.CoverageWarnRatio

func DecodeFrameRecords(br *grammar.Lecteur, w *grammar.World, cfg grammar.FrameConfig) ([]grammar.FrameRecord, error) {
	return grammar.DecodeFrameRecords(br, w, cfg)
}
func DefaultScanFilmOptions() grammar.ScanFilmOptions { return grammar.DefaultScanFilmOptions() }
func DetectI0LayoutOf(film *source.Film) (profile.I0Layout, grammar.I0LayoutReport, error) {
	return grammar.DetectI0LayoutOf(film)
}
func FilmChunkAt(f *source.Film, num int) ([]byte, []grammar.FilmPacket, bool) {
	return grammar.FilmChunkAt(f, num)
}
func FilmChunkNumbers(f *source.Film) []int { return grammar.FilmChunkNumbers(f) }

type FilmContext = grammar.FilmContext

func FilmMajorVersionFromHeader(chunk0 []byte) (int, bool) {
	return grammar.FilmMajorVersionFromHeader(chunk0)
}

const FilmMajorVersionUnknown = grammar.FilmMajorVersionUnknown

func FilmWeaponHitDistance(dir string, entry profile.MapQuantEntry, damages []grammar.WeaponDamage, n int) (grammar.WeaponHitDistanceFunc, int, error) {
	return grammar.FilmWeaponHitDistance(dir, entry, damages, n)
}

type FireEvent = grammar.FireEvent
type FrameConfig = grammar.FrameConfig

func HighlightProfileFromHeader(chunk0 []byte) profile.HighlightProfile {
	return grammar.HighlightProfileFromHeader(chunk0)
}

type KillSourceHealth = grammar.KillSourceHealth

const KnownRegistryFingerprint = grammar.KnownRegistryFingerprint

func LecteurSur(buf []byte) *grammar.Lecteur                { return grammar.LecteurSur(buf) }
func NewFilmContext(film *source.Film) *grammar.FilmContext { return grammar.NewFilmContext(film) }
func NewFilmContextForMap(film *source.Film, entry *profile.MapQuantEntry, forced *profile.I0Layout) *grammar.FilmContext {
	return grammar.NewFilmContextForMap(film, entry, forced)
}
func NewWorld(reg *grammar.Registry) *grammar.World { return grammar.NewWorld(reg) }
func PairWeaponHits(shots []grammar.WeaponShot, damages []grammar.WeaponDamage, window uint64, dist grammar.WeaponHitDistanceFunc) []grammar.WeaponHitStats {
	return grammar.PairWeaponHits(shots, damages, window, dist)
}
func ParseHighlightEvents(data []byte, filmMajorVersion int) ([]highlightevent.HighlightEvent, error) {
	return grammar.ParseHighlightEvents(data, filmMajorVersion)
}
func ParseRegistryChunk(data []byte) (*grammar.Registry, error) {
	return grammar.ParseRegistryChunk(data)
}

type ProfilDeBalayage = grammar.ProfilDeBalayage

func ProfilDeBalayageParDefaut() grammar.ProfilDeBalayage { return grammar.ProfilDeBalayageParDefaut() }
func ReadFilmChunk(dir string, chunk int) ([]byte, error) { return grammar.ReadFilmChunk(dir, chunk) }
func ReadPlayerTable(chunk0 []byte, ident profile.FilmIdentity) ([]types.PlayerSlot, grammar.PlayerTableReport, error) {
	return grammar.ReadPlayerTable(chunk0, ident)
}

type Registry = grammar.Registry

func RegistryFingerprint(reg *grammar.Registry) uint64 { return grammar.RegistryFingerprint(reg) }
func ScanBipedCreations(fc *grammar.FilmContext) ([]grammar.BipedCreation, types.BipedCreationStats, error) {
	return grammar.ScanBipedCreations(fc)
}
func ScanBipedPositions(fc *grammar.FilmContext, opt grammar.ScanFilmOptions) ([]grammar.BipedPosition, error) {
	return grammar.ScanBipedPositions(fc, opt)
}

type ScanFilmOptions = grammar.ScanFilmOptions

func ScanFilmWeaponDamages(dir string, reg *grammar.Registry, n int) ([]grammar.WeaponDamage, int, error) {
	return grammar.ScanFilmWeaponDamages(dir, reg, n)
}
func ScanFilmWeaponShots(dir string, n int) ([]grammar.WeaponShot, error) {
	return grammar.ScanFilmWeaponShots(dir, n)
}

const UnexplainedAlertRatio = grammar.UnexplainedAlertRatio
const UnexplainedWarnRatio = grammar.UnexplainedWarnRatio

func UnknownBuildExpvarPairs(build string) []grammar.ExpvarPair {
	return grammar.UnknownBuildExpvarPairs(build)
}

type WeaponDamage = grammar.WeaponDamage

func WeaponHitBucketCount() int { return grammar.WeaponHitBucketCount() }

type WeaponHitDistanceFunc = grammar.WeaponHitDistanceFunc

const WeaponHitPairWindowUS = grammar.WeaponHitPairWindowUS

type WeaponHitStats = grammar.WeaponHitStats
type World = grammar.World

// ---- killsource ----
type ApparStats = types.ApparStats
type Assist = types.Assist

const BotSuffix = killsource.BotSuffix

func CatalogueProvenance() damagetag.Provenance { return killsource.CatalogueProvenance() }
func CatalogueSize() int                        { return killsource.CatalogueSize() }

type Category = killsource.Category

const CategoryAttachedDamage = killsource.CategoryAttachedDamage
const CategoryCollisionDamage = killsource.CategoryCollisionDamage
const CategoryHeadshot = killsource.CategoryHeadshot
const CategoryHeadshotMultiplier = killsource.CategoryHeadshotMultiplier
const CategoryNone = killsource.CategoryNone
const CategorySilentMelee = killsource.CategorySilentMelee

type CoupleStats = types.CoupleStats
type Coverage = killsource.Coverage
type DamageShare = killsource.DamageShare

// Options / DefaultOptions : LA CONFIGURATION GELEE du decodage des morts, et la CARTE du match
// (`Options.Carte`, lot 3.4.1 — une donnee, pas une bascule). Re-exportees parce que les deux
// appelants de production qui resolvent la carte — `replaybuild` et `sync/killcollector` —
// n atteignent le decodeur que par cette facade.
type Options = killsource.Options

func DefaultOptions() Options { return killsource.DefaultOptions() }
func Decode(ctx context.Context, name string, film *source.Film, opts *killsource.Options) (*killsource.Result, error) {
	return killsource.Decode(ctx, name, film, opts)
}

var ErrNoKillFeed = killsource.ErrNoKillFeed

type FeedTruth = killsource.FeedTruth

const FilmTableNoSection = killsource.FilmTableNoSection

type FilmTablePinning = killsource.FilmTablePinning

const FilmTableRead = killsource.FilmTableRead
const FilmTableUnknownBuild = killsource.FilmTableUnknownBuild

type Kill = killsource.Kill
type Origin = killsource.Origin

const OriginBot = killsource.OriginBot
const OriginBotKiller = killsource.OriginBotKiller
const OriginCredit = killsource.OriginCredit
const OriginSelfSource = killsource.OriginSelfSource

type Path = killsource.Path

const PathScan = killsource.PathScan

type PathStats = killsource.PathStats

const PathWalk = killsource.PathWalk

func ProfilDeDepart() grammar.ProfilDeBalayage { return killsource.ProfilDeDepart() }

type Result = killsource.Result
type Stats = killsource.Stats

const XUIDNamePrefix = killsource.XUIDNamePrefix

// ---- objectives ----
func CaptureBurstTimes(film *source.Film) []int { return objectives.CaptureBurstTimes(film) }
func CountObjectiveFamily[E interface{ StatName() string }](evs []E) int {
	return objectives.CountObjectiveFamily(evs)
}

type DeathInstant = types.DeathInstant

const EventTypeCapture = objectives.EventTypeCapture

func Extract(matchID, gameVariantName string, film *source.Film, roster objectives.Roster) ([]domain.ObjectiveEvent, objectives.TeamControl) {
	return objectives.Extract(matchID, gameVariantName, film, roster)
}
func FlagFilmSignalsFrom(bursts []int, evs []objectives.NamedEvent) objectives.FlagFilmSignals {
	return objectives.FlagFilmSignalsFrom(bursts, evs)
}

type FlagGrabsNetPlayer = types.FlagGrabsNetPlayer
type IdentifiedEvent = objectives.IdentifiedEvent

func IdentifyNamedEvents(evs []objectives.NamedEvent, identity map[int]string) []objectives.IdentifiedEvent {
	return objectives.IdentifyNamedEvents(evs, identity)
}
func IdentifyNamedEventsByRound(evs []objectives.NamedEvent, identity objectives.RoundIdentity) ([]objectives.IdentifiedEvent, int) {
	return objectives.IdentifyNamedEventsByRound(evs, identity)
}

const IdentitySourceTotals = objectives.IdentitySourceTotals

type MapRoster = objectives.MapRoster
type NamedEvent = objectives.NamedEvent

func NamedEvents(film *source.Film, objectiveType string) []objectives.NamedEvent {
	return objectives.NamedEvents(film, objectiveType)
}
func NamedEventsFrom(recs []types.StatRecord, objectiveType string) []objectives.NamedEvent {
	return objectives.NamedEventsFrom(recs, objectiveType)
}
func NetFlagGrabs(tracks []types.FlagTrack, window time.Duration) objectives.FlagGrabsNetResult {
	return objectives.NetFlagGrabs(tracks, window)
}

const ObjectiveTypeBomb = objectives.ObjectiveTypeBomb
const ObjectiveTypeFlag = objectives.ObjectiveTypeFlag
const ObjectiveTypeHill = objectives.ObjectiveTypeHill

func ObjectiveTypeOf(gameVariantName string) string {
	return objectives.ObjectiveTypeOf(gameVariantName)
}

const ObjectiveTypeSkull = objectives.ObjectiveTypeSkull
const ObjectiveTypeZone = objectives.ObjectiveTypeZone
const OriginRoundResidue = objectives.OriginRoundResidue

type PlayerLine = types.PlayerLine

func RealRounds(recs []types.StatRecord) map[int]bool { return objectives.RealRounds(recs) }
func ResolveRoundIdentity(recs []types.StatRecord, deaths []types.DeathInstant) objectives.RoundIdentity {
	return objectives.ResolveRoundIdentity(recs, deaths)
}

const RoleScorer = objectives.RoleScorer

func RosterFitsStatborg(n int) bool { return objectives.RosterFitsStatborg(n) }

type RoundIdentity = objectives.RoundIdentity
type ScorePoint = types.ScorePoint

func SeriesByRound(recs []types.StatRecord, c objectives.StatComponent, teams bool) map[int]map[int][]types.ScorePoint {
	return objectives.SeriesByRound(recs, c, teams)
}
func SeriesTotal(recs []types.StatRecord, c objectives.StatComponent, teams bool) map[int][]types.ScorePoint {
	return objectives.SeriesTotal(recs, c, teams)
}
func SlotIdentityByDeaths(recs []types.StatRecord, deaths []types.DeathInstant) map[int]string {
	return objectives.SlotIdentityByDeaths(recs, deaths)
}
func SlotIdentityResolved(film *source.Film, lines []types.PlayerLine, deaths []types.DeathInstant) (map[int]string, objectives.IdentityStats) {
	return objectives.SlotIdentityResolved(film, lines, deaths)
}

type StatComponent = objectives.StatComponent

const StatFlagCaptures = objectives.StatFlagCaptures
const StatPlayerSlots = objectives.StatPlayerSlots

type StatRecord = types.StatRecord

func StatRecords(film *source.Film) []types.StatRecord { return objectives.StatRecords(film) }
func StatRecordsCtx(ctx context.Context, film *source.Film, matchID string) ([]types.StatRecord, bool) {
	return objectives.StatRecordsCtx(ctx, film, matchID)
}

type StatValue = types.StatValue

const StatZoneCaptures = objectives.StatZoneCaptures
const StatZoneSecures = objectives.StatZoneSecures

// ---- positions ----
type ChunkInput = positions.ChunkInput

func DecodeKeyframePositions(chunks []positions.ChunkInput) []playerposition.PlayerPosition {
	return positions.DecodeKeyframePositions(chunks)
}

// ---- profile ----
var ErrUnknownMapBounds = profile.ErrUnknownMapBounds

type I0Layout = profile.I0Layout

func LoadMapQuantCatalog(path string) (*profile.MapQuantCatalog, error) {
	return profile.LoadMapQuantCatalog(path)
}

type MapQuantCatalog = profile.MapQuantCatalog
type MapQuantEntry = profile.MapQuantEntry

const MapQuantSchemaVersion = profile.MapQuantSchemaVersion

func NormalizeMapName(s string) string { return profile.NormalizeMapName(s) }

// LargeurIndexDePlage : la LOI de `DAT_144632be0`, pour le PRODUCTEUR hors ligne du catalogue
// (`cmd/mapquant-build`), qui la recopiait en boucle a la main (lot 3.4.1, CLAUDE.md regle 6 :
// une meme largeur du jeu n a qu une ecriture).
func LargeurIndexDePlage(nbPlages int) uint { return profile.LargeurIndexDePlage(nbPlages) }

// ---- source ----
type ChunkMeta = types.ChunkMeta

func Decompresser(raw []byte) ([]byte, error) { return source.Decompresser(raw) }

type Film = source.Film

func Inflate(raw []byte) []byte { return source.Inflate(raw) }
func Load(src source.Source, meta []types.ChunkMeta) (*source.Film, error) {
	return source.Load(src, meta)
}
func LoadDir(dir string, meta []types.ChunkMeta) (*source.Film, error) {
	return source.LoadDir(dir, meta)
}

type MemoryChunks = source.MemoryChunks

func Paquets(chunk []byte, ch int) []types.Packet { return source.Paquets(chunk, ch) }

// ---- weaponscan ----
func FindFramePositions(data []byte) []int { return weaponscan.FindFramePositions(data) }
func ScanFireEventsB5(data []byte, estimateTS func(int) float64) []weaponscan.FireEvent {
	return weaponscan.ScanFireEventsB5(data, estimateTS)
}
func ScanFormulaA(data []byte) []weaponscan.FormulaAResult   { return weaponscan.ScanFormulaA(data) }
func ScanFormulaANS(data []byte) []weaponscan.FormulaAResult { return weaponscan.ScanFormulaANS(data) }
func TimestampEstimator(data []byte, startMS, durationMS int) func(int) float64 {
	return weaponscan.TimestampEstimator(data, startMS, durationMS)
}

// ---- weaponv3 ----
var KnownWeaponHigh32 = weaponv3.KnownWeaponHigh32

const PIBits = weaponv3.PIBits

func ResolveBest(rosterXuids []uint64, chunks [][]byte) map[uint64]int {
	return weaponv3.ResolveBest(rosterXuids, chunks)
}
func ResolveXuidToPI(rosterXuids []uint64, chunk []byte) map[uint64]int {
	return weaponv3.ResolveXuidToPI(rosterXuids, chunk)
}
