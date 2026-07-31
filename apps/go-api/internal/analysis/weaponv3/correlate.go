package weaponv3

// correlate.go — orchestrateur v3 (couche 3).
//
// Recolle les fondations v3 (pi bit-level, canon high-32, timing µs, scanners
// melee/grenade) au moteur de corrélation fire-event v2 GELÉ
// (internal/analysis). Le flux reprend EXACTEMENT le câblage du backfill v2
// (BuildWeaponTimelines + ScanFireEventsB5 par chunk + CorrelateKillsGlobal),
// mais en durcissant chaque attribution :
//   - pi résolu au niveau bit (ResolveBest) au lieu de l'ordre DB v2 ;
//   - timing fire-event µs-précis (USEstimator) au lieu du bucket grossier ;
//   - identité d'arme validée par high-32 (CanonWeaponID) — le bruit FormulaA
//     dont l'arme est inconnue est REJETÉ honnêtement (pas de faux "high") ;
//   - overlays DIRECTS melee/grenade qui surclassent le signal indirect quand un
//     marqueur film coïncide temporellement avec le kill.
//
// Réfs : .ai/PLAN_WEAPON_ATTRIBUTION_V3.md §1-§2, .ai/RESEARCH_THEATER_RE.md.

import (
	"strconv"

	"levelup/go-api/internal/analysis"
)

const (
	// gameplayChunkType — chunk_type du manifeste portant le gameplay (FormulaA,
	// fire events, marqueurs melee/grenade, xuid encodés). Aligné sur le backfill
	// v2 qui ne traite QUE les chunks type-2.
	gameplayChunkType = 2

	// meleeOverlayWindowMS — fenêtre |kill - melee| pour rattacher un MeleeHit à un
	// kill DÉJÀ étiqueté melee par highlight_events (chemin sentinel v2). Large car
	// le sentinel garantit déjà que c'est un kill melee : on ne fait que nommer l'arme.
	meleeOverlayWindowMS = 1500
	// grenadeOverlayWindowMS — fenêtre |kill - throw| pour rattacher un GrenadeThrow.
	// Plus large que melee : le projectile voyage avant d'exploser.
	grenadeOverlayWindowMS = 2500

	// meleeRecoveryWindowMS — fenêtre SERRÉE pour récupérer un kill laissé NONE par
	// le chemin fire via un MeleeHit LÉTAL (§K-bis) du même pi. highlight_events
	// N'ÉTIQUETTE PAS les kills melee (prouvé : v3Melee sentinel=0 alors que
	// match_participants.melee_kills>0), donc l'overlay sentinel ne récupère RIEN.
	// Cette fenêtre est BIEN plus serrée que l'overlay sentinel (1500ms) car ici on
	// n'a aucune garantie préalable que le kill EST un melee : seule la proximité
	// temporelle serrée + le type-byte létal + le pi font la preuve.
	//
	// VALEUR TUNÉE PAR MESURE (sweep 200/300/500/800/1200ms sur le panel snapshot,
	// cf. thought_log). Constat clé : un swing HIT (0x47/0x60) N'EST PAS un kill (un
	// coup qui touche sans tuer porte le même type-byte) → élargir la fenêtre
	// FABRIQUE du melee sur des joueurs à 0 kill melee API (000d5950 pi …760703 :
	// v3=8 api=0 à 800ms) et DÉGRADE la cohérence agrégats (30.8%→22.2%). À 200ms
	// cette fabrication Super-Fiesta tombe à ZÉRO, et les rares récupérations (CTF)
	// restent SOUS le compteur API (jamais au-dessus) → conforme à « préférer
	// sous-récupérer que fabriquer ». L'agreement v2=high est INCHANGÉ à toute
	// fenêtre (gate NONE-only). 200ms = choix conservateur retenu.
	meleeRecoveryWindowMS = 200
)

// ChunkInput — un chunk film DÉJÀ décompressé, prêt à scanner.
type ChunkInput struct {
	Index      int
	Data       []byte // décompressé (pas de zlib ici)
	StartMS    int
	DurationMS int
	ChunkType  int
}

// V3Input — entrée complète de l'orchestrateur pour un match.
type V3Input struct {
	MatchID     string
	Kills       []analysis.Kill
	RosterXuids []uint64 // xuids numériques du roster (pour la résolution pi bit-level)
	Chunks      []ChunkInput

	// FirePiLayout — layout de lecture du player_index du fire-event (plan §8).
	// Zéro-valeur (FirePi4High) = mode AUTO : 4-bit (v2) sur les matchs ≤16 joueurs,
	// 5-bit (FirePi5SpanBefore) UNIQUEMENT si la résolution pi révèle un pi>15 (BTB).
	// Une valeur explicite >0 FORCE ce layout sur tous les chunks (override mesure).
	FirePiLayout FirePiLayout
	// FireRelax3Set / FireRelax3 — recall relâché (§9 : marqueur 3-bit + validation
	// canon high-32). Si FireRelax3Set==false, l'orchestrateur prend le défaut mesuré
	// defaultFireRelax3 ; sinon FireRelax3 force la valeur (true/false explicite).
	FireRelax3Set bool
	FireRelax3    bool
}

// defaultFireRelax3 — recall relâché ACTIVÉ par défaut. MESURÉ (panel 6 matchs) :
// récupère des fires legit (53ce4390 HIGH 80→83.5 %, NULL 8.7→7.0 %), neutre sur
// le BTB, ZÉRO faux positif (agreement v2=high reste 100 % sur tous les matchs ≤16
// non-régressés) car chaque fire 3-bit est re-validé par canon high-32 (§9).
const defaultFireRelax3 = true

// bigMatchFirePiLayout — layout pi appliqué AUX SEULS matchs BTB (>16 joueurs).
// MESURÉ (plan §8) : FirePi5SpanBefore = le nibble pi 4-bit étendu
// d'UN bit vers le haut (bit avant b5). Pour pi≤15 ce bit emprunté vaut 0 → la
// lecture se réduit EXACTEMENT à la v2 (byte5>>4) ; pour pi∈[16,31] elle débloque
// le tueur BTB. Les variantes byte5>>3 / byte5&0x1f ont CASSÉ l'Arena (agreement
// 9-22 %) → rejetées. Appliqué globalement, SpanBefore régresse aussi les petits
// matchs (000d5950, 0215fe6b) car bit-31 n'y est pas du pi → le gating par TAILLE
// de roster (>16 joueurs) est OBLIGATOIRE (cf. firePiAutoLayout).
const bigMatchFirePiLayout = FirePi5SpanBefore

// bigMatchRosterThreshold — taille de roster au-delà de laquelle le champ pi du
// fire-event ne tient PLUS sur 4 bits (0-15) → lobby BTB nécessitant la lecture
// 5-bit. Le gating se fait sur le NOMBRE DE JOUEURS (déterministe, fiable), PAS
// sur le pi max résolu : ResolveBest peut produire de FAUX pi>15 par collision du
// motif xuid 64-bit bit-searché, ce qui déclencherait à tort le 5-bit sur un petit
// match (mesuré : 000d5950 8 joueurs régressait 100→59 % avec le gate par pi).
const bigMatchRosterThreshold = 16

// firePiAutoLayout décide du layout pi selon la TAILLE du roster : 5-bit
// (bigMatchFirePiLayout) si >16 joueurs (BTB), sinon 4-bit v2 (FirePi4High).
// Coeur du fix mesuré §8 : 5-bit uniquement là où le champ 4-bit déborde.
func firePiAutoLayout(rosterSize int) FirePiLayout {
	if rosterSize > bigMatchRosterThreshold {
		return bigMatchFirePiLayout
	}
	return FirePi4High
}

// BuildV3Attributions exécute le pipeline v3 complet : résolution pi, corrélation
// fire-event v2 durcie (timing µs + canon high-32), puis overlays melee/grenade.
// Renvoie une AttributionV3 par kill (même cardinalité que CorrelateKillsGlobal).
func BuildV3Attributions(in V3Input) []AttributionV3 {
	xuidToPI := resolvePIMap(in.RosterXuids, in.Chunks)

	chunkData := gameplayChunkData(in.Chunks)
	tl, chunksSorted := analysis.BuildWeaponTimelines(chunkData)

	// Layout pi : explicite si fourni, sinon AUTO selon la taille du roster (5-bit
	// seulement sur les matchs BTB >16 joueurs — sinon on régresse les petits, §8).
	layout := in.FirePiLayout
	if layout == FirePi4High {
		layout = firePiAutoLayout(len(in.RosterXuids))
	}
	relax := defaultFireRelax3
	if in.FireRelax3Set {
		relax = in.FireRelax3
	}
	fires := scanFiresUS(in.Chunks, layout, relax)
	attrs := analysis.CorrelateKillsGlobal(
		in.Kills, fires, xuidToPI,
		tl.Timeline, tl.SwapPIs, tl.Timing, chunksSorted, in.MatchID, tl.TimelineNS,
	)

	melees := collectMeleeHits(in.Chunks)
	throws := collectGrenadeThrows(in.Chunks)

	out := make([]AttributionV3, 0, len(attrs))
	for _, a := range attrs {
		out = append(out, buildOneAttribution(a, xuidToPI, melees, throws))
	}
	recoverMeleeOnNone(out, attrs, xuidToPI, melees)
	return out
}

// resolvePIMap collecte les Data des chunks gameplay, résout xuid→pi au niveau bit
// (ResolveBest, premier chunk gagnant) puis re-clé les pi sur la string décimale.
func resolvePIMap(roster []uint64, chunks []ChunkInput) map[string]int {
	datas := make([][]byte, 0, len(chunks))
	for _, c := range chunks {
		if c.ChunkType == gameplayChunkType {
			datas = append(datas, c.Data)
		}
	}
	resolved := ResolveBest(roster, datas)
	out := make(map[string]int, len(resolved))
	for xuid, pi := range resolved {
		out[strconv.FormatUint(xuid, 10)] = pi
	}
	return out
}

// gameplayChunkData construit le map index→ChunkData attendu par
// BuildWeaponTimelines (chunks type-2 uniquement, comme le backfill v2).
func gameplayChunkData(chunks []ChunkInput) map[int]analysis.ChunkData {
	out := make(map[int]analysis.ChunkData, len(chunks))
	for _, c := range chunks {
		if c.ChunkType != gameplayChunkType {
			continue
		}
		out[c.Index] = analysis.ChunkData{
			Data:       c.Data,
			StartMS:    c.StartMS,
			DurationMS: c.DurationMS,
		}
	}
	return out
}

// scanFiresUS scanne les fire events de chaque chunk gameplay avec un estimateur
// µs-précis (USEstimator) et concatène le tout (claim-and-remove global ensuite).
//
// Le layout pi du fire-event est CONFIGURABLE (plan §8) : layout==FirePi4High +
// relax3==false reproduit EXACTEMENT la v2 (analysis.ScanFireEventsB5, byte5>>4) ;
// sinon on passe par ScanFireEventsV3 (pi 5-bit BTB ou recall relâché). L'appelant
// (BuildV3Attributions) a déjà résolu le layout effectif (auto vs explicite).
func scanFiresUS(chunks []ChunkInput, layout FirePiLayout, relax3 bool) []analysis.FireEvent {
	useV2 := layout == FirePi4High && !relax3
	var fires []analysis.FireEvent
	for _, c := range chunks {
		if c.ChunkType != gameplayChunkType {
			continue
		}
		est := USEstimator(c.Data, c.StartMS)
		if useV2 {
			fires = append(fires, analysis.ScanFireEventsB5(c.Data, est)...)
			continue
		}
		fires = append(fires, ScanFireEventsV3(c.Data, est, layout, relax3)...)
	}
	return fires
}

// collectMeleeHits concatène les coups de melee weapon-validés de TOUS les chunks
// (le marqueur melee peut apparaître hors des chunks type-2 stricts ; on scanne
// large, le filtre high-32 écarte le bruit).
func collectMeleeHits(chunks []ChunkInput) []MeleeHit {
	var hits []MeleeHit
	for _, c := range chunks {
		est := USEstimator(c.Data, c.StartMS)
		hits = append(hits, ScanMeleeHits(c.Data, est)...)
	}
	return hits
}

// collectGrenadeThrows concatène les lancers de grenade des chunks gameplay.
func collectGrenadeThrows(chunks []ChunkInput) []GrenadeThrow {
	var throws []GrenadeThrow
	for _, c := range chunks {
		if c.ChunkType != gameplayChunkType {
			continue
		}
		est := USEstimator(c.Data, c.StartMS)
		throws = append(throws, ScanGrenadeThrows(c.Data, est)...)
	}
	return throws
}

// buildOneAttribution transforme une KillAttribution v2 en AttributionV3 :
// copie du coeur, dérivation du SourceSignal, canon high-32 (rejet du bruit), puis
// overlays melee/grenade prioritaires (signal DIRECT > fire-event indirect).
func buildOneAttribution(
	a analysis.KillAttribution,
	xuidToPI map[string]int,
	melees []MeleeHit,
	throws []GrenadeThrow,
) AttributionV3 {
	v := AttributionV3{
		MatchID:         a.MatchID,
		XUID:            a.XUID,
		TimeMS:          a.TimeMS,
		WeaponID:        a.WeaponID,
		DeltaMS:         a.DeltaMS,
		Confidence:      a.Confidence,
		AttributionPath: a.AttributionPath,
		PlayerIndex:     a.PlayerIndex,
		SourceChunkIdx:  a.SourceChunkIdx,
		SourceSignal:    signalFromPath(a.AttributionPath),
	}
	applyCanon(&v)
	applyMeleeGrenadeOverlay(&v, a, xuidToPI, melees, throws)
	return v
}

// signalFromPath mappe l'AttributionPath v2 vers le SourceSignal v3 (le défaut
// melee/grenade vient des overlays, pas du path).
func signalFromPath(path string) string {
	switch path {
	case analysis.AttributionPathFireEvent:
		return SignalFire
	case analysis.AttributionPathFormulaA:
		return SignalFormulaA
	default:
		return SignalNone
	}
}

// applyCanon valide l'arme par son high-32 et REJETTE le bruit. Si l'arme est
// connue → on expose le high-32. Si elle est inconnue → on ne prétend PAS la
// connaître : WeaponID/HighWeaponID sont remis à nil, et un signal fire dégrade en
// "low" (le tir existait, mais l'arme est du bruit FormulaA).
func applyCanon(v *AttributionV3) {
	if v.WeaponID == nil {
		return
	}
	high, known := CanonWeaponID(*v.WeaponID)
	if known {
		v.HighWeaponID = &high
		return
	}
	v.WeaponID = nil
	v.HighWeaponID = nil
	if v.SourceSignal == SignalFire {
		v.Confidence = confidenceLowV3
	}
}

// applyMeleeGrenadeOverlay surclasse l'attribution par un signal DIRECT si un
// marqueur film coïncide. Priorité : melee (arme RÉELLE ancrée sur le pi) puis,
// à défaut, grenade (sans pi validé → confiance medium honnête).
func applyMeleeGrenadeOverlay(
	v *AttributionV3,
	a analysis.KillAttribution,
	xuidToPI map[string]int,
	melees []MeleeHit,
	throws []GrenadeThrow,
) {
	if applyMeleeOverlay(v, a, xuidToPI, melees) {
		return
	}
	applyGrenadeOverlay(v, a, throws)
}

// applyMeleeOverlay rattache un MeleeHit (même pi killer, |Δt|<=1500ms) pour
// upgrader le sentinel melee générique vers l'arme réelle. Si le kill est flaggé
// melee mais qu'aucun hit ne matche, on garde l'attribution mais on signe melee à
// confiance basse (on sait que c'est du corps-à-corps, pas avec quelle arme).
// Renvoie true si l'overlay melee s'est appliqué (kill traité comme melee).
func applyMeleeOverlay(
	v *AttributionV3,
	a analysis.KillAttribution,
	xuidToPI map[string]int,
	melees []MeleeHit,
) bool {
	// GARDE-FOU (fix mesure panel Phase B) : l'overlay melee ne s'applique QU'AUX
	// kills que highlight_events a étiquetés melee (sentinel v2 MeleeWeaponID). Sans
	// elle, tout fire-kill ayant un swing du même pi à ±1500ms était reclassé melee
	// → sur-attribution massive sur Super Fiesta/Gravity Hammer (30 melee vs ~4 réels)
	// et régressions vs v2-high. Le scanner §K-bis ne sert qu'à nommer l'arme RÉELLE
	// d'un kill DÉJÀ confirmé melee, jamais à requalifier un fire-kill.
	if a.WeaponID == nil || *a.WeaponID != analysis.MeleeWeaponID {
		return false
	}
	if killerPI, hasPI := xuidToPI[a.XUID]; hasPI {
		if hit, ok := nearestMeleeHit(melees, killerPI, a.TimeMS); ok {
			wid := hit.WeaponID
			high, _ := CanonWeaponID(wid) // hit déjà validé high-32 connu par le scanner
			v.WeaponID = &wid
			v.HighWeaponID = &high
			v.SourceSignal = SignalMelee
			v.Confidence = confidenceHighV3
			return true
		}
	}
	// Kill melee confirmé mais aucun hit weapon-validé trouvé : on sait que c'est du
	// corps-à-corps, pas avec quelle arme → melee à confiance basse (honnête).
	v.SourceSignal = SignalMelee
	v.Confidence = confidenceLowV3
	return true
}

// recoverMeleeOnNone récupère les kills laissés NONE par le chemin fire (WeaponID
// ET HighWeaponID nil après corrélation+canon) en leur rattachant un MeleeHit
// LÉTAL (§K-bis : type-byte 0x47/0x60, 0x42 whiff exclu) du MÊME pi killer dans
// une fenêtre SERRÉE (meleeRecoveryWindowMS). C'est le SEUL chemin de récupération
// melee réellement actif : highlight_events n'étiquette pas les kills melee, donc
// le sentinel v2 (applyMeleeOverlay) ne se déclenche jamais.
//
// Appariement HIT-DRIVEN (un swing = un kill) : chaque MeleeHit létal réclame le
// kill NONE NON ENCORE RÉCUPÉRÉ du même pi le PLUS PROCHE en temps dans la fenêtre.
// Ainsi un hit est donné au kill qu'il a réellement causé (le plus proche), pas au
// premier kill rencontré dans l'ordre — et un hit sert au plus un kill.
//
// GARANTIES anti-régression :
//   - ne touche QUE les kills NONE → un kill déjà attribué fire/high/melee est
//     INTOUCHÉ (agreement v2=high inchangé, zéro régression Arena/CTF).
//   - claim-and-remove des DEUX côtés : un hit sert au plus un kill, un kill reçoit
//     au plus un hit (anti sur-attribution multi-kills).
//
// out et attrs sont indexés à l'identique (même cardinalité, même ordre).
func recoverMeleeOnNone(
	out []AttributionV3,
	attrs []analysis.KillAttribution,
	xuidToPI map[string]int,
	melees []MeleeHit,
) {
	lethal := lethalMeleeHits(melees)
	if len(lethal) == 0 {
		return
	}
	killClaimed := make([]bool, len(out))
	for _, hit := range lethal {
		i, ok := nearestNoneKill(out, attrs, xuidToPI, killClaimed, hit)
		if !ok {
			continue
		}
		killClaimed[i] = true
		applyRecoveredMelee(&out[i], hit)
	}
}

// lethalMeleeHits filtre les MeleeHit aux seuls coups LÉTAUX (HIT 0x47/0x60). Les
// 0x42 (miss/unpowered) sont exclus : un whiff ne tue pas (sinon sur-attribution).
func lethalMeleeHits(melees []MeleeHit) []MeleeHit {
	out := melees[:0:0]
	for _, h := range melees {
		if MeleeHitLethal(h.HitType) {
			out = append(out, h)
		}
	}
	return out
}

// nearestNoneKill renvoie l'indice du kill NONE (arme non résolue) NON ENCORE
// RÉCUPÉRÉ dont le pi == hit.PI et le plus proche en temps de hit.TimeMS dans la
// fenêtre meleeRecoveryWindowMS, ou ok=false si aucun. Les kills déjà attribués
// (WeaponID/HighWeaponID non nil) sont d'office exclus (jamais reclassés melee).
func nearestNoneKill(
	out []AttributionV3,
	attrs []analysis.KillAttribution,
	xuidToPI map[string]int,
	killClaimed []bool,
	hit MeleeHit,
) (int, bool) {
	best := -1
	bestDelta := meleeRecoveryWindowMS + 1
	for i := range out {
		if killClaimed[i] || out[i].WeaponID != nil || out[i].HighWeaponID != nil {
			continue
		}
		if pi, hasPI := xuidToPI[attrs[i].XUID]; !hasPI || pi != hit.PI {
			continue
		}
		d := abs(out[i].TimeMS - hit.TimeMS)
		if d <= meleeRecoveryWindowMS && d < bestDelta {
			best, bestDelta = i, d
		}
	}
	return best, best >= 0
}

// applyRecoveredMelee pose l'arme melee RÉELLE (déjà validée canon high-32 par le
// scanner) sur un kill récupéré : confiance HAUTE (arme lue + type-byte létal + pi
// + fenêtre serrée), SourceSignal=melee.
func applyRecoveredMelee(v *AttributionV3, hit MeleeHit) {
	wid := hit.WeaponID
	high, _ := CanonWeaponID(wid) // hit déjà validé high-32 connu par le scanner
	v.WeaponID = &wid
	v.HighWeaponID = &high
	v.SourceSignal = SignalMelee
	v.Confidence = confidenceHighV3
}

// applyGrenadeOverlay rattache un GrenadeThrow (|Δt|<=2500ms) à un kill grenade.
// Le marqueur grenade n'expose PAS de pi validé (§C) : on ne peut pas certifier le
// lanceur → medium. Le high-32 va dans HighWeaponID ; WeaponID encode le high-32
// dans ses bits hauts (le marqueur ne fournit pas le suffixe bas d'un fire-event).
func applyGrenadeOverlay(v *AttributionV3, a analysis.KillAttribution, throws []GrenadeThrow) {
	isGrenadeKill := a.WeaponID != nil && *a.WeaponID == analysis.GrenadeWeaponID
	if !isGrenadeKill {
		return
	}
	if th, ok := nearestGrenadeThrow(throws, a.TimeMS); ok {
		high := th.WeaponID
		wid := uint64(th.WeaponID) << 32
		v.HighWeaponID = &high
		v.WeaponID = &wid
		v.SourceSignal = SignalGrenade
		v.Confidence = confidenceMediumV3
	}
}

// nearestMeleeHit renvoie le MeleeHit du pi donné le plus proche (en temps) du
// kill dans la fenêtre meleeOverlayWindowMS, ou ok=false si aucun.
func nearestMeleeHit(melees []MeleeHit, pi, killMS int) (MeleeHit, bool) {
	best := MeleeHit{}
	bestDelta := meleeOverlayWindowMS + 1
	found := false
	for _, h := range melees {
		if h.PI != pi {
			continue
		}
		d := abs(h.TimeMS - killMS)
		if d <= meleeOverlayWindowMS && d < bestDelta {
			best, bestDelta, found = h, d, true
		}
	}
	return best, found
}

// nearestGrenadeThrow renvoie le GrenadeThrow le plus proche (en temps) du kill
// dans la fenêtre grenadeOverlayWindowMS, ou ok=false si aucun.
func nearestGrenadeThrow(throws []GrenadeThrow, killMS int) (GrenadeThrow, bool) {
	best := GrenadeThrow{}
	bestDelta := grenadeOverlayWindowMS + 1
	found := false
	for _, th := range throws {
		d := abs(th.TimeMS - killMS)
		if d <= grenadeOverlayWindowMS && d < bestDelta {
			best, bestDelta, found = th, d, true
		}
	}
	return best, found
}
