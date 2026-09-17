// Package service — compare_weapons.go : LE PROFIL D'ARMES DU FACE-À-FACE
// (plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md, lot 3).
//
// # CE QUE CE FICHIER ASSEMBLE
//
// Trois blocs par joueur — la part des frags par CLASSE, la portée par RÔLE, les trois armes
// les plus meurtrières — et rien d'autre. Les percentiles vivent dans `internal/analysis`, le
// SQL chez les repos, la répartition par classe dans `service/fragdist` : aucune de ces trois
// frontières ne bouge ici.
//
// # BEST-EFFORT ABSOLU : CE PROFIL NE FAIT JAMAIS TOMBER LA PAGE
//
// Il est ADDITIF à une page qui existe depuis longtemps. Toute panne de lecture, toute
// capability manquante, tout joueur non résolu rend simplement un profil (ou un bloc) absent.
// Aucune erreur ne remonte à `GetPage` — même régime que `loadWeaponAccuracy` et
// `buildWeaponRangeSection`, avec la même distinction Debug (absence légitime) / Warn
// (anomalie), parce qu'une capability manquante en Warn noierait les vrais bugs SQL.
//
// # AUCUNE COMPARAISON DE SLUG, AUCUN LIBELLÉ (D9/D10)
//
// Ce qui décide de la présence d'un bloc, c'est le REPO : lui seul sait si le titre produit
// des positions par kill, et il le dit par `games.ErrCapabilityNotSupported`. Les classes et
// les rôles voyagent en CLÉS ; les noms d'armes viennent de la chaîne `weapon_names.toml` du
// titre, portée par `port.WeaponKillRow`. Aucune chaîne FR/EN ne s'écrit ici.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/fragdist"
)

// compareTopWeaponsN : le nombre d'armes publiées par joueur (D1).
//
// TROIS, ET PAS VINGT comme la Synthèse : ici les deux colonnes se lisent CÔTE À CÔTE, et une
// liste longue ferait scruter deux classements au lieu de comparer deux styles. C'est un
// choix de lecture, pas une limite technique.
const compareTopWeaponsN = 3

// weaponImageFunc rend l'URL de l'icône d'une arme et dit si c'est un MASQUE à teinter.
//
// Injectée plutôt que résolue ici : la construction d'URL est un adaptateur d'ASSETS, propre
// au titre, et le paquet `service` n'a pas le droit d'en connaître la forme (ADR 0011). Nil =
// pas d'icône, jamais une panne.
type weaponImageFunc func(weaponID int64) (url string, tinted bool)

// WithWeaponProfile câble le profil d'armes. NIL-SAFE : sans les DEUX repos, `Weapons` reste
// absent de la réponse et la page sert ses métriques comme avant.
//
// POURQUOI LES DEUX ET PAS L'UN OU L'AUTRE. Le profil sans kills par arme n'aurait ni classes
// ni top 3 — il ne resterait que la portée, qui est le bloc le plus fragile (elle dépend d'un
// décodeur de film). Un demi-profil se lirait comme une page cassée ; son absence se lit comme
// une page qui n'a pas cette section.
func (s *CompareService) WithWeaponProfile(
	kills port.WeaponKillsRepository, rng port.WeaponRangeRepository, image weaponImageFunc,
) *CompareService {
	if kills == nil || rng == nil {
		return s
	}
	s.weaponKills = kills
	s.weaponRange = rng
	s.weaponImage = image
	return s
}

// buildWeaponProfile assemble le profil des deux joueurs, ou nil.
//
// # LE SCOPE DE CHAQUE CÔTÉ SUIT LA DOCTRINE DE LA PAGE (D2, amendé au lot 3-bis)
//
//	A et B -> tous leurs matchs présents dans la base partagée, campagne exclue
//	B sans xuid, ou absent de la base -> rien : il n'y a rien à mesurer
//
// UN SEUL CHEMIN, ET C'EST UN RÉSULTAT. Le plan prévoyait un scope « croisé » (les matchs
// communs avec A) pour un B non suivi ; il a été retiré le 2026-09-17 parce qu'il était un
// SOUS-ENSEMBLE du scope ci-dessus — même table, même exclusion, plus un `EXISTS`. La branche
// qui le servait ne pouvait donc jamais être prise.
//
// NIL SI L'UN DES DEUX CÔTÉS MANQUE, et c'est délibéré : la section compare deux joueurs. Un
// seul côté publié inviterait à lire un profil comme s'il avait un vis-à-vis.
func (s *CompareService) buildWeaponProfile(
	ctx context.Context, xuidB string,
) *domain.CompareWeaponProfile {
	if s.weaponKills == nil || s.weaponRange == nil || s.xuidA == "" || xuidB == "" {
		return nil
	}
	scopeA := s.weaponScope(ctx, s.xuidA)
	scopeB := s.weaponScope(ctx, xuidB)
	if scopeA == nil || scopeB == nil {
		return nil
	}
	profile := &domain.CompareWeaponProfile{
		PlayerA: s.buildWeaponSide(ctx, scopeA, s.xuidA),
		PlayerB: s.buildWeaponSide(ctx, scopeB, xuidB),
	}
	slog.DebugContext(ctx, "compare: profil d armes assemble",
		"title", s.titleSlug, "matchs_a", scopeA.Matches, "matchs_b", scopeB.Matches,
		"portee_a", profile.PlayerA.Range != nil, "portee_b", profile.PlayerB.Range != nil)
	return profile
}

// weaponScope lit le scope d'un joueur. nil si le joueur n'a aucun match dans la base
// partagée, ou si la lecture échoue (journalisée, jamais avalée).
func (s *CompareService) weaponScope(ctx context.Context, xuid string) *domain.CompareWeaponScope {
	scope, err := s.repo.GetWeaponScope(ctx, xuid, s.titleSlug)
	if err != nil {
		logBestEffortErr(ctx, "compare: scope d armes non disponible", err,
			"xuid", xuid, "title", s.titleSlug)
		return nil
	}
	return scope
}

// buildWeaponSide assemble les trois blocs d'UN joueur sur son scope.
//
// CHAQUE BLOC EST INDÉPENDANT (D9) : la portée peut manquer sans emporter les classes, et le
// top 3 peut être vide sans emporter la portée. Les trois pannes possibles sont distinctes, et
// les confondre cacherait ce qui est pourtant mesuré.
func (s *CompareService) buildWeaponSide(
	ctx context.Context, scope *domain.CompareWeaponScope, xuid string,
) domain.CompareWeaponSide {
	side := domain.CompareWeaponSide{
		Matches:     scope.Matches,
		TotalKills:  scope.Kills,
		FragClasses: []domain.CompareFragClass{},
		TopWeapons:  []domain.CompareTopWeapon{},
	}
	rows := s.loadCompareWeaponKills(ctx, scope.MatchIDs, xuid)
	if len(rows) > 0 {
		side.FragClasses = s.compareFragClasses(ctx, rows, scope, xuid)
		side.TopWeapons = s.compareTopWeapons(rows)
	}
	side.Range = s.compareWeaponRange(ctx, scope, xuid)
	return side
}

// loadCompareWeaponKills lit les frags par arme du joueur sur le scope, dimensions résolues.
//
// `ResolveRoles: true` : la classe ET le rôle arrivent dans LA MÊME passe registre, sans quoi
// il faudrait une seconde résolution pour la répartition par classe.
func (s *CompareService) loadCompareWeaponKills(
	ctx context.Context, matchIDs []string, xuid string,
) []port.WeaponKillRow {
	if xuid == "" || len(matchIDs) == 0 {
		return nil
	}
	rows, err := s.weaponKills.LoadWeaponKillsAggregated(ctx, s.titleSlug, port.WeaponKillFilters{
		MatchIDs: matchIDs, XUIDs: []string{xuid}, ResolveRoles: true,
	})
	if err != nil {
		logCompareWeaponFailure(ctx, "frags par arme", s.titleSlug, xuid, len(matchIDs), err)
		return nil
	}
	return rows
}

// compareFragClasses projette la répartition par classe en parts de frags.
//
// # LE DÉNOMINATEUR EST LE TOTAL DU SCOPE, PAS LA SOMME DES CLASSES
//
// `fragdist.Build` rend un résidu « non attribué » précisément parce que toutes les armes ne
// sont pas résolues. Le CONSERVER est ce qui fait sommer les parts à 100 : l'écarter ferait
// afficher des parts qui ne bouclent pas, et le lecteur mettrait la différence sur le compte
// d'un arrondi alors qu'elle mesure un trou de registre.
//
// Les classes à ZÉRO frag sont omises : une part de 0 % n'apprend rien et allonge la colonne.
func (s *CompareService) compareFragClasses(
	ctx context.Context, rows []port.WeaponKillRow, scope *domain.CompareWeaponScope, xuid string,
) []domain.CompareFragClass {
	fd := fragdist.Build(rows, domain.FragKillTypeCounts{
		Melee:   scope.MeleeKills,
		Grenade: scope.GrenadeKills,
		Total:   scope.Kills,
	}, titleHasNativeKillMechanics(s.titleSlug))
	logFragDistribution(ctx, "compare", s.titleSlug, xuid, fd)

	out := make([]domain.CompareFragClass, 0, len(fd.Classes))
	for _, c := range fd.Classes {
		if c.Kills <= 0 {
			continue
		}
		part := 0.0
		if fd.TotalKills > 0 {
			part = 100 * float64(c.Kills) / float64(fd.TotalKills)
		}
		out = append(out, domain.CompareFragClass{Class: c.Class, Kills: c.Kills, SharePct: part})
	}
	return out
}

// compareTopWeapons projette les trois armes les plus meurtrières, icône comprise.
//
// Le tri vient de `topWeaponKillRows`, définition canonique unique du dépôt (D11) : le
// recopier ici en ferait la troisième copie, et les trois divergeraient au premier changement
// de doctrine de départage.
func (s *CompareService) compareTopWeapons(rows []port.WeaponKillRow) []domain.CompareTopWeapon {
	top := topWeaponKillRows(rows, compareTopWeaponsN)
	out := make([]domain.CompareTopWeapon, 0, len(top))
	for _, r := range top {
		w := domain.CompareTopWeapon{
			Label: r.Label, LabelEN: r.LabelEN, Kills: r.Kills, Class: r.Class, Role: r.Role,
		}
		if s.weaponImage != nil {
			w.ImageURL, w.ImageTinted = s.weaponImage(r.WeaponID)
		}
		out = append(out, w)
	}
	return out
}

// compareWeaponRange assemble le bloc de portée PAR RÔLE, ou nil.
//
// # POURQUOI PAR RÔLE ET NON PAR ARME (D1)
//
// Par arme, une trentaine de lignes dont presque toutes passent sous le seuil de publication :
// le graphe serait vide et la liste « sous le seuil » illisible. Par classe, « lourde »
// mélangerait un fusil de précision et une épée — une portée qui ne décrit personne. Le rôle
// est le seul grain où les lignes ont assez de mesures ET un sens de distance homogène.
//
// # LE REKEYAGE SE FAIT AVANT L'AGRÉGAT, JAMAIS APRÈS
//
// `analysis.RegroupMeasuredKills` change la clé des frags, puis `buildWeaponRangeBlock`
// agrège. Fusionner des lignes DÉJÀ agrégées mélangerait des percentiles, ce qui n'a aucun
// sens : la médiane d'un ensemble ne se déduit pas des médianes de ses parties.
//
// # LES LIBELLÉS NE SONT PAS HYDRATÉS (D8)
//
// `hydrateWeaponRangeLabels` nommerait des CLÉS D'ARME ; ici les clés sont des RÔLES, que la
// metadata ne connaît pas sous ce nom. Les champs `label`/`label_en` restent donc vides et le
// front résout `frags.role.<clé>` depuis son manifeste — jamais un libellé écrit en Go.
func (s *CompareService) compareWeaponRange(
	ctx context.Context, scope *domain.CompareWeaponScope, xuid string,
) *domain.SynthesisWeaponRange {
	if xuid == "" || len(scope.MatchIDs) == 0 {
		return nil
	}
	filtres := port.WeaponRangeFilters{MatchIDs: scope.MatchIDs, XUIDs: []string{xuid}}
	mesures, err := s.weaponRange.LoadWeaponRange(ctx, s.titleSlug, filtres)
	if err != nil {
		logCompareWeaponFailure(ctx, "portee", s.titleSlug, xuid, len(scope.MatchIDs), err)
		return nil
	}
	if len(mesures) == 0 {
		return nil
	}
	roles := s.resolveWeaponRoles(ctx, mesures)
	regroupes, ecartes := analysis.RegroupMeasuredKills(mesures, func(cle string) string {
		return roles[cle]
	})
	if ecartes > 0 {
		// Une clé sans rôle est écartée (D8). Debug et non Warn : un registre incomplet
		// est un état connu, pas une panne — mais un silence total en ferait un zéro.
		slog.DebugContext(ctx, "compare: frags mesures ecartes faute de role",
			"title", s.titleSlug, "xuid", xuid, "ecartes", ecartes, "gardes", len(regroupes))
	}
	if len(regroupes) == 0 {
		return nil
	}
	return buildWeaponRangeBlock(regroupes, nil, weaponRangeScopeInfo{
		matchIDs: scope.MatchIDs, totalKills: scope.Kills, totalDeaths: scope.Deaths,
	})
}

// resolveWeaponRoles traduit les clés d'arme des frags mesurés en clés de RÔLE.
//
// UNE SEULE RÉSOLUTION POUR TOUT LE LOT : les clés sont dédupliquées avant l'appel. Résoudre
// clé par clé ferait une requête registre par arme.
//
// LA CLÉ EST SON PROPRE RÔLE QUAND LE REGISTRE N'EN DONNE PAS D'AUTRE (`grenade`, `melee`,
// `sidearm`, `equipment`, `vehicle`, `turret`, `environmental` : le registre y pose
// class == role). Rien de particulier n'est fait pour ces cas — le registre rend déjà le bon
// rôle ; une liste en dur ici serait une seconde source qui divergerait du TOML du titre.
func (s *CompareService) resolveWeaponRoles(
	ctx context.Context, mesures []analysis.MeasuredKill,
) map[string]string {
	cles := make([]string, 0, len(mesures))
	vues := make(map[string]bool, len(mesures))
	for _, m := range mesures {
		if m.WeaponKey != "" && !vues[m.WeaponKey] {
			vues[m.WeaponKey] = true
			cles = append(cles, m.WeaponKey)
		}
	}
	dims, err := s.weaponRange.ResolveWeaponDimensions(ctx, s.titleSlug, cles)
	if err != nil {
		logBestEffortErr(ctx, "compare: dimensions d armes non resolues", err,
			"title", s.titleSlug, "cles", len(cles))
		return nil
	}
	roles := make(map[string]string, len(dims))
	for cle, d := range dims {
		if d.Role != "" {
			roles[cle] = d.Role
		}
	}
	return roles
}

// logCompareWeaponFailure distingue l'ABSENCE LÉGITIME de l'ANOMALIE — parité
// `logWeaponRangeFailure`. Une capability manquante en Warn noierait les vrais bugs SQL sur
// tout titre sans décodeur de film, et le jour où un bug arrive personne ne le verrait.
func logCompareWeaponFailure(
	ctx context.Context, lecture, slug, xuid string, matchCount int, err error,
) {
	if errors.Is(err, games.ErrCapabilityNotSupported) {
		slog.DebugContext(ctx, "compare: profil d armes — capability absente",
			"lecture", lecture, "title", slug, "xuid", xuid)
		return
	}
	slog.WarnContext(ctx, "compare: profil d armes — lecture en echec (best-effort, bloc omis)",
		"lecture", lecture, "title", slug, "xuid", xuid, "match_count", matchCount, "err", err)
}
