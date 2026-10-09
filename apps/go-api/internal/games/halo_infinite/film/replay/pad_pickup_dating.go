package replay

// pad_pickup_dating.go — DATER LES OCCUPATIONS DE SOCLE AVEC L'ÉVÉNEMENT NATIF.
//
// LE PROBLÈME QU'IL RÈGLE. `padPickups` publiait « ce socle s'est vidé quelque part entre
// `tLow` et `tHigh` », un intervalle de vingt secondes, et `xuid` valait `null` PARTOUT. Ce
// n'était pas un oubli : le contrat de `PadPickup.XUID` porte la mesure qui l'a refusé
// (88,1 % en suivant le slot de vie, 79,7 % en suivant le joueur, contre >= 90 % exigé) et il
// nomme ce qui le lèverait — « un oracle plus RAPPROCHÉ que 20 s ».
//
// L'ÉVÉNEMENT NATIF EST CET ORACLE, et il ne fait aucune inférence : il est daté à la
// milliseconde et il porte son ramasseur (`512 + référence` = le slot, exact sur 32/32 paires
// de vérité terrain, deux films). Quand un ramassage natif de la MÊME FAMILLE tombe dans la
// fenêtre d'une occupation, on publie l'instant exact et le joueur.
//
// CE QU'ON NE FAIT PAS, ET C'EST LA RÈGLE : ON N'EFFACE RIEN. L'intervalle `[tLow, tHigh]`
// reste publié dans tous les cas — daté ou non. Une occupation que l'événement natif ne
// couvre pas garde exactement ce qu'elle avait avant ce lot ; le canal AJOUTE, il ne remplace
// pas. Le rappel du canal natif est une borne inférieure (il ne voit que les événements en
// tête de liste) : substituer serait échanger une donnée sûre contre une donnée partielle.
//
// LECTURE, PUIS REPLI. La lecture date une occupation quand UN SEUL ramassage natif de la même
// famille tombe dans sa fenêtre et qu'aucune autre occupation ne le revendique. Quand plusieurs
// tombent dans la fenêtre, rien dans l'événement ne dit de quel socle il vient (l'instance de
// l'objet n'est pas dans l'événement — hypothèse mesurée et réfutée) : deux joueurs ont pu
// prendre la même arme ailleurs sur la carte, ou reprendre au sol l'arme qu'un preneur a lâchée
// en mourant. La règle de jeu tranche alors, en repli nommé et compté
// (`repli_prise_de_socle_premiere_du_cycle`, pad_pickup_dating_cycle.go) : UNE seule prise de
// socle par réapparition de l'arme, la première faite AU SOCLE dans le cycle. Sans elle, on
// s'abstient et on le compte : choisir au hasard nommerait un ramasseur faux.
//
// ## LES DEUX CÔTÉS N'ÉCRIVENT PAS LA FAMILLE PAREIL, ET LA JOINTURE DOIT LE SAVOIR
//
// Revue adversariale du 2026-08-31 : la première version de ce fichier comparait `Pickup.W`
// (`fmt.Sprintf("%08x", …)` — huit hexa MINUSCULES, sans préfixe) à `WeaponPad.Weapon`
// (`formatWeaponFamily` — `"0x"` + huit hexa MAJUSCULES). Les deux espaces ne coïncidaient
// JAMAIS : `hits` était toujours vide, aucun `padPickups[].t` n'était jamais écrit, aucun
// `xuid` posé — et `coverage.padDating` publiait `{dated: 0, uncovered: N}` qui SE LISAIT
// COMME UNE MESURE alors que c'était un défaut de format. La cuisson pilote n'a pas pu le
// révéler : son film ne porte aucun socle.
//
// LA NORMALISATION SE FAIT ICI, AU POINT DE JOINTURE, ET NULLE PART AILLEURS. Les formes
// publiées ne bougent PAS : `weaponChanges[].w` s'écrit ainsi depuis le schéma 25 et des
// clients peuvent déjà le lire ; `weaponPads[].weapon` depuis bien plus tôt. Changer l'une des
// deux pour faire plaisir à une jointure interne casserait un contrat public pour un confort
// privé.
//
// LES SOCLES DE POWER-UP NE SONT PAS JOIGNABLES, et ce n'est pas un échec de couverture : leur
// identité (`gwPadWeaponID` -> `Appar.Family`) est un NOM CANONIQUE, pas un identifiant de
// famille. Aucun ramassage natif ne peut donc s'y apparier, jamais. Ils sont comptés à part
// (`PowerupOccupations`) au lieu d'être noyés dans `Uncovered`, qui laisserait croire que le canal
// natif a cherché et n'a pas trouvé.

import (
	"cmp"
	"slices"
)

// PadDatingStats dit ce que la datation a pu faire, et ce qu'elle n'a pas pu.
type PadDatingStats struct {
	// Occupations est le nombre d'occupations achevées examinées.
	Occupations int `json:"occupations"`
	// Dated est le nombre d'occupations dont l'instant exact a été publié.
	Dated int `json:"dated"`
	// Named est le nombre dont le RAMASSEUR a pu être nommé (sous-ensemble de Dated : un
	// ramassage natif peut être daté sans que le pont slot -> joueur nomme sa vie).
	Named int `json:"named"`
	// Ambiguous compte les fenêtres où PLUSIEURS ramassages natifs de la même famille
	// tombaient, ET celles dont l'unique ramassage était aussi le candidat unique d'une autre
	// occupation, quand le repli de la première prise du cycle ne les a pas tranchées : on
	// s'abstient plutôt que de nommer un ramasseur au hasard, ou de créditer deux prises pour
	// un seul ramassage.
	Ambiguous int `json:"ambiguous"`
	// Uncovered compte les fenêtres qu'aucun ramassage natif ne couvre — elles gardent leur
	// intervalle, intact.
	Uncovered int `json:"uncovered"`
	// PowerupOccupations compte les occupations de socle de POWER-UP, structurellement non
	// joignables : l'identité d'un tel socle est un NOM CANONIQUE, pas une famille d'arme.
	// Elles sont sorties d'`Uncovered` À DESSEIN — les y laisser ferait lire « le canal natif
	// a cherché et n'a pas trouvé » là où il n'y avait rien à chercher.
	//
	// LE NOM N'EST PAS `powerupPads`, ET C'EST DÉLIBÉRÉ (correctif de revue, ronde 2) :
	// `coverage.groundWeapons.powerupPads` existe déjà et compte des SOCLES PUBLIÉS. Deux clés
	// homonymes à dénominateurs différents dans le même document se liraient l'une pour
	// l'autre. Ici on compte des OCCUPATIONS écartées de la jointure — le nom le dit.
	PowerupOccupations int `json:"powerupOccupations"`
	// FirstOfCycle est le nombre d'occupations datées par le repli de la première prise du
	// cycle (sous-ensemble de Dated) : leur fenêtre portait plusieurs ramassages natifs, ou un
	// ramassage qu'une autre occupation revendiquait, et la règle de jeu a désigné le leur.
	FirstOfCycle int `json:"firstOfCycle"`
}

// PadWeaponFamilyKey rend la clé de comparaison d'une famille d'arme, quelle que soit la
// convention d'écriture de la source, et dit si la valeur EST une famille.
//
// Les deux écritures rencontrées : `fmt.Sprintf("%08x", fam)` (canaux `pickups` et
// `weaponChanges`) et `formatWeaponFamily(fam)` = `"0x" + huit majuscules` (`loadouts`,
// `weaponPads`). Le second retour est FAUX pour tout ce qui n'est pas huit chiffres
// hexadécimaux — c'est ainsi qu'un socle de power-up (nom canonique) se distingue d'un socle
// d'arme, sans avoir à connaître la liste des noms.
//
// EXPORTÉE LE 2026-09-04 (résumé d'usage de session) : c'est LA frontière socle d'ARME /
// socle de BONUS du dépôt, celle que `candidatsDeDatation` emploie pour sortir les bonus de la
// jointure (`PadDatingStats.PowerupOccupations`) et que le client rejoue côté web
// (`padControlLogic.ts`, note de pied `gapFmt.powerup`). Le résumé sidecar en avait besoin
// depuis `replaybuild` : en RÉ-ÉCRIRE le test hexadécimal là-bas aurait fait une troisième
// écriture d'une règle qui doit rester unique — un socle compté du mauvais côté fausse à la
// fois « prises de socle » et « bonus ramassés ».
func PadWeaponFamilyKey(s string) (string, bool) {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	if len(s) != 8 {
		return "", false
	}
	buf := make([]byte, 8)
	for i := range 8 {
		c := s[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
			buf[i] = c
		case c >= 'A' && c <= 'F':
			buf[i] = c + ('a' - 'A')
		default:
			return "", false
		}
	}
	return string(buf), true
}

// datationDesSocles porte la datation des occupations de socle entre ses deux phases : la LECTURE
// ([lireLesOccupations]) puis le REPLI de la première prise du cycle
// ([datationDesSocles.trancherParLeCycle]). Entre les deux, l'assemblage relève les socles hors de
// l'emprise jouée (ground_weapon_pads_releve.go) : le relevé ne lit que les occupations datées par
// la lecture, et le repli, qui juge si un ramasseur se tient AU SOCLE, lit la position relevée.
type datationDesSocles struct {
	pads     []WeaponPad
	picks    []PadPickup
	pickups  []Pickup
	fenetres [][]int
	retenu   []int
	st       PadDatingStats
}

// lireLesOccupations pose l'instant exact et le ramasseur sur les occupations que l'événement
// natif date sans ambiguïté. Modifie `picks` en place.
//
// `pads` sert à retrouver la FAMILLE d'arme du socle : `PadPickup.Pad` est un index dans
// `pads`, et c'est la famille qui apparie une occupation à un ramassage natif.
//
// UN RAMASSAGE NATIF DATE AU PLUS UNE OCCUPATION (lot J8.2 du plan de suite d audit, constat
// RB2-5, 2026-09-27). Deux socles de la même arme dont les fenêtres se chevauchent peuvent
// voir le MÊME ramassage comme leur candidat unique ; rien dans l'événement ne dit de quel
// socle il vient, et le poser sur les deux créditait le joueur de deux prises pour une seule
// (`BuildUsageSummary`). Un ramassage revendiqué par plusieurs lectures n'est donc CONSOMMÉ par
// aucune d'elles : ces occupations restent au repli, qui ne retient jamais un ramassage qu'une
// lecture a daté.
func lireLesOccupations(pads []WeaponPad, picks []PadPickup, pickups []Pickup) *datationDesSocles {
	d := &datationDesSocles{pads: pads, picks: picks, pickups: pickups, st: PadDatingStats{Occupations: len(picks)}}
	d.fenetres = candidatsDeDatation(pads, picks, pickups, &d.st)
	d.retenu = lectureDesFenetres(d.fenetres)
	for i, c := range d.retenu {
		if c >= 0 {
			d.poser(i, c)
		}
	}
	return d
}

// trancherParLeCycle applique le repli de la première prise du cycle aux occupations que la
// lecture a laissées ambiguës, pose leurs dates et rend les compteurs de la datation. `localiser`
// rend la position du ramasseur d'un ramassage natif ; nil, le repli ne localise rien et ne date
// donc rien.
func (d *datationDesSocles) trancherParLeCycle(localiser localiserRamasseur) PadDatingStats {
	lues := slices.Clone(d.retenu)
	d.st.FirstOfCycle = premieresPrisesDuCycle(entreesDuCycle{
		pads: d.pads, picks: d.picks, pickups: d.pickups, localiser: localiser,
	}, d.fenetres, d.retenu)
	for i, c := range d.retenu {
		switch {
		case len(d.fenetres[i]) == 0:
			// déjà classée par candidatsDeDatation
		case c < 0:
			d.st.Ambiguous++
		case lues[i] < 0:
			d.poser(i, c)
		}
	}
	return d.st
}

// poser date l'occupation `i` par le ramassage natif `c`, et la nomme quand le pont slot ->
// joueur nomme sa vie.
func (d *datationDesSocles) poser(i, c int) {
	k := &d.picks[i]
	t := d.pickups[c].T
	k.T = &t
	d.st.Dated++
	if d.pickups[c].XUID != "" {
		x := d.pickups[c].XUID
		k.XUID = &x
		d.st.Named++
	}
}

// candidatNonDate : la valeur retenue pour une occupation qui ne sera pas datée.
const candidatNonDate = -1

// lectureDesFenetres rend, pour chaque occupation, l'index dans `pickups` du ramassage natif que
// la LECTURE retient — l'unique de sa fenêtre, qu'aucune autre fenêtre ne voit comme son unique —,
// [candidatNonDate] sinon.
func lectureDesFenetres(fenetres [][]int) []int {
	revendications := map[int]int{}
	for _, f := range fenetres {
		if len(f) == 1 {
			revendications[f[0]]++
		}
	}
	out := make([]int, len(fenetres))
	for i, f := range fenetres {
		out[i] = candidatNonDate
		if len(f) == 1 && revendications[f[0]] == 1 {
			out[i] = f[0]
		}
	}
	return out
}

// candidatsDeDatation rend, pour chaque occupation, les INDEX dans `pickups` des ramassages
// natifs d'arme de la même famille tombés dans sa fenêtre, par instant croissant ; il compte
// dans `st` les occupations qui n'en ont aucun (hors bornes et non couvertes dans `Uncovered`,
// power-ups dans `PowerupOccupations`) — leur liste est vide.
func candidatsDeDatation(pads []WeaponPad, picks []PadPickup, pickups []Pickup, st *PadDatingStats) [][]int {
	// PAS DE RETOUR ANTICIPÉ QUAND LE CANAL NATIF EST VIDE, et c'est un correctif de revue
	// (ronde 2) : la première version versait alors TOUTES les occupations dans `Uncovered`,
	// power-ups compris — c'est-à-dire exactement la lecture mensongère (« le canal a cherché
	// et n'a pas trouvé ») que ce fichier prétend éliminer. La boucle ci-dessous classe
	// TOUJOURS, y compris avec zéro ramassage : un socle de power-up reste hors jointure, un
	// socle d'arme reste non couvert.
	//
	// Index par famille NORMALISÉE : une occupation ne s'apparie qu'à un ramassage de LA MÊME
	// arme, et les deux côtés ne l'écrivent pas pareil (cf. l'en-tête).
	byFamily := map[string][]int{}
	for j, p := range pickups {
		if p.Kind != PickupWeapon {
			continue // un socle d'arme ne rend pas de l'équipement
		}
		key, ok := PadWeaponFamilyKey(p.W)
		if !ok {
			continue
		}
		byFamily[key] = append(byFamily[key], j)
	}
	out := make([][]int, len(picks))
	for i := range picks {
		k := &picks[i]
		if k.Pad < 0 || k.Pad >= len(pads) {
			st.Uncovered++
			continue
		}
		key, ok := PadWeaponFamilyKey(pads[k.Pad].Weapon)
		if !ok {
			// Socle de POWER-UP (nom canonique) : rien à chercher, et on ne fait pas passer
			// ça pour une recherche infructueuse.
			st.PowerupOccupations++
			continue
		}
		for _, j := range byFamily[key] {
			if pickups[j].T >= k.TLow && pickups[j].T <= k.THigh {
				out[i] = append(out[i], j)
			}
		}
		if len(out[i]) == 0 {
			st.Uncovered++
			continue
		}
		// L'ORDRE DES CANDIDATS EST CELUI DU TEMPS, ex aequo par index : le repli prend le
		// premier, et l'ordre du canal publié n'est pas un contrat.
		slices.SortStableFunc(out[i], func(a, b int) int { return cmp.Compare(pickups[a].T, pickups[b].T) })
	}
	return out
}
