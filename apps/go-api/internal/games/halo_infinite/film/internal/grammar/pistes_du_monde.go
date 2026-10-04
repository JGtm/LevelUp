package grammar

// pistes_du_monde.go — LES PISTES DES OBJETS DU MONDE, RELEVEES EN UNE PASSE SUR L UNION DES
// BANDES (ADR 0037 IR-6 ; lot 2.5 du plan de l etape 2).
//
// # UNE PASSE, UN CURSEUR PAR BANDE
//
// Les poses, les socles et les projectiles balayaient chacun les payloads delta du film, bit a
// bit, pour la bande de leur archetype — celle de l equipement deux fois. Le prefixe de record, le
// handle et l appartenance du slot a la bande font l essentiel de ce cout ([enteteDObjetDuMonde]).
// La passe les lit une fois par position pour toutes les bandes, et chaque bande garde SON
// curseur : un record accepte n avance que le curseur de sa bande, exactement comme le balayage
// d une bande seule ([scanProjectileRecords]). Les pistes sont donc celles de ce balayage, bande
// par bande.
//
// # CE QUI EST RELEVE
//
// Au premier balayage de pistes : les bandes a pistes de la cuisson ([archetypesAPistes]) et la
// bande demandee, aux bornes et aux largeurs du moment. Une demande suivante de la meme bande aux
// memes bornes et largeurs rend une COPIE de ce qui est releve : un lecteur peut trier ses pistes.

import (
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// archetypesAPistes rend les archetypes dont la cuisson releve les pistes delta : l equipement
// (poses et socles), les armes au sol (socles) et les projectiles.
func archetypesAPistes() []int {
	return []int{EquipmentTypeIndex, GroundWeaponTypeIndex, ProjectileTypeIndex}
}

// maxBandesParPasse borne les bandes d une passe : l appartenance d un slot est un masque de bits.
const maxBandesParPasse = 32

// pistesRelevees : les pistes d une bande, aux bornes et aux largeurs ou elles ont ete relevees.
type pistesRelevees struct {
	wr     profile.Vec3Range
	lg     profile.PrecisionDescriptor
	slots  []uint32
	pistes []types.ProjectileTrack
}

// pistesDeLaBande rend une copie des pistes de la bande `band` aux bornes `wr` et aux largeurs
// `lg`, relevees au premier appel avec celles des bandes a pistes de la cuisson.
func (c *FilmContext) pistesDeLaBande(wr profile.Vec3Range, lg profile.PrecisionDescriptor,
	band map[uint32]bool) []types.ProjectileTrack {
	demandee := slotsDeLaBande(band)
	if p, ok := c.recup.pistesDe(wr, lg, demandee); ok {
		return copierLesPistes(p)
	}
	bandes := [][]uint32{demandee}
	for _, ti := range archetypesAPistes() {
		s := slotsDeLaBande(worldObjectSlotBand(c, ti))
		_, deja := c.recup.pistesDe(wr, lg, s)
		if len(s) == 0 || deja || slices.ContainsFunc(bandes, func(b []uint32) bool { return slices.Equal(b, s) }) {
			continue
		}
		bandes = append(bandes, s)
	}
	for i, pistes := range releverLesPistes(c.Film(), wr, lg, bandes) {
		c.recup.pistes = append(c.recup.pistes, pistesRelevees{wr: wr, lg: lg, slots: bandes[i], pistes: pistes})
	}
	p, _ := c.recup.pistesDe(wr, lg, demandee)
	return copierLesPistes(p)
}

// pistesDe rend les pistes relevees pour ces bornes, ces largeurs et cette bande.
func (m *memoDesRecuperations) pistesDe(wr profile.Vec3Range, lg profile.PrecisionDescriptor,
	slots []uint32) ([]types.ProjectileTrack, bool) {
	for _, p := range m.pistes {
		if p.wr == wr && p.lg == lg && slices.Equal(p.slots, slots) {
			return p.pistes, true
		}
	}
	return nil, false
}

// slotsDeLaBande rend les slots d une bande, tries.
func slotsDeLaBande(band map[uint32]bool) []uint32 {
	out := make([]uint32, 0, len(band))
	for s, ok := range band {
		if ok {
			out = append(out, s)
		}
	}
	slices.Sort(out)
	return out
}

// copierLesPistes rend une copie profonde de pistes.
func copierLesPistes(p []types.ProjectileTrack) []types.ProjectileTrack {
	out := make([]types.ProjectileTrack, len(p))
	for i, t := range p {
		out[i] = types.ProjectileTrack{Slot: t.Slot, Gen: t.Gen, Pts: slices.Clone(t.Pts)}
	}
	return out
}

// releverLesPistes releve, en une passe sur les payloads delta du film, les pistes de chacune des
// `bandes` (au plus [maxBandesParPasse]).
func releverLesPistes(film *source.Film, wr profile.Vec3Range, lg profile.PrecisionDescriptor,
	bandes [][]uint32) [][]types.ProjectileTrack {
	appartenance := appartenanceDesBandes(bandes)
	vies := make([]map[vieDePiste][]types.ProjectileSample, len(bandes))
	for i := range vies {
		vies[i] = map[vieDePiste][]types.ProjectileSample{}
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
			for i, echantillons := range echantillonsDesBandes(p.Payload(chunk), appartenance, len(bandes), &wr, lg) {
				for _, s := range echantillons {
					s.TimestampUS, s.Chunk = p.TimestampUS, c
					k := vieDePiste{s.slot, s.gen}
					vies[i][k] = append(vies[i][k], s.ProjectileSample)
				}
			}
		}
	}
	out := make([][]types.ProjectileTrack, len(bandes))
	for i := range vies {
		out[i] = pistesDesVies(vies[i])
	}
	return out
}

// appartenanceDesBandes rend, par slot, le masque des bandes qui le portent.
func appartenanceDesBandes(bandes [][]uint32) []uint32 {
	taille := 0
	for _, b := range bandes {
		if len(b) > 0 {
			taille = max(taille, int(b[len(b)-1])+1)
		}
	}
	out := make([]uint32, taille)
	for i, b := range bandes {
		for _, s := range b {
			out[s] |= 1 << uint(i)
		}
	}
	return out
}

// echantillonsDesBandes balaye UN payload delta pour `n` bandes a la fois, dont `appartenance` dit
// quels slots elles portent : la lecture d une bande seule ([scanProjectileRecords]), chaque bande
// avec son curseur. Un record se juge sans sa bande ([echantillonDuRecord]) : il est lu une fois
// pour toutes les bandes qui le portent.
func echantillonsDesBandes(pay []byte, appartenance []uint32, n int, wr *profile.Vec3Range,
	lg profile.PrecisionDescriptor) [][]projSample {
	out := make([][]projSample, n)
	posBits := projPosBits(lg)
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
	var curseurs [maxBandesParPasse]int
	for p := 0; p <= limit; p++ {
		h, ok := enteteDObjetDuMonde(pay, p)
		if !ok || int(h.Slot) >= len(appartenance) || appartenance[h.Slot] == 0 {
			continue
		}
		var s projSample
		lu, accepte := false, false
		for i := range n {
			if appartenance[h.Slot]&(1<<uint(i)) == 0 || curseurs[i] > p {
				continue
			}
			if !lu {
				lu = true
				s, accepte = echantillonDuRecord(pay, p, h, wr, lg)
			}
			if !accepte {
				break // un record refuse l est pour toutes les bandes
			}
			out[i] = append(out[i], s)
			curseurs[i] = p + posBits + 1 // un record accepte n est pas re-balaye dans sa bande
		}
	}
	return out
}

// echantillonDuRecord juge le record de handle `h` à la position p — masque, i0 présent (c'est la
// position), position déquantifiée — et rend son échantillon. Le jugement ne dépend d'aucune bande.
func echantillonDuRecord(pay []byte, p int, h types.LifeKey, wr *profile.Vec3Range,
	lg profile.PrecisionDescriptor) (projSample, bool) {
	rec, ok := masqueDObjetDuMonde(pay, p, h)
	if !ok || rec.Idx[0] != 0 { // i0 doit être présent : c'est la position
		return projSample{}, false
	}
	v, ok := decodeWorldObjectPos(pay, rec.After, wr, lg)
	if !ok {
		return projSample{}, false
	}
	rest := false
	for _, i := range rec.Idx {
		if i == projectileRestComponent {
			rest = true
		}
	}
	return projSample{
		ProjectileSample: types.ProjectileSample{X: v[0], Y: v[1], Z: v[2], AtRest: rest},
		slot:             rec.Slot, gen: rec.Gen,
	}, true
}
