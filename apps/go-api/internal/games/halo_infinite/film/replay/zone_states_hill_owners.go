package replay

// zone_states_hill_owners.go — LE PROPRIETAIRE D UNE COLLINE ET SES TRANCHES PUBLIEES : les
// periodes deja etablies (designateur ou rampes) y rencontrent le canal de propriete, et en
// sortent les `ZoneState` que l artefact porte.
//
// DEPLACEMENT PUR depuis `zone_states_hill.go` (lot 2.7 volet publication, 2026-09-16) : le
// fichier passait 500 lignes en portant DEUX sujets — QUELLE colline est active a quel moment
// (le designateur, les rampes, le vote, la fusion des periodes : restes la-bas) et QUI la tient
// (ici). Aucune ligne n a change : memes fonctions, meme ordre, memes commentaires de mesure.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
)

// hillStatesOf regroupe les periodes par zone et rend les intervalles ACTIFS, avec leur
// PROPRIETAIRE quand le canal le donne.
//
// # LE PROPRIETAIRE DE LA COLLINE, ET LE NIVEAU DE PREUVE QUI L'A AUTORISE (2026-08-26)
//
// Le canal est le tag 4 du slot VOISIN du designateur (`d.slot+1`) — celui-la meme que la
// condition d'election du designateur exige deja (`hillDesignatorMinOwnerSamples`). Trois
// campagnes de mesure l'ont confronte a trois oracles differents, et il faut lire leur verdict
// ensemble parce qu'il n'est PAS unanime :
//
//	D2      score de MODE : REFUTE COMME ORACLE — en KOTH il compte des collines GAGNEES
//	        (3-2, 4-2 : les scores de l'API), pas des secondes de garde, et deux films sur
//	        quatre n'en repliquent qu'UN camp. Aucun denominateur exploitable.
//	D2-bis  prises `th=10` : **88-89 % d'accord sur les deux films longs, contre un temoin de
//	        decalage a 56 %** — plus de trente points d'ecart, sur 64 et 92 confrontations.
//	        Sous le seuil de 90 % que le plan s'etait fixe. Sur les deux films COURTS (13 et 35
//	        emissions du canal) signal et temoins se confondent.
//	D2-ter  score PERSONNEL : REFUTE COMME ORACLE — delta dominant median de 150 points contre
//	        0-25 pour le camp domine, quand un frag vaut ~100 et un tic de colline quelques
//	        points. Il mesure qui a TUE, pas qui tient.
//
// **LE SEUIL DE 90 % N'A JAMAIS ETE ATTEINT NI REBAISSE.** Ce qui a change, c'est la DECISION :
// l'utilisateur a accepte ce niveau de preuve le 2026-08-26 pour ce calque, avec le precedent de
// la garde de l'ouvrier de rejeu (retenue a 88 %). Le canal n'a jamais ete refute — il a ete
// mesure sous le seuil, ce qui n'est pas la meme chose.
//
// **OU VIT L'ERREUR RESIDUELLE, ET C'EST CE QUI REND LE RISQUE ACCEPTABLE** : elle est
// concentree aux BASCULES. L'oracle `th=10` date ses prises au bloc de temps fort, pas a
// l'action — d'ou une fenetre d'appariement de +/- 20 s. Les 11 a 12 % de desaccord se lisent
// donc comme un flottement AUTOUR de l'instant du changement de main, pas comme une teinte
// fausse sur toute la duree d'une garde. Un lecteur qui trouverait une colline de la mauvaise
// couleur PENDANT une garde entiere tiendrait une regression, pas cette reserve.
//
// # CE QUE CE PRODUCTEUR NE PUBLIE PAS, ET POURQUOI
//
// `OwnerChecked` / `OwnerAgreed` restent a ZERO sur ce chemin. Ce sont les compteurs du CONTROLE
// INDEPENDANT de la methode par captures (la valeur du tag 4 confrontee a l'equipe du capteur) ;
// la colline n'a pas d'equivalent en production — son controle vit dans les instruments de D2-bis,
// sous garde `ZONE_FILM`. Publier des compteurs a zero comme s'ils avaient ete verifies serait
// pire que leur absence.
func hillStatesOf(periods []hillPeriod, owner []zoneSample, teams map[uint64]bool,
	cov *ZonesCoverage, fb *fallback.Compteur,
) []ZoneState {
	runs := hillOwnerRuns(owner, teams, cov, fb)
	byRef := map[int][]ZoneSpan{}
	for _, p := range periods {
		if p.t1 < p.t0 {
			continue
		}
		// LE SOMMET DE JAUGE D'UNE PERIODE DE COLLINE SE CONVERTIT PAR LA MEME FONCTION QUE LES
		// ZONES SIMULTANEES (lot C-ter, fusion des volets 1 et 3, 2026-08-19) : l'echelle DU JEU
		// (gaugeProgressOf), pas l'ancienne echelle par excursion de match (zoneGaugeScales,
		// supprimee par le volet 3 — cf. zone_states.go). ABSENT quand aucune rampe n'a contribue
		// (colline gardee mais jamais approchee : hasTop faux), jamais une invention a zero.
		var prog *float32
		if p.hasTop {
			v := gaugeProgressOf(p.top)
			prog = &v
		}
		byRef[p.ref] = append(byRef[p.ref], hillSpansOf(p, runs, prog)...)
	}
	refs := make([]int, 0, len(byRef))
	for ref := range byRef {
		refs = append(refs, ref)
	}
	sort.Ints(refs)
	out := make([]ZoneState, 0, len(refs))
	for _, ref := range refs {
		spans := byRef[ref]
		sort.SliceStable(spans, func(i, j int) bool { return spans[i].T0 < spans[j].T0 })
		out = append(out, ZoneState{ZoneRef: ref, Spans: spans})
	}
	return out
}

// hillOwnerRun est un intervalle de propriete CONSTANTE, bornes incluses. `team` vaut nil pour
// la valeur neutre du canal — « personne ne la tient » est une MESURE, pas une absence.
type hillOwnerRun struct {
	t0, t1 int
	team   *int
}

// hillOwnerRuns decoupe la serie du canal de propriete en intervalles de valeur constante.
//
// LA SEGMENTATION EST CELLE DE LA METHODE PAR CAPTURES (`mergeZoneRuns` puis « chaque groupe
// court jusqu'a la veille du suivant ») : une seconde ecriture de ce decoupage divergerait au
// premier correctif, et l'ecart serait invisible. Une valeur qui n'est ni le neutre ni un camp
// connu n'ouvre AUCUN intervalle et se compte (`UnknownOwner`) — publier un camp qu'aucun
// joueur n'occupe serait une invention, et la taire empecherait de la voir arriver.
//
// LA DERNIERE VALEUR COURT JUSQU'A LA FIN DE L'AXE, comme sur les zones simultanees : le canal
// est un ETAT, pas un evenement — il ne re-emet pas tant que rien ne change.
func hillOwnerRuns(owner []zoneSample, teams map[uint64]bool, cov *ZonesCoverage,
	fb *fallback.Compteur,
) []hillOwnerRun {
	groups := mergeZoneRuns(owner)
	out := make([]hillOwnerRun, 0, len(groups))
	for i, g := range groups {
		t1 := int(^uint(0) >> 1) // le dernier groupe court jusqu'a la fin : borne ouverte a droite
		if i+1 < len(groups) {
			t1 = groups[i+1].t - 1
		} else {
			// REPLI NOMME ET COMPTE (D14) : le canal est un ETAT, pas un evenement — rien n'ecrit
			// la FIN de la derniere propriete, et elle court jusqu'a la fin de l'axe.
			fb.Declenche(fallback.NomCollineDernierIntervalleOuvert)
		}
		if t1 < g.t {
			continue
		}
		team, known := zoneOwnerTeam(g.v, teams)
		if !known {
			cov.UnknownOwner++
			continue
		}
		out = append(out, hillOwnerRun{t0: g.t, t1: t1, team: team})
	}
	return out
}

// hillSpansOf decoupe UNE periode de colline par les changements de proprietaire qui la
// traversent. Rend un seul intervalle — sans proprietaire — quand le canal ne dit rien d'elle.
//
// # LE SOMMET DE JAUGE NE SE DUPLIQUE PAS SUR LES SOUS-INTERVALLES, ET C'EST DELIBERE
//
// `Progress` est le sommet atteint sur LA PERIODE. Quand la colline change de main en cours de
// periode, ce sommet n'est la propriete d'AUCUN des sous-intervalles : le recopier sur chacun
// affirmerait que chacun l'a atteint. Il n'est donc porte que par la periode qui sort d'un seul
// tenant. C'est une perte assumee, et elle ne coute rien a l'ecran : le client ne dessine plus
// `progress` depuis le schema 18 (le sommet statique se lisait comme une jauge en cours).
// # LA PERIODE EST COUVERTE ENTIEREMENT, MEME LA OU LE CANAL SE TAIT
//
// Un trou entre deux intervalles de propriete — ou avant la premiere emission du canal — ne
// FERME PAS la colline : elle est active, on ne sait simplement pas qui la tient. Ces morceaux
// sortent donc en intervalles SANS camp. Les omettre eteindrait la surbrillance au milieu d'une
// garde, ce qui se lirait comme « il ne se passe rien ici » — un contresens, et c'est le defaut
// que ce decoupage a eu avant d'etre corrige (le cas `606d9844` / `8076f97f`, ou le film ne
// replique qu'un camp et se tait sur tout le debut du match).
func hillSpansOf(p hillPeriod, runs []hillOwnerRun, prog *float32) []ZoneSpan {
	var out []ZoneSpan
	curseur := p.t0
	for _, r := range runs {
		t0, t1 := max(r.t0, p.t0), min(r.t1, p.t1)
		if t0 > t1 {
			continue
		}
		if t0 > curseur {
			out = append(out, ZoneSpan{T0: curseur, T1: t0 - 1, Active: true})
		}
		out = append(out, ZoneSpan{T0: t0, T1: t1, Owner: r.team, Active: true})
		curseur = t1 + 1
	}
	if curseur <= p.t1 {
		// Trou de fin — ou periode entiere quand le canal ne dit RIEN d'elle : la colline reste
		// ACTIVE et sans proprietaire, exactement ce que ce producteur publiait avant le
		// 2026-08-26. Une colline dont on ne lit pas le camp n'est pas une colline neutre.
		out = append(out, ZoneSpan{T0: curseur, T1: p.t1, Active: true})
	}
	if len(out) == 1 {
		out[0].Progress = prog
	}
	return out
}
