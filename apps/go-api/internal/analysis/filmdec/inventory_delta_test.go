package filmdec

import "testing"

// inventory_delta_test.go — LES GARDE-RAILS PURS du suivi delta de l'inventaire (aucun film,
// aucune I/O : ils valent en CI, contrairement à l'instrument de corpus
// inventory_delta_corpus_test.go qui exige les films).
//
// CE QU'ILS VERROUILLENT. (1) Le déser d'i47 consommait ses neuf bits pour rester aligné et
// les JETAIT ; la publication ne doit RIEN changer au nombre de bits consommés — un décalage
// d'un seul bit désynchroniserait tout le reste du record (même point sensible que le lot
// i48 du 2026-08-14). (2) Le test réfutable d'i22 — compteur == 4 et valeurs bornées — est
// la seule chose qui distingue une lecture d'un curseur perdu ; s'il s'assouplit, le scanner
// se met à publier du bruit sans que rien ne rougisse.

// grenadeSetBits construit un i47 isolé : R(6) masque puis R(3) sélection, suivis d'une queue
// de garde qu'un désalignement déplacerait.
func grenadeSetBits(mask, sel uint64) []byte {
	w := &bitWriter{}
	w.bits(mask, i47MaskBits)
	w.bits(sel, i47SelBits)
	w.bits(0x2A, 8)
	return w.buf
}

func TestConsumeBipedDesiredGrenadeSetPublieEtNeDecalePas(t *testing.T) {
	cases := []struct {
		nom      string
		mask     uint64
		sel      uint64
		wantMask uint32
		wantSel  int
	}{
		{"un seul type porté, sélectionné", 0b000001, 1, 0b000001, 1},
		{"deux types portés, second sélectionné", 0b000011, 2, 0b000011, 2},
		{"aucune sélection (codage 1-base : 0 = absence)", 0b000101, 0, 0b000101, GrenadeSetNoSelection},
		{"masque plein, sélection maximale", 0b111111, 7, 0b111111, 7},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			var got struct {
				mask  uint32
				sel   int
				calls int
			}
			prev := grenadeSetHook
			SetGrenadeSetHook(func(mask uint32, sel int) {
				got.mask, got.sel, got.calls = mask, sel, got.calls+1
			})
			defer SetGrenadeSetHook(prev)

			br := NewBitReader(grenadeSetBits(c.mask, c.sel))
			consumeBipedDesiredGrenadeSet(br)

			// LA LARGEUR D'ABORD : c'est elle qui tient l'alignement du record.
			if w := br.BitPos(); w != i47MaskBits+i47SelBits {
				t.Fatalf("largeur consommée = %d bits, attendu %d", w, i47MaskBits+i47SelBits)
			}
			if got.calls != 1 {
				t.Fatalf("hook appelé %d fois, attendu 1", got.calls)
			}
			if got.mask != c.wantMask || got.sel != c.wantSel {
				t.Fatalf("publié (masque=%b, sel=%d), attendu (%b, %d)",
					got.mask, got.sel, c.wantMask, c.wantSel)
			}
		})
	}
}

func TestConsumeBipedDesiredGrenadeSetSansHookConsommeAutant(t *testing.T) {
	prev := grenadeSetHook
	SetGrenadeSetHook(nil)
	defer SetGrenadeSetHook(prev)

	br := NewBitReader(grenadeSetBits(0b001011, 3))
	consumeBipedDesiredGrenadeSet(br)
	if w := br.BitPos(); w != i47MaskBits+i47SelBits {
		t.Fatalf("hook absent : largeur = %d bits, attendu %d", w, i47MaskBits+i47SelBits)
	}
}

// TestInvDeltaPlausibleEstLeTestRefutable fige la règle qui sépare une lecture d'un curseur
// perdu. Chaque cas REJETÉ est une chose qu'un curseur au hasard produirait couramment.
func TestInvDeltaPlausibleEstLeTestRefutable(t *testing.T) {
	cases := []struct {
		nom   string
		count uint64
		vals  []uint64
		want  bool
	}{
		{"quatre compteurs dans les bornes", 4, []uint64{0, 1, 2, 0}, true},
		{"quatre compteurs tous nuls (une mesure, pas un échec)", 4, []uint64{0, 0, 0, 0}, true},
		{"compteur 3 : signature d'un curseur mal placé", 3, []uint64{0, 1, 2}, false},
		{"compteur 7 : idem", 7, []uint64{0, 0, 0, 0, 0, 0, 0}, false},
		{"compteur 0 : idem", 0, nil, false},
		{"une valeur au-delà de la borne de jeu", 4, []uint64{0, 3, 0, 0}, false},
		{"un octet plein : la signature la plus nette", 4, []uint64{255, 0, 0, 0}, false},
		{"compteur juste mais valeurs manquantes", 4, []uint64{0, 1}, false},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			if got := invDeltaPlausible(c.count, c.vals); got != c.want {
				t.Fatalf("invDeltaPlausible(%d, %v) = %v, attendu %v", c.count, c.vals, got, c.want)
			}
		})
	}
}

// TestCollectI47RameneLaSelectionEnBase0 verrouille la CONVERSION de grandeur : le flux code
// la sélection en base 1, le document la publie en RANG base 0 — la même grandeur que le canal
// des images-clés. Publier deux grandeurs sous un seul nom est le défaut qui a coûté le
// chantier de la capacité d'armure.
func TestCollectI47RameneLaSelectionEnBase0(t *testing.T) {
	cases := []struct {
		nom           string
		mask          uint32
		sel           int
		wantSel       int
		wantNoSel     int
		wantOutOfMask int
	}{
		{"sélection 1 -> rang 0", 0b0001, 1, 0, 0, 0},
		{"sélection 4 -> rang 3", 0b1000, 4, 3, 0, 0},
		{"sélection 0 : aucun type désigné", 0b0011, 0, InventoryDeltaNoSel, 1, 0},
		{"sélection hors masque : non publiée", 0b0001, 2, InventoryDeltaNoSel, 0, 1},
		{"sélection hors des rangs du titre", 0b1111, 7, InventoryDeltaNoSel, 0, 1},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			sc := &invDeltaScanner{got47: true, last47mask: c.mask, last47sel: c.sel}
			var rec InventoryDelta
			if !sc.collectI47(&rec) {
				t.Fatal("collectI47 a rendu false alors que la lecture a abouti")
			}
			if !rec.SelRead || rec.Mask != c.mask {
				t.Fatalf("masque publié %b (lu=%v), attendu %b", rec.Mask, rec.SelRead, c.mask)
			}
			if rec.Sel != c.wantSel {
				t.Fatalf("sel = %d, attendu %d", rec.Sel, c.wantSel)
			}
			if sc.st.NoSelection != c.wantNoSel || sc.st.SelOutsideMask != c.wantOutOfMask {
				t.Fatalf("compteurs (noSel=%d, horsMasque=%d), attendus (%d, %d)",
					sc.st.NoSelection, sc.st.SelOutsideMask, c.wantNoSel, c.wantOutOfMask)
			}
		})
	}
}

// TestCollectI22NePubliePasUneLectureImplausible : le scanner COMPTE le bruit et ne le publie
// pas. Sans ce contrôle, une dérive de largeur en amont remplirait les fiches de valeurs
// inventées sans qu'aucun test ne rougisse.
func TestCollectI22NePubliePasUneLectureImplausible(t *testing.T) {
	sc := &invDeltaScanner{got22: true, last22c: 4, last22v: []uint64{0, 200, 0, 0}}
	var rec InventoryDelta
	if sc.collectI22(&rec) {
		t.Fatal("une lecture implausible a été publiée")
	}
	if rec.Grenades != nil {
		t.Fatalf("compteurs publiés %v, attendu nil", rec.Grenades)
	}
	if sc.st.Implausible != 1 || sc.st.I22Read != 1 {
		t.Fatalf("stats (lues=%d, implausibles=%d), attendues (1, 1)", sc.st.I22Read, sc.st.Implausible)
	}
}

// ammoBits construit un `weapon-state-ammo` isolé. LES DEUX PORTES SONT ACTIVES-BAS : le
// champ est présent quand son bit vaut 0 (relu au désassemblage le 2026-07-26, cf.
// consumeWeaponStateAmmo). Se tromper de polarité décale de 8 ou 12 bits tout ce qui suit.
func ammoBits(hasMag bool, mag uint64, hasFrac bool, frac uint64) []byte {
	w := &bitWriter{}
	if hasMag {
		w.bits(0, 1)
		w.bits(mag, 8)
	} else {
		w.bits(1, 1)
	}
	if hasFrac {
		w.bits(0, 1)
		w.bits(frac, 12)
	} else {
		w.bits(1, 1)
	}
	w.bits(0x2A, 8) // queue de garde
	return w.buf
}

func TestConsumeWeaponStateAmmoPublieEtNeDecalePas(t *testing.T) {
	cases := []struct {
		nom             string
		hasMag, hasFrac bool
		mag, frac       uint64
		wantWidth       int
	}{
		{"chargeur et fraction présents", true, true, 36, 2048, 1 + 8 + 1 + 12},
		{"chargeur seul (fraction absente)", true, false, 80, 0, 1 + 8 + 1},
		{"fraction seule (chargeur absent)", false, true, 0, 4095, 1 + 1 + 12},
		{"les deux portes fermées : le film n'écrit rien", false, false, 0, 0, 1 + 1},
		{"chargeur nul : une mesure, pas une absence", true, false, 0, 0, 1 + 8 + 1},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			var got struct {
				hasMag, hasFrac bool
				mag, frac       uint32
				calls           int
			}
			prev := weaponAmmoHook
			SetWeaponAmmoHook(func(hasMag bool, mag uint32, hasFrac bool, fracQ uint32) {
				got.hasMag, got.mag, got.hasFrac, got.frac = hasMag, mag, hasFrac, fracQ
				got.calls++
			})
			defer SetWeaponAmmoHook(prev)

			br := NewBitReader(ammoBits(c.hasMag, c.mag, c.hasFrac, c.frac))
			consumeWeaponStateAmmo(br)

			if w := br.BitPos(); w != c.wantWidth {
				t.Fatalf("largeur consommée = %d bits, attendu %d", w, c.wantWidth)
			}
			if got.calls != 1 {
				t.Fatalf("hook appelé %d fois, attendu 1", got.calls)
			}
			if got.hasMag != c.hasMag || got.hasFrac != c.hasFrac {
				t.Fatalf("présences publiées (mag=%v, frac=%v), attendues (%v, %v)",
					got.hasMag, got.hasFrac, c.hasMag, c.hasFrac)
			}
			if c.hasMag && uint64(got.mag) != c.mag {
				t.Fatalf("chargeur publié %d, attendu %d", got.mag, c.mag)
			}
			if c.hasFrac && uint64(got.frac) != c.frac {
				t.Fatalf("fraction publiée %d, attendue %d", got.frac, c.frac)
			}
		})
	}
}

func TestConsumeWeaponStateRoundsInventoryPublieEtNeDecalePas(t *testing.T) {
	for _, v := range []uint64{0, 4, 240, 2047} {
		var got struct {
			rounds uint32
			calls  int
		}
		prev := weaponRoundsHook
		SetWeaponRoundsHook(func(r uint32) { got.rounds, got.calls = r, got.calls+1 })

		w := &bitWriter{}
		w.bits(v, weaponRoundsBits)
		w.bits(0x2A, 8)
		br := NewBitReader(w.buf)
		consumeWeaponStateRoundsInventory(br)
		SetWeaponRoundsHook(prev)

		if br.BitPos() != weaponRoundsBits {
			t.Fatalf("réserve %d : largeur = %d bits, attendu %d", v, br.BitPos(), weaponRoundsBits)
		}
		if got.calls != 1 || uint64(got.rounds) != v {
			t.Fatalf("réserve %d : publié %d en %d appels", v, got.rounds, got.calls)
		}
	}
}

// TestCollectAmmoAppliqueLesEnveloppes fige LA règle des munitions : une valeur au-delà de
// l'enveloppe mesurée n'est pas publiée, elle est comptée. L'enveloppe est délibérément plus
// large que le maximum observé (80 / 240) et bien plus étroite que le champ (255 / 2047) : ce
// qu'elle rejette, c'est une distribution qui remplit le champ, pas une arme inconnue.
func TestCollectAmmoAppliqueLesEnveloppes(t *testing.T) {
	t.Run("chargeur et réserve dans l'enveloppe", func(t *testing.T) {
		sc := &invDeltaScanner{}
		sc.ammo[0] = invDeltaAmmoAcc{Read: true, HasMag: true, Mag: 36, HasFrac: true, FracQ: 2048}
		sc.rounds[0], sc.roundsRead[0] = 240, true
		var rec InventoryDelta
		if !sc.collectAmmo(&rec) {
			t.Fatal("collectAmmo a rendu false")
		}
		if len(rec.Ammo) != 1 || rec.Ammo[0].WeaponSlot != 0 {
			t.Fatalf("emplacements publiés %+v", rec.Ammo)
		}
		a := rec.Ammo[0]
		if a.Mag == nil || *a.Mag != 36 || a.Res == nil || *a.Res != 240 || a.FracQ == nil || *a.FracQ != 2048 {
			t.Fatalf("valeurs publiées %+v", a)
		}
		if sc.st.MagOutOfEnvelope != 0 || sc.st.ResOutOfEnvelope != 0 {
			t.Fatalf("dépassements comptés à tort : %+v", sc.st)
		}
	})
	t.Run("chargeur hors enveloppe : compté, pas publié", func(t *testing.T) {
		sc := &invDeltaScanner{}
		sc.ammo[1] = invDeltaAmmoAcc{Read: true, HasMag: true, Mag: 200}
		var rec InventoryDelta
		if sc.collectAmmo(&rec) {
			t.Fatal("une valeur hors enveloppe a été publiée")
		}
		if sc.st.MagOutOfEnvelope != 1 || sc.st.MagRead != 1 {
			t.Fatalf("stats %+v, attendu MagRead=1 MagOutOfEnvelope=1", sc.st)
		}
	})
	t.Run("porte fermée : ni valeur ni dépassement", func(t *testing.T) {
		sc := &invDeltaScanner{}
		sc.ammo[0] = invDeltaAmmoAcc{Read: true}
		var rec InventoryDelta
		if sc.collectAmmo(&rec) {
			t.Fatal("un emplacement sans contenu a été publié")
		}
		if sc.st.AmmoRead != 1 || sc.st.MagRead != 0 {
			t.Fatalf("stats %+v", sc.st)
		}
	})
	t.Run("réserve seule, sans composant de chargeur au masque", func(t *testing.T) {
		sc := &invDeltaScanner{}
		sc.rounds[1], sc.roundsRead[1] = 12, true
		var rec InventoryDelta
		if !sc.collectAmmo(&rec) {
			t.Fatal("une réserve seule doit être publiée")
		}
		if len(rec.Ammo) != 1 || rec.Ammo[0].Mag != nil || rec.Ammo[0].Res == nil || *rec.Ammo[0].Res != 12 {
			t.Fatalf("publié %+v", rec.Ammo)
		}
	})
}
