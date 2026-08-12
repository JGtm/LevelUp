package replay

// ground_weapon_research_test.go — INSTRUMENT DE MESURE des ARMES AU SOL (Phase 2.1 du plan
// .ai/V7.5/replay2d/PLAN_FICHES_ENRICHIES_REJEU2D.md).
//
// CE QU'IL MESURE, et pourquoi il existe. La décision d'inclure ou non le calque « armes au sol »
// dans la v7.5 se prend sur un CHIFFRE DE COUVERTURE, pas sur une intention. Deux grandeurs
// distinctes, à ne jamais confondre :
//
//	IDENTITÉ  combien de records d'arme au sol (ti=42) des keyframes portent une famille d'arme
//	          reconnue (l'ancrage anti-hasard annonce 397 occurrences sur 000d5950) ;
//	POSITION  combien de ces lectures peuvent être POSÉES sur la carte, c'est-à-dire appariées à
//	          une position monde décodée des paquets delta.
//
// TÉMOIN DE CONTRÔLE OBLIGATOIRE sur la position : une bande de slots FANTÔME de même
// cardinalité (des slots jamais vus porter ti=42 dans aucun keyframe) passe par le MÊME
// décodeur. Si le fantôme rend autant d'échantillons que la vraie bande, le signal est sous le
// bruit — c'est exactement le verdict consigné le 2026-07-28 dans .ai/SUIVI_REPLAY_2D.md
// (« 5 échantillons contre 1 006 sur un jeu de slots fantôme »), que cet instrument doit
// reproduire ou réfuter sur pièces.
//
// IL NE MODIFIE RIEN : ni le décodeur, ni l'assemblage, ni l'artefact. Il lit le film en LECTURE
// SEULE et n'écrit dans aucun répertoire de données. SOUS GARDE D'ENVIRONNEMENT (GW_FILM), donc
// sauté partout ailleurs, CI comprise.
//
// VERDICT RENDU LE 2026-08-12 sur 000d5950 — c'est la RAISON D'ÊTRE de ce fichier : garder la
// réfutation REJOUABLE plutôt que de la réduire à une phrase de documentation.
//
//	IDENTITÉ ACQUISE  26 images-clés · 269 records ti=42 · 196 lectures portant une famille connue
//	                  · 169 vies (slot, gen) · 22 armes nommées · 0 fuite d'alias sur 172 paires.
//	POSITION RÉFUTÉE  témoin fantôme (899 slots jamais vus en image-clé) -> 493 slots peuplés /
//	                  10 950 échantillons contre 661 / 44 498 pour la vraie bande ; dispersion :
//	                  3,3 % seulement des 458 slots réels à >=3 échantillons tiennent dans 0,5 u,
//	                  62,4 % s'étalent au-delà de 20 u (fantôme 0,5 % / 82,4 %).
//
// Conséquence : le calque cartographique « armes au sol » est HORS DE PORTÉE offline-pur. Phase 2
// spatiale abandonnée pour v7.5, reportée avec sa condition de reprise (default-state ti=42 résolu,
// OU événements de cycle de vie décodés) — .ai/V7.5/REGISTRE_REPORTS.md. Ce verdict CONFIRME celui
// du 2026-07-28 (.ai/SUIVI_REPLAY_2D.md) que le plan de Phase 2 avait contredit.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 GW_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  GW_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  GW_MAP=Cliffhanger \
//	  go test ./internal/analysis/replay/ -run GroundWeaponCoverage -timeout 30m -v

import (
	"fmt"
	"math"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

const (
	gwFilmEnv   = "GW_FILM"   // répertoire des chunks du film
	gwBoundsEnv = "GW_BOUNDS" // catalogue map_quant_bounds.json
	gwMapEnv    = "GW_MAP"    // nom de carte du film
)

// gwGapBuckets borne l'écart temporel entre une lecture d'arme au sol (instant du keyframe) et
// l'échantillon de position le plus proche du même slot. Le premier seuil est la tolérance de
// rattachement des tirs (120 ms) ; les suivants séparent « la réplication est clairsemée » de
// « cet objet n'a aucune trace ici ». Un keyframe tombe toutes les ~20 s : au-delà, l'appariement
// n'a plus de sens.
var gwGapBuckets = []uint64{120_000, 1_000_000, 5_000_000, 20_000_000}

func TestGroundWeaponCoverage(t *testing.T) {
	filmDir := os.Getenv(gwFilmEnv)
	if filmDir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", gwFilmEnv)
	}
	known := loadoutFamilies()

	// --- IDENTITÉ -----------------------------------------------------------------------
	ground, err := filmdec.ScanFilmKeyframeGroundWeapons(filmDir, known)
	if err != nil {
		t.Fatalf("balayage des armes au sol : %v", err)
	}
	census := gwKeyframeCensus(t, filmDir)
	occ, named := 0, map[string]int{}
	slots, lives := map[uint32]bool{}, map[[2]uint32]bool{}
	for _, g := range ground {
		occ += len(g.Families)
		slots[g.Slot] = true
		lives[[2]uint32{g.Slot, g.Gen}] = true
		for _, f := range g.Families {
			if n := weaponv3.WeaponName(f); n != "" {
				named[n]++
			}
		}
	}
	t.Logf("IDENTITÉ — keyframes %d · records ti=42 %d (slots distincts %d) · lectures avec famille %d"+
		" · occurrences de famille %d · vies (slot,gen) %d · armes nommées distinctes %d",
		census.keyframes, census.recordsTI[filmdec.GroundWeaponTypeIndex],
		len(census.slotsTI[filmdec.GroundWeaponTypeIndex]), len(ground), occ, len(lives), len(named))
	for _, n := range gwTopNames(named, 12) {
		t.Logf("    %-34s %d", n.name, n.count)
	}
	// TÉMOIN DE COMPARABILITÉ : le MÊME balayage sur le MÊME film, côté armes PORTÉES (ti=35).
	// L'en-tête de keyframe_loadout.go annonce 495 occurrences biped et 397 au sol ; si le compte
	// biped mesuré ici retombe lui aussi à ~la moitié de 495, l'écart vient de la RÈGLE DE COMPTE
	// de la mesure d'origine (occurrences brutes, alias compris), pas d'occurrences manquées.
	if worn, err := filmdec.ScanFilmKeyframeLoadouts(filmDir, known); err == nil {
		wornOcc := 0
		for _, l := range worn {
			wornOcc += len(l.Families)
		}
		t.Logf("COMPARABILITÉ — armes portées (ti=35) : %d lectures / %d occurrences de famille"+
			" (en-tête du décodeur : 495) ; armes au sol : %d / %d (en-tête : 397)",
			len(worn), wornOcc, len(ground), occ)
	}

	// --- POSITION -----------------------------------------------------------------------
	wr, ok := gwWorldRange(t)
	if !ok {
		t.Logf("POSITION — bornes de carte absentes (%s/%s) : mesure d'identité seule", gwBoundsEnv, gwMapEnv)
		return
	}
	band := filmdec.GroundWeaponSlotBand(filmDir)
	real := filmdec.WorldObjectPositionsForBand(filmDir, &wr, band)
	phantom := filmdec.WorldObjectPositionsForBand(filmDir, &wr, gwPhantomBand(band, census.allSlots))
	t.Logf("POSITION — bande ti=42 %d slots -> %d slots peuplés / %d échantillons"+
		" || TÉMOIN FANTÔME %d slots -> %d slots peuplés / %d échantillons",
		len(band), len(real), gwTotalSamples(real),
		len(gwPhantomBand(band, census.allSlots)), len(phantom), gwTotalSamples(phantom))

	// Appariement lecture -> position : la seule grandeur qui décide du calque.
	hist := make([]int, len(gwGapBuckets)+1)
	none := 0
	for _, g := range ground {
		pts := real[g.Slot]
		_, gap, found := filmdec.NearestWorldObjectSample(pts, g.TimestampUS)
		if !found {
			none++
			continue
		}
		b := len(gwGapBuckets)
		for i, lim := range gwGapBuckets {
			if gap <= lim {
				b = i
				break
			}
		}
		hist[b]++
	}
	t.Logf("APPARIEMENT — lectures %d · sans aucune position du slot %d · <=120ms %d · <=1s %d"+
		" · <=5s %d · <=20s %d · >20s %d",
		len(ground), none, hist[0], hist[1], hist[2], hist[3], hist[4])

	// DISCRIMINANT SPATIAL — le seul qui puisse départager signal et bruit slot par slot. Une
	// arme posée NE BOUGE PAS : ses échantillons doivent former un point. Un slot de bruit tire
	// des positions au hasard dans l'AABB de la carte, donc s'étale. On compare la DISPERSION
	// (diagonale de la boîte englobante des échantillons d'un slot) entre la vraie bande et la
	// bande fantôme : sans écart de distribution, aucune position n'est publiable.
	t.Logf("DISPERSION vraie bande   — %s", gwDispersion(real))
	t.Logf("DISPERSION bande fantôme — %s", gwDispersion(phantom))
}

// TestGroundWeaponAliasLeak mesure la FUITE D'ALIAS entre records d'armes au sol voisins : le
// même canon apparaît deux fois dans le bitstream sous deux identifiants de famille (le second à
// +97 bits, cf. keyframe_loadout.go). Les records d'arme au sol sont COURTS : si l'alias tombe
// au-delà du début du record suivant, il est attribué à CE record — et une arme au sol serait
// alors nommée d'après sa voisine. Signature mesurable : deux lectures consécutives du même
// keyframe portant le même NOM canonique sous deux familles DIFFÉRENTES.
func TestGroundWeaponAliasLeak(t *testing.T) {
	filmDir := os.Getenv(gwFilmEnv)
	if filmDir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", gwFilmEnv)
	}
	ground, err := filmdec.ScanFilmKeyframeGroundWeapons(filmDir, loadoutFamilies())
	if err != nil {
		t.Fatalf("balayage des armes au sol : %v", err)
	}
	suspects, sameFamily, pairs := 0, 0, 0
	for i := 1; i < len(ground); i++ {
		a, b := ground[i-1], ground[i]
		if a.TimestampUS != b.TimestampUS || len(a.Families) == 0 || len(b.Families) == 0 {
			continue
		}
		pairs++
		na, nb := weaponv3.WeaponName(a.Families[0]), weaponv3.WeaponName(b.Families[0])
		if na == "" || na != nb {
			continue
		}
		if a.Families[0] == b.Families[0] {
			sameFamily++ // deux armes du même modèle au sol : légitime
			continue
		}
		suspects++ // même nom, familles différentes : signature de l'alias du voisin
	}
	t.Logf("ALIAS — paires consécutives d'un même keyframe %d · même nom & MÊME famille %d"+
		" (deux armes identiques au sol, légitime) · même nom & famille DIFFÉRENTE %d"+
		" (signature de fuite d'alias)", pairs, sameFamily, suspects)
}

// gwDispersion résume la dispersion spatiale par slot : combien de slots ont au moins 3
// échantillons, et parmi eux la répartition de la diagonale de leur boîte englobante. Les seuils
// sont en unités monde (~mètres) : 0,5 = un objet immobile au quantum près, 5 = déjà une pièce
// entière, 20+ = la carte.
func gwDispersion(m map[uint32][]filmdec.WorldObjectSample) string {
	buckets := []float32{0.5, 2, 5, 20}
	counts := make([]int, len(buckets)+1)
	eligible := 0
	for _, pts := range m {
		if len(pts) < 3 {
			continue
		}
		eligible++
		d := gwBoxDiagonal(pts)
		b := len(buckets)
		for i, lim := range buckets {
			if d <= lim {
				b = i
				break
			}
		}
		counts[b]++
	}
	return fmtGwDispersion(eligible, counts)
}

func fmtGwDispersion(eligible int, counts []int) string {
	pct := func(n int) float64 {
		if eligible == 0 {
			return 0
		}
		return 100 * float64(n) / float64(eligible)
	}
	return fmt.Sprintf("slots >=3 échantillons %d · <=0,5u %d (%.1f %%) · <=2u %d · <=5u %d"+
		" · <=20u %d · >20u %d (%.1f %%)",
		eligible, counts[0], pct(counts[0]), counts[1], counts[2], counts[3], counts[4], pct(counts[4]))
}

// gwBoxDiagonal rend la diagonale de la boîte englobante des échantillons d'un slot.
func gwBoxDiagonal(pts []filmdec.WorldObjectSample) float32 {
	mn, mx := [3]float32{pts[0].X, pts[0].Y, pts[0].Z}, [3]float32{pts[0].X, pts[0].Y, pts[0].Z}
	for _, p := range pts[1:] {
		for a, v := range [3]float32{p.X, p.Y, p.Z} {
			if v < mn[a] {
				mn[a] = v
			}
			if v > mx[a] {
				mx[a] = v
			}
		}
	}
	var sum float64
	for a := 0; a < 3; a++ {
		d := float64(mx[a] - mn[a])
		sum += d * d
	}
	return float32(math.Sqrt(sum))
}

// gwCensus est le recensement des archétypes du monde de keyframe.
type gwCensus struct {
	keyframes int
	recordsTI map[int]int
	slotsTI   map[int]map[uint32]bool
	allSlots  map[uint32]bool // tout slot vu dans un keyframe, quel que soit l'archétype
}

// gwKeyframeCensus recense les records de keyframe par archétype. Il réutilise le walker déjà
// validé (249/250 entités) plutôt que de relire la table à la main.
func gwKeyframeCensus(t *testing.T, dir string) gwCensus {
	c := gwCensus{recordsTI: map[int]int{}, slotsTI: map[int]map[uint32]bool{}, allSlots: map[uint32]bool{}}
	n := filmdec.CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	for i := 1; i <= n; i++ {
		data, err := filmdec.ReadFilmChunk(dir, i)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			c.keyframes++
			for _, r := range filmdec.WalkKeyframeWorld(p.Payload(data)) {
				c.recordsTI[r.TI]++
				if c.slotsTI[r.TI] == nil {
					c.slotsTI[r.TI] = map[uint32]bool{}
				}
				c.slotsTI[r.TI][uint32(r.Slot)] = true
				c.allSlots[uint32(r.Slot)] = true
			}
		}
	}
	return c
}

// gwPhantomBand construit une bande de contrôle de MÊME CARDINALITÉ que band, faite de slots
// JAMAIS vus dans un keyframe — donc ne portant aucun objet du monde. Tout échantillon qu'elle
// rend est du bruit par construction.
func gwPhantomBand(band, seen map[uint32]bool) map[uint32]bool {
	out := make(map[uint32]bool, len(band))
	for s := uint32(1); len(out) < len(band) && s < 8192; s++ {
		if band[s] || seen[s] {
			continue
		}
		out[s] = true
	}
	return out
}

func gwTotalSamples(m map[uint32][]filmdec.WorldObjectSample) int {
	n := 0
	for _, v := range m {
		n += len(v)
	}
	return n
}

type gwNameCount struct {
	name  string
	count int
}

// gwTopNames rend les armes les plus fréquentes, par compte décroissant puis par nom (ordre
// TOTAL : une map d'itération aléatoire ferait varier le rapport d'une exécution à l'autre).
func gwTopNames(named map[string]int, max int) []gwNameCount {
	out := make([]gwNameCount, 0, len(named))
	for n, c := range named {
		out = append(out, gwNameCount{n, c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].name < out[j].name
	})
	if len(out) > max {
		out = out[:max]
	}
	return out
}

// gwWorldRange lit les bornes de la carte dans le catalogue versionné. Absentes, la mesure de
// position est simplement sautée : un quantum sans bornes n'est pas une position.
func gwWorldRange(t *testing.T) (filmdec.Vec3Range, bool) {
	boundsPath, mapName := os.Getenv(gwBoundsEnv), os.Getenv(gwMapEnv)
	if boundsPath == "" || mapName == "" {
		return filmdec.Vec3Range{}, false
	}
	cat, err := filmdec.LoadMapQuantCatalog(boundsPath)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", boundsPath, err)
	}
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("carte %s absente du catalogue : %v", mapName, err)
	}
	filmdec.SetWorldObjectPrecisionFromLayout(filmdec.I0Layout{AxisW: [3]uint{
		uint(entry.AxisWidths[0]), uint(entry.AxisWidths[1]), uint(entry.AxisWidths[2])}})
	return entry.Range(), true
}
