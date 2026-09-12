package filmdec

import (
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// keyframe_ground_weapons.go — ARMES AU SOL, lues dans les keyframes type-2.
//
// CE QUE C'EST. Le même balayage de familles d'arme que keyframe_loadout.go, mais attribué aux
// records de l'archétype ARME AU SOL (ti=42) au lieu du biped (ti=35). L'ancrage anti-hasard est
// commun aux deux (911 occurrences de famille sur 29 997 624 bits contre 0,52 attendue par pur
// hasard, cf. l'en-tête de keyframe_loadout.go) et sa répartition par archétype porteur — biped
// 495, arme au sol 397 — est la mesure qui fonde CE fichier : les 397 occurrences d'armes au sol
// existaient déjà, elles étaient simplement écartées par le filtre d'archétype.
//
// CE QUE CE DÉCODEUR NE DONNE PAS, et il ne faut pas le lui demander :
//   - le PORTEUR / le lien arme -> joueur (composant i10 object-parent-state : consumed_only,
//     largeur dépendante de param_4 non résolue hors capture) ;
//   - le POSER et le RAMASSER : le film ne porte AUCUN événement typé pickup/drop (mesuré,
//     investigation 2026-08-12) et les records NEW/DEL d'entité ne sont pas appariés ici ;
//   - la POSITION : MESURÉE ET RÉFUTÉE le 2026-08-12 (cf. GroundWeaponPositions ci-dessous). Il
//     n'existe aucune coordonnée publiable pour une arme au sol.
//
// Un keyframe toutes les ~20 s : ce que ce fichier rend est un ÉTAT DE RÉFÉRENCE (quelles armes
// gisent au sol à cet instant), pas un suivi continu.
//
// COUVERTURE MESURÉE (film 000d5950, catalogue de production weaponv3.KnownWeaponHigh32) :
// 26 images-clés, 269 records ti=42, 196 lectures portant une famille connue, 169 vies (slot, gen),
// 22 armes nommées distinctes, et 0 fuite d'alias sur la voisine (172 paires consécutives testées).
//
// HORS LIGNE par construction (I/O disque sur tout le film) — jamais depuis un chemin de requête.

// GroundWeaponTypeIndex est l'archétype (typeIndex) des ARMES AU SOL. Même famille de records que
// les autres objets du monde (projectile ti=41, équipement ti=37, corps rigide ti=38) : cf.
// WorldObjectPrecision. À VÉRIFIER par le nom des composants du registre plutôt qu'à câbler —
// c'est un index de build, pas une constante du format (même réserve que ProjectileTypeIndex).
const GroundWeaponTypeIndex = 42

// KeyframeGroundWeapon est une arme AU SOL vue à l'instant d'un keyframe.
type KeyframeGroundWeapon struct {
	// TimestampUS est l'horodatage du paquet keyframe — MÊME horloge que
	// BipedPosition.TimestampUS et KeyframeLoadout.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent le keyframe dans le film.
	Chunk, PacketIndex int
	// Slot / Gen identifient l'ENTITÉ arme au sol. Le pool de slots reboucle : c'est la PAIRE
	// qui désigne une vie d'objet, comme pour les projectiles (cf. ProjectileTrack).
	Slot, Gen uint32
	// Families liste les familles (high-32 du weapon-id) trouvées dans le record, dans l'ORDRE
	// DES BITS. Les alias ne sont PAS repliés (même convention que KeyframeLoadout.Families) :
	// le repli appartient à la couche qui possède le catalogue d'armes.
	Families []uint32
}

// ScanFilmKeyframeGroundWeapons décode les armes AU SOL de tous les keyframes du film de dir.
// `known` est le prédicat d'appartenance au catalogue de familles : c'est LUI qui fait la
// sélectivité du balayage (cf. l'ancrage anti-hasard de keyframe_loadout.go).
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
//
// ScanFilmKeyframeGroundWeapons est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanKeyframeGroundWeapons].
func ScanFilmKeyframeGroundWeapons(dir string, known map[uint32]bool) ([]KeyframeGroundWeapon, error) {
	if len(known) == 0 {
		return nil, nil // catalogue vide : rien a chercher, et rien a charger
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanKeyframeGroundWeapons(film, known)
}

// ScanKeyframeGroundWeapons décode les armes au sol des images-clés d'un film DEJA CHARGE.
func ScanKeyframeGroundWeapons(film *filmsource.Film, known map[uint32]bool) ([]KeyframeGroundWeapon, error) {
	if len(known) == 0 {
		return nil, nil
	}
	var out []KeyframeGroundWeapon
	read := 0
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		read++
		for _, p := range pks {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			for _, g := range keyframeGroundWeapons(p.Payload(chunk), known) {
				g.TimestampUS, g.Chunk, g.PacketIndex = p.TimestampUS, c, p.Index
				out = append(out, g)
			}
		}
	}
	if read == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	return out, nil
}

// keyframeGroundWeapons balaye un payload de keyframe et rend une arme au sol par record ti=42
// porteur d'au moins une famille connue. PUR (aucune I/O).
func keyframeGroundWeapons(pay []byte, known map[uint32]bool) []KeyframeGroundWeapon {
	rf := familiesByRecord(pay, known, GroundWeaponTypeIndex)
	if len(rf) == 0 {
		return nil
	}
	out := make([]KeyframeGroundWeapon, 0, len(rf))
	for _, r := range rf {
		out = append(out, KeyframeGroundWeapon{
			Slot:     uint32(r.Rec.Slot),
			Gen:      uint32(r.Rec.Gen),
			Families: r.Families,
		})
	}
	return out
}

// GroundWeaponSlotBand rend les slots utilisables pour les armes au sol, lus dans les keyframes :
// la bande de l'archétype comblée, PUIS privée de tout slot vu porter un AUTRE archétype (règle
// mesurée en 2026-07-26, cf. worldObjectSlotBand — combler seul contamine, s'en tenir à l'observé
// rate l'essentiel).
//
// HORS LIGNE (I/O disque sur tout le film).
// ENVELOPPE D2, HORS PRODUCTION : consommee par les instruments de mesure des armes au sol.
func GroundWeaponSlotBand(dir string) map[uint32]bool {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil
	}
	return worldObjectSlotBand(film, GroundWeaponTypeIndex)
}

// GroundWeaponPositions rend, par slot d'arme au sol, les positions monde décodées des paquets
// DELTA (`object-position-component`, chemin world-object 45 bits déjà validé sur les trajectoires
// de grenade — cf. decodeWorldObjectPos). Les échantillons d'un slot sont triés par instant.
//
// ATTENTION — CE QUE CETTE FONCTION REND N'EST PAS PUBLIABLE, et c'est mesuré. Son unique
// consommateur est l'INSTRUMENT DE MESURE `replay/ground_weapon_research_test.go` (sous garde
// d'environnement GW_FILM), qui existe pour garder la réfutation REJOUABLE. Verdict du 2026-08-12
// sur 000d5950 : témoin fantôme de même cardinalité (899 slots jamais vus en image-clé) -> 493
// slots peuplés / 10 950 échantillons contre 661 / 44 498 pour la vraie bande, et discriminant
// spatial (une arme posée ne bouge pas) -> 3,3 % seulement des 458 slots réels à >=3 échantillons
// tiennent dans 0,5 u, 62,4 % s'étalent au-delà de 20 u. La majorité de ces « positions » est du
// BRUIT. Ne pas la brancher sur un artefact ni sur une réponse d'API.
//
// DEUX CAUSES STRUCTURELLES ÉTAIENT INVOQUÉES ; LA PREMIÈRE EST LEVÉE (2026-08-17) :
//   - le default-state de l'archétype (vtable[0x60]) n'avait PAS d'entrée 42 dans
//     `defaultStateDeserByTI`. Il en a une depuis le 2026-08-17 : `consumeDefaultStateTI42`
//     (default_state_ti42.go), portée depuis `FUN_1407f0c68` et VALIDÉE par un oracle de
//     position — 282 atterrissages exacts sur 289 records de création (97,6 %) sur trois films,
//     contre 0 sur 289 pour trois déserialiseurs faux passés par le même code. C'est ce qui
//     rend lisible le record de CRÉATION d'une arme au sol
//     (`ScanFilmGroundWeaponCreations`, ground_weapon_creation.go) ;
//   - en DELTA, le record ne porte aucun typeIndex (il résout son archétype par le World) et la
//     bande de slots comblée est contaminée par les archétypes voisins. CETTE CAUSE-LÀ TIENT, et
//     c'est elle qui gouverne la fonction ci-dessous : le record de création, lui, DIT `ti=42`.
//
// CE QUE LA SECONDE LECTURE A MESURÉ (2026-08-17, 6 films / 5 cartes, aux largeurs de chaque
// carte). Restreindre la bande aux VIES (slot, génération) confirmées par un record de création
// fait tomber la part des vies étalées au-delà de 20 u de 59,1 % à 19,0 % (n = 1 965 -> 1 140)
// et monter la part sous 2 u de 25,5 % à 53,3 % ; mais la part STABLE au sens strict (0,5 u)
// ne fait que doubler, 3,2 % -> 6,2 %, loin du saut d'un ordre de grandeur exigé. La raison
// tient à la nature du flux : un objet posé cesse d'émettre sa position (même acquis que `ti=37`,
// cf. EquipmentLifeSpan), donc les échantillons delta d'une arme au sol sont sa phase MOBILE —
// mesurer leur immobilité n'a pas de sens. La position publiable d'une arme au sol est celle de
// son record de CRÉATION, pas la dispersion de ses deltas.
//
// Report et condition de reprise : .ai/V7.5/REGISTRE_REPORTS.md (2026-08-12, amendé les
// 2026-08-17).
//
// HORS LIGNE (I/O disque sur tout le film).
//
// ENVELOPPE D2, HORS PRODUCTION (marquee a l'item 1.7, 2026-09-03 : elle chargeait deux films
// par appel — un par enveloppe deleguee — sans porter le marqueur que l'inventaire des
// enveloppes cherche). Son unique consommateur est l'instrument de mesure cite ci-dessus.
func GroundWeaponPositions(dir string, wr *Vec3Range) map[uint32][]WorldObjectSample {
	if wr == nil {
		return map[uint32][]WorldObjectSample{}
	}
	return WorldObjectPositionsForBand(dir, wr, GroundWeaponSlotBand(dir))
}

// WorldObjectPositionsForBand décode les positions des paquets delta pour une BANDE DE SLOTS
// donnée. La bande est un paramètre pour que le témoin de contrôle (une bande FANTÔME de même
// cardinalité, faite de slots jamais vus porter cet archétype) passe par le MÊME code que la
// mesure — sans quoi le contrôle ne contrôlerait pas le décodeur mais une variante de lui.
//
// MÊME AVERTISSEMENT QUE GroundWeaponPositions : consommateur unique = l'instrument de mesure
// (`replay/ground_weapon_research_test.go`, garde GW_FILM). Ce n'est pas un décodeur de production.
//
// HORS LIGNE (I/O disque sur tout le film).
//
// ENVELOPPE D2, HORS PRODUCTION : son unique consommateur est l'instrument de mesure ci-dessus.
func WorldObjectPositionsForBand(dir string, wr *Vec3Range, band map[uint32]bool) map[uint32][]WorldObjectSample {
	out := map[uint32][]WorldObjectSample{}
	if wr == nil || len(band) == 0 {
		return out
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return out
	}
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, p := range pks {
			if p.Type != PacketTypeDelta {
				continue
			}
			for _, s := range scanProjectileRecords(p.Payload(chunk), band, wr) {
				out[s.slot] = append(out[s.slot], WorldObjectSample{
					TimestampUS: p.TimestampUS,
					Gen:         s.gen,
					X:           s.X, Y: s.Y, Z: s.Z,
				})
			}
		}
	}
	for slot := range out {
		pts := out[slot]
		sort.Slice(pts, func(i, j int) bool { return pts[i].TimestampUS < pts[j].TimestampUS })
	}
	return out
}

// WorldObjectSample est une position monde d'un objet du monde à un instant.
type WorldObjectSample struct {
	TimestampUS uint64
	Gen         uint32
	X, Y, Z     float32
}

// NearestWorldObjectSample rend l'échantillon de position le plus proche dans le temps de atUS,
// et l'écart en microsecondes. ok=false si la liste est vide.
func NearestWorldObjectSample(pts []WorldObjectSample, atUS uint64) (WorldObjectSample, uint64, bool) {
	var best WorldObjectSample
	var bestGap uint64
	found := false
	for _, p := range pts {
		gap := p.TimestampUS - atUS
		if p.TimestampUS < atUS {
			gap = atUS - p.TimestampUS
		}
		if !found || gap < bestGap {
			best, bestGap, found = p, gap, true
		}
	}
	return best, bestGap, found
}
