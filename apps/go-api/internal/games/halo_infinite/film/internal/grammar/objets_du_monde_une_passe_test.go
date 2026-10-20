package grammar

// objets_du_monde_une_passe_test.go — LES OBJETS DU MONDE RELEVES EN UNE PASSE (lot 2.5 du plan de
// l etape 2) : la passe sur l union des bandes rend, bande par bande, les pistes du balayage d une
// bande seule ; la passe des creations rend, archetype par archetype, les creations de la marche
// d un archetype seul ; ce que le contexte memorise se rend en copie, et ne se reutilise que sous
// les memes entrees.

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// contexteDesObjetsDuMonde ouvre la mini-bobine contigue de `killsource` (registre, images-cles et
// trames delta d un match reel) et rend son contexte et les bornes du film 000d5950.
func contexteDesObjetsDuMonde(t *testing.T) (*FilmContext, profile.Vec3Range) {
	t.Helper()
	film, err := source.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	return contexteDeBobine(film), profile.QuantRangeCEBiped()
}

// TestLesPistesDUnePasseSontCellesDesBandesSeules : sur chaque payload delta de la bobine, la passe
// sur l union des bandes a pistes rend, bande par bande, les echantillons du balayage de cette bande
// seule.
// MUTATION — la passe donne chaque record accepte a toutes les bandes, sans regarder si elles
// portent son slot : ROUGE.
func TestLesPistesDUnePasseSontCellesDesBandesSeules(t *testing.T) {
	fc, wr := contexteDesObjetsDuMonde(t)
	lg := fc.ProfilDeBalayage().LargeursObjetDuMonde()
	var bandes [][]uint32
	var cartes []map[uint32]bool
	for _, ti := range archetypesAPistes() {
		b := worldObjectSlotBand(fc, ti)
		bandes, cartes = append(bandes, slotsDeLaBande(b)), append(cartes, b)
	}
	appartenance := appartenanceDesBandes(bandes)
	echantillons := 0
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			unePasse := echantillonsDesBandes(pay, appartenance, len(bandes), &wr, lg)
			for i, carte := range cartes {
				seule := scanProjectileRecords(pay, carte, &wr, lg)
				if !reflect.DeepEqual(unePasse[i], seule) {
					t.Fatalf("chunk %d paquet %d, bande %d : %d echantillon(s) en une passe, %d seule", c, pk.Index, i,
						len(unePasse[i]), len(seule))
				}
				echantillons += len(seule)
			}
		}
	}
	if echantillons == 0 {
		t.Fatal("aucun echantillon d objet du monde sur la bobine : le test ne prouve rien")
	}
	t.Logf("%d echantillon(s) identiques, %d bande(s)", echantillons, len(bandes))
}

// marcheDUnArchetype est l ORACLE du test suivant : la marche d un archetype seul, comme elle
// s ecrivait avant la passe commune — un curseur, l en-tete reconnu en entier a chaque bit.
func marcheDUnArchetype(fc *FilmContext, w equipCreationWalk) ([]types.EquipmentCreation,
	types.EquipmentCreationStats) {
	var out []types.EquipmentCreation
	st := types.EquipmentCreationStats{Slots: len(w.band)}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			for p := 0; p <= len(pay)*8-woNewHeaderBits; p++ {
				slot, gen, ok := matchWorldObjectNewHeader(pay, p, w.band, w.archetype())
				if !ok {
					continue
				}
				cre, ok := w.creationA(pay, p, types.LifeKey{Slot: slot, Gen: gen}, lieuDuPaquet{chunk: c, pk: pk}, &st)
				if !ok {
					continue
				}
				out = append(out, cre)
				p = cre.AfterBit - 1
			}
		}
	}
	return out, st
}

// TestLesCreationsDUnePasseSontCellesDesArchetypesSeuls : la passe des creations rend, archetype
// par archetype, les creations et les comptes de la marche de cet archetype seul (l oracle
// [marcheDUnArchetype]).
// MUTATION — les curseurs de la passe ne sont pas remis a zero d un payload au suivant : ROUGE.
func TestLesCreationsDUnePasseSontCellesDesArchetypesSeuls(t *testing.T) {
	fc, wr := contexteDesObjetsDuMonde(t)
	var marches []equipCreationWalk
	for _, ti := range archetypesDeCreation() {
		band := worldObjectSlotBand(fc, int(ti))
		if len(band) == 0 {
			continue
		}
		w, err := fc.marcheDeCreation(ti, &wr, band)
		if err != nil {
			t.Fatalf("marche de creation ti=%d : %v", ti, err)
		}
		marches = append(marches, w)
	}
	if len(marches) < 2 {
		t.Fatalf("%d archetype(s) de creation sur la bobine : la passe commune n est pas exercee", len(marches))
	}
	ensemble, comptes := releverLesCreations(fc, marches)
	acceptees := 0
	for i, w := range marches {
		seule, compte := marcheDUnArchetype(fc, w)
		if !reflect.DeepEqual(ensemble[i], seule) || comptes[i] != compte {
			t.Fatalf("ti=%d : %d creation(s) en une passe (%+v), %d seule (%+v)", w.archetype(), len(ensemble[i]),
				comptes[i], len(seule), compte)
		}
		acceptees += compte.Accepted
	}
	if acceptees == 0 {
		t.Fatal("aucune creation acceptee sur la bobine : le test ne prouve rien")
	}
	t.Logf("%d creation(s) identiques, %d archetype(s)", acceptees, len(marches))
}

// TestLesObjetsDuMondeReleveSeRendentEnCopie : deux demandes de la meme bande rendent les memes
// pistes, et modifier ce que rend la premiere ne change pas la seconde ; meme regle pour les
// creations, qui ne se reutilisent que sous le meme profil.
// MUTATION — rendre les pistes relevees sans les copier : ROUGE.
func TestLesObjetsDuMondeReleveSeRendentEnCopie(t *testing.T) {
	fc, wr := contexteDesObjetsDuMonde(t)
	band := worldObjectSlotBand(fc, EquipmentTypeIndex)
	premieres, err := ScanWorldObjectsForBand(fc, &wr, EquipmentTypeIndex, band)
	if err != nil || len(premieres) == 0 || len(premieres[0].Pts) == 0 {
		t.Fatalf("pistes d equipement : %v (%d piste(s))", err, len(premieres))
	}
	attendues := copierLesPistes(premieres)
	premieres[0].Pts[0].X, premieres[0] = premieres[0].Pts[0].X+1, types.ProjectileTrack{}
	secondes, err := ScanWorldObjectsForBand(fc, &wr, EquipmentTypeIndex, band)
	if err != nil || !reflect.DeepEqual(secondes, attendues) {
		t.Fatalf("seconde demande de pistes : %v, differente de la premiere avant modification", err)
	}
	cre, st, err := ScanEquipmentCreationsForBand(fc, &wr, band)
	if err != nil || st.Accepted == 0 {
		t.Fatalf("creations d equipement : %v (%+v)", err, st)
	}
	avant := len(fc.recup.creations)
	cre2, st2, _ := ScanEquipmentCreationsForBand(fc, &wr, band)
	if len(fc.recup.creations) != avant || !reflect.DeepEqual(cre, cre2) || st != st2 {
		t.Fatalf("seconde demande de creations : %d marche(s) memorisee(s) au lieu de %d, ou un resultat different",
			len(fc.recup.creations), avant)
	}
	mpp := fc.ProfilDeBalayage().MPP
	prev := fc.PoserMPP(profile.MPPWidths{Lead: mpp.Lead + 1, Index: mpp.Index})
	defer fc.PoserMPP(prev)
	if _, _, err := ScanEquipmentCreationsForBand(fc, &wr, band); err != nil || len(fc.recup.creations) == avant {
		t.Fatalf("creations sous un autre profil : %v ; aucune marche nouvelle (%d memorisee(s))", err,
			len(fc.recup.creations))
	}
}

// releverLesObjetsDUnPayloadQuelconque passe `pay` dans les deux passes des objets du monde, sous la
// bande `band` (pistes : la bande et une seconde vide ; creations : l equipement et les armes au sol)
// — le contrat du harnais [FuzzFilmRecordReaders] : aucune panique, quelle que soit l entree.
func releverLesObjetsDUnPayloadQuelconque(pay []byte, band map[uint32]bool, wr *profile.Vec3Range) {
	lg := ProfilDeBalayageParDefaut().LargeursObjetDuMonde()
	bandes := [][]uint32{slotsDeLaBande(band), nil}
	_ = echantillonsDesBandes(pay, appartenanceDesBandes(bandes), len(bandes), wr, lg)
	var marches []equipCreationWalk
	for _, ti := range []uint32{EquipmentTypeIndex, GroundWeaponTypeIndex} {
		cur := &equipCreationRead{}
		marches = append(marches, equipCreationWalk{obs: installCreationHooks(cur), prof: ProfilDeBalayageParDefaut(),
			comps: 40, wr: wr, band: band, cur: cur, ti: ti})
	}
	nouvellePasseDesCreations(marches).payload(pay, lieuDuPaquet{pk: FilmPacket{Type: PacketTypeDelta, Size: len(pay)}})
}
