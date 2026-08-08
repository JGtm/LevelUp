// Commande de recherche (jetable, gitignoree) : LES PROJECTILES D'ARME SONT-ILS DES ENTITES
// `ti=41` ?
//
// LA QUESTION, ET POURQUOI ELLE DECIDE D'UNE VOIE ENTIERE. La derniere voie non essayee vers
// le NUMERATEUR de la precision (les touches par arme) est : lire la trajectoire du projectile
// et regarder ou elle finit. Elle repose sur une prémisse jamais verifiee — que le tir d'une
// arme a projectile CREE une entite `ti=41`. Les 70 lancers de grenade sur 70 de
// `filmdec/projectiles.go` ne prouvent que le cas des GRENADES.
//
// LE TEST. Pour chaque record de TIR (type 105, avec identifiant d'arme), on demande : une
// naissance d'entite `ti=41` survient-elle dans la fenetre qui suit ? Puis on ventile par arme.
//
// LE CONTROLE POSITIF EST DANS LA FORME MEME DU RESULTAT, et c'est ce qui rend le chiffre
// interpretable : les armes a TRACE (BR75, MA40, Sidekick, Sniper) ne creent aucun projectile
// et doivent tomber au niveau de la nulle ; les armes a PROJECTILE (Needler, Plasma, Ravager,
// Mangler, Cindershot, Hydra, Rocket) doivent s'en detacher. Si tout le monde est au meme
// niveau, l'instrument ne lit rien — et ce sera un negatif sur l'instrument, pas sur la
// question.
//
// LA NULLE : les memes naissances, decalees dans le temps d'un offset circulaire. Elle mesure
// le taux de coincidence fortuite — avec jusqu'a 32 joueurs qui tirent, il n'est pas petit.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	projectileTI = 41
	// recType, bitShooter, bitWeapon* : la grammaire du record de tir, reprise de tmp_pjcnt.
	recType     = 105
	bitShooter  = 35
	widthShoot  = 5
	bitWeaponHi = 44
	bitWeaponLo = 76
	widthWeapon = 32
	minBitsHead = 40
	minBitsWeap = 108
	// commonWeaponSuffix est la moitie basse partagee par 95 % des identifiants d'arme.
	commonWeaponSuffix = 0x42c9679f
	// lowPosBits / hiPosBits : les deux longueurs d'`object-position-component`.
	lowPosBits = 45
	hiPosBits  = 60
	lifeGapUS  = 250_000
)

type fire struct {
	tsUS    uint64
	weapon  uint64
	shooter int
}

type birth struct {
	tsUS uint64
	slot uint32
}

func main() {
	films := flag.String("films", "", "racine du cache de films (LECTURE SEULE)")
	limit := flag.Int("limit", 20, "nombre maximum de films")
	weapCSV := flag.String("armes", "", "CSV weapon_id,name_en exporte de metadata.duckdb")
	// 200 ms est la fenetre de PRODUCTION (`grenadeBirthWindowUS`), symetrique, et elle est
	// mesuree : 65 des 70 lancers y apparient, contre 11 a 13 pour les memes lancers decales
	// en bloc. Ne pas la choisir soi-meme.
	windowMS := flag.Int("fenetre", 200, "demi-fenetre en ms autour du tir (production : 200)")
	shiftMS := flag.Int("decalage", 5000, "decalage des naissances pour la nulle")
	minShots := flag.Int("mintirs", 300, "tirs minimum pour publier une arme")
	catalog := flag.String("catalogue", "", "chemin de map_quant_bounds.json")
	mapsCSV := flag.String("cartes", "", "CSV matchID,carte")
	flag.Parse()

	if *films == "" || *catalog == "" || *mapsCSV == "" {
		fmt.Fprintln(os.Stderr, "usage: tmp_projorig -films <dir> -catalogue <json> -cartes <csv> [-armes weapons.csv]")
		os.Exit(2)
	}
	names := loadWeaponNames(*weapCSV)

	cat, err := filmdec.LoadMapQuantCatalog(*catalog)
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalogue:", err)
		os.Exit(1)
	}
	dirs, ranges, err := listFilmsWithBounds(*films, *mapsCSV, cat, *limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "films:", err)
		os.Exit(1)
	}

	// tally[arme] = [tirs, tirs suivis d'une naissance, idem sur la nulle]
	tally := map[uint64]*[3]int{}
	var totFires, totBirths int

	for di, d := range dirs {
		fires := scanFires(d)
		// Les lancers de grenade entrent dans la MEME population, sous une cle synthetique :
		// ils sont le controle positif de l'instrument.
		fires = append(fires, scanThrows(d)...)
		wr := ranges[di]
		births := scanBirthsProd(d, &wr)
		totFires += len(fires)
		totBirths += len(births)
		if len(fires) == 0 || len(births) == 0 {
			continue
		}
		ts := make([]uint64, len(births))
		for i, b := range births {
			ts[i] = b.tsUS
		}
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })

		// La nulle decale toutes les naissances du meme offset : elle conserve exactement la
		// densite temporelle des naissances, et ne casse que leur lien aux tirs.
		shifted := make([]uint64, len(ts))
		for i, t := range ts {
			shifted[i] = t + uint64(*shiftMS)*1000
		}

		win := uint64(*windowMS) * 1000
		for _, f := range fires {
			t := tally[f.weapon]
			if t == nil {
				t = &[3]int{}
				tally[f.weapon] = t
			}
			t[0]++
			if hasBirthIn(ts, f.tsUS, win) {
				t[1]++
			}
			if hasBirthIn(shifted, f.tsUS, win) {
				t[2]++
			}
		}
	}

	fmt.Printf("%d films : %d records de tir, %d naissances d'entite ti=41\n", len(dirs), totFires, totBirths)
	fmt.Printf("fenetre %d ms apres le tir ; nulle = naissances decalees de %d ms\n\n", *windowMS, *shiftMS)
	report(tally, names, *minShots)
}

// hasBirthIn dit si une naissance tombe dans [t-win, t+win] — la fenetre SYMETRIQUE de
// `birthNear` en production. ts est TRIE.
func hasBirthIn(ts []uint64, t, win uint64) bool {
	i := sort.Search(len(ts), func(k int) bool { return ts[k] >= t })
	for _, k := range []int{i - 1, i} {
		if k < 0 || k >= len(ts) {
			continue
		}
		d := ts[k] - t
		if ts[k] < t {
			d = t - ts[k]
		}
		if d <= win {
			return true
		}
	}
	return false
}

func report(tally map[uint64]*[3]int, names map[uint64]string, minShots int) {
	type row struct {
		id        uint64
		name      string
		shots     int
		hit, null float64
		lift      float64
	}
	var rows []row
	for id, t := range tally {
		if t[0] < minShots {
			continue
		}
		h := float64(t[1]) / float64(t[0])
		n := float64(t[2]) / float64(t[0])
		rows = append(rows, row{id, weaponName(names, id), t[0], h, n, h - n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].lift > rows[j].lift })

	fmt.Printf("%-34s %8s %8s %8s %9s\n", "arme", "tirs", "suivi", "nulle", "ecart")
	fmt.Println("------------------------------------------------------------------------")
	for _, r := range rows {
		fmt.Printf("%-34s %8d %8.4f %8.4f %+9.4f\n", r.name, r.shots, r.hit, r.null, r.lift)
	}
	fmt.Println()
	fmt.Println("LECTURE : `ecart` = taux observe moins taux de coincidence fortuite. Les armes a")
	fmt.Println("TRACE doivent etre a ~0. Si les armes a PROJECTILE ne s'en detachent pas, le tir")
	fmt.Println("d'arme ne cree pas d'entite ti=41 — et la voie de la trajectoire est morte pour")
	fmt.Println("la precision. Si TOUT est a ~0, c'est l'instrument qui ne lit rien.")
}

func weaponName(names map[uint64]string, id uint64) string {
	if id == grenadeWeaponID {
		return ">> LANCER DE GRENADE (controle positif)"
	}
	if n, ok := names[id]; ok && n != "" {
		return n
	}
	if n, ok := names[id>>32]; ok && n != "" {
		return n
	}
	return fmt.Sprintf("id %#x", id)
}
