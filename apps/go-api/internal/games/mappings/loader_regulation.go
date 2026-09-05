package mappings

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/analysis/modelabel"
)

// RegulationSet porte le temps RÉGLEMENTAIRE par variante de jeu d'un titre
// (data-driven, config/titles/{slug}/mappings/regulation.toml). C'est la source
// unique du seuil qui sépare un match terminé dans le temps d'un match parti en
// PROLONGATION (cf. analysis.ComputeOvertime).
//
// La clé est `match_registry.game_variant_name` = un NOM D'ASSET UGC, PAS un
// identifiant stable : il bouge au fil des saisons du titre. Une variante
// inconnue n'a donc pas de temps réglementaire → aucun flag (dégradation sûre,
// jamais de faux positif). Fichier absent = table vide, jamais une erreur.
type RegulationSet struct {
	titleSlug     string
	schemaVersion int
	// seconds : game_variant_name → temps réglementaire en secondes (> 0).
	seconds map[string]int
	// targets : game_variant_name → CIBLE DE VICTOIRE du mode (score qui termine le match
	// quand il est atteint). Même doctrine que seconds : valeurs MESURÉES (plateau du score
	// du vainqueur au registre), variante inconnue → pas de cible, jamais une devinette.
	// Consommateur : le constructeur d'artefact de rejeu (ScoreTimeline.TargetScore).
	targets map[string]int
	// roundsDecide : game_variant_name → la variante se décide aux MANCHES, donc son
	// `CoreStats.Score` (un cumul de points sur toutes les manches) ne dit PAS le résultat.
	// Même doctrine encore : contenu MESURÉ (`.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`),
	// variante absente → on garde les points. Consommateur : analysis.TeamScoreDisplay.
	roundsDecide map[string]bool
	// holdTicks : game_variant_name → TICS DE GARDE qui valent un point, sur un mode
	// où l'on marque en TENANT une zone (KOTH : la colline se prend instantanément, c'est la
	// garde qui compte). Même doctrine que targets : valeur MESURÉE, variante inconnue → pas
	// de dénominateur, donc aucune jauge de progression — jamais une jauge au jugé.
	// Consommateur : le constructeur d'artefact (ScoreTimeline.HoldTicksPerPoint).
	holdTicks map[string]int
	// scoreTimeline : JETON DE MODE → comment le score se montre dans le temps
	// (`hidden` / `events` / `curve`). Contrairement aux quatre tables ci-dessus, la clé
	// n'est PAS un game_variant_name mais un jeton de mode apparié comme dans
	// objective_roles.toml : la règle porte sur la FAMILLE de mode, pas sur une déclinaison
	// de playlist qu'un renommage de saison ferait tomber. Mode non déclaré → repli sûr
	// `curve`, le comportement d'avant la table. Consommateur : MatchViewHeader.
	scoreTimeline map[string]string
	// scoreTimelineTokens : les jetons déclarés, dans un ordre stable — l'appariement
	// (mot entier, jeton le plus long gagnant) les prend tels quels.
	scoreTimelineTokens []string
}

// Les trois lectures possibles du bloc « Score dans le temps » de la vue match. Ce sont
// les SEULES valeurs admises par la table `[score_timeline]` : toute autre est une erreur
// de configuration refusée au chargement, jamais un silence.
const (
	// ScoreTimelineHidden : le bloc ne s'affiche pas (le mode marque au frag — la courbe
	// redirait « Frags cumulés », juste au-dessus dans le même onglet).
	ScoreTimelineHidden = "hidden"
	// ScoreTimelineEvents : des barres verticales aux INSTANTS de marque (le mode marque
	// en 3 à 5 points sur tout le match : une courbe y serait un escalier vide).
	ScoreTimelineEvents = "events"
	// ScoreTimelineCurve : la courbe en escalier — et le REPLI de tout mode non déclaré.
	ScoreTimelineCurve = "curve"
)

// scoreTimelineKinds — la liste FERMÉE des lectures admises.
var scoreTimelineKinds = map[string]bool{
	ScoreTimelineHidden: true,
	ScoreTimelineEvents: true,
	ScoreTimelineCurve:  true,
}

// regulationTOML — projection brute de regulation.toml.
type regulationTOML struct {
	Meta          metaSection       `toml:"meta"`
	Seconds       map[string]int    `toml:"regulation_seconds"`
	Targets       map[string]int    `toml:"score_target"`
	RoundsDecide  map[string]bool   `toml:"rounds_decide"`
	HoldTicks     map[string]int    `toml:"hold_ticks_per_point"`
	ScoreTimeline map[string]string `toml:"score_timeline"`
}

// Seconds retourne le temps réglementaire de la variante et true s'il est connu.
// nil-safe et variante inconnue → (0, false) : l'appelant ne flague pas.
func (s *RegulationSet) Seconds(gameVariantName string) (int, bool) {
	if s == nil {
		return 0, false
	}
	v, ok := s.seconds[strings.TrimSpace(gameVariantName)]
	return v, ok
}

// ScoreTarget retourne la cible de victoire de la variante et true si elle est connue.
// nil-safe et variante inconnue → (0, false) : l'appelant retombe sur son repli.
func (s *RegulationSet) ScoreTarget(gameVariantName string) (int, bool) {
	if s == nil {
		return 0, false
	}
	v, ok := s.targets[strings.TrimSpace(gameVariantName)]
	return v, ok
}

// HoldTicksPerPoint retourne le nombre de secondes de GARDE qui valent un point sur la
// variante, et true s'il est connu.
//
// nil-safe et variante inconnue → (0, false) : l'appelant ne publie aucun dénominateur, donc
// le client n'affiche AUCUNE jauge de progression. Une jauge absente ne ment pas ; une jauge
// remplie sur un dénominateur inventé, si.
func (s *RegulationSet) HoldTicksPerPoint(gameVariantName string) (int, bool) {
	if s == nil {
		return 0, false
	}
	v, ok := s.holdTicks[strings.TrimSpace(gameVariantName)]
	return v, ok
}

// ScoreTimelineKind dit COMMENT le score du mode se montre dans le temps :
// `hidden` (le mode marque au frag — la courbe redirait « Frags cumulés »),
// `events` (3 à 5 points sur tout le match — des barres aux instants de marque), ou
// `curve` (le score monte en continu — la courbe en escalier).
//
// `pairName` est le `pair_name` BRUT du match, dont on n'a retiré que le suffixe de CARTE
// (modelabel.StripMapSuffix). La table est indexée par JETON de mode, cherché comme mot
// entier, insensible à la casse, jeton le plus long gagnant (analysis/modelabel, une seule
// implémentation dans le dépôt).
//
// ET NON PAS UN LIBELLÉ NORMALISÉ, contrairement à objective_roles.toml : la normalisation
// MANGE le jeton de mode sur toute une famille de pair_name — « Super Fiesta:Slayer » y
// devient « Super Fiesta », « Team Slayer:Arena » devient « Arena ». Mesure du 2026-09-03
// sur le registre local : 460 matchs recevraient le mauvais verdict, dont les 429 du mode le
// plus joué du corpus. Le détail des neuf familles est dans le commentaire de la table
// (config/titles/halo_infinite/mappings/regulation.toml).
//
// LE RETRAIT DU SUFFIXE DE CARTE, LUI, EST INDISPENSABLE : c'est lui qui empêche un nom de
// carte de porter un jeton de mode.
//
// nil-safe, table absente, libellé vide ou mode non déclaré → `curve` : le REPLI SÛR,
// c'est-à-dire le comportement d'avant la table. Un mode inconnu ne fait jamais
// disparaître le bloc.
func (s *RegulationSet) ScoreTimelineKind(pairName string) string {
	if s == nil || len(s.scoreTimeline) == 0 {
		return ScoreTimelineCurve
	}
	token := modelabel.ExtractKnownMode(pairName, s.scoreTimelineTokens)
	if token == "" {
		return ScoreTimelineCurve
	}
	if kind, ok := s.scoreTimeline[token]; ok {
		return kind
	}
	return ScoreTimelineCurve
}

// RoundsDecide dit si le RÉSULTAT de la variante se lit en MANCHES plutôt qu'en points.
// nil-safe et variante inconnue → false : l'appelant garde les points (dégradation sûre,
// jamais un affichage inventé).
func (s *RegulationSet) RoundsDecide(gameVariantName string) bool {
	if s == nil {
		return false
	}
	return s.roundsDecide[strings.TrimSpace(gameVariantName)]
}

// RoundsDecideVariants retourne les variantes déclarées, triées (ordre déterministe pour
// les appelants qui les injectent dans une requête ou un journal). nil-safe : liste vide.
// Utilisé par `cmd/backfill-team-rounds` pour ne re-lire que l'historique qui en a besoin.
func (s *RegulationSet) RoundsDecideVariants() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.roundsDecide))
	for k := range s.roundsDecide {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// RoundsDecideMap retourne une copie de la table variante → « se lit en manches ». nil-safe
// (map vide). Pendant de SecondsMap : le wiring injecte la table dans les services sans
// exposer le type interne.
func (s *RegulationSet) RoundsDecideMap() map[string]bool {
	out := make(map[string]bool)
	if s == nil {
		return out
	}
	for k, v := range s.roundsDecide {
		out[k] = v
	}
	return out
}

// SecondsMap retourne une copie de la table variante → secondes. nil-safe (map
// vide). Utilisé par le wiring pour injecter la table dans les services sans
// exposer le type interne.
func (s *RegulationSet) SecondsMap() map[string]int {
	out := make(map[string]int)
	if s == nil {
		return out
	}
	for k, v := range s.seconds {
		out[k] = v
	}
	return out
}

// TitleSlug retourne le slug déclaré.
func (s *RegulationSet) TitleSlug() string {
	if s == nil {
		return ""
	}
	return s.titleSlug
}

// SchemaVersion retourne la version de schéma déclarée.
func (s *RegulationSet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadRegulationFromFile lit et valide un regulation.toml à un chemin donné.
func LoadRegulationFromFile(path string) (*RegulationSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadRegulationFromBytes(path, raw)
}

// LoadRegulationFromBytes parse et valide un payload TOML déjà en mémoire.
// Validation stricte : title_slug + schema_version obligatoires ; chaque entrée
// a une clé non vide et une valeur > 0. Une table VIDE est VALIDE (titre sans
// temps réglementaire mesuré, ex. Halo 5 → aucun flag).
func LoadRegulationFromBytes(path string, raw []byte) (*RegulationSet, error) {
	var doc regulationTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	seconds := make(map[string]int, len(doc.Seconds))
	for rawName, secs := range doc.Seconds {
		key := strings.TrimSpace(rawName)
		if key == "" {
			return nil, fmt.Errorf("%s: game_variant_name vide", path)
		}
		if secs <= 0 {
			return nil, fmt.Errorf("%s: variante %q : temps réglementaire doit être > 0 (reçu %d)", path, key, secs)
		}
		seconds[key] = secs
	}
	targets := make(map[string]int, len(doc.Targets))
	for rawName, target := range doc.Targets {
		key := strings.TrimSpace(rawName)
		if key == "" {
			return nil, fmt.Errorf("%s: [score_target] game_variant_name vide", path)
		}
		if target <= 0 {
			return nil, fmt.Errorf("%s: variante %q : cible de victoire doit être > 0 (reçu %d)", path, key, target)
		}
		targets[key] = target
	}
	holds := make(map[string]int, len(doc.HoldTicks))
	for rawName, secs := range doc.HoldTicks {
		key := strings.TrimSpace(rawName)
		if key == "" {
			return nil, fmt.Errorf("%s: [hold_ticks_per_point] game_variant_name vide", path)
		}
		if secs <= 0 {
			return nil, fmt.Errorf("%s: variante %q : secondes de garde par point doit être > 0 (reçu %d)", path, key, secs)
		}
		holds[key] = secs
	}
	rounds := make(map[string]bool, len(doc.RoundsDecide))
	for rawName, decides := range doc.RoundsDecide {
		key := strings.TrimSpace(rawName)
		if key == "" {
			return nil, fmt.Errorf("%s: [rounds_decide] game_variant_name vide", path)
		}
		// Une entrée `false` n'existe pas : l'absence EST le « non ». L'accepter ferait
		// croire qu'on peut désactiver quelque chose depuis cette table, alors que la
		// dégradation se lit par l'absence de clé.
		if !decides {
			return nil, fmt.Errorf("%s: [rounds_decide] variante %q à false — retirer la ligne (l'absence vaut « non »)", path, key)
		}
		rounds[key] = true
	}
	timeline, timelineTokens, err := parseScoreTimeline(path, doc.ScoreTimeline)
	if err != nil {
		return nil, err
	}
	return &RegulationSet{
		titleSlug:           doc.Meta.TitleSlug,
		schemaVersion:       doc.Meta.SchemaVersion,
		seconds:             seconds,
		targets:             targets,
		roundsDecide:        rounds,
		holdTicks:           holds,
		scoreTimeline:       timeline,
		scoreTimelineTokens: timelineTokens,
	}, nil
}

// parseScoreTimeline valide la table `[score_timeline]` : jeton non vide, lecture DANS la
// liste fermée (`hidden` / `events` / `curve`).
//
// UNE VALEUR INCONNUE EST UNE ERREUR DE CHARGEMENT, JAMAIS UN SILENCE. Une faute de frappe
// (`event` au lieu de `events`) tomberait sinon sur le repli `curve` et se lirait à l'écran
// comme une décision produit — le mode garderait sa courbe alors que la table dit le
// contraire, et rien ne le signalerait.
//
// Les jetons sortent TRIÉS : l'appariement (`ExtractKnownMode`) départage sur la longueur,
// mais l'ordre stable garde le comportement reproductible et les journaux comparables.
func parseScoreTimeline(path string, raw map[string]string) (map[string]string, []string, error) {
	kinds := make(map[string]string, len(raw))
	tokens := make([]string, 0, len(raw))
	for rawToken, rawKind := range raw {
		token := strings.TrimSpace(rawToken)
		if token == "" {
			return nil, nil, fmt.Errorf("%s: [score_timeline] jeton de mode vide", path)
		}
		kind := strings.TrimSpace(rawKind)
		if !scoreTimelineKinds[kind] {
			return nil, nil, fmt.Errorf(
				"%s: [score_timeline] jeton %q : lecture %q inconnue — attendu %q, %q ou %q",
				path, token, rawKind, ScoreTimelineHidden, ScoreTimelineEvents, ScoreTimelineCurve)
		}
		kinds[token] = kind
		tokens = append(tokens, token)
	}
	sort.Strings(tokens)
	return kinds, tokens, nil
}
