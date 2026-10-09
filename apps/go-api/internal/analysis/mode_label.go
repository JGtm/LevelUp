// Package analysis — mode_label.go : normalisation canonique des labels de mode de jeu.
//
// Port Go de src.analysis.mode_display (branche Python v7/cockpit).
// Logique unifiée utilisée par la home, l'historique de matchs et les filtres.
// Aucun accès DB — function pure.
//
// COMPLÉMENTAIRE de mode_category.go : ce fichier extrait le SOUS-MODE
// ("Arena:Slayer" → "Slayer"), tandis que mode_category.go infère la
// CATÉGORIE PARENTE ("Arena:Slayer" → "Assassin"). Voir l'en-tête de
// mode_category.go pour le détail des 2 niveaux orthogonaux.
//
// LES DEUX GRAMMAIRES D'UN pair_name (depuis le 2026-09-19) : « Conteneur:Mode on Carte »
// (« Arena:Slayer on Bazaar ») ET la forme INVERSÉE « Mode:Conteneur [qualificatif] on Carte »
// (« Slayer:Arena on Live Fire », « CTF:Arena Neutral Flag on Cliffhanger »). Les jetons de
// conteneur vivent dans `analysis/modelabel` (une seule liste, paquet feuille) ; ce fichier
// est le chokepoint unique — ses appelants ne connaissent pas la grammaire.
package analysis

import (
	"regexp"
	"strings"

	"levelup/go-api/internal/analysis/modelabel"
)

// Regex partagées pour la normalisation des modes.
var (
	// Strip " - Forge" et " - Ranked" (suffixes Halo Infinite).
	modeLabelForgeRe  = regexp.MustCompile(`(?i)\s*-\s*Forge\b`)
	modeLabelRankedRe = regexp.MustCompile(`(?i)\s*-\s*Ranked\b`)
)

// Labels d'identité de playlists Halo Infinite — partagés avec mode_category.go
// (côté games/halo_infinite). Re-déclarés ici pour éviter un cycle d'import
// analysis ← halo_infinite ← analysis.
const (
	modeLabelSuperFiesta    = "Super Fiesta"
	modeLabelHuskyRaid      = "Husky Raid"
	modeLabelSuperHuskyRaid = "Super Husky Raid"
)

// playlistIdentityPrefixes : préfixes pair_name qui SONT le label utilisateur,
// donc à conserver tels quels au lieu d'extraire le sous-mode après ":".
//
// Exemple : "Super Fiesta:Slayer on Forbidden - Forge" doit afficher
// "Super Fiesta" (et non "Slayer", qui serait traduit FR en "Assassin" via
// mode_name_tr et noierait l'identité de la playlist).
//
// Liste alignée sur les catégories promues côté `halo_infinite/mode_category.go`
// (ModeCategorySuperFiesta, ModeCategoryHuskyRaid). Les "containers"
// (Arena/Tactical/Assault/Community) restent extraits au sous-mode.
//
// Comparaison case-insensitive — les pair_name peuvent arriver avec une casse
// inconsistante depuis l'API Halo.
var playlistIdentityPrefixes = map[string]string{
	"super fiesta":     modeLabelSuperFiesta,
	"husky raid":       modeLabelHuskyRaid,
	"super husky raid": modeLabelSuperHuskyRaid,
}

// NormalizeModeLabel normalise un label brut de mode de jeu Halo Infinite.
//
// Logique (alignée sur Python resolve_display_mode + translate_pair_name) :
//  1. Strip map-label connu : " sur {map}" / " on {map}" → retiré en priorité.
//  2. Extraction du mode depuis le format pair_name (extractModeFromPairName) :
//     - Format FR avec séparateur espacé " : " → prend la partie avant
//     ("Assassin : Classé" → "Assassin").
//     - Format technique "Conteneur:Mode" → prend la partie après le dernier ":"
//     ("Arena:Slayer" → "Slayer", "BTB:CTF" → "CTF") ; identité de playlist à gauche
//     conservée ("Super Fiesta:Slayer" → "Super Fiesta").
//     - Forme INVERSÉE "Mode:Conteneur [qualificatif]" → le mode est à gauche, recollé
//     derrière le qualificatif ("Slayer:Arena" → "Slayer", "CTF:Arena Neutral Flag" →
//     "Neutral Flag CTF", "Slayer:Arena Super Fiesta" → "Super Fiesta").
//  3. Strip générique " sur .+" / " on .+" (FR + EN) si non retiré à l'étape 1.
//  4. Strip " - Forge" et " - Ranked".
//
// mapLabels est optionnel : les noms de map connus pour l'étape 1.
func NormalizeModeLabel(raw string, mapLabels ...string) string {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return ""
	}

	// Étape 1 — strip map-label connu (prioritaire, avant extraction du préfixe)
	for _, mapLabel := range mapLabels {
		trimmedMap := strings.TrimSpace(mapLabel)
		if trimmedMap == "" {
			continue
		}
		mapSpecificRe := regexp.MustCompile(`(?i)\s+(?:on|sur)\s+` + regexp.QuoteMeta(trimmedMap) + `$`)
		updated := mapSpecificRe.ReplaceAllString(normalized, "")
		if updated != normalized {
			normalized = strings.TrimSpace(updated)
			break
		}
	}

	// Étape 2 — extraction du mode depuis le format pair_name (les deux grammaires)
	normalized = extractModeFromPairName(normalized)

	// Étape 3 — strip générique " sur/on <carte>" résiduel. La regex vit dans le paquet
	// feuille `modelabel` : l'appariement du bloc « Score dans le temps » a besoin du MÊME
	// retrait sans le reste de cette normalisation, et deux expressions du même découpage
	// finiraient par couper différemment (règle CLAUDE.md n°6).
	normalized = modelabel.StripMapSuffix(normalized)

	// Étape 4 — strip suffixes Forge / Ranked
	normalized = modeLabelForgeRe.ReplaceAllString(normalized, "")
	normalized = modeLabelRankedRe.ReplaceAllString(normalized, "")

	return strings.TrimSpace(normalized)
}

// extractModeFromPairName — l'étape 2 de NormalizeModeLabel : le MODE d'un pair_name.
//
//   - Format FR " : " espacé : « Assassin : Classé » → « Assassin » (partie avant).
//   - Format technique « left:right » (découpé sur le DERNIER « : ») :
//     1. left est une identité de playlist (Super Fiesta, Husky Raid, Super Husky Raid) →
//     l'identité canonique, pour ne pas afficher « Slayer/Assassin » sur une tuile
//     Super Fiesta (cf. thought_log 2026-05-08). Testé AVANT la règle du conteneur : ces
//     identités sont aussi des conteneurs de grammaire.
//     2. left est un conteneur (modelabel.IsContainer) → right : « Arena:Slayer » → « Slayer »,
//     « Ranked:Doubles Slayer » → « Doubles Slayer ».
//     3. right COMMENCE par un conteneur → forme INVERSÉE « Mode:Conteneur [qualificatif] » :
//     cf. invertedModeLabel.
//     4. sinon → right (défaut) : « Infection:Alpha Zombies » → « Alpha Zombies ».
//
// Sans « : » (ou « : » final), le libellé sort intact.
func extractModeFromPairName(label string) string {
	if idx := strings.Index(label, " : "); idx > 0 {
		return strings.TrimSpace(label[:idx])
	}
	idx := strings.LastIndex(label, ":")
	if idx < 0 || idx >= len(label)-1 {
		return label
	}
	left := strings.TrimSpace(label[:idx])
	right := strings.TrimSpace(label[idx+1:])
	if canonical, ok := playlistIdentityPrefixes[strings.ToLower(left)]; ok {
		return canonical
	}
	if modelabel.IsContainer(left) {
		return right
	}
	// Forme inversée : le suffixe de carte suit le qualificatif (« Arena Neutral Flag on
	// Cliffhanger ») — il se retire ICI, avant le recollage, sinon l'étape 3 mangerait le mode
	// recollé derrière lui (« Neutral Flag on Cliffhanger CTF » → « Neutral Flag »).
	if _, rest, ok := modelabel.SplitContainer(modelabel.StripMapSuffix(right)); ok {
		return invertedModeLabel(left, rest)
	}
	return right
}

// invertedModeLabel recolle le mode d'un pair_name en grammaire inversée « left:Conteneur rest » :
//   - rest vide → left : « Slayer:Arena » → « Slayer », « Gruntpocalypse:Fiesta » → « Gruntpocalypse » ;
//   - rest est une identité de playlist → l'identité canonique : « Slayer:Arena Super Fiesta »
//     → « Super Fiesta » (même règle que le préfixe gauche : l'identité prime sur le sous-mode) ;
//   - sinon → « rest left », l'ordre naturel des libellés de mode_name_tr : « CTF:Arena Neutral
//     Flag » → « Neutral Flag CTF », « Slayer:Arena Tactical » → « Tactical Slayer »,
//     « CTF:BTB Fiesta » → « Fiesta CTF ».
func invertedModeLabel(left, rest string) string {
	if rest == "" {
		return left
	}
	if canonical, ok := playlistIdentityPrefixes[strings.ToLower(rest)]; ok {
		return canonical
	}
	return rest + " " + left
}

// ResolveModeUI applique la formule canonique de résolution du libellé de mode
// affiché côté UI (home tiles, match-view header, historique).
//
// Source unique de vérité : COALESCE(pair_name_fr, pair_name) → NormalizeModeLabel.
// Tout caller qui a besoin d'un libellé de mode propre (sans suffixe map, sans
// préfixe technique "Arena:") doit appeler ce helper plutôt que d'inventer
// sa propre cascade — la divergence home/match-view qui produisait
// "Slayer on Streets" sur la vue détail est née d'une cascade ad hoc.
//
// Retourne nil si les deux sources sont vides après normalisation.
func ResolveModeUI(pairName, pairNameFR *string) *string {
	src := ""
	if pairNameFR != nil && *pairNameFR != "" {
		src = *pairNameFR
	} else if pairName != nil {
		src = *pairName
	}
	out := NormalizeModeLabel(src)
	if out == "" {
		return nil
	}
	return &out
}

// ResolveModeUIWithVariant retourne le libellé de MODE côté UI : pair prioritaire
// (FR sinon EN, normalisé via NormalizeModeLabel), sinon fallback game_variant
// (FR sinon EN, normalisé) pour les titres sans pair_name (Halo 5,
// games/halo_5/mapping.go PairMode nil). Source unique de la convention
// pair-sinon-variant — ne pas recopier ce pattern ailleurs. Retourne nil si
// aucune source (pair ET variant vides).
//
// Différence assumée avec ResolveModeUI : si une source non vide est sur-nettoyée
// par NormalizeModeLabel (résultat vide, ex. un brut réduit à un pur suffixe
// "- Forge"), on retombe sur la valeur brute trimée au lieu de perdre le libellé.
// Un pair_name / game_variant réel porte toujours un token de mode, donc ce repli
// ne se déclenche pas sur les données de production (iso-comportement effectif).
func ResolveModeUIWithVariant(pairName, pairNameFR, variantName, variantNameFR *string) *string {
	if v := resolveModeSource(pairNameFR, pairName); v != nil {
		return v
	}
	return resolveModeSource(variantNameFR, variantName)
}

// resolveModeSource prend la première source non vide (préférence au 1er argument
// = FR), applique NormalizeModeLabel, et retombe sur la valeur brute trimée si la
// normalisation vide le libellé. Retourne nil si les deux sources sont vides.
func resolveModeSource(preferred, fallback *string) *string {
	src := ""
	if preferred != nil && *preferred != "" {
		src = *preferred
	} else if fallback != nil && *fallback != "" {
		src = *fallback
	}
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}
	if norm := NormalizeModeLabel(src); norm != "" {
		return &norm
	}
	return &src
}

// ExtractKnownMode retrouve un mode CANONIQUE connu à l'intérieur d'un label déjà
// normalisé mais non reconnu tel quel — typiquement une variante d'arme/saison
// ("Legacy Slayer BR" → "Slayer", "Tactical Slayer" → "Slayer"). Cherche le mode
// connu LE PLUS LONG apparaissant comme mot entier (case-insensitive).
//
// knownModesEN = noms EN canoniques (clés mode_en de mode_name_tr), chargés par le
// caller (repo). Retourne "" si aucun match → le caller garde le label d'origine.
// Fonction pure (aucun accès DB), à appliquer APRÈS NormalizeModeLabel.
//
// LA RÈGLE D'APPARIEMENT ELLE-MÊME VIT DANS `analysis/modelabel` depuis le 2026-09-03 :
// `games/mappings` en a besoin pour la table `[score_timeline]` de regulation.toml et ne
// peut pas importer `analysis` (cycle). Ce point d'entrée est conservé — tous ses
// appelants sont inchangés — mais il n'y a qu'UNE implémentation.
func ExtractKnownMode(label string, knownModesEN []string) string {
	return modelabel.ExtractKnownMode(label, knownModesEN)
}
