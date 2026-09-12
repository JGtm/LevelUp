package replay

import "levelup/go-api/internal/games/halo_infinite/film/filmdec"

// killpos_opening.go — OÙ L'ENGAGEMENT A COMMENCÉ, ET POURQUOI CE N'EST QU'UN PROXY.
//
// # LA QUESTION
//
// La position d'une mort dit où le coup fatal est tombé. Elle ne dit pas où l'engagement a
// COMMENCÉ — et c'est cette distance-là qui a un sens tactique : on ouvre à 20 m au fusil de
// précision et on finit au contact, ou l'inverse.
//
// # POURQUOI UN PROXY ET PAS LE VRAI PREMIER DÉGÂT
//
// La vraie ouverture exigerait le PREMIER dégât de l'échange. Mesuré le 2026-09-06 (sonde
// `.ai/V7.5/film_re/SONDE_DUELS_2026-09-06.md`) : le film n'émet que 91 à 428
// enregistrements de dégât pour 90 à 117 morts, et le dégât fatal lui-même n'est capturé que
// dans 10 à 50 % des morts selon le film. Le flux de dégâts complet N'EST PAS dans le film ;
// aucune amélioration de décodage n'y changera rien (le sous-type 0xC0 non décodé n'ouvre
// aucun réservoir). Il n'y a donc PAS d'événement d'ouverture à lire.
//
// Les trajectoires, elles, sont denses et continues. D'où le proxy : la position des deux
// joueurs UN TEMPS-POUR-TUER avant le coup fatal. C'est un décalage d'horloge, pas un
// événement — et c'est précisément pour ça qu'il se valide (voir plus bas).
//
// # IL N'Y A PAS DE SECONDE FONCTION DE PLACEMENT
//
// La position d'entame se compose, elle ne se réécrit pas : `BuildKillOpenings` décale par
// `ShiftKillRefs` puis appelle `placeKillPositions`, LA fonction de placement, celle-là même
// que `BuildKillPositions` publie. Un second producteur de position divergerait du premier à
// la première correction portée d'un seul côté — c'est la règle « deux décodeurs du même fait
// divergeraient » qui gouverne déjà killpos.go et killpos_bridge.go.
//
// # CE QUE LE PLACEMENT SEUL NE SAIT PAS FAIRE, ET POURQUOI CE FICHIER EXISTE
//
// Décaler l'instant NE SUFFIT PAS, et l'affirmation contraire a vécu ici jusqu'au 2026-09-06,
// où deux relectures indépendantes l'ont démontée. `placeKillPositions` ne connaît AUCUNE
// frontière de vie : elle rend l'échantillon le plus proche de l'instant demandé, dans une
// tolérance de 120 ms. Si un joueur a RÉAPPARU entre l'entame et le coup fatal, l'instant
// décalé tombe dans la tolérance du premier échantillon de la nouvelle vie — et le placement
// rend le POINT D'APPARITION, présenté comme une entame. Reproduit : mort à 1 550 ms,
// échantillons d'apparition à t = 0, entame « résolue » sur les deux points d'apparition,
// 1 131 m d'écart pour une mort survenue à 5 m.
//
// D'où le filtre de vie de `BuildKillOpenings` : un côté n'est gardé que si le slot qui a
// fourni la position d'entame porte l'instant du KILL dans la MÊME vie (découpage
// `buildLifeSpans`, trou de `lifeGapUS`). Sinon ce côté est nil et compté dans
// `KillPosReport.OpeningOutOfLife` — mieux vaut pas d'entame qu'une position de réapparition
// présentée comme une entame, et ce qui manque est COMPTÉ plutôt que tu.

// OpeningLeadMS : l'avance, en millisecondes, du proxy d'entame sur le coup fatal.
//
// D'OÙ VIENT LA VALEUR : c'est UN TEMPS-POUR-TUER, pas un réglage. La sonde du 2026-09-06 le
// mesure à 1,2-1,5 s au fusil de combat sur les quatre films — c'est la durée pendant
// laquelle les deux joueurs se sont vus. Un décalage plus court retomberait sur la position
// du coup fatal (donc ne dirait rien de neuf) ; un décalage plus long remonterait avant le
// contact visuel, là où les deux trajectoires ne parlent plus du même engagement.
//
// LA VALEUR EST VALIDÉE, PAS SUPPOSÉE : `duels_ouverture_research_test.go` compare, sur les
// morts dont le premier dégât de l'échange EST capturé, la distance à cet instant et la
// distance à T − OpeningLeadMS. Gate écrit avant la mesure : écart médian <= 2 m.
const OpeningLeadMS int64 = 1_500

// ShiftKillRefs rend une COPIE des couples, tous décalés de deltaMS sur l'horloge du match.
//
// PURE : l'entrée n'est ni mutée ni réordonnée. Pour remonter à l'entame, l'appelant passe
// `-OpeningLeadMS` — le signe est à lui, cette fonction ne présume d'aucun sens.
//
// ELLE NE FILTRE RIEN, ET C'EST VOULU : elle décale, un point c'est tout. Ce qui est plaçable
// ou non, ce qui appartient ou non à la vie du kill, se décide chez `BuildKillOpenings` — qui
// est le seul appelant légitime pour une entame. L'utiliser à nu puis appeler
// `BuildKillPositions` rend des points d'apparition (voir l'en-tête).
func ShiftKillRefs(kills []KillRef, deltaMS int64) []KillRef {
	if len(kills) == 0 {
		return nil
	}
	out := make([]KillRef, len(kills))
	for i, k := range kills {
		k.TimeMS += deltaMS
		out[i] = k
	}
	return out
}

// BuildKillOpenings rend la position d'ENTAME — `OpeningLeadMS` avant le coup fatal — des deux
// joueurs de chaque mort, UNIQUEMENT si l'instant décalé tombe dans la MÊME VIE que le kill.
//
// PURE, comme tout ce paquet : aucune I/O. Mêmes paramètres que `BuildKillPositions`, et pour
// cause — elle place par la même fonction.
//
// L'INSTANT PORTÉ EN SORTIE EST CELUI DU KILL, jamais l'instant décalé. `time_ms` est la clé de
// jointure d'une entame vers sa mort (`kill_positions` / `kill_openings`) : un consommateur qui
// persisterait l'instant décalé ne retrouverait aucun kill en face.
//
// LE RAPPORT EST RECOMPTÉ après le filtre : retirer un côté change la classe de la mort
// (`Both` -> `KillerOnly`, et une mort dont les deux côtés tombent n'est pas écrite du tout,
// elle passe en `Dropped`). `OpeningOutOfLife` compte les CÔTÉS écartés par le filtre de vie.
func BuildKillOpenings(pos []filmdec.BipedPosition, reg IdentityRegistry,
	kills []KillRef, offsetUS int64) ([]KillPosition, KillPosReport) {
	p := placeKillPositions(pos, reg, ShiftKillRefs(kills, -OpeningLeadMS), offsetUS)
	rep := p.report
	if len(p.positions) == 0 {
		return nil, rep
	}
	lives := livesBySlot(p.tracks)
	rep.Both, rep.KillerOnly, rep.VictimOnly = 0, 0, 0
	out := make([]KillPosition, 0, len(p.positions))
	for i, kp := range p.positions {
		kp = keepOpeningInLife(kp, p.slots[i], lives, offsetUS, &rep)
		kp.TimeMS += OpeningLeadMS
		if !countKillPosition(&rep, kp) {
			continue
		}
		out = append(out, kp)
	}
	return out, rep
}

// keepOpeningInLife écarte les côtés dont l'entame et le kill ne partagent pas la même vie.
// `kp.TimeMS` est encore l'instant DÉCALÉ à l'entrée : le fatal s'en déduit.
func keepOpeningInLife(kp KillPosition, sides killSides, lives map[uint32][]lifeSpan,
	offsetUS int64, rep *KillPosReport) KillPosition {
	ouvertureUS := kp.TimeMS*1000 + offsetUS
	fatalUS := ouvertureUS + OpeningLeadMS*1000
	if kp.Killer != nil && !sameLife(lives[sides.killer], ouvertureUS, fatalUS) {
		kp.Killer = nil
		rep.OpeningOutOfLife++
	}
	if kp.Victim != nil && !sameLife(lives[sides.victim], ouvertureUS, fatalUS) {
		kp.Victim = nil
		rep.OpeningOutOfLife++
	}
	return kp
}

// livesBySlot regroupe par slot les vies découpées par `buildLifeSpans` — le découpage est
// celui du reste du paquet (trou de `lifeGapUS`), il n'est pas refait ici.
func livesBySlot(tracks map[uint32]slotTrack) map[uint32][]lifeSpan {
	out := map[uint32][]lifeSpan{}
	for _, l := range buildLifeSpans(tracks) {
		out[l.slot] = append(out[l.slot], l)
	}
	return out
}

// sameLife dit si UNE MÊME vie du slot couvre les deux instants (horloge du FILM, µs). Les
// vies d'un slot sont disjointes d'au moins `lifeGapUS` (5 s) : aucune ambiguïté à craindre
// entre deux d'entre elles à la tolérance près.
func sameLife(lives []lifeSpan, aUS, bUS int64) bool {
	for _, l := range lives {
		if coversInstant(l, aUS) && coversInstant(l, bUS) {
			return true
		}
	}
	return false
}

// coversInstant dit si l'instant tUS tombe dans la vie.
//
// LA TOLÉRANCE EST ASYMÉTRIQUE, ET C'EST TOUT LE PROPOS DE CE FICHIER :
//
//   - APRÈS la dernière position (`to`), on accorde les mêmes 120 ms que `positionOf`. La vie
//     d'une victime se TERMINE au coup fatal, et l'instant du fil des morts n'est pas
//     rigoureusement celui du dernier échantillon répliqué ; sans cette marge, aucune entame
//     de victime ne passerait jamais.
//   - AVANT la première position (`from`), AUCUNE marge. La première position d'une vie EST la
//     réapparition ; accorder 120 ms en amont rendrait exactement ce que ce fichier interdit,
//     un point d'apparition présenté comme une entame.
func coversInstant(l lifeSpan, tUS int64) bool {
	return tUS >= l.from && tUS <= l.to+killPosToleranceUS
}
