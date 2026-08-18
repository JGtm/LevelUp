package replay

// zone_states.go — LA REGLE qui remplit `zoneStates` : comment un slot `ti=13` devient une zone,
// et comment la valeur d'un canal devient un proprietaire.
//
// LA FORME publiee vit dans `document_zones.go`, le CABLAGE dans `build_zones.go` — meme partage
// que le drapeau vivant et les socles d'arme.
//
// LES TROIS PONTS, ET AUCUN N'EST DEVINE :
//
//	slot -> zone       la coincidence d'un SOMMET DE JAUGE avec une capture nommee attribuee
//	                   geometriquement. Vote modal sur tout le match, puis les slots qu'aucune
//	                   capture ne rattache sont ECARTES (jamais poses sur une zone plausible).
//	valeur -> equipe   la valeur du tag 4 EST l'index d'equipe (mesure : 100 % et 91,1 % hors
//	                   emissions neutres). Le roster valide, il n'invente pas : une valeur qui
//	                   n'est pas un camp connu n'ouvre aucun intervalle et se compte.
//	instant -> frame   les lectures de `ti=13` sont datees sur l'HORLOGE MOTEUR, la meme que les
//	                   positions : la conversion est une division, sans correction d'origine
//	                   (contrairement aux calques dates depuis le premier paquet du film).

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// SEUILS DE L'APPARIEMENT — repris TELS QUELS de la mesure (lot C-bis phase 2a). Les faire
// bouger ici rendrait les chiffres du journal incomparables a ceux du document.
const (
	// zoneWindowMS : demi-fenetre autour d'une capture. C'est celle de l'arbitrage.
	zoneWindowMS = 2000
	// zoneCaptureDistanceM : tolerance geometrique de l'attribution d'une capture a une zone.
	// Cinq metres, ecrits avant la mesure : a l'instant d'une action de zone la mediane du
	// joueur a la zone vaut 6,6 m et seuls ~10 % sont DEDANS (la statistique est repliquee
	// apres l'action). Le taux STRICT des captures de zone vaut malgre tout 73,2 % — elles se
	// font bien dans la zone ; la tolerance ne sert qu'a ne pas juger sur une poignee de cas.
	zoneCaptureDistanceM = 5.0
	// Ce qui compte comme une RAMPE de la jauge : trois emissions croissantes et une amplitude
	// d'au moins 4 096 quanta sur 24 bits.
	zoneRampMinSamples   = 3
	zoneRampMinAmplitude = 4096

	// zoneNeutralOwner est la valeur « personne » du canal de propriete : -1 sur 32 bits.
	zoneNeutralOwner = 0xFFFFFFFF
)

// ZoneInput est CE QUE L'APPELANT FOURNIT de l'etat des zones, plus ce que `BuildFromFilm` y
// depose.
//
// LES LECTURES VOYAGENT ENSEMBLE, ET `Scanned` DIT QU'ELLES ONT ABOUTI : un calque vide sans lui
// serait indistinguable d'un film qu'on n'a pas su balayer (meme regle que [FlagInput]).
type ZoneInput struct {
	// Scanned dit que le balayage de `ti=13` a eu lieu. Faux : ni calque ni couverture.
	Scanned bool
	// Reads sont les lectures de proprietes reseau, deposees par `BuildFromFilm`. L'appelant ne
	// les remplit pas.
	Reads []filmdec.ManagedPropertyRead
	// Zones est le catalogue de zones de la carte, DANS L'ORDRE OU LE SERVICE SERT
	// `mapObjectives.zones` (role par role de la table du titre, puis rang spatial). C'est cet
	// ordre qui donne son sens a `ZoneState.ZoneRef` — d'ou `Roles`, publie a cote.
	Zones []Zone
	// Roles nomme ces roles, dans l ordre et separes par une virgule, pour que la jointure soit
	// verifiable (cf. ZonesCoverage.Roles).
	Roles string
	// TeamByXUID est le roster (xuid -> camp). Il VALIDE la valeur du canal de propriete, il ne
	// la remplace pas. Vide : seuls les camps 0 et 1 — les deux valeurs mesurees — sont acceptes.
	TeamByXUID map[string]int
}

// zoneCtx porte l'axe de temps et les entrees deja assemblees du document (regle des 5
// parametres).
type zoneCtx struct {
	origin, step uint64
	frames       int
	intervalMS   int
	tracks       []Track
	// actions sont les actions d'objectif DEJA posees sur la grille de frames (doc.Objectives) :
	// un seul decodage du statborg pour tout le document.
	actions []ObjectiveAction
	matchID string
}

// zoneSample est une emission scalaire posee sur la grille de frames.
type zoneSample struct {
	t int
	v uint64
}

// zoneSeries regroupe les emissions scalaires par tag puis par slot.
type zoneSeries struct {
	gauge map[uint32][]zoneSample // tag 3 : la jauge de capture
	owner map[uint32][]zoneSample // tag 4 : le proprietaire
	keys  map[uint32]uint32       // tag 5 : la cle de nommage, une par slot
	slots int
}

// buildZoneStates rend l'etat des zones et sa couverture. Rend (nil, nil) quand l'appelant n'a
// rien fourni a lire — ce qui ne dit PAS la meme chose qu'un calque vide.
func buildZoneStates(in ZoneInput, c zoneCtx) ([]ZoneState, *ZonesCoverage) {
	if !in.Scanned {
		return nil, nil
	}
	cov := &ZonesCoverage{Method: ZoneMethodCaptures, Roles: in.Roles, Catalog: len(in.Zones)}
	if len(in.Zones) == 0 || c.frames <= 0 || c.step == 0 {
		slog.Warn("rejeu : etat des zones sans catalogue de carte — aucun intervalle publie",
			"match_id", c.matchID, "zones", len(in.Zones), "frames", c.frames)
		return nil, cov
	}
	cat := zoneCatalogOf(in.Zones)
	ser := zoneSeriesOf(in.Reads, c)
	cov.Slots = ser.slots
	caps := zoneCapturesOf(c.actions)
	cov.Captures = len(caps)
	att, _ := AttributeZones(caps, c.tracks, cat, AttributeOptions{MaxDistanceM: zoneCaptureDistanceM})
	pairs := zonePairsOf(att)
	cov.Attributed = len(pairs)
	if len(pairs) == 0 {
		return buildHillStates(cat, ser, c, cov), cov
	}
	states := zoneOwnerStates(in, ser, pairs, c, cov)
	tallyZoneSpans(states, cov)
	return states, cov
}

// zoneSeriesOf pose les lectures scalaires sur la grille de frames et les range par tag.
//
// LES LECTURES PAR JOUEUR SONT ECARTEES : en mode a zones, leur trafic apparent est de la
// CONTAMINATION d'ancrage (0,6 a 1,5 % de chainage, au niveau de la bande fantome — mesure de la
// phase 2a). Les retenir ferait entrer du bruit dans les series.
func zoneSeriesOf(reads []filmdec.ManagedPropertyRead, c zoneCtx) zoneSeries {
	out := zoneSeries{gauge: map[uint32][]zoneSample{}, owner: map[uint32][]zoneSample{},
		keys: map[uint32]uint32{}}
	// L'ensemble des slots QUI PARLENT. Ce sont les slots de l'archetype 13, PAS les slots de
	// vie publies du rejeu : deux espaces distincts, qui n'ont ni la meme origine ni le meme
	// sens — d'ou l'ensemble local plutot qu'un appel au helper des pistes publiees.
	seen := map[uint32]struct{}{}
	for _, r := range reads {
		if r.Field != filmdec.ManagedPropertyScalar || !r.HasValue {
			continue
		}
		seen[r.Slot] = struct{}{}
		t, ok := zoneFrameOf(r.TimestampUS, c)
		if !ok {
			continue
		}
		switch r.Tag {
		case filmdec.ManagedPropertyTagQuant:
			out.gauge[r.Slot] = append(out.gauge[r.Slot], zoneSample{t: t, v: r.Value})
		case filmdec.ManagedPropertyTagU32:
			out.owner[r.Slot] = append(out.owner[r.Slot], zoneSample{t: t, v: r.Value})
		case filmdec.ManagedPropertyTagStringID:
			out.keys[r.Slot] = uint32(r.Value)
		}
	}
	out.slots = len(seen)
	for _, m := range []map[uint32][]zoneSample{out.gauge, out.owner} {
		for s := range m {
			ss := m[s]
			sort.SliceStable(ss, func(i, j int) bool { return ss[i].t < ss[j].t })
			m[s] = ss
		}
	}
	return out
}

// zoneFrameOf convertit un horodatage MOTEUR en index de frame. Faux hors de l'axe publie.
func zoneFrameOf(ts uint64, c zoneCtx) (int, bool) {
	if ts < c.origin || c.step == 0 {
		return 0, false
	}
	f := int((ts - c.origin) / c.step)
	if f >= c.frames {
		return 0, false
	}
	return f, true
}

// zoneWindowFrames rend la demi-fenetre d'appariement en frames (au moins une).
func zoneWindowFrames(intervalMS int) int {
	if intervalMS <= 0 {
		return 1
	}
	if w := zoneWindowMS / intervalMS; w > 1 {
		return w
	}
	return 1
}

// zoneCapturesOf retient les seules actions qui PRENNENT une zone. Les frags et assistances sont
// nommes par le meme decodeur mais ne disent rien d'une zone.
func zoneCapturesOf(actions []ObjectiveAction) []ObjectiveAction {
	out := make([]ObjectiveAction, 0, len(actions))
	for _, a := range actions {
		if a.Stat == objectiveevents.StatZoneCaptures || a.Stat == objectiveevents.StatZoneSecures {
			out = append(out, a)
		}
	}
	return out
}

// zoneCatalogOf renumerote le catalogue sur L'ORDRE SERVI : le rang spatial de chaque zone
// devient son index dans `mapObjectives.zones`, c'est-a-dire la seule cle que le client sache
// joindre.
//
// POURQUOI IL FAUT RENUMEROTER, ET C'EST UNE MESURE. `AttributeZones` rend, de la zone retenue,
// son rang spatial et son `InstanceID`. Ni l'un ni l'autre n'est utilisable tel quel :
//
//	l'InstanceID   vaut 0 SUR TOUTES LES ENTREES du catalogue versionne (releve du 2026-08-18,
//	               `map_objectives.json`) — il ne designe rien ;
//	le rang        est attribue PAR ROLE (`ZonesOfRole` trie et numerote son propre ensemble),
//	               donc deux roles servis cote a cote portent tous deux une zone de rang 0.
//
// Renumeroter sur la liste servie rend la cle unique SANS TOUCHER A L'ORDRE, qui est le contrat
// de `zoneRef`. La copie est locale : le catalogue de l'appelant n'est pas modifie.
func zoneCatalogOf(zones []Zone) []Zone {
	out := make([]Zone, len(zones))
	copy(out, zones)
	for i := range out {
		out[i].SpatialRank = i
	}
	return out
}

// zonePair est une capture attribuee, ramenee a l'index de zone du catalogue servi.
type zonePair struct {
	t    int
	ref  int
	xuid string
}

// zonePairsOf traduit les attributions en couples (frame, zone) exploitables.
func zonePairsOf(att []ZoneAttribution) []zonePair {
	out := make([]zonePair, 0, len(att))
	for _, a := range att {
		if !a.Attributed {
			continue
		}
		out = append(out, zonePair{t: a.Action.T, ref: a.SpatialRank, xuid: a.Action.XUID})
	}
	return out
}

// zoneRamp est une montee monotone de la jauge : le support d'une capture en cours.
type zoneRamp struct {
	slot       uint32
	t0, tPeak  int
	start, top uint64
}

// findZoneRamps decoupe une serie chronologique en montees monotones.
func findZoneRamps(slot uint32, ss []zoneSample) []zoneRamp {
	var out []zoneRamp
	for i := 0; i < len(ss); {
		j := i
		for j+1 < len(ss) && ss[j+1].v >= ss[j].v {
			j++
		}
		if j-i+1 >= zoneRampMinSamples && ss[j].v-ss[i].v >= zoneRampMinAmplitude {
			out = append(out, zoneRamp{slot: slot, t0: ss[i].t, tPeak: ss[j].t,
				start: ss[i].v, top: ss[j].v})
		}
		if j == i {
			i++
			continue
		}
		i = j + 1
	}
	return out
}

// zoneRampsOf rend toutes les rampes de la jauge du film, triees par sommet.
func zoneRampsOf(ser zoneSeries) []zoneRamp {
	out := make([]zoneRamp, 0, len(ser.gauge))
	for _, s := range sortedZoneSlots(ser.gauge) {
		out = append(out, findZoneRamps(s, ser.gauge[s])...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tPeak < out[j].tPeak })
	return out
}

// sortedZoneSlots rend les cles d'une carte de slots, triees — determinisme du parcours.
func sortedZoneSlots[T any](m map[uint32]T) []uint32 {
	out := make([]uint32, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// zoneGauge porte l'excursion MESUREE de la jauge d'une zone sur ce match : le plancher et le
// sommet observes, qui donnent son echelle a la progression publiee.
type zoneGauge struct {
	low, high uint64
	seen      bool
}

// zoneGaugeOf releve l'excursion d'une serie de jauge.
//
// POURQUOI L'ECHELLE EST CELLE DU MATCH, ET PAS LA PLAGE DECLAREE DU DESER. Le deser annonce
// [-100, +100] (constantes du jeu), mais la mesure du 2026-08-18 dit que la jauge n'y fait qu'une
// EXCURSION MINUSCULE autour de zero : ramenee lineairement sur la plage declaree, toute valeur
// reelle vaut 0,50 a trois decimales pres — un arc a moitie plein en permanence, c'est-a-dire une
// information nulle affichee comme une mesure. La progression publiee est donc la part de
// l'excursion OBSERVEE sur ce match pour CETTE zone : 1 = le sommet atteint (une capture menee a
// son terme), 0 = le plancher. C'est une echelle relative, et le champ le dit.
func zoneGaugeOf(ss []zoneSample) zoneGauge {
	var g zoneGauge
	for _, s := range ss {
		switch {
		case !g.seen:
			g.low, g.high, g.seen = s.v, s.v, true
		case s.v < g.low:
			g.low = s.v
		case s.v > g.high:
			g.high = s.v
		}
	}
	return g
}

// progressOf ramene un quantum a [0, 1] sur l'excursion mesuree. Rend nil quand l'excursion est
// vide ou plate : une jauge qui ne bouge pas n'a pas de progression a montrer.
func (g zoneGauge) progressOf(q uint64) *float32 {
	if !g.seen || g.high <= g.low {
		return nil
	}
	if q < g.low {
		q = g.low
	}
	if q > g.high {
		q = g.high
	}
	p := float32(float64(q-g.low) / float64(g.high-g.low))
	return &p
}
