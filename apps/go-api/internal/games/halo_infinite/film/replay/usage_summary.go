package replay

// usage_summary.go — LE RÉSUMÉ D'USAGE D'UN MATCH, dérivé de l'artefact de rejeu
// DÉJÀ CONSTRUIT (jamais d'un re-décodage de film) et destiné à la persistance
// (`shared.match_usage_players` + `shared.match_usage_films`, chantier session du
// 2026-09-04 — décision utilisateur : « il faut les sauvegarder en BDD lors du sync »).
//
// # POURQUOI CE FICHIER EXISTE
//
// Les grandeurs d'équipement et de socle ne vivent QUE dans les artefacts
// (1,8 Mo pièce) : la page Sessions devrait sinon ouvrir ~9 artefacts (~16 Mo de
// JSON) par requête. Ce module PROJETTE un artefact en quelques centaines d'octets
// par joueur — une fonction PURE (document -> lignes), zéro DB, zéro HTTP, comme
// tout ce paquet.
//
// # LES RÈGLES D'ATTRIBUTION SONT CELLES DU CLIENT WEB, REPRODUITES À L'IDENTIQUE
//
// La vue match affiche déjà ces comptes (equipmentUsageLogic.ts, padControlLogic.ts).
// Deux écritures d'une même règle divergeraient : chaque canal reprend donc ici la
// jointure exacte du web —
//
//   - grappleLines / equipmentEpisodes : par la VIE qui couvre l'instant du geste
//     (`usageOwners.at`) ; equipmentPlacements par la vie qui le couvre OU qui vient
//     de s'achever (`atOrJustBefore`, jumeau d'`ownerAtFrameOrLast` du web), parce
//     qu'un objet lâché à la mort porte `t0 = finVie + 1`. L'agrégat « dernier
//     gagnant » (indexBySlot de rosterLogic.ts) créditait tout au SECOND occupant
//     d'un slot recyclé, gestes de la vie du premier compris — corrigé ici le
//     2026-09-06 (constat C5 de la revue REG-R1, complété par N-3 en seconde ronde) :
//     l'instant est publié à chacun des trois sites, il n'y avait rien à deviner.
//     LE WEB RESTE LE MIROIR pour la formule, mais deux de ses appelants
//     (equipmentUsageLogic.ts, equipmentKillBadges.ts) sont encore sur `indexBySlot`
//     alors que `ownerAtFrame`/`ownerAtFrameOrLast` existent : l'écart est inscrit au
//     registre des reports, et c'est la règle d'ici qui est juste ;
//   - grenades : par INDEX DE FILM (`grenades[].i`), jamais par slot — mesuré côté
//     web : joindre par slot perd la quasi-totalité des lancers ;
//   - padPickups : par le XUID PUBLIÉ (`padPickups[].xuid`, événement natif daté) —
//     jamais deviné ; une occupation sans ramasseur nommé n'est comptée pour
//     PERSONNE, elle reste au niveau du match (pad_unnamed).
//
// # LE PIÈGE CENTRAL : SOCLE D'ARME CONTRE SOCLE DE BONUS
//
// `weaponPads[].weapon` mélange familles d'arme (hexadécimal) et bonus (nom
// canonique `powerup_*`). LA frontière est [PadWeaponFamilyKey] — exportée pour ce
// chantier précisément, jamais réécrite ici : `pad_pickups` ne compte QUE les
// socles d'ARME ; les occupations de socle de BONUS partent dans
// `powerup_pad_pickups`, anonymes et au grain du match (aucun ramassage natif ne
// peut nommer leur ramasseur — cf. pad_pickup_dating.go).
//
// # CE QUE LE RÉSUMÉ NE PORTE PAS
//
// Les effectifs d'équipe (dérivables de match_participants — ne pas dupliquer,
// §3 du handoff), et aucune grandeur des vies anonymes ou des bots : une ligne est
// keyée par xuid, un geste sans xuid attributable n'entre dans aucune ligne (les
// totaux de match, eux, restent complets : pad_occupancies compte tout).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
)

// UsageSummaryRev — révision des RÈGLES DE PROJECTION de ce fichier (attribution par
// slot, frontière socle d'arme/bonus, familles déployables...), PAS de l'artefact :
// l'artefact a SchemaVersion, et les deux voyagent ensemble dans les tables
// (`summary_rev` + `artifact_schema`). C'est la clé de reprise du backfill : un match
// dont la passe courante porte (UsageSummaryRev, SchemaVersion) est à jour ; changer
// une règle d'attribution ici DOIT incrémenter cette révision, sinon le backfill
// sautera des matchs à re-résumer. CE QUE CHAQUE RÉVISION A CHANGÉ, ET CE QU'ELLE
// EXIGE COMME RECUISSON : usage_summary_chronicle.go, sa seule et unique chronique.
const UsageSummaryRev = "us6"

// UsagePlayerSummary — les usages d'UN joueur sur UN match, prêt à persister.
type UsagePlayerSummary struct {
	// XUID en décimal, même forme que la base et que Track.XUID.
	XUID string
	// GrapplePulls : tractions de grappin (`grappleLines`), la seule ACTIVATION de
	// capacité que le film mesure et attribue.
	GrapplePulls int
	// Camo* / Overshield* : les épisodes d'état ACTIF (`equipmentEpisodes`), leur
	// durée cumulée en millisecondes réelles (frames x frameIntervalMs — 0 si
	// l'artefact n'a pas d'échelle de temps) et les frags du porteur pendant ses
	// épisodes (somme des `k` ; zéro NE distingue PAS « rien tué » de « jointure
	// non tentée » — c'est coverage.equipment.killsRead, côté artefact, qui le dit).
	CamoEpisodes       int
	CamoMS             int64
	CamoKills          int
	OvershieldEpisodes int
	OvershieldMS       int64
	OvershieldKills    int
	// DeployedByFamily : les DÉPLOIEMENTS par famille du document (mur compté sur
	// ses panneaux uniquement — cf. usage_summary_families.go). Nil quand aucun.
	DeployedByFamily map[string]int
	// DroppedObjects : les objets LÂCHÉS à la mort, HORS grenades (décision
	// utilisateur 2026-09-04). DroppedByFamily les ventile — la somme des valeurs
	// vaut DroppedObjects, l'invariant est testé. LA VENTILATION EST PERSISTÉE
	// DEPUIS `us4` (étape E3, 2026-09-09) : sans elle la page Sessions ne pouvait
	// servir qu'un total de lâchers, jamais l'issue d'une famille.
	DroppedObjects  int
	DroppedByFamily map[string]int
	// TakenByFamily / SpentByFamily : les RAMASSAGES et les CONSOMMATIONS du canal
	// `equipmentChanges`, ventilés par famille du bilan (vocabulaire des POSES —
	// cf. usage_summary_outcomes.go). Une prise dont le rang n'a pas de libellé
	// dans ce film n'entre dans AUCUNE famille : elle se compte à la couverture
	// (UsageMatchSummary.EquipmentChanges), jamais dans une grandeur.
	TakenByFamily map[string]int
	SpentByFamily map[string]int
	// KeptByFamily : GARDÉ SANS L'UTILISER — la troisième issue (décision P1),
	// DÉRIVÉE et jamais lue d'un canal : `max(0, taken - utilisé - lâché)`, la
	// formule exacte de la vue match (equipmentUsageLogic.ts, étape E2). Le
	// clamp absorbe le fait qu'une POSE EST UNE CHARGE, PAS UN OBJET.
	KeptByFamily map[string]int
	// GrenadesThrown : les lancers de grenade (`grenades[]`). Produit mais non
	// affiché par la page Sessions (§3 du handoff) — il coûte trois lignes.
	GrenadesThrown int
	// PadPickups : prises de socle d'ARME nommées par l'événement natif.
	// PadPickupsByWeapon les ventile par la CLÉ NORMALISÉE de la famille d'arme
	// ([PadWeaponFamilyKey] : huit hexa minuscules, sans « 0x ») — jamais la forme
	// verbatim du document : deux artefacts peuvent écrire la même famille sous deux
	// conventions, et des clés verbatim couperaient une famille en deux à l'agrégat.
	PadPickups         int
	PadPickupsByWeapon map[string]int
}

// UsageWeaponPad — un socle d'ARME présent sur la carte, avec ses occupations
// achevées et celles dont le ramasseur est nommé. Les socles de BONUS n'y sont
// pas : ils vivent dans PowerupPadPickups. `Weapon` porte la clé NORMALISÉE de la
// famille ([PadWeaponFamilyKey]), la même que PadPickupsByWeapon — jointure directe.
// Les étiquettes JSON servent la sérialisation en base (`weapon_pads_json`).
type UsageWeaponPad struct {
	Weapon      string `json:"weapon"`
	Occupations int    `json:"occupations"`
	Named       int    `json:"named"`
}

// UsageMatchSummary — les grandeurs de niveau MATCH du résumé.
type UsageMatchSummary struct {
	// SchemaVersion de l'ARTEFACT résumé : c'est la clé de reprise du backfill
	// (un artefact re-cuit à un schéma plus récent doit se voir « à re-résumer »).
	SchemaVersion int
	// L'axe de temps de l'artefact, verbatim : sans lui les cadences « par dix
	// minutes » de l'agrégat de session n'auraient pas de dénominateur mesuré.
	FrameIntervalMS int
	FrameCount      int
	DurationMS      int64
	// PadOccupancies : TOUTES les occupations achevées (`padPickups`), socles
	// d'arme ET de bonus. PadNamed/PadUnnamed ne ventilent que les socles d'ARME
	// (nommée = xuid publié) ; les occupations de socle de BONUS sont dans
	// PowerupPadPickups, par famille. L'invariant testé :
	// PadNamed + PadUnnamed + somme(PowerupPadPickups) + horsBornes == PadOccupancies.
	PadOccupancies int
	PadNamed       int
	PadUnnamed     int
	// PowerupPadPickups : occupations de socle de BONUS par famille (la clé est
	// `weaponPads[].weapon`, ex. "powerup_camo"). ANONYMES et au grain du match —
	// aucun ramassage natif ne peut les nommer (cf. pad_pickup_dating.go).
	PowerupPadPickups map[string]int
	// WeaponPads : les socles d'ARME présents, un par socle du document, dans
	// l'ordre du document (stable côté build).
	WeaponPads []UsageWeaponPad
	// EquipmentChanges : ce que le canal des ramassages n'a PAS su rattacher.
	// NON PERSISTÉ et hors de toute métrique (décision utilisateur 2026-09-09) —
	// c'est un témoin d'outillage, journalisé par les deux producteurs de passes
	// (post-sync `sync/replayartifacts/usage.go`, backfill `cmd/levelup`).
	EquipmentChanges UsageChangeCoverage
	// Fallbacks : LES REPLIS QUE CETTE PROJECTION A DÉCLENCHÉS, par nom du registre
	// (`film/replay/fallback`), triés, les zéros absents. NON PERSISTÉ, hors de toute
	// métrique : même statut de témoin qu'[UsageMatchSummary.EquipmentChanges].
	//
	// IL EXISTE PARCE QUE CE COMPTE NE PEUT PAS VOYAGER DANS `coverage.fallbacks[]`
	// (câblage du 2026-09-16, revue de jalon M1). [BuildUsageSummary] lit un document
	// DÉJÀ CUIT : ses replis se déclenchent APRÈS la cuisson, donc hors du compteur qui
	// alimente la couverture de l'artefact. Sans ce canal, les trois replis de
	// l'attribution par slot (`repli_geste_dernier_occupant_du_match`,
	// `repli_geste_premiere_vie_du_slot`, `repli_garde_equipement_negatif_a_zero`)
	// resteraient à compte INCONNU — et D14 (d) fait supprimer un repli dont le compte
	// est à zéro : confondre « jamais déclenché » et « jamais instrumenté » ferait
	// supprimer un repli actif.
	//
	// QUI LE LIT, ET C'EST LA RÉPONSE COMPLÈTE (revue de jalon M1, ronde 2, constat F2 —
	// ce champ a été écrit un jour sans lecteur, et trois cibles de retrait nommaient
	// alors un instrument, le `replay-corpus-gate`, qui NE PEUT PAS le produire
	// puisqu'il lit les artefacts) :
	//
	//	LE JOURNAL DES PASSES   les deux producteurs — post-sync
	//	                        (`sync/replayartifacts/usage.go`,
	//	                        `journaliserReplisUsage`) et backfill CLI
	//	                        (`cmd/levelup/cmd_backfill_usage_summary.go`,
	//	                        `journaliserReplisUsageCorpus`) — écrivent une ligne
	//	                        `slog.Info` par match déclenchant ET une par passe, celle-ci
	//	                        TOUJOURS, « aucun » compris. C'est l'instrument du parc.
	//	LA MESURE SUR 8 BUILDS  `usage_summary_replis_test.go`, sur les fixtures
	//	                        d'assemblage : le chiffre reproductible, hors production.
	//
	// Il reste NON PERSISTÉ : rien de tout cela n'entre en base ni dans l'artefact.
	Fallbacks []fallback.Declenchement
}

// UsageSummary — la projection complète d'un artefact.
type UsageSummary struct {
	Match   UsageMatchSummary
	Players []UsagePlayerSummary
}

// BuildUsageSummary projette un document de rejeu en résumé d'usage. Fonction
// PURE : elle ne lit que le document, ne modifie rien, et rend des lignes triées
// par xuid (déterminisme des passes et des tests).
func BuildUsageSummary(doc *ReplayDocument) UsageSummary {
	// LE COMPTEUR EST PAR PROJECTION, jamais de paquet (critères S1 et S2 du plan) : il naît
	// ici, il voyage dans `out.Match.Fallbacks`, et deux projections concurrentes ne se
	// mélangent pas.
	fb := fallback.NouveauCompteur()
	out := UsageSummary{
		Match: UsageMatchSummary{
			SchemaVersion:   doc.SchemaVersion,
			FrameIntervalMS: doc.FrameIntervalMS,
			FrameCount:      doc.FrameCount,
			DurationMS:      int64(doc.DurationMS),
		},
	}
	players := newUsageTallies()
	slotOwner := usageSlotOwners(doc, fb)
	filmIndexOwner := usageFilmIndexOwners(doc, slotOwner)

	for i := range doc.GrappleLines {
		gl := &doc.GrappleLines[i]
		if t := players.of(slotOwner.at(gl.Slot, gl.T0)); t != nil {
			t.GrapplePulls++
		}
	}
	tallyUsageEpisodes(doc, players, slotOwner)
	tallyUsagePlacements(doc, players, slotOwner)
	for i := range doc.Grenades {
		if t := players.of(filmIndexOwner[doc.Grenades[i].Idx]); t != nil {
			t.GrenadesThrown++
		}
	}
	tallyUsagePads(doc, players, &out.Match)
	// LES ISSUES EN DERNIER, ET DANS CET ORDRE : `deriveUsageKept` soustrait des
	// grandeurs que les tallies ci-dessus viennent de poser (poses déployées,
	// épisodes, lâchers) ET les CONSOMMATIONS que la ligne juste au-dessus ventile
	// (côté « utilisé » des déployables sans pièce engendrée, `us6`). L'inverser
	// rendrait un gardé égal aux prises.
	out.Match.EquipmentChanges = tallyUsageEquipmentChanges(doc, players, slotOwner)
	deriveUsageKept(players, fb)

	out.Players = players.rows()
	out.Match.Fallbacks = fb.Rapport()
	return out
}

// tallyUsageEpisodes cumule les épisodes camo/surbouclier par propriétaire de slot.
// La durée est convertie en millisecondes RÉELLES (frameToMs côté web) ; un
// artefact sans échelle de temps rend 0 ms — le compte d'épisodes reste.
func tallyUsageEpisodes(doc *ReplayDocument, players *usageTallies, slotOwner usageOwners) {
	for i := range doc.EquipmentEpisodes {
		e := &doc.EquipmentEpisodes[i]
		t := players.of(slotOwner.at(e.Slot, e.T0))
		if t == nil {
			continue
		}
		ms := int64(0)
		if doc.FrameIntervalMS > 0 && e.T1 > e.T0 {
			ms = int64(e.T1-e.T0) * int64(doc.FrameIntervalMS)
		}
		switch e.Fam {
		case EquipFamilyCamo:
			t.CamoEpisodes++
			t.CamoMS += ms
			t.CamoKills += e.K
		case EquipFamilyOvershield:
			t.OvershieldEpisodes++
			t.OvershieldMS += ms
			t.OvershieldKills += e.K
		}
		// Une famille hors des deux mesurées n'a pas de colonne : elle n'entre pas
		// (même règle que EPISODE_FAMILIES côté web).
	}
}

// tallyUsagePlacements ventile les poses : déploiements par famille, lâchers hors
// grenades. `owner` -1 = aucun bipède contemporain assez proche : la pose est
// réelle, son auteur ne l'est pas — elle n'entre dans aucune ligne.
func tallyUsagePlacements(doc *ReplayDocument, players *usageTallies, slotOwner usageOwners) {
	for i := range doc.EquipmentPlacements {
		p := &doc.EquipmentPlacements[i]
		if p.Owner < 0 {
			continue
		}
		t := players.of(slotOwner.atOrJustBefore(uint32(p.Owner), p.T0))
		if t == nil {
			continue
		}
		switch {
		case usageDeployedCounts(p):
			if t.DeployedByFamily == nil {
				t.DeployedByFamily = map[string]int{}
			}
			t.DeployedByFamily[p.Family]++
		case p.Origin == OriginDropped && usageFamilyIsDroppable(p.Family):
			t.DroppedObjects++
			if t.DroppedByFamily == nil {
				t.DroppedByFamily = map[string]int{}
			}
			t.DroppedByFamily[p.Family]++
		}
	}
}

// tallyUsagePads compte les occupations de socle, des deux côtés de LA frontière
// ([PadWeaponFamilyKey]) : socles d'ARME (nommées par joueur + anonymes au match),
// socles de BONUS (anonymes par famille). Un index de socle hors bornes ne compte
// pour aucun socle voisin — il reste dans PadOccupancies seul, comme côté web.
func tallyUsagePads(doc *ReplayDocument, players *usageTallies, m *UsageMatchSummary) {
	pads := make([]UsageWeaponPad, 0, len(doc.WeaponPads))
	padIdx := make(map[int]int, len(doc.WeaponPads)) // index document -> index pads
	for i := range doc.WeaponPads {
		// La clé NORMALISÉE remplace la forme verbatim du document dès l'entrée :
		// tout ce qui sort d'ici (liste des socles, ventilation par joueur) parle la
		// même langue que le reste du dépôt (PadWeaponFamilyKey).
		key, ok := PadWeaponFamilyKey(doc.WeaponPads[i].Weapon)
		if !ok {
			continue // socle de BONUS : il n'entre pas dans la liste des socles d'arme
		}
		padIdx[i] = len(pads)
		pads = append(pads, UsageWeaponPad{Weapon: key})
	}

	m.PadOccupancies = len(doc.PadPickups)
	for i := range doc.PadPickups {
		pick := &doc.PadPickups[i]
		if pick.Pad < 0 || pick.Pad >= len(doc.WeaponPads) {
			continue // hors bornes : compté dans PadOccupancies, nulle part ailleurs
		}
		j, estArme := padIdx[pick.Pad]
		if !estArme {
			// Socle de BONUS : occupation anonyme, par famille — JAMAIS dans
			// pad_pickups, même si un xuid était publié (il ne l'est jamais :
			// aucun ramassage natif ne s'apparie à un nom canonique). La clé reste
			// le nom canonique VERBATIM (`powerup_camo`, ...) : il est déjà normal.
			if m.PowerupPadPickups == nil {
				m.PowerupPadPickups = map[string]int{}
			}
			m.PowerupPadPickups[doc.WeaponPads[pick.Pad].Weapon]++
			continue
		}
		pads[j].Occupations++
		if pick.XUID == nil || *pick.XUID == "" {
			m.PadUnnamed++
			continue
		}
		pads[j].Named++
		m.PadNamed++
		t := players.of(*pick.XUID)
		t.PadPickups++
		if t.PadPickupsByWeapon == nil {
			t.PadPickupsByWeapon = map[string]int{}
		}
		t.PadPickupsByWeapon[pads[j].Weapon]++
	}
	m.WeaponPads = pads
}

// usageTallies — les lignes par joueur, créées à la demande. La clé vide (vie
// anonyme, bot) ne crée jamais de ligne : of("") rend nil et l'appelant passe.
type usageTallies struct {
	byXUID map[string]*UsagePlayerSummary
}

func newUsageTallies() *usageTallies {
	return &usageTallies{byXUID: map[string]*UsagePlayerSummary{}}
}

func (u *usageTallies) of(xuid string) *UsagePlayerSummary {
	if xuid == "" {
		return nil
	}
	t := u.byXUID[xuid]
	if t == nil {
		t = &UsagePlayerSummary{XUID: xuid}
		u.byXUID[xuid] = t
	}
	return t
}

// rows rend les lignes triées par xuid — l'ordre d'itération d'une map n'est pas
// une sortie.
func (u *usageTallies) rows() []UsagePlayerSummary {
	out := make([]UsagePlayerSummary, 0, len(u.byXUID))
	for _, t := range u.byXUID {
		out = append(out, *t)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].XUID < out[b].XUID })
	return out
}
