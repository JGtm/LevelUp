package grammar

// components_navpoint_test.go — LES LARGEURS DE `ti=12`, MESUREES SUR LA CHAINE DE PRODUCTION.
//
// Le test n'appelle PAS les deserialiseurs directement : il passe par `consumeByName`, la meme
// porte que la traversee d'un record. C'est le seul moyen d'exercer aussi le cablage — le
// routage du nom, et surtout le `param_4` de `paramByComponent`, qui decide de deux largeurs du
// bloc de filtres et dont l'absence etait le mode de panne annonce par la note
// (`.ai/V7.5/film_re/NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17.md` § 17 pt 1).
//
// AUCUN FILM N'EST OUVERT : les flux sont fabriques bit a bit. La confrontation a un film entier
// est le lot 5.1.2, et elle porte sur la VALEUR, pas sur la largeur.

import "testing"

// navpointTi12 est l'index d'archetype des points de navigation geres.
const navpointTi12 = navpointRadialArchIndex

// consommerNavpoint fait passer `flux` par la chaine de dispatch pour le composant `nom` et rend
// le nombre de bits consommes et le verdict de portage.
//
// LE NIVEAU EST CELUI DU REGISTRE DU FILM, et c'est ce qui a change au lot 5.1.7 : `param_4` EST
// le `level` de l'entree de composant, et la chaine de dispatch le recoit par ce parametre. Le
// harnais passait `1` en dur du temps ou la table par nom decidait ; il passe desormais la valeur
// du composant, que `TestParamByComponentEgaleLeNiveauDuRegistre` tient egale au registre.
// [consommerNavpointAuNiveau] sert aux cas qui veulent MESURER un autre niveau.
func consommerNavpoint(t *testing.T, nom string, flux []byte) (int, bool) {
	t.Helper()
	return consommerNavpointAuNiveau(t, nom, flux, paramMesureDuComposant(nom))
}

// consommerNavpointAuNiveau fait la meme chose a un niveau IMPOSE.
func consommerNavpointAuNiveau(t *testing.T, nom string, flux []byte, niveau uint32) (int, bool) {
	t.Helper()
	br := lecteurDInstrument(flux)
	_, _, porte := consumeByName(br, nom, navpointTi12, niveau)
	return br.BitPos(), porte
}

// fluxDeBits fabrique un flux depuis une suite d'ecritures, complete a l'octet.
func fluxDeBits(ecr func(w *bitw)) []byte {
	w := &bitw{}
	ecr(w)
	w.pad(64) // du rab : le lecteur ne doit jamais toucher au bourrage si la largeur est juste
	return w.buf
}

// TestNavpointLargeursPlates : les six composants de `ti=12` a largeur FIXE, celle que le
// desassemblage donne. Ce sont les six lignes que le controle G4 fige.
func TestNavpointLargeursPlates(t *testing.T) {
	cas := []struct {
		nom    string
		compo  string
		attend int
	}{
		{"i0 sous-type R(32)", compNavpointSubType, 32},
		{"i1 drapeaux R(8)", compNavpointFlags, 8},
		{"i7 ordre d accostage R(8)", compNavpointDockingOrder, 8},
		{"i8 nom de groupe R(32)", compNavpointDockingGroupName, 32},
		{"i10 deux index de minuteur 2 x R(7)", compNavpointTimers, 14},
		{"i11 duree initiale R(17)", compNavpointManualTimerInitial, 17},
		{"i12 duree courante R(17)", compNavpointManualTimerCurrent, 17},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			for _, motif := range []byte{0x00, 0xFF, 0xAA} {
				flux := make([]byte, 16)
				for i := range flux {
					flux[i] = motif
				}
				n, porte := consommerNavpoint(t, c.compo, flux)
				if !porte {
					t.Fatalf("motif 0x%02X : le composant n est pas porte", motif)
				}
				if n != c.attend {
					t.Fatalf("motif 0x%02X : %d bits consommes, %d attendus", motif, n, c.attend)
				}
			}
		})
	}
}

// TestNavpointBlocDeFiltresLargeurs : le bloc `FUN_140dbe400` et les queues de `i2`..`i6`, avec
// le `param_4` que la table pose. Les largeurs attendues sont calculees a la main depuis la
// table des tags de la note § 3.
func TestNavpointBlocDeFiltresLargeurs(t *testing.T) {
	cas := []struct {
		nom    string
		compo  string
		ecr    func(w *bitw)
		attend int
	}{
		{
			// i5 : masque vide -> masque R(4) + drapeau R(1) (param_4 = 2, donc v = 1).
			nom: "i5 masque vide", compo: compNavpointVisibilityFilter,
			ecr: func(w *bitw) { w.put(0, 4); w.put(0, 1) }, attend: 5,
		},
		{
			// i6 : meme grammaire que i5, un filtre de tag 0 (le rappel ne lit RIEN).
			nom: "i6 un filtre de tag 0", compo: compNavpointDockingFilter,
			ecr:    func(w *bitw) { w.put(1, 4); w.put(0, 1); w.put(navpointFilterTagVide, 4) },
			attend: 9,
		},
		{
			// i5 : deux filtres, tag 1 (R(1)) et tag 8 (R(8)), chacun precede du R(1) commun.
			nom: "i5 tags 1 et 8", compo: compNavpointVisibilityFilter,
			ecr: func(w *bitw) {
				w.put(3, 4)
				w.put(0, 1)
				w.put(navpointFilterTagBool, 4)
				w.put(0, 1)
				w.put(0, 1)
				w.put(navpointFilterTagByte, 4)
				w.put(0, 1)
				w.put(0, 8)
			},
			attend: 5 + 6 + 13,
		},
		{
			// i5 : tag 5, porte INVERSEE — le R(13) n est lu que si le bit vaut ZERO.
			nom: "i5 tag 5, deux entrees dont une seule ouverte", compo: compNavpointVisibilityFilter,
			ecr: func(w *bitw) {
				w.put(1, 4)
				w.put(0, 1)
				w.put(navpointFilterTagStrList, 4)
				w.put(0, 1)
				w.put(2, 3) // deux entrees
				w.put(0, 1) // porte inversee : ouverte
				w.put(0, 13)
				w.put(1, 1) // porte inversee : fermee, rien de plus
			},
			attend: 4 + 1 + 4 + 1 + 3 + (1 + 13) + 1,
		},
		{
			// i5 : tag 6, une reference d entite du DOMAINE 0 derriere une porte DROITE.
			nom: "i5 tag 6, une reference d entite", compo: compNavpointVisibilityFilter,
			ecr: func(w *bitw) {
				w.put(1, 4)
				w.put(0, 1)
				w.put(navpointFilterTagRefList, 4)
				w.put(0, 1)
				w.put(1, 4) // une entree
				w.put(1, 1) // porte droite : la reference suit
				w.put(0, int(refDomWidth(navpointFilterRefDomain)))
				w.put(0, 2) // la generation
			},
			attend: 4 + 1 + 4 + 1 + 4 + 1 + int(refDomWidth(navpointFilterRefDomain)) + 2,
		},
		{
			// i3 : bloc vide, puis le R(1) de drapeau, puis aucune entree d ordre (K = 0).
			nom: "i3 masque vide", compo: compNavpointOffscreenFilters,
			ecr: func(w *bitw) { w.put(0, 4); w.put(0, 1); w.put(0, 1) }, attend: 6,
		},
		{
			// i4 : un filtre de tag 0 -> K = 1, donc un R(1) par filtre et UNE entree d ordre
			// de 3 bits (param_4 = 2, donc v = 1).
			nom: "i4 un filtre", compo: compNavpointOccludedFilters,
			ecr: func(w *bitw) {
				w.put(1, 4)
				w.put(0, 1)
				w.put(navpointFilterTagVide, 4)
				w.put(0, 1) // le drapeau du composant
				w.put(0, 1) // le drapeau du filtre present
				w.put(0, 3) // l entree d ordre
			},
			attend: 4 + 1 + 4 + 1 + 1 + 3,
		},
		{
			// i2 : bloc vide, puis les deux distances de 16 bits, et K = 0.
			nom: "i2 masque vide", compo: compNavpointDistanceFilters,
			ecr:    func(w *bitw) { w.put(0, 4); w.put(0, 1); w.put(0, 16); w.put(0, 16) },
			attend: 4 + 1 + 32,
		},
		{
			// i2 : un filtre -> deux distances de plus, et une entree d ordre de 3 bits
			// (param_4 = 3, donc v = 1 ; avec le defaut 1 le drapeau ferait 32 bits).
			nom: "i2 un filtre", compo: compNavpointDistanceFilters,
			ecr: func(w *bitw) {
				w.put(1, 4)
				w.put(0, 1)
				w.put(navpointFilterTagVide, 4)
				w.put(0, 16)
				w.put(0, 16)
				w.put(0, 16)
				w.put(0, 16)
				w.put(0, 3)
			},
			attend: 4 + 1 + 4 + 64 + 3,
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			n, porte := consommerNavpoint(t, c.compo, fluxDeBits(c.ecr))
			if !porte {
				t.Fatalf("le composant n est pas porte")
			}
			if n != c.attend {
				t.Fatalf("%d bits consommes, %d attendus", n, c.attend)
			}
		})
	}
}

// TestNavpointParam4DecideLaLargeur : le `param_4` de la table n'est pas decoratif. Sans lui le
// drapeau du bloc ferait 32 bits au lieu d'un, et l'entree d'ordre 2 au lieu de 3 — la marche se
// desynchroniserait des le premier filtre. Le test le MESURE, sur le meme flux.
func TestNavpointParam4DecideLaLargeur(t *testing.T) {
	for _, compo := range []string{
		compNavpointDistanceFilters, compNavpointOffscreenFilters,
		compNavpointOccludedFilters, compNavpointVisibilityFilter, compNavpointDockingFilter,
	} {
		t.Run(compo, func(t *testing.T) {
			p := paramMesureDuComposant(compo)
			if compo == compNavpointDistanceFilters {
				if p != 3 {
					t.Fatalf("param_4 = %d, 3 attendu (slot +0x10 = MOV EAX,0x3)", p)
				}
				return
			}
			if p != 2 {
				t.Fatalf("param_4 = %d, 2 attendu (slot +0x10 = MOV EAX,0x2)", p)
			}
		})
	}
	// LA CONSEQUENCE, MESUREE DES DEUX COTES SUR LE MEME FLUX : un bloc a masque vide vaut 5 bits
	// au niveau du registre (4 + 1) et 36 au niveau 1 (4 + 32), la mise en page LEGACY. C'est la
	// difference que le lot 5.1.7 fait passer par le canal du FILM : le niveau n'est plus une
	// entree de table consultee par nom, c'est le parametre que le traverseur descend.
	flux := func() []byte {
		return fluxDeBits(func(w *bitw) {
			w.put(0, 4)
			w.put(0, 1)
		})
	}
	attendu := navpointFilterMaskBits + navpointFilterFlagBitsRecent
	n, porte := consommerNavpoint(t, compNavpointVisibilityFilter, flux())
	if !porte || n != attendu {
		t.Fatalf("bloc a masque vide : %d bits (porte=%v), %d attendus", n, porte, attendu)
	}
	legacy := navpointFilterMaskBits + navpointFilterFlagBitsLegacy
	n, porte = consommerNavpointAuNiveau(t, compNavpointVisibilityFilter, flux(), 1)
	if !porte || n != legacy {
		t.Fatalf("au niveau 1 le bloc a masque vide doit valoir %d bits (mise en page legacy) : "+
			"%d obtenus (porte=%v). Si les deux niveaux rendent la meme largeur, le parametre "+
			"`level` n'atteint plus le lecteur de filtres.", legacy, n, porte)
	}
}

// TestNavpointTagInvalideArreteLaMarche : un tag hors [0, 14] ne se devine pas. Chez le jeu il
// tombe sur `FUN_1411c8f80`, qui ne revient pas ; ici la traversee s'arrete proprement.
func TestNavpointTagInvalideArreteLaMarche(t *testing.T) {
	flux := fluxDeBits(func(w *bitw) {
		w.put(1, 4)
		w.put(0, 1)
		w.put(15, 4) // le tag que le moteur refuse
		w.put(0, 1)
	})
	if _, porte := consommerNavpoint(t, compNavpointVisibilityFilter, flux); porte {
		t.Fatal("un tag de filtre invalide doit rendre ported=false, pas une largeur devinee")
	}
}

// TestNavpointMinuteurManuelDequantification : le pas de 50 ms, et la zone morte.
//
// C'est la seule chose qui rende `i11`/`i12` confrontables a une duree affichee : la note etablit
// que `min + (q + 0,5) * 0,05` se simplifie en `q * 0,05` sur les bornes du deserialiseur.
func TestNavpointMinuteurManuelDequantification(t *testing.T) {
	cas := []struct {
		q      uint64
		attend float32
	}{
		{0, 0},            // et la zone morte y ramene aussi les valeurs infimes
		{1, 0.05},         // le pas
		{20, 1},           // une seconde
		{240, 12},         // « le drapeau revient dans 0:12 »
		{600, 30},         // la duree de retour d'un drapeau de CTF
		{131071, 6553.55}, // le quantum le plus haut
	}
	for _, c := range cas {
		v := NavpointManualTimerValue(c.q)
		if d := v - c.attend; d > 1e-3 || d < -1e-3 {
			t.Errorf("quantum %d : %.6f s, %.6f s attendues", c.q, v, c.attend)
		}
	}
}

// TestNavpointMinuteurManuelPublie : les deux durees sortent sur le canal de `ti=12`, chacune
// sous son champ, et le quantum est publie BRUT — la dequantification est chez l'appelant.
func TestNavpointMinuteurManuelPublie(t *testing.T) {
	cas := []struct {
		compo string
		champ NavpointField
	}{
		{compNavpointManualTimerInitial, NavpointManualTimerInitial},
		{compNavpointManualTimerCurrent, NavpointManualTimerCurrent},
	}
	for _, c := range cas {
		t.Run(c.compo, func(t *testing.T) {
			const q = 240 // douze secondes
			var vu []uint64
			var champ NavpointField
			prev := observateur.NavpointHook
			SetNavpointHook(func(f NavpointField, values []uint64) {
				champ, vu = f, append([]uint64(nil), values...)
			})
			defer SetNavpointHook(prev)

			n, porte := consommerNavpoint(t, c.compo, fluxDeBits(func(w *bitw) {
				w.put(q, navpointManualTimerBits)
			}))
			if !porte || n != navpointManualTimerBits {
				t.Fatalf("%d bits consommes (porte=%v), %d attendus", n, porte, navpointManualTimerBits)
			}
			if champ != c.champ {
				t.Fatalf("champ publie %v, attendu %v", champ, c.champ)
			}
			if len(vu) != 1 || vu[0] != q {
				t.Fatalf("valeurs publiees %v, attendu [%d]", vu, q)
			}
		})
	}
}

// TestNavpointTexteFormateReutiliseLeSacDeLObjectif : `i9` est une boucle `R(8)` autour du sac
// texte de `ti=11 i2`. Le test mesure les deux bornes — zero entree, et une entree presente.
func TestNavpointTexteFormateReutiliseLeSacDeLObjectif(t *testing.T) {
	// Zero entree : le compte seul.
	n, porte := consommerNavpoint(t, compNavpointFormattedText, fluxDeBits(func(w *bitw) {
		w.put(0, navpointTextEntriesBits)
	}))
	if !porte || n != navpointTextEntriesBits {
		t.Fatalf("compte nul : %d bits (porte=%v), %d attendus", n, porte, navpointTextEntriesBits)
	}
	// Une entree, texte present, un argument de tag 3 (R(32)) :
	// 8 (compte) + 32 (textStringId) + 1 (presence) + 32 (identifiant) + 3 (compte) + 3 + 32.
	n, porte = consommerNavpoint(t, compNavpointFormattedText, fluxDeBits(func(w *bitw) {
		w.put(1, navpointTextEntriesBits)
		w.put(0, navpointTextIDBits)
		w.put(1, 1)
		w.put(0, 32)
		w.put(1, 3)
		w.put(3, 3)
		w.put(0, 32)
	}))
	if !porte || n != 8+32+1+32+3+3+32 {
		t.Fatalf("une entree : %d bits (porte=%v), %d attendus", n, porte, 8+32+1+32+3+3+32)
	}
}
