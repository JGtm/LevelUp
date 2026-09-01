package filmdec

// biped_pickup_ref0_couverture_test.go — LOT 2, SUITE. La base 512 est etablie sur les
// paires de classe R(3)=0 (ramassages d'arme). Restent deux questions :
//
//	1. COUVERTURE — l'identite `slot = 512 + ref0.index` designe-t-elle un bipede pour TOUS
//	   les evenements, ou seulement pour ceux de la classe 0 ? Mesure par classe.
//	2. LES CLASSES 2 ET 3 (equipement, grenades) — la base tient-elle pour elles ? Le lot 1 a
//	   montre qu'elles ne portent JAMAIS une arme d'i43..i46 ; leur verite terrain est donc
//	   ailleurs : le canal EQUIPEMENT (`ScanFilmEquipmentChanges`, i48), dont chaque emission
//	   est lue sur un record delta ancre et porte donc le SLOT du bipede.
//
// L'APPARTENANCE A LA BANDE DE BIPEDES EST UN PROXY FAIBLE, et c'est assume : la bande fait
// une centaine de slots, un index tire au hasard y tombe souvent. Elle n'est publiee que
// comme mesure de couverture. LE JUGE reste la correspondance EXACTE par evenement contre une
// verite terrain, avec temoin — c'est ce que fait la seconde partie.
//
// Garde BIPED_PICKUP_FILM.

import (
	"os"
	"testing"
)

// bpkBaseDom2 est la base du domaine 2 etablie par TestBipedPickupRef0Base : ecart
// `slot du ramasseur - index de ref0` mesure a 512 sur 21/21 puis 11/11 paires non ambigues,
// une seule valeur distincte, sur deux films. C'est la meme valeur que la base de plage des
// bipedes utilisee pour la reference domaine 1 de damage_aftermath.
const bpkBaseDom2 = 512

// bpkSlot rend le slot de bipede designe par ref0.
func bpkSlot(e bpkEvent) uint32 { return uint32(bpkBaseDom2 + int(e.Ref0)) }

// TestBipedPickupRef0Couverture — mesure de COUVERTURE, par classe.
//
// SEUIL ECRIT AVANT : on publie, par classe R(3), la part des evenements dont
// `512 + ref0.index` tombe dans la bande de bipedes du film, ET le TEMOIN obtenu en tirant
// l'index d'un AUTRE evenement (permutation d'un cran). Si la part reelle ne depasse pas
// nettement le temoin, la mesure ne dit rien — la bande est large.
func TestBipedPickupRef0Couverture(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()

	evs := bpkCollecte(t, f)
	if len(evs) == 0 {
		t.Skip("aucun biped_pickup sur ce film")
	}
	chunks := make([]int, 0, f.chunks)
	for i := 1; i <= f.chunks; i++ {
		chunks = append(chunks, i)
	}
	bande := bipedSlotBand(f.dir, chunks)
	if len(bande) == 0 {
		t.Skip("bande de bipedes vide : pas de mesure possible")
	}
	minS, maxS := uint32(1<<30), uint32(0)
	for s := range bande {
		if s < minS {
			minS = s
		}
		if s > maxS {
			maxS = s
		}
	}
	t.Logf("== COUVERTURE de `slot = %d + ref0` · %s ==", bpkBaseDom2, f.dir)
	t.Logf("bande de bipedes : %d slots, de %d a %d", len(bande), minS, maxS)

	dans, total := map[uint64]int{}, map[uint64]int{}
	temoin := map[uint64]int{}
	slots := map[uint64]int{}
	for i, e := range evs {
		total[e.Kind]++
		s := bpkSlot(e)
		slots[uint64(s)]++
		if bande[s] {
			dans[e.Kind]++
		}
		if bande[uint32(bpkBaseDom2+int(evs[(i+1)%len(evs)].Ref0))] {
			temoin[e.Kind]++
		}
	}
	for k := uint64(0); k < 8; k++ {
		if total[k] == 0 {
			continue
		}
		t.Logf("  classe R(3)=%d : %d / %d dans la bande (%.1f %%) · temoin permute %.1f %%",
			k, dans[k], total[k], bpkPct(dans[k], total[k]), bpkPct(temoin[k], total[k]))
	}
	tous, tousDans := 0, 0
	for k := range total {
		tous += total[k]
		tousDans += dans[k]
	}
	t.Logf("  TOUTES CLASSES : %d / %d (%.1f %%) · %d slots distincts designes : %s",
		tousDans, tous, bpkPct(tousDans, tous), len(slots), bpkTop(slots, 10))
	t.Log("LECTURE : ce taux est un PROXY FAIBLE (la bande fait ~100 slots). Le juge exact est " +
		"TestBipedPickupRef0Base (classe 0) et TestBipedPickupRef0Equipement (classes 2 et 3).")
}

// TestBipedPickupRef0Equipement — LE JUGE EXACT DES CLASSES 2 ET 3. Verite terrain
// independante : le canal equipement (i48), dont chaque emission est ancree sur un record
// delta et porte donc le slot du bipede.
//
// NOTE SUR LE CANAL : `ScanFilmEquipmentChanges` est appele SANS temoin de naissance, ce qui
// fait qu'il sur-classe les premieres emissions en `taken`. Cela n'affecte PAS cette mesure :
// on n'utilise pas la nature du changement, seulement le couple (instant, slot) — et celui-la
// est lu, pas deduit.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	E1 — parmi les evenements type 9 de classe 2 ou 3 qui ont AU MOINS une emission
//	     d'equipement a moins de 500 ms, la part dont `512 + ref0` egale EXACTEMENT le slot
//	     d'une de ces emissions doit valoir >= 60 %.
//	E2 — le TEMOIN (les memes evenements, horodatages decales de +37 / -53 / +91 s, pire des
//	     trois) doit rester sous 25 %.
//	E3 — la classe 0 sert de CONTROLE POSITIF sur ce meme canal : elle ne doit PAS mieux
//	     s'apparier a l'equipement que les classes 2/3 (sinon l'appariement ne mesure que la
//	     densite d'emissions, pas la semantique).
func TestBipedPickupRef0Equipement(t *testing.T) {
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()

	evs := bpkCollecte(t, f)
	if len(evs) == 0 {
		t.Skip("aucun biped_pickup sur ce film")
	}
	chg, st, err := ScanFilmEquipmentChanges(f.dir, nil)
	if err != nil {
		t.Fatalf("balayage des changements d'equipement : %v", err)
	}
	t.Logf("== CLASSES 2/3 contre le canal EQUIPEMENT (i48) · %s ==", f.dir)
	t.Logf("emissions d'equipement : %d sur %d vies (emissions manquees estimees par le canal lui-meme : %d)",
		len(chg), st.Lives, st.MissedEstimate)
	if len(chg) == 0 {
		t.Skip("aucune emission d'equipement : pas de verite terrain sur ce film")
	}

	mesure := func(classes map[uint64]bool, decalUS int64) (avec, exact int) {
		for _, e := range evs {
			if !classes[e.Kind] {
				continue
			}
			trouve, bon := false, false
			for _, c := range chg {
				d := int64(c.TimestampUS) + decalUS - int64(e.TimestampUS)
				if d < 0 {
					d = -d
				}
				if d > bpkTolUS {
					continue
				}
				trouve = true
				if c.Slot == bpkSlot(e) {
					bon = true
				}
			}
			if trouve {
				avec++
				if bon {
					exact++
				}
			}
		}
		return avec, exact
	}
	nonArmes := map[uint64]bool{2: true, 3: true}
	armes := map[uint64]bool{0: true, 1: true}

	avec, exact := mesure(nonArmes, 0)
	pire := 0.0
	for _, d := range bpkDecalages {
		a, x := mesure(nonArmes, d)
		p := bpkPct(x, a)
		t.Logf("  TEMOIN decale de %d s : %d / %d (%.1f %%)", d/1_000_000, x, a, p)
		if p > pire {
			pire = p
		}
	}
	t.Logf("E1 — classes 2/3 : %d evenements ont une emission d'equipement a <= %d ms ; parmi eux "+
		"`512 + ref0` == le slot emetteur : %d (%.1f %%)",
		avec, bpkTolUS/1000, exact, bpkPct(exact, avec))
	aA, xA := mesure(armes, 0)
	t.Logf("E3 — CONTROLE classes 0/1 (armes) sur le MEME canal : %d / %d (%.1f %%)",
		xA, aA, bpkPct(xA, aA))
	okE1 := bpkPct(exact, avec) >= 60 && avec >= 8
	okE2 := pire < 25
	t.Logf("VERDICT E1 (>= 60 %%, n >= 8) : %s · E2 (temoin < 25 %%) : %s · base 512 valable pour les classes 2/3 : %s",
		bpkVerdict(okE1), bpkVerdict(okE2), bpkVerdict(okE1 && okE2))
}

// TestBipedPickupRef0HypotheseB — H-B : ref0 designe-t-il l'OBJET ramasse plutot que le
// ramasseur ? La question est tranchee par construction une fois H-A etablie (un slot est
// unique : il ne peut pas designer a la fois le bipede ramasseur et l'objet ramasse), mais on
// le VERIFIE au lieu de le supposer : si `512 + ref0` etait un objet, ce slot ne serait pas
// dans la bande des bipedes et ne coinciderait pas avec le slot du porteur connu.
//
// SEUIL ECRIT AVANT : H-B est REFUTEE si, sur les paires de verite terrain, `512 + ref0`
// egale le slot du RAMASSEUR dans >= 90 % des cas — un objet au sol n'a pas le slot de celui
// qui le ramasse.
func TestBipedPickupRef0HypotheseB(t *testing.T) {
	if os.Getenv(bpkFilmEnv) == "" {
		t.Skipf("%s absent : instrument de recherche saute", bpkFilmEnv)
	}
	f, ok := bpkOpen(t)
	if !ok {
		return
	}
	release := LockProcessDecode()
	defer release()

	evs := bpkCollecte(t, f)
	ref := hwKeyframeRef(t, f.dir)
	chg, _, err := ScanFilmHeldWeaponChanges(f.dir, ref.setAt)
	if err != nil {
		t.Fatalf("balayage des changements d'arme : %v", err)
	}
	paires, _ := bpkPaires(evs, chg)
	if len(paires) < 8 {
		t.Skipf("%d paires : trop peu pour trancher", len(paires))
	}
	ramasseur := 0
	for _, p := range paires {
		if bpkSlot(p.ev) == p.slot {
			ramasseur++
		}
	}
	t.Logf("== H-B : ref0 = l'OBJET ramasse ? · %s ==", f.dir)
	t.Logf("`512 + ref0` == slot du RAMASSEUR connu : %d / %d (%.1f %%)",
		ramasseur, len(paires), bpkPct(ramasseur, len(paires)))
	t.Logf("VERDICT : H-B (ref0 = l'objet) %s — un slot est unique, il ne peut designer les deux.",
		map[bool]string{true: "REFUTEE", false: "non refutee"}[bpkPct(ramasseur, len(paires)) >= 90])
	t.Log("CONSEQUENCE : l'evenement ne porte AUCUNE reference a l'INSTANCE monde de l'objet " +
		"ramasse. Il porte son identifiant de CATALOGUE (le R(32) du lot 1), pas son handle. " +
		"Le lien vers le SOCLE d'origine reste donc l'affaire du canal spatial (schema 26).")
}
