// Package fragdist héberge le builder PUR de la « Répartition des frags » v2
// (sunburst hiérarchique classe→rôle). Extrait de package service (P6) pour être
// réutilisable SANS duplication par toutes les couches qui en ont besoin —
// Synthesis / Match view / Timeseries / Sessions (package service) ET la page
// Escouade (package service/teammates, qui ne peut pas importer son parent
// service). Cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md §2 + §6 (D-P6-1).
//
// Le builder reste PUR (aucune IO, aucun log — le câblage service loggue les
// compteurs via logFragDistribution). Les types (FragDistribution/FragClassEntry/
// FragRoleEntry/FragKillTypeCounts) vivent dans internal/domain (leaf partagé).
package fragdist

import (
	"sort"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// canonicalFragClassOrder fixe l'ordre déterministe des classes du sunburst v2
// (§4 du plan). Toute sortie de Build suit cet ordre.
// Véhicule et tourelle se placent APRÈS les classes API et AVANT le résidu : ce sont
// des outils de destruction secondaires, et « Non attribué » reste en dernier.
// Équipement et environnement se placent APRÈS les engins et AVANT le résidu : ce sont
// les outils de destruction les plus périphériques, et « Non attribué » reste dernier.
var canonicalFragClassOrder = []string{
	domain.FragClassShoulder, domain.FragClassSidearm, domain.FragClassHeavy,
	domain.FragClassMelee, domain.FragClassGrenade, domain.FragClassSpartanAbility,
	domain.FragClassVehicle, domain.FragClassTurret,
	domain.FragClassEquipment, domain.FragClassEnvironmental,
	domain.FragClassUnattributed,
}

// gunFragClasses = classes d'ARME à main dont la ventilation vient du REGISTRE (estimé),
// avec un niveau 2 par RÔLE de combat. Les classes melee/grenade/spartan sont servies par
// les totaux API ; les classes d'ENGIN (vehicle/turret) viennent aussi du registre mais se
// ventilent par engin (domain.IsPerWeaponFragClass) ; environnement/autre et les ids hors
// registre retombent dans « Non attribué ».
var gunFragClasses = map[string]bool{
	domain.FragClassShoulder: true,
	domain.FragClassSidearm:  true,
	domain.FragClassHeavy:    true,
}

// isRegistryFragClass : classe dont les KILLS proviennent des rows du registre (par
// opposition aux compteurs API). Recouvre les deux familles de niveau 2 — par rôle
// (gun) et par engin (véhicule/tourelle).
func isRegistryFragClass(class string) bool {
	return gunFragClasses[class] || domain.IsPerWeaponFragClass(class)
}

// Build assemble la répartition hiérarchique des frags (sunburst v2).
// Builder PUR (aucune IO, aucun log — le câblage service loggue les compteurs).
//
// Provenance (anti-double-source, §2) : classes gun shoulder/sidearm/heavy + rôles
// d'arme = registre (rows, Authoritative=false) ; classes d'ENGIN véhicule/tourelle +
// leur niveau 2 par engin = registre également (V73-3.2) ; classes melee/grenade/
// spartan_ability + total = compteurs kill-type API canoniques (counts,
// Authoritative=true) ; unattributed = counts.Total − Σ classes (résidu, ajouté si
// > 0). hasMechanics gate spartan_ability et le niveau 2 de Mêlée (invariant d — cap
// off ⇒ pas de spartan + Mêlée feuille).
//
// Les classes d'engin sont DATA-DRIVEN, jamais gated par titre : un titre dont le
// registre ne déclare aucune arme de classe vehicle/turret ne produit aucune row de
// cette classe, donc aucun arc (cas Halo Infinite au 2026-08-02).
//
// Niveau 2 de la classe Grenade (V72-15.2) = TYPE (frag/plasma/dynamo/splinter) dérivé de
// la FAMILLE des rows class=grenade, réconcilié au total API (grenadeRoles). Anti-double-
// comptage H5 (V72-15.3) : les MechanicKills (mêlée/assassinat attribués à l'arme tenue,
// kill_kind <> 'weapon') sont retranchés des classes gun — servis par les compteurs natifs.
//
// counts est un struct NEUTRE (domain.FragKillTypeCounts) : le builder est ainsi
// partagé tel quel par Synthesis (agrégat SynthesisDetailedStats), Match view
// (ligne scoreboard native du viewer) ET Escouade (agrégat par gamertag) — jamais
// dupliqué (règle ≤2 copies).
func Build(
	rows []port.WeaponKillRow,
	sources []port.KillSourceClassRow,
	counts domain.FragKillTypeCounts,
	hasMechanics bool,
) domain.FragDistribution {
	byClass := make(map[string]domain.FragClassEntry, len(canonicalFragClassOrder))
	for _, e := range buildRegistryFragClasses(rows) {
		byClass[e.Class] = e
	}
	for _, e := range buildAPIFragClasses(rows, counts, hasMechanics) {
		byClass[e.Class] = e
	}
	// sources vide (titre sans décodeur de film, match jamais décodé, appelant qui ne les
	// charge pas encore) => AUCUNE entrée ajoutée, donc sortie byte-identique à ce qu'elle
	// était avant cette provenance. C'est le test de non-régression du lot.
	for _, e := range buildKillSourceFragClasses(sources) {
		byClass[e.Class] = e
	}
	// Non attribué = résidu calculé (invariant a : Σ classes == total ; invariant c :
	// clamp >= 0). Un dépassement (Σ attribué > total) est une anomalie de données —
	// non ajouté ici, signalé par le câblage service (jamais avalé).
	sum := 0
	for _, e := range byClass {
		sum += e.Kills
	}
	if unattr := counts.Total - sum; unattr > 0 {
		byClass[domain.FragClassUnattributed] = domain.FragClassEntry{
			Class: domain.FragClassUnattributed, Kills: unattr, Authoritative: false,
		}
	}
	classes := make([]domain.FragClassEntry, 0, len(byClass))
	for _, c := range canonicalFragClassOrder {
		if e, ok := byClass[c]; ok {
			classes = append(classes, e)
		}
	}
	return domain.FragDistribution{TotalKills: counts.Total, Classes: classes}
}

// buildKillSourceFragClasses agrège les classes servies par la SOURCE DE DÉGÂT du film
// (3ᵉ provenance, 2026-08-29) : Équipement et Environnement, niveau 2 PAR OBJET.
//
// POURQUOI UN CHEMIN À PART, et pas un élargissement de `isRegistryFragClass`. Élargir
// l'ensemble des classes servies par le registre ferait aussi remonter les lignes
// `h5_environmental` de Halo 5, qui ont un id numérique et vivent dans `weapon_kills` :
// le sunburst de Halo 5 changerait, hors du périmètre de ce lot. Les deux provenances
// restent donc séparées, comme le sont déjà registre et compteurs API.
//
// Niveau 2 par OBJET (weapon_key + libellé du registre) et jamais par rôle : sur ces
// classes, `role` vaut la classe elle-même — un niveau 2 par rôle serait un arc unique
// sans information. « Bobine à plasma » est une information, « environnement » n'en est
// pas une (même exigence que les engins, V73-3.2).
//
// Aucune collision possible avec les classes du registre : `equipment` et `environmental`
// ne sont pas dans `isRegistryFragClass`, donc `buildRegistryFragClasses` ne les produit
// jamais.
func buildKillSourceFragClasses(sources []port.KillSourceClassRow) []domain.FragClassEntry {
	type acc struct {
		kills    int
		byKey    map[string]int
		labels   map[string]string
		labelsEN map[string]string
	}
	agg := map[string]*acc{}
	for _, s := range sources {
		if s.Class == "" || s.WeaponKey == "" || s.Kills <= 0 {
			continue
		}
		a := agg[s.Class]
		if a == nil {
			a = &acc{byKey: map[string]int{}, labels: map[string]string{}, labelsEN: map[string]string{}}
			agg[s.Class] = a
		}
		a.kills += s.Kills
		a.byKey[s.WeaponKey] += s.Kills
		if s.Label != "" {
			a.labels[s.WeaponKey] = s.Label
		}
		if s.LabelEN != "" {
			a.labelsEN[s.WeaponKey] = s.LabelEN
		}
	}
	out := make([]domain.FragClassEntry, 0, len(agg))
	for class, a := range agg {
		out = append(out, domain.FragClassEntry{
			Class: class, Kills: a.kills, Authoritative: false,
			Roles: perWeaponRoles(a.byKey, a.labels, a.labelsEN),
		})
	}
	return out
}

// registryAcc accumule une classe servie par le registre : total + ventilation de
// niveau 2 keyée (rôle de combat, ou weapon_key d'engin), les libellés d'engin, et le
// volume NON keyable (qui interdit une ventilation partielle trompeuse).
type registryAcc struct {
	kills    int
	byKey    map[string]int
	labels   map[string]string // key → libellé registre FR-first (classes par engin uniquement)
	labelsEN map[string]string // key → libellé registre EN-first (idem, V2.1 2026-08-29)
	unkeyed  int
}

// buildRegistryFragClasses agrège les classes servies par le REGISTRE (rows) : les
// classes gun (shoulder/sidearm/heavy, niveau 2 par RÔLE) et les classes d'ENGIN
// (véhicule/tourelle, niveau 2 par weapon_key — V73-3.2). Exclut les sentinels
// grenade/melee et les rows sans class résolue (dégradation propre).
//
// Distinction de niveau 2 : sur une classe d'engin, le registre porte
// class == role == family == « vehicle » — ventiler par rôle donnerait un arc unique
// sans information. La clé est donc le weapon_key, et le libellé vient du registre
// (weapon_names.toml du titre), jamais d'un nom en dur.
//
// Anti-double-comptage V72-15.3 : on retranche MechanicKills. Sur H5 une mêlée/un
// assassinat est attribué à l'ARME TENUE dans weapon_kills (kill_kind <> 'weapon') → sans
// ce retrait le kill compte à la fois dans la classe ET dans le compteur natif Mêlée/
// Capacités spartanes (Σ classes > total). Sur Infinite MechanicKills == 0 → inchangé.
func buildRegistryFragClasses(rows []port.WeaponKillRow) []domain.FragClassEntry {
	agg := make(map[string]*registryAcc, len(gunFragClasses)+2)
	for _, r := range rows {
		if r.IsGrenadeMelee || r.Class == "" || !isRegistryFragClass(r.Class) {
			continue
		}
		perWeapon := domain.IsPerWeaponFragClass(r.Class)
		// Classe gun sans rôle résolu : row ignorée (comportement historique conservé).
		if !perWeapon && r.Role == "" {
			continue
		}
		weaponKills := r.Kills - r.MechanicKills // kills d'ARME seuls (hors mécaniques natives)
		if weaponKills <= 0 {
			continue
		}
		a := agg[r.Class]
		if a == nil {
			a = &registryAcc{byKey: make(map[string]int), labels: make(map[string]string), labelsEN: make(map[string]string)}
			agg[r.Class] = a
		}
		a.kills += weaponKills
		key := r.Role
		if perWeapon {
			key = r.WeaponKey
		}
		if key == "" {
			// Engin hors registre : le kill compte dans la classe, mais la ventilation
			// devient partielle → la classe retombera en feuille (cf. registryRoles).
			a.unkeyed += weaponKills
			continue
		}
		a.byKey[key] += weaponKills
		if perWeapon && r.Label != "" {
			a.labels[key] = r.Label
		}
		if perWeapon && r.LabelEN != "" {
			a.labelsEN[key] = r.LabelEN
		}
	}
	out := make([]domain.FragClassEntry, 0, len(agg))
	for class, a := range agg {
		if a.kills <= 0 {
			continue
		}
		out = append(out, domain.FragClassEntry{
			Class: class, Kills: a.kills, Authoritative: false,
			Roles: registryRoles(class, a),
		})
	}
	return out
}

// registryRoles choisit la ventilation de niveau 2 d'une classe registre : par ENGIN
// (véhicule/tourelle) ou par RÔLE (classes gun). Sur une classe d'engin, un reliquat
// non keyable (engin absent du registre) force la FEUILLE : mieux vaut aucun niveau 2
// qu'une ventilation qui ne somme pas au total de la classe (invariant b).
func registryRoles(class string, a *registryAcc) []domain.FragRoleEntry {
	if domain.IsPerWeaponFragClass(class) {
		if a.unkeyed > 0 {
			return nil
		}
		return perWeaponRoles(a.byKey, a.labels, a.labelsEN)
	}
	return rolesFromMap(a.byKey, class)
}

// perWeaponRoles trie les engins (kills desc, tie-break sur la clé → ordre stable) et
// porte leur libellé registre, FR (labels) ET EN (labelsEN, V2.1 2026-08-29) — le web
// choisit entre les deux selon la locale (cf. fragRoleLabel.ts). Aucun repli en feuille à
// un seul engin : « Warthog » est une information, « véhicule » n'en est pas une
// (exigence V73-3.2).
func perWeaponRoles(byKey map[string]int, labels map[string]string, labelsEN map[string]string) []domain.FragRoleEntry {
	roles := make([]domain.FragRoleEntry, 0, len(byKey))
	for key, kills := range byKey {
		if kills <= 0 {
			continue
		}
		roles = append(roles, domain.FragRoleEntry{Role: key, Kills: kills, Label: labels[key], LabelEN: labelsEN[key]})
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Kills != roles[j].Kills {
			return roles[i].Kills > roles[j].Kills
		}
		return roles[i].Role < roles[j].Role
	})
	return roles
}

// rolesFromMap trie les rôles (kills desc, tie-break alpha → ordre stable) et
// replie en FEUILLE une classe dont l'unique rôle porte son propre nom (ex. Poing/
// sidearm — D4 : pas de niveau 2 significatif).
func rolesFromMap(byRole map[string]int, class string) []domain.FragRoleEntry {
	roles := make([]domain.FragRoleEntry, 0, len(byRole))
	for role, kills := range byRole {
		if kills <= 0 {
			continue
		}
		roles = append(roles, domain.FragRoleEntry{Role: role, Kills: kills})
	}
	if len(roles) == 1 && roles[0].Role == class {
		return nil
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Kills != roles[j].Kills {
			return roles[i].Kills > roles[j].Kills
		}
		return roles[i].Role < roles[j].Role
	})
	return roles
}

// buildAPIFragClasses construit les classes servies par les compteurs kill-type API
// canoniques (Authoritative=true) : Mêlée, Grenade, et — si hasMechanics — Capacités
// spartanes. rows alimente UNIQUEMENT le niveau 2 de la classe Grenade (TYPE de grenade,
// V72-15.2) : le total de la classe reste le compteur API autoritatif.
func buildAPIFragClasses(rows []port.WeaponKillRow, counts domain.FragKillTypeCounts, hasMechanics bool) []domain.FragClassEntry {
	out := make([]domain.FragClassEntry, 0, 3)
	// Total de la classe Mêlée = mêlées directes + assassinats. Sur H5 l'API expose
	// deux compteurs DISJOINTS (melee_kills n'inclut PAS les assassinats) : on additionne
	// donc pour obtenir le total de la classe. Sur Infinite, Assassination=0 → total =
	// Melee (inchangé). Cf. verdict empirique dans meleeRoles ci-dessous.
	if meleeTotal := counts.Melee + counts.Assassination; meleeTotal > 0 {
		out = append(out, domain.FragClassEntry{
			Class: domain.FragClassMelee, Kills: meleeTotal, Authoritative: true,
			Roles: meleeRoles(counts.Melee, counts.Assassination, hasMechanics),
		})
	}
	if gk := counts.Grenade; gk > 0 {
		out = append(out, domain.FragClassEntry{
			Class: domain.FragClassGrenade, Kills: gk, Authoritative: true,
			Roles: grenadeRoles(rows, gk),
		})
	}
	if hasMechanics {
		if sk := counts.GroundPound + counts.ShoulderBash; sk > 0 {
			out = append(out, domain.FragClassEntry{
				Class: domain.FragClassSpartanAbility, Kills: sk, Authoritative: true,
				Roles: spartanRoles(counts.GroundPound, counts.ShoulderBash),
			})
		}
	}
	return out
}

// meleeRoles construit le niveau 2 de la classe Mêlée. FEUILLE sur Infinite (pas de
// mécanique native → invariant d). Sur H5 (hasMechanics) : Assassinat = melee_kills API
// (compteur total_assassinations) ; Corps-à-corps direct = melee_kills (compteur
// total_melee_kills), SANS soustraction ni clamp.
//
// VERDICT (probe empirique, 2026-07-19) : assassinat ⊄ mêlée — les deux compteurs API
// sont DISJOINTS (melee_kills n'inclut PAS les assassinats). Preuve : match
// 3066a511-ebd0-428f-9555-50422caebaba / xuid 2535421586125737 → melee_kills=6,
// assassination_kills=4, per-kill 6 mêlées + 4 assassinats ; et match 9bb09267… a
// melee_kills=2 < assassination_kills=4 (impossible sous inclusion). 1212/1213 couples
// disjoints. Donc Σ rôles = melee + assass = total de la classe (invariant b tenu).
func meleeRoles(melee, assassinations int, hasMechanics bool) []domain.FragRoleEntry {
	if !hasMechanics {
		return nil
	}
	roles := make([]domain.FragRoleEntry, 0, 2)
	if assassinations > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleAssassination, Kills: assassinations})
	}
	if melee > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleDirectMelee, Kills: melee})
	}
	return roles
}

// spartanRoles construit le niveau 2 des Capacités spartanes (ordre sémantique fixe :
// Frappe au sol puis Charge d'épaule). Σ rôles == kills de la classe (invariant b).
func spartanRoles(groundPound, shoulderBash int) []domain.FragRoleEntry {
	roles := make([]domain.FragRoleEntry, 0, 2)
	if groundPound > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleGroundPound, Kills: groundPound})
	}
	if shoulderBash > 0 {
		roles = append(roles, domain.FragRoleEntry{Role: domain.FragRoleShoulderBash, Kills: shoulderBash})
	}
	return roles
}

// grenadeRoles construit le niveau 2 de la classe Grenade par TYPE (V72-15.2), dérivé de
// la FAMILLE registre des rows class=grenade (Infinite frag/plasma/dynamo ; H5 frag/plasma/
// splinter ; famille inconnue → « Autre grenade »). Le total de la classe reste le compteur
// API autoritatif (classTotal = counts.Grenade) ; les types en sont une VENTILATION,
// réconciliée pour tenir l'invariant b (Σ rôles == kills de la classe) :
//   - Σ types < classTotal : résidu « Autre grenade » = classTotal − Σ types ;
//   - Σ types > classTotal (registre sur-attribue vs compteur natif) : FEUILLE (nil) — pas
//     de ventilation trompeuse ; l'anomalie reste visible via le total central.
//
// FEUILLE (nil) si aucune row grenade typée : dégradation DATA-DRIVEN (ni Infinite ni H5 en
// dur — un titre/scope sans grenade dans weapon_kills garde une classe Grenade feuille).
func grenadeRoles(rows []port.WeaponKillRow, classTotal int) []domain.FragRoleEntry {
	if classTotal <= 0 {
		return nil
	}
	byType := make(map[string]int, len(domain.GrenadeTypeRoleOrder))
	typedSum := 0
	for _, r := range rows {
		if r.IsGrenadeMelee || r.Class != domain.FragClassGrenade || r.Kills <= 0 {
			continue
		}
		role, _ := domain.GrenadeTypeRoleForFamily(r.Family)
		byType[role] += r.Kills
		typedSum += r.Kills
	}
	if typedSum == 0 || typedSum > classTotal {
		return nil // pas de type distinguable, ou sur-attribution → classe feuille
	}
	if typedSum < classTotal {
		byType[domain.FragRoleGrenadeOther] += classTotal - typedSum // résidu non typé
	}
	return sortedGrenadeRoles(byType)
}

// sortedGrenadeRoles trie les types de grenade (kills desc, tie-break ordre canonique
// GrenadeTypeRoleOrder → « Autre grenade » en dernier à kills égaux) et écarte les zéros.
func sortedGrenadeRoles(byType map[string]int) []domain.FragRoleEntry {
	rank := make(map[string]int, len(domain.GrenadeTypeRoleOrder))
	for i, role := range domain.GrenadeTypeRoleOrder {
		rank[role] = i
	}
	roles := make([]domain.FragRoleEntry, 0, len(byType))
	for role, kills := range byType {
		if kills > 0 {
			roles = append(roles, domain.FragRoleEntry{Role: role, Kills: kills})
		}
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Kills != roles[j].Kills {
			return roles[i].Kills > roles[j].Kills
		}
		return rank[roles[i].Role] < rank[roles[j].Role]
	})
	return roles
}
