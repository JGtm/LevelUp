package main

// player.go — ETAPE A DU VERDICT : la deconvolution AU GRAIN JOUEUR.
//
// POURQUOI CHANGER DE GRAIN. Au grain du match, le systeme n est pas identifiable : les
// melanges d armes se ressemblent trop d un match a l autre, et les armes de faible volume
// sortent des taux negatifs. Le grain joueur multiplie les observations par ~10 ET decorrele
// les melanges — un joueur au Needler dans un match ou un autre joue au BR75 donne deux
// lignes que le grain match confondait en une.
//
// CE QUE ÇA COÛTE : l appariement indice de film -> xuid, qui n etait pas necessaire au grain
// match. Il se fait par `weaponv3.ResolveXuidToPI` (motif xuid 8 octets LE relu en BE, cherche
// au bit pres, 5 bits AVANT le motif) — la meme regle que `resolveFilmPlayerIndices` en
// production.
//
// ET IL APPORTE LA REFERENCE QUI MANQUAIT : une fois les joueurs nommes, la population a arme
// dominante >= 80 % redonne la reference API PAR ARME (celle de GUIDE_WEAPON_SHOTS §3bis.1,
// dont seuls le MA40 et le Sidekick sont cites en clair). Sans elle, le controle positif de la
// deconvolution n a que deux points.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// playerRow est une observation (match x joueur) du systeme.
type playerRow struct {
	pfx      string
	famille  string
	xuid     string
	apiFired float64
	apiHits  float64
	shots    map[uint64]float64 // records type 105 de ce joueur, par arme
	carriers map[uint64]float64
}

// scanWithPI lit les records d un film ET resout l appariement indice -> xuid dans la meme
// passe : un chunk decompresse sert aux deux, il n est jamais garde apres usage.
func scanWithPI(dir string, roster []uint64) ([]rec, map[int]string) {
	n := filmdec.CountFilmChunks(dir)
	var out []rec
	xuidByPI := map[int]string{}
	seen := map[uint64]bool{}
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		// appariement : premiere occurrence gagnante, comme ResolveBest
		var pending []uint64
		for _, x := range roster {
			if !seen[x] {
				pending = append(pending, x)
			}
		}
		if len(pending) > 0 {
			for x, pi := range resolvePIFast(chunk, pending) {
				seen[x] = true
				if _, taken := xuidByPI[pi]; !taken {
					xuidByPI[pi] = strconv.FormatUint(x, 10)
				}
			}
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != recType || len(pay)*8 < minBitsWeapon {
				continue
			}
			r := rec{
				shooter: int(readBits(pay, bitShooter, widthShoot)),
				long:    pay[0]&1 == 0,
				tsUS:    p.TimestampUS,
			}
			if !r.long {
				continue
			}
			r.weapon = readBits(pay, bitWeaponHi, widthWeapon)<<32 | readBits(pay, bitWeaponLo, widthWeapon)
			r.hasWeap = true
			r.porteur = readBits(pay, bitCountersNull, 1) == 0
			out = append(out, r)
		}
	}
	return out, xuidByPI
}

// buildPlayerRows construit les observations (match x joueur) sur les films retenus.
func buildPlayerRows(root string, pfxs []string, refs map[string]*matchRef, normalize bool) []playerRow {
	var rows []playerRow
	var filmsOK, joueursApparies, joueursTotal int
	for _, pfx := range pfxs {
		m := refs[pfx]
		var roster []uint64
		api := map[string]apiRow{}
		for _, p := range m.players {
			joueursTotal++
			x, err := strconv.ParseUint(p.xuid, 10, 64)
			if err != nil {
				continue // bot ou xuid non numerique
			}
			roster = append(roster, x)
			api[p.xuid] = p
		}
		if len(roster) == 0 {
			continue
		}
		recs, xuidByPI := scanWithPI(filepath.Join(root, pfx), roster)
		if len(recs) == 0 {
			continue
		}
		filmsOK++
		byXuid := map[string]*playerRow{}
		for _, r := range recs {
			if !isFire(r) {
				continue
			}
			xu, ok := xuidByPI[r.shooter]
			if !ok {
				continue // indice non rattache au roster : la ligne est jetee, jamais devinee
			}
			pr := byXuid[xu]
			if pr == nil {
				a := api[xu]
				pr = &playerRow{pfx: pfx, famille: familyOf(m.pairName), xuid: xu,
					apiFired: float64(a.shotsFired), apiHits: float64(a.shotsHit),
					shots: map[uint64]float64{}, carriers: map[uint64]float64{}}
				byXuid[xu] = pr
			}
			pr.shots[r.weapon]++
			if r.porteur {
				pr.carriers[r.weapon]++
			}
		}
		for _, pr := range byXuid {
			if pr.apiFired <= 0 || pr.apiHits <= 0 {
				continue
			}
			joueursApparies++
			// NORMALISATION DE VISIBILITE, au grain JOUEUR cette fois. Le film ne montre
			// qu une FRACTION des tirs, et elle varie de 0,31 (Fiesta) a 0,92 (Tactical) :
			// sans correction, cette fraction entre dans les coefficients, qui absorbent
			// 1/visibilite et saturent a 1. Mesure a l appui — sans elle, MA40 +0,1595 et
			// BR75 +0,2096 contre leur reference, toutes les armes rares butant sur la borne.
			if normalize {
				var tot float64
				for _, v := range pr.shots {
					tot += v
				}
				if tot <= 0 {
					continue
				}
				k := pr.apiFired / tot
				for id := range pr.shots {
					pr.shots[id] *= k
				}
			}
			rows = append(rows, *pr)
		}
	}
	fmt.Fprintf(os.Stderr, "films exploitables: %d — joueurs apparies: %d / %d\n",
		filmsOK, joueursApparies, joueursTotal)
	return rows
}

// referenceParArme rend la reference API par arme, mesuree sur la population a ARME DOMINANTE.
// C est la methode de GUIDE_WEAPON_SHOTS §3bis.1 : la precision API d un joueur dont une seule
// arme porte >= purete de ses tirs decodes EST la precision de cette arme, au biais de
// contamination pres. On publie l effectif avec le chiffre, toujours.
func referenceParArme(rows []playerRow, purete float64, minShots float64) map[uint64][3]float64 {
	agg := map[uint64][3]float64{} // tirs API, touches API, effectif
	for _, r := range rows {
		var tot, best float64
		var bestID uint64
		for id, v := range r.shots {
			tot += v
			if v > best {
				best, bestID = v, id
			}
		}
		if tot < minShots || best/tot < purete {
			continue
		}
		a := agg[bestID]
		agg[bestID] = [3]float64{a[0] + r.apiFired, a[1] + r.apiHits, a[2] + 1}
	}
	return agg
}

// runPlayer execute l etape A : reference par arme, puis deconvolution bornee hors echantillon.
func runPlayer(root string, pfxs []string, refs map[string]*matchRef, names map[uint64]string,
	minShots float64, purete float64, outCSV string, normalize bool) {
	rows := buildPlayerRows(root, pfxs, refs, normalize)
	if len(rows) == 0 {
		fmt.Fprintln(os.Stderr, "aucune observation joueur")
		return
	}

	ref := referenceParArme(rows, purete, 50)
	fmt.Printf("REFERENCE API PAR ARME — population a arme dominante >= %.0f %%\n", purete*100)
	fmt.Printf("%-22s %9s %11s %11s\n", "arme", "joueurs", "tirs_API", "precision_API")
	type refLine struct {
		id                    uint64
		n, fired, hits, preci float64
	}
	var refs2 []refLine
	for id, a := range ref {
		if a[2] < 20 {
			continue
		}
		refs2 = append(refs2, refLine{id, a[2], a[0], a[1], a[1] / a[0]})
	}
	sort.Slice(refs2, func(i, j int) bool { return refs2[i].n > refs2[j].n })
	for _, l := range refs2 {
		fmt.Printf("%-22s %9.0f %11.0f %11.4f\n", nameOf(names, l.id), l.n, l.fired, l.preci)
	}

	// Deconvolution bornee, grain joueur, moities hors echantillon.
	ids := retainedWeaponsPlayer(rows, minShots)
	var a, b []playerRow
	for i, r := range rows {
		if i%2 == 0 {
			a = append(a, r)
		} else {
			b = append(b, r)
		}
	}
	pAll := boundedFit(rows, ids, 300000)
	pA := boundedFit(a, ids, 300000)
	pB := boundedFit(b, ids, 300000)

	fmt.Printf("\nDECONVOLUTION BORNEE [0,1] — grain JOUEUR, %d observations, %d armes\n", len(rows), len(ids))
	fmt.Printf("%-22s %10s %10s %10s %12s %9s\n", "arme", "coef_tot", "coef_A", "coef_B", "ref_API", "ecart")
	w, closeW := openCSV(outCSV, []string{"weapon_id", "arme", "coef_total", "coef_A", "coef_B", "ref_api", "ecart", "joueurs_ref"})
	defer closeW()
	for i, id := range ids {
		var refv, refn float64
		if r, ok := ref[id]; ok && r[0] > 0 {
			refv, refn = r[1]/r[0], r[2]
		}
		ecart := ""
		if refv > 0 && refn >= 20 {
			ecart = fmt.Sprintf("%+.4f", pAll[i]-refv)
		}
		fmt.Printf("%-22s %10.4f %10.4f %10.4f %12s %9s\n", nameOf(names, id),
			pAll[i], pA[i], pB[i], fmtRef(refv, refn), ecart)
		writeRow(w, strconv.FormatUint(id, 10), nameOf(names, id),
			fmt.Sprintf("%.4f", pAll[i]), fmt.Sprintf("%.4f", pA[i]), fmt.Sprintf("%.4f", pB[i]),
			fmt.Sprintf("%.4f", refv), ecart, fmt.Sprintf("%.0f", refn))
	}
}

func fmtRef(v, n float64) string {
	if v <= 0 || n < 20 {
		return "-"
	}
	return fmt.Sprintf("%.4f", v)
}

func nameOf(names map[uint64]string, id uint64) string {
	if n := names[id]; n != "" {
		return n
	}
	return fmt.Sprintf("0x%016x", id)
}

func retainedWeaponsPlayer(rows []playerRow, minShots float64) []uint64 {
	tot := map[uint64]float64{}
	for _, r := range rows {
		for id, v := range r.shots {
			tot[id] += v
		}
	}
	var ids []uint64
	for id, v := range tot {
		if v >= minShots {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// resolvePIFast rend le MEME resultat que weaponv3.ResolveXuidToPI, en O(taille du chunk)
// au lieu d un balayage bit a bit par xuid.
//
// POURQUOI IL A FALLU LE REECRIRE : le resolveur du depot relit 64 bits a CHAQUE position de
// bit et pour CHAQUE xuid — sur 60 films il ne finit pas en 10 minutes. Or le motif cherche
// est un motif d OCTETS a un decalage de bit inconnu : il suffit donc de construire les 8
// versions decalees du chunk et d y chercher les 8 octets par `bytes.Index`. Meme reponse,
// trois ordres de grandeur plus vite.
//
// Le motif est l ecriture LITTLE-ENDIAN du xuid, et les 5 bits qui PRECEDENT portent l indice
// (weaponv3.PIBits) — les deux regles viennent de `pi_resolver.go`, elles ne sont pas
// redecouvertes ici.
func resolvePIFast(chunk []byte, roster []uint64) map[uint64]int {
	out := map[uint64]int{}
	if len(chunk) < 9 {
		return out
	}
	needles := make(map[uint64][]byte, len(roster))
	for _, x := range roster {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, x)
		needles[x] = b
	}
	// LE PIEGE, ET IL A ETE PRIS AU GATE : le resolveur du depot retient la PREMIERE
	// occurrence EN POSITION DE BIT. Balayer decalage par decalage retient la premiere de
	// CHAQUE decalage, ce qui n est pas la meme occurrence — mesure : 277 accords contre
	// 299 desaccords. Il faut donc collecter le minimum sur les huit decalages.
	best := make(map[uint64]int, len(roster))
	shifted := make([]byte, len(chunk)-1)
	for s := 0; s < 8; s++ {
		var buf []byte
		if s == 0 {
			buf = chunk
		} else {
			for i := 0; i < len(chunk)-1; i++ {
				shifted[i] = chunk[i]<<uint(s) | chunk[i+1]>>uint(8-s)
			}
			buf = shifted
		}
		for x, needle := range needles {
			idx := bytes.Index(buf, needle)
			if idx < 0 {
				continue
			}
			bitPos := idx*8 + s
			if cur, ok := best[x]; !ok || bitPos < cur {
				best[x] = bitPos
			}
		}
	}
	for x, bitPos := range best {
		if bitPos < weaponv3.PIBits {
			continue
		}
		out[x] = int(readBits(chunk, bitPos-weaponv3.PIBits, weaponv3.PIBits))
	}
	return out
}

// runPIGate confronte `resolvePIFast` au resolveur DU DEPOT, chunk par chunk, sur quelques
// films. Un resolveur reecrit pour la vitesse ne vaut que s il rend la MEME reponse : la regle
// du chantier est qu on ne remplace pas un instrument sans le confronter a celui qu il remplace.
func runPIGate(root string, pfxs []string, refs map[string]*matchRef) {
	var accord, desaccord, absentRapide, absentDepot int
	for _, pfx := range pfxs {
		m := refs[pfx]
		var roster []uint64
		for _, p := range m.players {
			if x, err := strconv.ParseUint(p.xuid, 10, 64); err == nil {
				roster = append(roster, x)
			}
		}
		if len(roster) == 0 {
			continue
		}
		n := filmdec.CountFilmChunks(root)
		_ = n
		for c := 1; c <= filmdec.CountFilmChunks(filepath.Join(root, pfx)); c++ {
			chunk, err := filmdec.ReadFilmChunk(filepath.Join(root, pfx), c)
			if err != nil {
				continue
			}
			ref := weaponv3.ResolveXuidToPI(roster, chunk)
			fast := resolvePIFast(chunk, roster)
			for x, pi := range ref {
				if f, ok := fast[x]; !ok {
					absentRapide++
				} else if f == pi {
					accord++
				} else {
					desaccord++
				}
			}
			for x := range fast {
				if _, ok := ref[x]; !ok {
					absentDepot++
				}
			}
		}
	}
	fmt.Printf("GATE resolveur : accord=%d  desaccord=%d  absent du rapide=%d  absent du depot=%d\n",
		accord, desaccord, absentRapide, absentDepot)
}
