package replay

import "sort"

// closures.go — REFERMER LE PONT SANS JAMAIS DEVINER.
//
// # LE DÉFAUT QUE CE FICHIER RÉPARE
//
// `lives.go` nomme chaque vie par LA MORT QUI LA TERMINE. Une vie que nulle mort ne termine —
// celle qui court de la dernière mort d'un joueur jusqu'au coup de sifflet — reste donc anonyme,
// et tout ce qu'elle porte est rejeté « slot introuvable ». Mesuré le 2026-08-08 sur sept films :
// **63 à 92 % des tirs perdus tombent dans une vie non nommée**, et le dernier décile du film
// perd de 40 à 74 % de ses tirs — dans TOUS les modes, pas seulement en CTF.
//
// # POURQUOI CE N'EST PAS LE VOTE RETIRÉ LE 2026-07-28
//
// La distinction est de nature, pas de degré, et elle est la seule chose qui autorise ce fichier
// à exister après le retrait du repli voté :
//
//	le vote        plusieurs candidats, on garde le mieux placé        -> un CHOIX
//	la fermeture   un seul candidat POSSIBLE, les autres sont exclus   -> une DÉDUCTION
//
// Dès que deux candidats subsistent, RIEN n'est attribué. La règle de l'utilisateur reste
// intacte : « je préfère rien afficher que quelque chose de complètement faux ».
//
// # LES DEUX FERMETURES
//
//	A — par le corps disponible   un joueur tire alors qu'aucune de ses vies nommées ne le
//	                              couvre : son corps est l'une des vies anonymes dont l'INTERVALLE
//	                              couvre cet instant. S'il n'y en a QU'UNE, c'est elle.
//	B — par la réapparition       une vie commence UNE RÉAPPARITION après la mort qui l'a causée,
//	                              et le fil des morts NOMME cette victime. Si UNE SEULE mort tombe
//	                              dans la fenêtre, ET qu'aucune autre vie ne revendique cette
//	                              mort, la vie est la sienne.
//
// # LES GARDE-FOUS, ET ILS MORDENT
//
//	contestation   l'unicité manque, dans l'un des quatre sens -> aucune attribution
//	               deux corps possibles pour un même tir (A) · deux joueurs revendiquent le même
//	               corps (A) · deux corps revendiquent le même joueur (A — un joueur n'a qu'un
//	               DERNIER corps) · deux corps revendiquent la même mort (B — une mort ne rend
//	               qu'un corps, symétrique du refus de la vie qui voit deux morts)
//	corroboration  l'unicité du candidat ne prouve RIEN de son appartenance (A) : le corps déduit
//	               n'est attribué que si le tireur peut le PROLONGER — il a un corps connu, et
//	               tous ses corps connus s'achèvent avant que le candidat ne commence -> sinon
//	               rejetée. Ce contrôle absorbe et durcit l'ancien « recouvrement ».
//	recouvrement   un joueur n'a qu'un corps : si le corps déduit chevauche dans le temps une vie
//	               DÉJÀ nommée du même joueur, l'attribution est impossible -> rejetée (B)
//
// Mesuré sur les sept films, APRÈS LA RONDE DE CORROBORATION DU 2026-08-11 : **31 vies attribuées,
// 99 refusées** — 87 contestées, 12 rejetées. Ce compte de refus NE SE COMPARE PAS aux 17 du
// 2026-08-08 : la contestation couvre depuis le 2026-08-09 les vies écartées faute d'unicité
// d'INTERVALLE, qui n'étaient alors comptées nulle part. Un contrôle qui ne rejette jamais rien ne
// prouve rien ; celui-ci refuse plus de trois fois pour une attribution. C'est le pendant du
// critère « huit entités distinctes » qui a réfuté la piste i19 le 2026-07-28.
//
// # RÉSULTAT MESURÉ (lecture seule -> fermetures, sept films, 2026-08-11)
//
//	0edb8512 Team Slayer  93,4 -> 96,4      db7b8c3c CTF   88,5 -> 94,5
//	9aeca4b3 Team Slayer  89,0 -> 91,3      64e8adfa CTF   80,3 -> 87,4
//	000d5950 Fiesta       91,5 -> 93,1      829abef9 CTF   79,7 -> 88,7
//	01e1f945 KOTH         86,4 -> 89,2
//
// LE PLANCHER DU CORPUS EST PASSÉ SOUS LE CRITÈRE DU GARDE LOCAL, ET CE N'EST PAS UN DÉFAUT DE CE
// FICHIER : 87,4 % sur `64e8adfa` contre 88 % exigés. La corroboration a retiré 3 fermetures A sur
// ce film (98 tirs), toutes attribuées à des tireurs que rien ne situait — exactement le « corps
// d'autrui » que le chantier interdit. Le seuil n'a PAS été touché et la règle n'a PAS été
// relâchée pour faire remonter le chiffre : l'arbitrage entre la justesse et le seuil
// d'activation appartient à l'utilisateur (cf. `api/handlers/replay_local_gate.go`).
//
// LES CHIFFRES BAISSENT, ET C'EST LE SENS ATTENDU d'un correctif de justesse : quatre films sont
// inchangés, trois reculent (90,8 -> 87,4 · 89,7 -> 89,2 · 91,6 -> 91,3). Un correctif de ce genre
// qui ferait MONTER le compte serait le signal qu'il a élargi au lieu de resserrer.
//
// Détail, méthode et échec de réglage : `.ai/V7.5/RECHERCHE_CTF_TIRS_PERDUS.md` §7.5, §7.5bis
// et §7.5ter.

// closureReport compte ce que les fermetures ont attribué ET refusé. Publier l'un sans l'autre
// laisserait croire que la déduction ne se trompe jamais.
type closureReport struct {
	byShot, byRespawn  int
	contested, refused int
	// closedLife retient, pour chaque slot attribué, l'INDICE de la vie que la fermeture a
	// désignée — jamais le slot seul. -1 = deux vies distinctes désignées, on ne tranche pas.
	//
	// POURQUOI LA VIE ET PAS LE SLOT (2026-09-06, instruction des régressions du balayage du
	// parc). Les deux fermetures raisonnent sur UNE VIE : A désigne « l'unique corps libre qui
	// couvre l'instant du tir », B « la vie qui commence une réapparition après cette mort ». Ne
	// rendre que le slot jetait cette désignation, et le nommage des vies devait la RE-DEVINER
	// (« l'unique vie anonyme du slot ») : sur un slot qui en porte plusieurs il s'abstenait, et
	// la piste restait anonyme alors que le document publiait déjà les TIRS de ce même slot.
	// Mesure sur `145908d1` au schéma 40 : 53 slots au pont, 51 pistes nommées, 29 tirs posés
	// sur deux pistes sans nom (slots 562 et 570).
	closedLife map[uint32]int
}

// noteLife enregistre la vie qu'une fermeture vient de désigner pour ce slot.
//
// LA MAP S'ALLOUE ICI, pas au constructeur : les fermetures s'appellent aussi depuis les tests
// avec un `closureReport` nu, et un rapport partiellement construit ne doit pas paniquer.
//
// DEUX VIES DIFFÉRENTES POUR UN MÊME SLOT NE SE TRANCHENT PAS. Le cas existe (deux corps libres
// du même slot, chacun seul candidat à un instant distinct) ; le pont n'en retient qu'un
// propriétaire, mais rien ne dit LAQUELLE des deux vies est la sienne. -1 vaut abstention.
func (r *closureReport) noteLife(slot uint32, life int) {
	if r.closedLife == nil {
		r.closedLife = map[uint32]int{}
	}
	if prev, seen := r.closedLife[slot]; seen && prev != life {
		r.closedLife[slot] = -1
		return
	}
	r.closedLife[slot] = life
}

// closeBridge applique les deux fermetures, dans l'ordre mesuré (A puis B), et rend le pont
// augmenté avec son compte rendu. Le pont d'entrée n'est jamais modifié.
func closeBridge(tracks map[uint32]slotTrack, owner map[uint32]int, lives []lifeSpan,
	deaths []Death, off int64, byXUID map[uint64]int, fire []FireEventRef) (map[uint32]int, closureReport) {
	var rep closureReport
	out := copyOwners(owner)
	closeByAvailableBody(tracks, out, lives, fire, &rep)
	closeByRespawn(tracks, out, lives, deaths, off, byXUID, &rep)
	return out, rep
}

// FireEventRef est ce que les fermetures ont besoin de savoir d'un tir : QUI et QUAND. Le type
// est réduit à dessein — la fermeture ne doit pas pouvoir dépendre de l'arme ni de la visée, qui
// n'ont aucun pouvoir de désignation.
type FireEventRef struct {
	FilmIndex   int
	TimestampUS uint64
}

// closeByAvailableBody — FERMETURE A. Un tir dont l'auteur n'a aucune vie nommée à cet instant
// désigne l'unique vie anonyme QUI COUVRE cet instant, s'il n'y en a qu'une.
//
// L'UNICITÉ SE JUGE SUR L'INTERVALLE DE LA VIE, PAS SUR SES ÉCHANTILLONS, et ce point a été payé.
// Le premier jet exigeait du candidat une position répliquée à moins de `shotPosToleranceUS`
// (120 ms) de l'instant du tir. Or une vie survit à un trou de réplication de `lifeGapUS` (5 s) :
// deux vies anonymes pouvaient couvrir l'instant, celle du VRAI tireur être dans son trou, et
// l'autre — la seule échantillonnée — passer pour l'unique candidate. Le corps d'autrui était
// alors attribué sans qu'aucun garde-fou ne s'en aperçoive, puisque du point de vue du code il
// n'y avait qu'un candidat.
//
// L'échantillon ne sert donc qu'au RATTACHEMENT (`slotFor`, qui refuse toujours de poser un
// événement sans position proche), jamais à trancher l'unicité.
func closeByAvailableBody(tracks map[uint32]slotTrack, owner map[uint32]int,
	lives []lifeSpan, fire []FireEventRef, rep *closureReport) {
	free := freeLives(owner, lives)
	if len(free) == 0 {
		return
	}
	c := claimsFromShots(tracks, owner, lives, free, fire)
	attributeClaimedBodies(tracks, owner, c, rep)
	// Une vie contestée à un instant peut être seule candidate à un autre : seules celles qui
	// n'ont JAMAIS été revendiquées sont des déductions abandonnées. Le comptage ne dépend pas
	// de l'ordre d'itération de la map, donc rien à trier ici.
	for slot := range c.blocked {
		if _, claimed := c.byBody[slot]; !claimed {
			rep.contested++
		}
	}
}

// shotClaims est le dépouillement des tirs orphelins de la fermeture A.
//
// `byBody` et `blocked` portent la DÉCISION (qui revendique quel corps, et quels corps sont
// écartés faute d'unicité) ; `life` porte la DÉSIGNATION — l'indice de la vie que les tirs ont
// pointée pour ce slot, -1 quand ils en pointent plusieurs. Les deux se séparent parce que la
// décision se prend PAR SLOT (un slot n'a qu'un propriétaire) tandis que la désignation est PAR
// VIE : c'est cette seconde information que l'ancien code jetait.
type shotClaims struct {
	byBody  map[uint32]map[int]int
	blocked map[uint32]bool
	life    map[uint32]int
}

// claimsFromShots dépouille les tirs ORPHELINS — ceux dont l'auteur n'a aucun corps rattachable
// à cet instant — et rend qui revendique quel corps libre. Les corps écartés faute d'unicité y
// figurent aussi : sans eux, l'abstention la plus fréquente de la fermeture A ne serait comptée
// nulle part.
func claimsFromShots(tracks map[uint32]slotTrack, owner map[uint32]int, lives []lifeSpan,
	free []int, fire []FireEventRef) shotClaims {
	c := shotClaims{byBody: map[uint32]map[int]int{}, blocked: map[uint32]bool{}, life: map[uint32]int{}}
	for _, e := range fire {
		if _, r := slotFor(tracks, owner, e.FilmIndex, e.TimestampUS); r == reasonAttached {
			continue
		}
		cand := livesCoveringAt(lives, free, e.TimestampUS)
		if len(cand) != 1 { // plusieurs corps possibles : on ne tranche pas
			for _, i := range cand {
				c.blocked[lives[i].slot] = true
			}
			continue
		}
		slot := lives[cand[0]].slot
		if c.byBody[slot] == nil {
			c.byBody[slot] = map[int]int{}
			c.life[slot] = cand[0]
		} else if c.life[slot] != cand[0] {
			c.life[slot] = -1 // deux vies du même slot revendiquées : la vie ne se tranche pas
		}
		c.byBody[slot][e.FilmIndex]++
	}
	return c
}

// attributeClaimedBodies pose au pont les corps qu'UN SEUL tireur revendique, que ce tireur ne
// revendique QU'UNE FOIS, et qu'il peut PROLONGER. Chacun des trois refus est compté.
func attributeClaimedBodies(tracks map[uint32]slotTrack, owner map[uint32]int,
	c shotClaims, rep *closureReport) {
	twice := shootersClaimingTwoBodies(c.byBody)
	for _, slot := range sortedClaimSlots(c.byBody) {
		if len(c.byBody[slot]) != 1 { // deux joueurs pour un même corps
			rep.contested++
			continue
		}
		pi := onlyPlayerIndex(c.byBody[slot])
		if twice[pi] { // deux corps pour un même joueur
			rep.contested++
			continue
		}
		if !bodyExtendsShooter(tracks, owner, slot, pi) {
			rep.refused++
			continue
		}
		owner[slot] = pi
		rep.noteLife(slot, c.life[slot])
		rep.byShot++
	}
}

// shootersClaimingTwoBodies rend les tireurs que DEUX corps libres revendiquent.
//
// UN JOUEUR N'A QU'UN DERNIER CORPS, et sans ce contrôle l'ORDRE DES SLOTS déciderait laquelle
// des deux déductions passe : la première attribuée devient un corps connu du tireur, ce qui fait
// tomber la seconde — ou pas, selon que son numéro de slot est plus petit ou plus grand. Deux
// déductions qui s'excluent ne sont pas deux déductions ; les deux tombent. C'est le symétrique
// exact du refus « deux joueurs revendiquent le même corps » ici, et de « deux corps revendiquent
// la même mort » à la fermeture B.
func shootersClaimingTwoBodies(claims map[uint32]map[int]int) map[int]bool {
	n := map[int]int{}
	for _, m := range claims {
		if len(m) != 1 {
			continue
		}
		n[onlyPlayerIndex(m)]++
	}
	out := map[int]bool{}
	for pi, c := range n {
		if c > 1 {
			out[pi] = true
		}
	}
	return out
}

// bodyExtendsShooter — LA CORROBORATION, ET ELLE EST POSITIVE.
//
// L'UNICITÉ DU CANDIDAT NE PROUVE RIEN DE SON APPARTENANCE, et c'est le défaut que cette fonction
// ferme. `livesCoveringAt` rend les corps libres dont l'intervalle couvre l'instant du tir ; qu'il
// n'y en ait qu'un dit seulement qu'UN SEUL CORPS LIBRE EST LÀ — jamais qu'il est celui du tireur.
// Le tireur, lui, peut n'être nulle part dans les positions à cet instant : l'événement de tir est
// un record INDÉPENDANT de la réplication des bipeds, un trou de plus de `lifeGapUS` scinde une vie
// en deux dont aucune ne couvre l'instant, et une vie peut s'achever avant lui. Sans corroboration,
// le corps d'AUTRUI était alors attribué au tireur — silencieusement, puisque du point de vue du
// code il ne restait qu'un candidat. C'est exactement ce que le chantier interdit.
//
// CE QUE LE MODÈLE AUTORISE, ET RIEN D'AUTRE. Une vie anonyme est une vie QUE NULLE MORT NE TERMINE
// (`lives.go`) : dans la vie d'un joueur, c'est donc sa DERNIÈRE — celle qui court de sa dernière
// mort au coup de sifflet, précisément la zone où les tirs se perdent. Le corps déduit n'est donc
// attribuable au tireur que sous DEUX conditions, toutes deux positives :
//
//	ancrage        le tireur possède AU MOINS UN corps déjà connu. Sans ancre, rien ne le situe
//	               dans le film, et l'unicité du candidat ne dit rien de lui.
//	terminalité    TOUS ses corps connus s'achèvent AVANT que le corps candidat ne commence. Un
//	               corps connu postérieur ferait du candidat une vie INTERMÉDIAIRE du tireur —
//	               donc une vie terminée par une mort, qui l'aurait NOMMÉE. Elle ne l'a pas été :
//	               le candidat n'est pas de lui.
//
// LA SECONDE CONDITION ABSORBE LE CONTRÔLE DE RECOUVREMENT et le durcit. « S'achever avant que le
// candidat ne commence » interdit à la fois le chevauchement (un joueur n'a qu'un corps) et la
// postériorité. C'est le même invariant, énoncé dans le sens qui PROUVE au lieu de celui qui ne
// réfute pas — un test de non-contradiction laisse passer tout ce qu'il ne voit pas.
//
// CE QU'ELLE NE COUVRE PAS, ET IL FAUT LE DIRE : un tireur dont TOUS les corps connus précèdent le
// candidat reste attribuable même s'il était en réalité invisible à cet instant. Ce cas-là ne se
// distingue pas du cas nominal avec les pièces disponibles — c'est la limite de la fermeture A,
// pas un contrôle oublié.
func bodyExtendsShooter(tracks map[uint32]slotTrack, owner map[uint32]int, slot uint32, pi int) bool {
	cand := tracks[slot].pts
	if len(cand) == 0 {
		return false
	}
	from := cand[0].TimestampUS
	anchored := false
	for s, p := range owner {
		if p != pi || s == slot {
			continue
		}
		pts := tracks[s].pts
		if len(pts) == 0 {
			continue
		}
		if pts[len(pts)-1].TimestampUS >= from {
			return false // chevauchement, ou corps postérieur : le candidat n'est pas terminal
		}
		anchored = true
	}
	return anchored
}

// freeLives rend les vies sans identité dont le slot n'est pas DÉJÀ au pont : un slot nommé par
// une autre de ses vies n'a rien à déduire.
//
// CE SONT DES INDICES, pas des copies : la vie désignée doit rester nommable en fin de course
// (cf. `closureReport.closedLife`), et une copie n'est plus reliée à rien.
func freeLives(owner map[uint32]int, lives []lifeSpan) []int {
	var out []int
	for i, l := range lives {
		if l.xuid != 0 {
			continue
		}
		if _, known := owner[l.slot]; known {
			continue
		}
		out = append(out, i)
	}
	return out
}

// livesCoveringAt rend les vies libres dont l'INTERVALLE contient tUS — c'est-à-dire tous les
// corps qui pouvaient être là, y compris ceux dont la réplication a un trou à cet instant.
//
// C'EST VOLONTAIREMENT PLUS LARGE QUE LE RATTACHEMENT. Filtrer ici sur la présence d'un
// échantillon rendrait INVISIBLES les candidats en trou de réplication, et une candidature
// invisible ne conteste rien : la fermeture croirait n'avoir qu'un candidat là où elle en a deux.
func livesCoveringAt(lives []lifeSpan, free []int, tUS uint64) []int {
	var out []int
	t := int64(tUS)
	for _, i := range free {
		if t < lives[i].from || t > lives[i].to {
			continue
		}
		out = append(out, i)
	}
	return out
}

func copyOwners(in map[uint32]int) map[uint32]int {
	out := make(map[uint32]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// sortedClaimSlots rend les slots revendiqués dans un ordre STABLE : itérer une map donnerait un
// pont différent d'une exécution à l'autre, donc un artefact non reproductible.
func sortedClaimSlots(m map[uint32]map[int]int) []uint32 {
	out := make([]uint32, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
