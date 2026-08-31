package replay

// equipment_pickup_link_research_test.go — LOT 4, ÉTAPE 3 : LE LIEN OBJET-AU-SOL ↔ RAMASSAGE
// D'ÉQUIPEMENT, RE-MESURÉ À L'INSTANT NATIF EXACT.
//
// ## CE QUI A DÉJÀ ÉTÉ RÉFUTÉ, ET POURQUOI ON A LE DROIT D'Y REVENIR
//
// La réfutation est ÉCRITE dans `equipment_placements.go` (mesure D, 2026-08-30) : « l'équipement
// tombe à la mort AVEC les grenades du mort — plusieurs objets au mètre carré — et le lien
// spatial vers la prise i48 attrape le mauvais objet (matrice GlobalID × rang non diagonale ; à
// candidat unique il reste 0 à 2 paires par film, incohérentes) ». La règle du dépôt est de ne
// PAS retenter sans idée neuve.
//
// L'IDÉE NEUVE, ET IL FAUT ÊTRE HONNÊTE SUR SA FORCE. On m'a présenté « l'instant natif exact »
// comme la nouveauté. Ce n'en est qu'une à moitié : les émissions i48 sont DÉJÀ datées à la
// milliseconde (elles sont ancrées sur un record delta), donc la réfutation n'a jamais souffert
// d'un flou temporel — sa cause écrite est la DENSITÉ d'objets, pas la datation. Ce que le canal
// natif apporte réellement est ailleurs, et c'est mesuré (lot 2) : **70 à 72 % des ramassages
// non-arme n'ont AUCUNE émission i48 sur le même slot à moins de 500 ms**. La population que
// cette mesure examine n'est donc pas la même que celle de la mesure D — elle est trois fois
// plus grande, et les deux tiers en étaient invisibles.
//
// ## LE PRÉAMBULE N'EST PAS OPTIONNEL
//
// `glResolve` pose le verrou de process ET les largeurs d'axe DE LA CARTE ; les vies d'objets
// passent par la chaîne de PRODUCTION (`decodeFilmPlacements` pour la calibration MPP, puis
// `decodeFilmPadScans`). Les trois premières versions de la mesure voisine refaisaient la
// chaîne à la main et rendaient des nuages fantômes (médiane 24 m, puis 117 m, ÉGALE au témoin).
// On ne recopie pas la chaîne : on l'appelle.
//
// ## SEUILS ÉCRITS AVANT LA MESURE
//
//	L1 — la réfutation est LEVÉE si la médiane de distance ramasseur → objet ti=37 vivant tombe
//	     au niveau mesuré pour les ARMES (0,61-0,75 m), disons < 1,0 m, ET si la part < 1 m
//	     dépasse 60 %.
//	L2 — le TÉMOIN « autre bipède vivant au même instant » doit être au moins 3× plus loin en
//	     médiane. Sans cet écart, la proximité ne dit rien : elle ne mesurerait que la densité
//	     d'objets sur la carte.
//	L3 — le TÉMOIN « même ramasseur, instants décalés » doit lui aussi être nettement plus loin.
//	L4 — si la médiane tient à 1,4 m ou plus, la réfutation est CONFIRMÉE avec l'instrument
//	     amélioré, et on l'écrit tel quel.
//
// Gardes PICKUP_FILM + PICKUP_MAP (celles de `glResolve`).

import (
	"math"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// eqlLife est la fenêtre de présence d'un objet du monde et sa position de repos.
//
// DEUXIÈME COPIE ASSUMÉE de `glRestLife`/`glRestLives` (règle CLAUDE.md n° 6 : centraliser à la
// TROISIÈME). Celle-ci lit la voie POWER-UPS (ti=37), l'autre la voie ARMES (ti=42) ; les
// paramétrer ensemble demanderait de modifier un fichier de recherche partagé pendant qu'une
// ronde de revue lit la borne d'à côté. Si une troisième voie apparaît, on centralise.
type eqlLife struct {
	t0, tEnd uint64
	x, y, z  float32
}

// eqlLivesFromScan reconstruit les vies d'un balayage d'objets du monde : une vie par piste
// delta, bornée en fin par le recensement des images-clés de la MÊME paire (slot, génération),
// contenu à la fenêtre de la vie — la paire reboucle, son recensement mêle plusieurs vies.
func eqlLivesFromScan(scan WorldObjectScan) []eqlLife {
	byPair := map[filmdec.EquipmentLifeKey][]filmdec.ProjectileTrack{}
	for _, tr := range scan.Tracks {
		if len(tr.Pts) == 0 {
			continue
		}
		byPair[filmdec.EquipmentLifeKey{Slot: tr.Slot, Gen: tr.Gen}] = append(
			byPair[filmdec.EquipmentLifeKey{Slot: tr.Slot, Gen: tr.Gen}], tr)
	}
	var out []eqlLife
	for k, list := range byPair {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Pts[0].TimestampUS < list[j].Pts[0].TimestampUS
		})
		for i, tr := range list {
			first, last := tr.Pts[0], tr.Pts[len(tr.Pts)-1]
			next := uint64(math.MaxUint64)
			if i+1 < len(list) {
				next = list[i+1].Pts[0].TimestampUS
			}
			end := last.TimestampUS
			for _, seen := range scan.Keyframes.SeenUS[k] {
				if seen >= first.TimestampUS && seen < next && seen > end {
					end = seen
				}
			}
			out = append(out, eqlLife{t0: first.TimestampUS, tEnd: end, x: last.X, y: last.Y, z: last.Z})
		}
	}
	return out
}

// eqlNearest rend la distance au plus proche objet VIVANT à l'instant `at`, et si un tel objet
// existe.
func eqlNearest(lives []eqlLife, p filmdec.BipedPosition, at uint64) (float64, bool) {
	best, ok := math.MaxFloat64, false
	for _, l := range lives {
		if at < l.t0 || at > l.tEnd {
			continue
		}
		if d := glDist(p.X, p.Y, p.Z, l.x, l.y, l.z); d < best {
			best, ok = d, true
		}
	}
	return best, ok
}

// eqlStats rend médiane et part sous un seuil.
func eqlStats(ds []float64, seuil float64) (mediane float64, sous float64, n int) {
	if len(ds) == 0 {
		return 0, 0, 0
	}
	s := append([]float64(nil), ds...)
	sort.Float64s(s)
	m := s[len(s)/2]
	c := 0
	for _, d := range s {
		if d < seuil {
			c++
		}
	}
	return m, 100 * float64(c) / float64(len(s)), len(s)
}

func TestEquipmentPickupGroundLinkAtNativeInstant(t *testing.T) {
	s := glResolve(t)
	pickups, _, err := filmdec.ScanFilmBipedPickups(s.dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	// LA CHAÎNE DE PRODUCTION, APPELÉE — pas recopiée (cf. l'en-tête).
	_, pst := decodeFilmPlacements(s.dir, &s.wr)
	scan := decodeFilmPadScans(s.dir, &s.wr, pst.Calibration.Widths).Powerups
	if !scan.Scanned || len(scan.Tracks) == 0 {
		t.Fatalf("chaîne des power-ups muette : scanned=%v pistes=%d", scan.Scanned, len(scan.Tracks))
	}
	lives := eqlLivesFromScan(scan)
	t.Logf("== ÉTAPE 3 — LIEN RAMASSAGE ↔ OBJET ti=37, À L'INSTANT NATIF · %s ==", s.dir)
	t.Logf("ramassages natifs : %d · vies ti=37 (chaîne de production) : %d sur %d pistes delta · calibration MPP : %s",
		len(pickups), len(lives), len(scan.Tracks), pst.Calibration)

	var reel, temoinAutre, temoinDecale []float64
	sansPos, sansObjet := 0, 0
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		pos, ok := glAt(s.pos, p.Slot, p.TimestampUS)
		if !ok {
			sansPos++
			continue
		}
		d, ok := eqlNearest(lives, pos, p.TimestampUS)
		if !ok {
			sansObjet++
			continue
		}
		reel = append(reel, d)
		// TÉMOIN 1 — les AUTRES bipèdes vivants au même instant. Si la carte est saturée
		// d'objets, ils sont aussi près que le ramasseur, et la proximité ne prouve rien.
		for slot := range s.pos {
			if slot == p.Slot {
				continue
			}
			q, ok := glAt(s.pos, slot, p.TimestampUS)
			if !ok {
				continue
			}
			if dd, ok := eqlNearest(lives, q, p.TimestampUS); ok {
				temoinAutre = append(temoinAutre, dd)
			}
		}
		// TÉMOIN 2 — le MÊME ramasseur, à des instants décalés.
		for _, dec := range eqnDecalages {
			at := int64(p.TimestampUS) + dec
			if at < 0 {
				continue
			}
			q, ok := glAt(s.pos, p.Slot, uint64(at))
			if !ok {
				continue
			}
			if dd, ok := eqlNearest(lives, q, uint64(at)); ok {
				temoinDecale = append(temoinDecale, dd)
			}
		}
	}
	mr, sr, nr := eqlStats(reel, 1.0)
	ma, sa, na := eqlStats(temoinAutre, 1.0)
	md, sd, nd := eqlStats(temoinDecale, 1.0)
	t.Logf("écartés : %d sans position du ramasseur · %d sans aucun objet vivant à cet instant", sansPos, sansObjet)
	t.Logf("RÉEL          — n=%d · médiane %.2f m · part < 1 m : %.1f %%", nr, mr, sr)
	t.Logf("TÉMOIN autre bipède, même instant — n=%d · médiane %.2f m · part < 1 m : %.1f %%", na, ma, sa)
	t.Logf("TÉMOIN même ramasseur, décalé     — n=%d · médiane %.2f m · part < 1 m : %.1f %%", nd, md, sd)
	if nr == 0 {
		t.Log("VERDICT : aucun ramassage non-arme mesurable sur ce film — rien à conclure.")
		return
	}
	t.Logf("VERDICT L1 (médiane < 1,0 m ET part < 1 m > 60 %%) : %v", mr < 1.0 && sr > 60)
	t.Logf("VERDICT L2 (témoin autre bipède >= 3x la médiane réelle) : %v (%.2f vs %.2f)", ma >= 3*mr, ma, mr)
	t.Logf("VERDICT L3 (témoin décalé nettement plus loin) : %v (%.2f vs %.2f)", md >= 3*mr, md, mr)
	t.Logf("VERDICT L4 (réfutation CONFIRMÉE : médiane >= 1,4 m) : %v", mr >= 1.4)
}
