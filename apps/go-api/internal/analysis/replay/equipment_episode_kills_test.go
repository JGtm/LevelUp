package replay

import "testing"

// equipment_episode_kills_test.go — la jointure épisodes x kills, sur données 100 %
// synthétiques (aucun film, aucune base) : PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.1.

// ekEp est un raccourci pour construire un épisode de test — Slot/Fam fixes par défaut,
// seuls T0/T1 varient d'un cas à l'autre.
func ekEp(t0, t1 int) EquipmentEpisode {
	return EquipmentEpisode{Slot: 512, Fam: EquipFamilyCamo, T0: t0, T1: t1}
}

func TestAttachEpisodeKills_FragDansLaFenetre(t *testing.T) {
	eps := []EquipmentEpisode{ekEp(10, 20)}
	slotXUID := map[uint32]uint64{512: 111}
	// originMs=0, interval=100 : TimeMS=1500 -> frame 15, dans [10,20].
	attachEpisodeKills(eps, []EquipmentKillRef{{XUID: 111, TimeMS: 1500}}, slotXUID, 0, 100)
	if eps[0].K != 1 {
		t.Fatalf("K = %d, attendu 1 (frag dans la fenêtre)", eps[0].K)
	}
}

func TestAttachEpisodeKills_FragHorsFenetre(t *testing.T) {
	eps := []EquipmentEpisode{ekEp(10, 20)}
	slotXUID := map[uint32]uint64{512: 111}
	// frame 25 : après T1.
	attachEpisodeKills(eps, []EquipmentKillRef{{XUID: 111, TimeMS: 2500}}, slotXUID, 0, 100)
	if eps[0].K != 0 {
		t.Fatalf("K = %d, attendu 0 (frag hors fenêtre)", eps[0].K)
	}
}

func TestAttachEpisodeKills_BornesExactesT0EtT1(t *testing.T) {
	slotXUID := map[uint32]uint64{512: 111}

	epsT0 := []EquipmentEpisode{ekEp(10, 20)}
	attachEpisodeKills(epsT0, []EquipmentKillRef{{XUID: 111, TimeMS: 1000}}, slotXUID, 0, 100) // frame 10
	if epsT0[0].K != 1 {
		t.Errorf("borne T0 : K = %d, attendu 1 (bornes incluses)", epsT0[0].K)
	}

	epsT1 := []EquipmentEpisode{ekEp(10, 20)}
	attachEpisodeKills(epsT1, []EquipmentKillRef{{XUID: 111, TimeMS: 2000}}, slotXUID, 0, 100) // frame 20
	if epsT1[0].K != 1 {
		t.Errorf("borne T1 : K = %d, attendu 1 (bornes incluses)", epsT1[0].K)
	}

	epsApresT1 := []EquipmentEpisode{ekEp(10, 20)}
	attachEpisodeKills(epsApresT1, []EquipmentKillRef{{XUID: 111, TimeMS: 2100}}, slotXUID, 0, 100) // frame 21
	if epsApresT1[0].K != 0 {
		t.Errorf("juste après T1 : K = %d, attendu 0", epsApresT1[0].K)
	}
}

func TestAttachEpisodeKills_PorteurDifferentDuTueur(t *testing.T) {
	// L'épisode appartient au slot 512, porté par le xuid 111. Le frag est crédité à 222 :
	// un AUTRE joueur a tué pendant que 111 était sous camo — 111 n'a rien fait, aucun crédit.
	eps := []EquipmentEpisode{ekEp(10, 20)}
	slotXUID := map[uint32]uint64{512: 111}
	attachEpisodeKills(eps, []EquipmentKillRef{{XUID: 222, TimeMS: 1500}}, slotXUID, 0, 100)
	if eps[0].K != 0 {
		t.Fatalf("K = %d, attendu 0 (porteur de l'épisode != tueur crédité)", eps[0].K)
	}
}

func TestAttachEpisodeKills_AssistanceConnueCrediteA(t *testing.T) {
	// Deux épisodes : le tueur (111, slot 512) et l'assistant (222, slot 700) sont TOUS
	// LES DEUX sous effet actif au même instant — chacun reçoit son propre compteur.
	epTueur := ekEp(10, 20)
	epAssist := EquipmentEpisode{Slot: 700, Fam: EquipFamilyOvershield, T0: 10, T1: 20}
	eps := []EquipmentEpisode{epTueur, epAssist}
	slotXUID := map[uint32]uint64{512: 111, 700: 222}
	attachEpisodeKills(eps, []EquipmentKillRef{
		{XUID: 111, TimeMS: 1500, AssistXUID: 222, AssistKnown: true},
	}, slotXUID, 0, 100)
	if eps[0].K != 1 {
		t.Errorf("K du tueur = %d, attendu 1", eps[0].K)
	}
	if eps[1].A != 1 {
		t.Errorf("A de l'assistant = %d, attendu 1", eps[1].A)
	}
	if eps[0].A != 0 || eps[1].K != 0 {
		t.Errorf("croisement K/A inattendu : tueur=%+v assistant=%+v", eps[0], eps[1])
	}
}

func TestAttachEpisodeKills_AssistanceInconnueNeCrediteRien(t *testing.T) {
	// AssistKnown=false : on ne sait pas s'il y avait un assistant. Poser un XUID quand
	// même dans le champ serait une erreur d'appelant, mais la jointure doit rester sourde
	// à AssistXUID tant que AssistKnown est faux — jamais un crédit sur un « peut-être ».
	epAssist := EquipmentEpisode{Slot: 700, Fam: EquipFamilyOvershield, T0: 10, T1: 20}
	eps := []EquipmentEpisode{ekEp(10, 20), epAssist}
	slotXUID := map[uint32]uint64{512: 111, 700: 222}
	attachEpisodeKills(eps, []EquipmentKillRef{
		{XUID: 111, TimeMS: 1500, AssistXUID: 222, AssistKnown: false},
	}, slotXUID, 0, 100)
	if eps[0].K != 1 {
		t.Errorf("K du tueur = %d, attendu 1", eps[0].K)
	}
	if eps[1].A != 0 {
		t.Errorf("A de l'assistant = %d, attendu 0 (AssistKnown=false)", eps[1].A)
	}
}

func TestAttachEpisodeKills_EpisodeFermeParLaMortNeCompterPasApres(t *testing.T) {
	// Épisode ouvert à la mort (EndRead=false) : T1 est la fin de vie, pas une transition
	// mesurée. Un frag après T1 n'existe pas pour ce porteur mort — il ne compte pas.
	eps := []EquipmentEpisode{{Slot: 512, Fam: EquipFamilyCamo, T0: 10, T1: 42, EndRead: false}}
	slotXUID := map[uint32]uint64{512: 111}
	attachEpisodeKills(eps, []EquipmentKillRef{{XUID: 111, TimeMS: 5000}}, slotXUID, 0, 100) // frame 50 > 42
	if eps[0].K != 0 {
		t.Fatalf("K = %d, attendu 0 (frag après la fin de vie qui ferme l'épisode)", eps[0].K)
	}
}

func TestAttachEpisodeKills_OrigineNonNulleEtConversionHorloge(t *testing.T) {
	// originMs != 0 : vérifie replayMs = TimeMS - originMs, PAS TimeMS seul.
	eps := []EquipmentEpisode{ekEp(10, 20)}
	slotXUID := map[uint32]uint64{512: 111}
	// TimeMS=4500, originMs=3000 -> replayMs=1500 -> frame 15, dans [10,20].
	attachEpisodeKills(eps, []EquipmentKillRef{{XUID: 111, TimeMS: 4500}}, slotXUID, 3000, 100)
	if eps[0].K != 1 {
		t.Fatalf("K = %d, attendu 1 (conversion TimeMS-originMs)", eps[0].K)
	}
}

func TestAttachEpisodeKills_MemeFragCrediteDeuxFamillesSimultanees(t *testing.T) {
	// Le même porteur a camo ET surbouclier actifs en même temps (deux Track distincts
	// n'existent pas ici : c'est le MÊME slot qui porte deux épisodes de familles
	// différentes, ce que la machine à états d'equipment_episodes.go autorise).
	camo := ekEp(10, 20)
	overshield := EquipmentEpisode{Slot: 512, Fam: EquipFamilyOvershield, T0: 5, T1: 30}
	eps := []EquipmentEpisode{camo, overshield}
	slotXUID := map[uint32]uint64{512: 111}
	attachEpisodeKills(eps, []EquipmentKillRef{{XUID: 111, TimeMS: 1500}}, slotXUID, 0, 100)
	if eps[0].K != 1 || eps[1].K != 1 {
		t.Fatalf("K camo=%d K surbouclier=%d, attendu 1 et 1 (les deux familles actives)", eps[0].K, eps[1].K)
	}
}

func TestAttachEpisodeKills_ListesVidesNePaniquentPas(t *testing.T) {
	attachEpisodeKills(nil, nil, nil, 0, 100)
	attachEpisodeKills([]EquipmentEpisode{ekEp(0, 10)}, nil, map[uint32]uint64{512: 111}, 0, 100)
	attachEpisodeKills(nil, []EquipmentKillRef{{XUID: 111, TimeMS: 100}}, map[uint32]uint64{512: 111}, 0, 100)
}

func TestAttachAllEquipmentKills_LectureNonTentee(t *testing.T) {
	eps := []EquipmentEpisode{ekEp(10, 20)}
	slotXUID := map[uint32]uint64{512: 111}
	kills := KillsInput{Read: false, Kills: []EquipmentKillRef{{XUID: 111, TimeMS: 1500}}}
	read := attachAllEquipmentKills(eps, kills, slotXUID, int64Ptr(0), 100)
	if read {
		t.Fatalf("Read=false doit rendre killsRead=false, quels que soient les EquipmentKillRef fournis")
	}
	if eps[0].K != 0 {
		t.Fatalf("K = %d, attendu 0 : aucune jointure ne doit avoir lieu quand Read=false", eps[0].K)
	}
}

func TestAttachAllEquipmentKills_OrigineNonEtablie(t *testing.T) {
	eps := []EquipmentEpisode{ekEp(10, 20)}
	slotXUID := map[uint32]uint64{512: 111}
	kills := KillsInput{Read: true, Kills: []EquipmentKillRef{{XUID: 111, TimeMS: 1500}}}
	read := attachAllEquipmentKills(eps, kills, slotXUID, nil, 100)
	if read {
		t.Fatalf("origine nil doit rendre killsRead=false : la conversion TimeMS -> frame ne veut rien dire sans elle")
	}
	if eps[0].K != 0 {
		t.Fatalf("K = %d, attendu 0 : sans origine, aucun frag ne doit être joint", eps[0].K)
	}
}

func TestAttachAllEquipmentKills_LectureReussie(t *testing.T) {
	eps := []EquipmentEpisode{ekEp(10, 20)}
	slotXUID := map[uint32]uint64{512: 111}
	kills := KillsInput{Read: true, Kills: []EquipmentKillRef{{XUID: 111, TimeMS: 1500}}}
	read := attachAllEquipmentKills(eps, kills, slotXUID, int64Ptr(0), 100)
	if !read {
		t.Fatalf("killsRead doit être vrai quand Read=true et l'origine est établie")
	}
	if eps[0].K != 1 {
		t.Fatalf("K = %d, attendu 1", eps[0].K)
	}
}

func int64Ptr(v int64) *int64 { return &v }
