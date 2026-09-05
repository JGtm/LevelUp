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
//   - grappleLines / equipmentEpisodes / equipmentPlacements : par SLOT, via
//     l'agrégat « dernier gagnant » (indexBySlot côté web) : ce qui appartient au
//     joueur redescend sur chacune de ses vies, effondré en une valeur par slot.
//     C'est un agrégat de MATCH, pas une lecture par image — en multi-manche, un
//     slot réattribué crédite son dernier propriétaire (dette connue et assumée
//     par le web, reproduite pour rester comparable) ;
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

import "sort"

// UsageSummaryRev — révision des RÈGLES DE PROJECTION de ce fichier (attribution par
// slot, frontière socle d'arme/bonus, familles déployables...), PAS de l'artefact :
// l'artefact a SchemaVersion, et les deux voyagent ensemble dans les tables
// (`summary_rev` + `artifact_schema`). C'est la clé de reprise du backfill : un match
// dont la passe courante porte (UsageSummaryRev, SchemaVersion) est à jour ; changer
// une règle d'attribution ici DOIT incrémenter cette révision, sinon le backfill
// sautera des matchs à re-résumer.
const UsageSummaryRev = "us1"

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
	// vaut DroppedObjects, l'invariant est testé. LA VENTILATION N'EST PAS
	// PERSISTÉE : la DDL ne porte que dropped_objects (§3 du handoff), le champ
	// n'existe que pour prouver l'invariant de classification à la projection.
	DroppedObjects  int
	DroppedByFamily map[string]int
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
	out := UsageSummary{
		Match: UsageMatchSummary{
			SchemaVersion:   doc.SchemaVersion,
			FrameIntervalMS: doc.FrameIntervalMS,
			FrameCount:      doc.FrameCount,
			DurationMS:      int64(doc.DurationMS),
		},
	}
	players := newUsageTallies()
	slotOwner := usageSlotOwners(doc)
	filmIndexOwner := usageFilmIndexOwners(doc, slotOwner)

	for i := range doc.GrappleLines {
		if t := players.of(slotOwner[doc.GrappleLines[i].Slot]); t != nil {
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

	out.Players = players.rows()
	return out
}

// usageSlotOwners — l'agrégat « dernier gagnant » slot -> propriétaire, copie
// conforme de la construction web (buildPlayers + indexBySlot de rosterLogic.ts) :
// l'ordre des joueurs est celui du roster du film puis des pistes, les vies de
// chacun sont triées par frame de début, et le dernier propriétaire d'un slot
// contesté gagne. La clé rendue est le xuid, ou "" pour une vie de BOT ou anonyme
// (un bot n'a pas de xuid : ses gestes n'entrent dans aucune ligne persistée, mais
// il OCCUPE ses slots — les attribuer au précédent occupant humain serait faux).
func usageSlotOwners(doc *ReplayDocument) map[uint32]string {
	type joueur struct {
		xuid  string // "" pour un bot : identité non persistable
		lives []*Track
	}
	index := map[string]int{}
	var ordre []*joueur
	ajouter := func(cle, xuid string) *joueur {
		if i, ok := index[cle]; ok {
			return ordre[i]
		}
		index[cle] = len(ordre)
		j := &joueur{xuid: xuid}
		ordre = append(ordre, j)
		return j
	}
	for i := range doc.Roster {
		e := &doc.Roster[i]
		switch {
		case e.XUID != "":
			ajouter(e.XUID, e.XUID)
		case e.Bot && e.Name != "":
			ajouter("bot:"+e.Name, "")
		}
	}
	for i := range doc.Tracks {
		tr := &doc.Tracks[i]
		var j *joueur
		switch {
		case tr.XUID != "":
			j = ajouter(tr.XUID, tr.XUID)
		case tr.Bot != "":
			j = ajouter("bot:"+tr.Bot, "")
		default:
			continue // vie anonyme : ni ligne, ni occupation de slot
		}
		j.lives = append(j.lives, tr)
	}
	owners := make(map[uint32]string)
	for _, j := range ordre {
		sort.SliceStable(j.lives, func(a, b int) bool {
			return j.lives[a].StartFrame < j.lives[b].StartFrame
		})
		for _, tr := range j.lives {
			owners[tr.Slot] = j.xuid
		}
	}
	return owners
}

// usageFilmIndexOwners — index de film -> xuid, via le roster. Même garde que le
// web : seul un joueur dont AU MOINS UNE VIE est publiée reçoit des lancers (une
// entrée de roster sans piste n'a été mesurée sur aucun canal).
func usageFilmIndexOwners(doc *ReplayDocument, slotOwner map[uint32]string) map[int]string {
	avecVie := make(map[string]bool, len(slotOwner))
	for _, xuid := range slotOwner {
		if xuid != "" {
			avecVie[xuid] = true
		}
	}
	out := make(map[int]string, len(doc.Roster))
	for i := range doc.Roster {
		e := &doc.Roster[i]
		if e.XUID != "" && avecVie[e.XUID] {
			out[e.FilmIndex] = e.XUID
		}
	}
	return out
}

// tallyUsageEpisodes cumule les épisodes camo/surbouclier par propriétaire de slot.
// La durée est convertie en millisecondes RÉELLES (frameToMs côté web) ; un
// artefact sans échelle de temps rend 0 ms — le compte d'épisodes reste.
func tallyUsageEpisodes(doc *ReplayDocument, players *usageTallies, slotOwner map[uint32]string) {
	for i := range doc.EquipmentEpisodes {
		e := &doc.EquipmentEpisodes[i]
		t := players.of(slotOwner[e.Slot])
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
func tallyUsagePlacements(doc *ReplayDocument, players *usageTallies, slotOwner map[uint32]string) {
	for i := range doc.EquipmentPlacements {
		p := &doc.EquipmentPlacements[i]
		if p.Owner < 0 {
			continue
		}
		t := players.of(slotOwner[uint32(p.Owner)])
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
