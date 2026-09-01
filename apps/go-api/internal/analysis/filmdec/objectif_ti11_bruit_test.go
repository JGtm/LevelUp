package filmdec

// objectif_ti11_bruit_test.go — ti=11 i12 / i13 / i14 SANS LE FILTRE `Chained` : CHERCHER UN
// EFFET STATISTIQUE SOUS UN BRUIT ASSUME.
//
// # LA QUESTION, ET EN QUOI ELLE DIFFERE DE TOUS LES LOTS PRECEDENTS
//
// Les instruments qui precedent demandaient QUELLE VALEUR porte la jauge d'armement. Tous ont
// bute sur le meme mur : la voie DELTA de ti=11 est du bruit d'ancrage — l'oracle de legalite de
// `i0` la mesure a 45,7 % la ou le hasard donne 53,1 % — et le filtre `Chained`, seul garde
// disponible, ne laisse passer que 1 a 35 lectures par film, ce qui interdit toute serie.
//
// Ce lot pose une question strictement plus faible, et c'est ce qui la rend repondable :
//
//	NON PAS « quelle valeur »  MAIS  « y a-t-il PLUS de lectures juste avant une explosion
//	                                  qu'ailleurs dans le meme film ? »
//
// Un ancrage bruite ne date rien, mais il n'efface pas forcement le TRAFIC : si le mode ecrit
// davantage sur ses objectifs pendant l'armement, la DENSITE des records ti=11 monte avant
// l'explosion, meme quand leur contenu reste illisible. C'est un effet de COMPTAGE, pas de
// lecture — et un comptage survit a une largeur fausse.
//
// # CE QUE L'INSTRUMENT MESURE, EXACTEMENT
//
// Pour chaque lecture datee `t` d'un film d'Assaut, le DELAI jusqu'a l'explosion SUIVANTE du
// meme film (oracle `ti12Explosions`, recopie de `a5Explosions` : 28 explosions, 9 films).
// L'histogramme de ces delais est confronte a DEUX TEMOINS construits sur le MEME film, avec le
// MEME nombre de tirages :
//
//	TEMOIN A (prescrit par la mission)  instants UNIFORMES sur le support temporel du film.
//	TEMOIN B (decalage circulaire)      les instants REELS, decales d'un offset aleatoire modulo
//	                                    la duree du film. Il preserve la structure d'AMAS des
//	                                    lectures (un paquet delta porte des dizaines de records
//	                                    au MEME horodatage) et ne detruit que l'alignement avec
//	                                    les explosions.
//
// C'EST LE TEMOIN B QUI FAIT FOI, ET IL FAUT DIRE POURQUOI AVANT DE LIRE UN CHIFFRE. Compter des
// lectures comme si elles etaient independantes alors qu'elles arrivent par paquets fabrique de
// la significativite a partir de rien : c'est la pseudo-replication, et elle rendrait n'importe
// quel canal « significatif ». Le temoin A est publie parce que la mission le prescrit ; il est
// systematiquement le plus optimiste des deux, et c'est attendu.
//
// btRepetitions repetitions, graine figee. La p-valeur est EMPIRIQUE : la part des repetitions
// dont le compte egale ou depasse l'observe, majoree de un au numerateur et au denominateur
// (aucune p-valeur nulle ne peut sortir d'un tirage fini).
//
// # LE CRITERE, ECRIT AVANT LA MESURE
//
// FENETRE D'INTERET : delai dans [5 s, 60 s[ avant l'explosion — la plage que la mission designe.
//
//	EXCES           le compte observe dans la fenetre vaut au moins btEnrichMin = 1,5 fois la
//	                moyenne du TEMOIN B ;
//	SIGNIFICATIVITE p-valeur empirique <= btPMax = 0,01 contre le TEMOIN B (et <= btPMax / 8
//	                apres correction de Bonferroni quand les huit valeurs de `i14` sont testees
//	                ensemble) ;
//	CONSISTANCE     l'exces ne vient pas d'une poignee d'explosions : le nombre d'explosions dont
//	                la fenetre porte PLUS de lectures que son attendu doit lui aussi depasser le
//	                TEMOIN B a la meme p-valeur. Avec 28 explosions, un effet porte par trois
//	                d'entre elles n'est pas un canal, c'est une coincidence ;
//	TEMOIN CROISE   les films NON-Assaut, passes au meme instrument avec des PSEUDO-explosions
//	                (le motif d'un film d'Assaut de duree voisine), ne doivent PAS montrer
//	                d'enrichissement. Un temoin qui s'allume disqualifie l'instrument.
//
// LES QUATRE ENSEMBLE -> CANDIDAT. UN SEUL QUI MANQUE -> NEGATIF, publie tel quel.
//
// # CE QUE CHAQUE ISSUE VOUDRAIT DIRE — ecrit ici AVANT de lancer
//
//	exces net, temoin croise muet   le trafic ti=11 monte avant l'armement. Le canal ne DONNE pas
//	                                la jauge, mais il DATE une fenetre, et le lot suivant a une
//	                                cible.
//	pas d'exces, temoins muets      NEGATIF NET. La derniere voie ouverte sur ti=11 se ferme : ni
//	                                la valeur, ni meme la densite ne parlent de la bombe.
//	exces ET temoin croise allume   L'INSTRUMENT FABRIQUE DES PICS — le plus souvent parce que la
//	                                densite des paquets n'est pas stationnaire. Aucun verdict sur
//	                                l'Assaut ne vaut alors, et c'est ce qu'il faut ecrire.
//	trop peu de lectures            ECHANTILLON INSUFFISANT, a dire et non a maquiller. Le nombre
//	                                d'INSTANTS DISTINCTS est publie a cote du nombre de lectures :
//	                                c'est lui, et non le second, qui borne ce que la mesure peut
//	                                trancher.
//
// # LES TROIS VOIES, DECIDEES AVANT LA MESURE
//
//	DELTA           toutes les lectures delta, chainees ou non — la voie de la mission ;
//	DELTA CHAINEE   le sous-ensemble chaine, publie a cote pour qu'aucun des deux ne soit choisi
//	                apres coup ;
//	IMAGE-CLE       la voie dont l'ancrage est PROUVE (legalite 100 % sur i0), mais dont les
//	                records sont rares et la valeur de `i12` constante par slot. Controle de
//	                forme, pas candidat.
//
// # LE RESULTAT DE LA PASSE UNIQUE (2026-09-01) — A LIRE AVANT D'EN ECRIRE UN AUTRE
//
// NEGATIF SUR LES DIX-SEPT EPREUVES. 11 films decodes en 134 s, pic memoire 0,02 Gio.
//
//	voie / champ        lectures   fenetre observee   temoin B          verdict
//	DELTA i12                277        65            67,3 +- 10,5      x0,97  p=0,61
//	DELTA i13                326        83            88,2 +- 14,9      x0,94  p=0,63
//	DELTA i14                437       114            99,9 +- 11,0      x1,14  p=0,09
//	IMAGE-CLE i12            291       100            77,3 +- 16,1      x1,29  p=0,09
//
// Aucun des huit etats de i14 ne ressort non plus (le meilleur : etat 5 a x1,40, p=0,16, quand le
// seuil de Bonferroni est 0,00125). La consistance ne bouge pas davantage : 13, 10 et 16
// explosions en exces sur 28, pour des temoins a 12,1, 11,4 et 13,2.
//
// LE NEGATIF EST BORNE, ET C'EST CE QUI LUI DONNE SA VALEUR. Un exces de x1,5 aurait valu +3,2
// ecarts-types du temoin sur i12, +2,9 sur i13, +4,6 sur i14 : il aurait ete vu. Les valeurs
// observees se tiennent a -0,2, -0,4 et +1,3 ecart-type. Ce n'est donc pas « l'echantillon est
// trop petit pour repondre » — la mesure EXCLUT tout enrichissement au-dela d'environ x1,4.
//
// DEUX RESULTATS DE BORD QUI VALENT AUTANT QUE LE VERDICT.
//
//  1. LA VOIE DELTA N'A AUCUN AMAS : 277 lectures pour 276 instants DISTINCTS. Les temoins A et B
//     coincident donc, et la taille d'echantillon effective est bien celle qu'on croit. La voie
//     IMAGE-CLE, elle, en a : 291 lectures pour 85 instants. Et c'est exactement la qu'A et B
//     divergent — A rend p=0,001, B rend p=0,088 sur le MEME chiffre. Sans le temoin B, ce lot
//     aurait annonce une trouvaille a p=0,001 qui n'est qu'un effet de grappe.
//  2. HORS CRITERE, ET DIT ICI POUR QU'IL NE SOIT PAS RELU COMME UN RESULTAT : le seau [0 s, 5 s[
//     est au-dessus du temoin sur les trois champs delta (12/6,1 · 13/8,0 · 11/9,1, soit x1,55
//     ensemble), et l'etat 7 y met 5 lectures pour 0,8 attendue. Ce seau n'est PAS dans la fenetre
//     preenregistree ; sur 63 seaux publies, trois depassements a p=0,05 sont l'esperance du
//     hasard. Et une ecriture d'objectif dans les cinq secondes qui precedent l'explosion serait
//     de toute facon la CONSEQUENCE de la fin de meche, pas le debut de l'armement. A verifier par
//     une mesure dont ce serait le critere ECRIT D'AVANCE, jamais en relisant celle-ci.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee, UN SEUL
// decodage a la fois (`LockProcessDecode`). Aucun chemin de production n'est modifie.
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run ObjectifTi11Bruit -v -timeout 60m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// Les trois champs interroges. i12 et i13 sont R(32) bruts, i14 est R(3).
const (
	btChampProgres = iota // i12 — LA JAUGE
	btChampRequis         // i13 — LE SEUIL
	btChampEtat           // i14 — L'ETAT, huit valeurs possibles
	btChamps
)

// Les trois voies, decidees avant la mesure (cf. l'en-tete).
const (
	btVoieDelta = iota
	btVoieDeltaChainee
	btVoieCle
	btVoies
)

// Les seuils du critere, figes AVANT la mesure.
const (
	// btFenetreBasMS / btFenetreHautMS bornent la fenetre d'interet, en millisecondes AVANT
	// l'explosion : la plage « 5 a 60 secondes » que la mission designe.
	btFenetreBasMS  = 5_000
	btFenetreHautMS = 60_000
	// btEnrichMin est le facteur d'enrichissement minimal exige contre le temoin B.
	btEnrichMin = 1.5
	// btPMax est le seuil de p-valeur empirique, avant correction de Bonferroni.
	btPMax = 0.01
	// btEtats est le domaine de i14 : R(3).
	btEtats = 8
	// btGraine fixe le tirage : la mesure doit se rejouer a l'identique.
	btGraine = 20260901
	// btSeaux est le nombre de seaux de l'histogramme des delais.
	btSeaux = 7
)

// btBornesMS decoupe les delais en seaux (millisecondes). Le dernier seau est ouvert.
var btBornesMS = [btSeaux]int{0, 5_000, 10_000, 20_000, 30_000, 60_000, 120_000}

// btEch est une lecture datee sur l'horloge du MANIFESTE — la meme base que l'oracle des
// explosions (cf. `mntChargerHorloge`, dont l'ecart moteur -> manifeste est mesure sous 1e-4 en
// relatif, donc les delais ABSOLUS sont lisibles et pas seulement leur dispersion).
type btEch struct {
	tMS int32
	v   uint64
}

// btVoie porte les echantillons d'UN champ dans UNE voie.
//
// `distincts` est publie a cote de `len(ech)` parce qu'il est la VRAIE taille d'echantillon : un
// paquet delta porte des dizaines de records au meme horodatage, et compter ces records comme
// autant d'observations independantes serait une pseudo-replication.
type btVoie struct {
	ech       []btEch
	distincts int
}

// btFilmBilan porte ce qu'un film a rendu, une fois ses lectures digerees et relachees.
type btFilmBilan struct {
	id, mode string
	// debutMS / finMS bornent le support temporel du film : le domaine des tirages temoins.
	debutMS, finMS int32
	voies          [btVoies][btChamps]*btVoie
	sansHorloge    int
	sc             ObjectiveScan
	duree          time.Duration
	// expl est la liste TRIEE des explosions du film — ou des PSEUDO-explosions d'un temoin.
	expl []int
	// motif nomme le film d'Assaut dont un temoin a emprunte le motif d'explosions.
	motif string
}

// TestObjectifTi11Bruit interroge i12, i13 et i14 sur les neuf films d'Assaut et les temoins.
func TestObjectifTi11Bruit(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestObjectifTi11Bruit", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("pic memoire observe : %.2f Gio (plafond souple %d Gio)",
			float64(g.Peak())/(1<<30), filmproc.MeasureLimitGiB)
	}()
	release := LockProcessDecode()
	defer release()

	var assauts, temoins []*btFilmBilan
	for _, f := range ti11Corpus {
		b := btMesurerFilm(t, cache, f.id, f.mode)
		if b == nil {
			continue
		}
		btJournalFilm(t, b)
		if expl, ok := ti12Explosions[b.id]; ok {
			b.expl = append([]int(nil), expl...)
			sort.Ints(b.expl)
			assauts = append(assauts, b)
			continue
		}
		temoins = append(temoins, b)
	}
	btMotifsTemoins(t, temoins, assauts)
	btGate0(t, assauts)
	btGate1(t, temoins)
	btGate2(t, assauts)
	btGate3(t, assauts, temoins)
}

// btMesurerFilm balaye UN film et digere ses lectures. Les lectures brutes sont relachees avant
// de rendre : seul le bilan survit d'un film au suivant (garde memoire).
func btMesurerFilm(t *testing.T, cache, id, mode string) *btFilmBilan {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	if CountFilmChunks(dir) == 0 {
		t.Logf("%-9s %-26s AUCUN CHUNK — film absent du cache, saute", id, mode)
		return nil
	}
	h, ok := mntChargerHorloge(dir)
	if !ok {
		t.Logf("%-9s %-26s MANIFESTE ABSENT — sans horloge la lecture n'est pas confrontable",
			id, mode)
		return nil
	}
	depart := time.Now()
	sc, err := ScanFilmObjectives(dir)
	if err != nil {
		t.Logf("%-9s %-26s balayage impossible (%v) — saute", id, mode, err)
		return nil
	}
	b := &btFilmBilan{id: id, mode: mode}
	b.debutMS, b.finMS = btSupport(h)
	for v := range b.voies {
		for c := range b.voies[v] {
			b.voies[v][c] = &btVoie{}
		}
	}
	btDigerer(b, sc.Reads, h)
	b.finaliser()
	sc.Reads = nil
	b.sc, b.duree = sc, time.Since(depart)
	return b
}

// btSupport rend les bornes du support temporel du film : le domaine sur lequel les temoins
// tirent. Il est pris sur TOUS les paquets dates, pas sur les seules lectures — un temoin tire
// sur le film, pas sur ce que l'instrument y a trouve.
func btSupport(h mntHorloge) (int32, int32) {
	premier := true
	var bas, haut int32
	for _, ms := range h.ms {
		if premier {
			bas, haut, premier = ms, ms, false
			continue
		}
		if ms < bas {
			bas = ms
		}
		if ms > haut {
			haut = ms
		}
	}
	return bas, haut
}

// btDigerer range les lectures des trois champs dans leurs voies.
func btDigerer(b *btFilmBilan, reads []ObjectiveRead, h mntHorloge) {
	for _, r := range reads {
		c, ok := btChampDe(r.Field)
		if !ok {
			continue
		}
		ms, date := h.ms[r.TimestampUS]
		if !date {
			b.sansHorloge++
			continue
		}
		e := btEch{tMS: ms, v: r.Value}
		if r.FromKeyframe {
			b.voies[btVoieCle][c].ech = append(b.voies[btVoieCle][c].ech, e)
			continue
		}
		b.voies[btVoieDelta][c].ech = append(b.voies[btVoieDelta][c].ech, e)
		if r.Chained {
			b.voies[btVoieDeltaChainee][c].ech = append(b.voies[btVoieDeltaChainee][c].ech, e)
		}
	}
}

// btChampDe traduit un champ publie de ti=11 en index de ce lot. Les trois autres champs publies
// (i0 timers, i3 object-reference, i5 type) sont hors mission et ecartes.
func btChampDe(f ObjectiveField) (int, bool) {
	switch f {
	case ObjectiveFieldProgress:
		return btChampProgres, true
	case ObjectiveFieldRequiredProgress:
		return btChampRequis, true
	case ObjectiveFieldState:
		return btChampEtat, true
	}
	return 0, false
}

// finaliser trie chaque voie par instant et compte les INSTANTS DISTINCTS.
func (b *btFilmBilan) finaliser() {
	for v := range b.voies {
		for c := range b.voies[v] {
			s := b.voies[v][c]
			sort.Slice(s.ech, func(i, j int) bool { return s.ech[i].tMS < s.ech[j].tMS })
			vus := map[int32]bool{}
			for _, e := range s.ech {
				vus[e.tMS] = true
			}
			s.distincts = len(vus)
		}
	}
}

// btNomChamp rend l'etiquette d'un champ de ce lot.
func btNomChamp(c int) string {
	switch c {
	case btChampProgres:
		return "i12 progress"
	case btChampRequis:
		return "i13 required-progress"
	case btChampEtat:
		return "i14 state"
	}
	return champInconnu
}

// btNomVoie rend l'etiquette d'une voie.
func btNomVoie(v int) string {
	switch v {
	case btVoieDelta:
		return "DELTA"
	case btVoieDeltaChainee:
		return "DELTA CHAINEE"
	case btVoieCle:
		return "IMAGE-CLE"
	}
	return champInconnu
}
