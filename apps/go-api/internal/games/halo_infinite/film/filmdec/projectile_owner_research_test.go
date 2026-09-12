package filmdec

// projectile_owner_research_test.go — LOT 1 : RELIER UN PROJECTILE A SON TIREUR PAR UN CHAMP
// (l'owner/instigateur), pas par une fenetre temporelle.
//
// RAISONNEMENT (utilisateur, solide) : le jeu SAIT qui a blesse sans tuer (precision par arme +
// assistance aux armes lourdes). L'instigateur d'un degat de projectile est donc trace PAR DEGAT
// dans la simulation. Ghidra le confirme : l'objet porte un DamageOwner
// (`Object_Get/SetDamageOwnerObject/Player`, `Unit_GetReceivedDamage_WeakDamageOwnerObject/Player`,
// `Equipment_GetOwnerUnit`). La question OUVERTE : ce champ est-il REPLIQUE dans le film sur
// l'entite projectile (ti=41), ou n'existe-t-il qu'au runtime, materialise dans le film SEULEMENT
// a la mort (dead-state tueur +0x08) ?
//
// LE CANDIDAT CONCRET, NON TESTE AVANT : le composant i10 `object-parent-state` de ti=41. Sur un
// projectile EN VOL (branche libre, recordStateParam < 2) il lit, derriere une porte, un
// identifiant a largeur variable (FUN_1408f0ac4 -> FUN_1406d3140) : un index R(13), le MEME espace
// de handle que les bipedes (dom1). Si cet id pointe le tireur, c'est le lien de champ cherche.
// (La valeur etait jetee jusqu'ici ; on la publie via ObjectParentState.FreeID.)
//
// VERITE TERRAIN (le point fort) : pour les KILLS, le tueur est deja resolu par le dead-state i11
// de la victime (EnumB = killer-absolute-participant-index, 97,6 %). M2 le re-mesure comme ANCRE :
// il PROUVE qu'un lien de champ projectile->tueur existe AU MOINS a la mort.
//
// MESURES :
//   M1 — CENSUS i10 sur ti=41 : combien de lectures, part en branche libre, part a porte ouverte
//        (id transmis). Pour les ids transmis : cardinalite (owner => poignee de valeurs),
//        resolution en bipede (balayage de bases), stabilite par vie (un owner est constant).
//   M2 — ORACLE dead-state : les tueurs (EnumB) sur les memes chunks — compte et cardinalite
//        (<= roster). Ancre du lien de champ a la mort.
//   M3 — PONT : un damage_aftermath explosif (ref1 non-bipede) resout-il ref1 a un slot ti=41
//        vivant, condition necessaire pour remonter du degat au projectile puis a son owner ?
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borne a
// deltaWitnessChunks (12). Lancer une fois par film (000d5950, 01e1f945, 00502e52).
// Collecteurs et utilitaires : projectile_owner_helpers_test.go.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectileOwner(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	// Precision des objets du monde (largeurs de position ti=41) : sans elle les records
	// projectile desynchronisent plus tot et la sonde i10 rend moins.
	if lay, _, err := DetectI0Layout(dir); err == nil {
		prev := WorldObjectPrecision
		t.Cleanup(func() { WorldObjectPrecision = prev })
		SetWorldObjectPrecisionFromLayout(lay)
	}

	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	t.Logf("== film %s · %d chunks · ti projectile=%d · base bipede reference %d ==",
		filepath.Base(dir), n, ProjectileTypeIndex, lot1chReferenceBase)

	coll := projOwnerCollect(t, dir, reg, n)
	t.Logf("collecte : %d lectures i10 sur ti=41 · %d kills (dead-state) · %d degats explosifs (ref1 non-bipede)",
		len(coll.reads), len(coll.kills), coll.dmgExplo)

	projOwnerM1b(t, dir, n)
	projOwnerM1(t, coll)
	projOwnerM2(t, coll)
	projOwnerM3(t, coll)
}

// projOwnerM1b : censure ROBUSTE des masques ti=41 (independante du rendement des trames
// propres). Combien de records de projectile portent i10 (parent-state, ou vivrait l'owner
// candidat) et i9 (multiplayer-properties) ? Si i10 est quasi absent, l'avenue est close par
// simple rarete, sans meme decoder le champ.
func projOwnerM1b(t *testing.T, dir string, n int) {
	t.Helper()
	total, hist := projOwnerMaskCensus(t, dir, n)
	t.Logf("M1b CENSUS ROBUSTE des masques ti=41 (%d records de projectile ancres sur i0) :", total)
	if total == 0 {
		t.Logf("   aucun record ti=41 ancre : bande de slots vide ou position illisible.")
		return
	}
	t.Logf("   histogramme index de composant :%s", projOwnerMaskLine(hist, total))
	t.Logf("   i9 (multiplayer-properties) : %d (%.1f %%) · i10 (parent-state) : %d (%.1f %%)",
		hist[9], lot1Pct(hist[9], total), hist[10], lot1Pct(hist[10], total))
}

// projOwnerM1 : le champ owner candidat existe-t-il sur le projectile en vol ?
func projOwnerM1(t *testing.T, coll projOwnerColl) {
	t.Helper()
	reads := coll.reads
	var free, withID, attached int
	var ids []uint64
	var word16s []uint64
	for _, r := range reads {
		if r.attached {
			attached++
			word16s = append(word16s, uint64(r.word16))
		}
		if r.freeRead() {
			free++
		}
		if r.hasFreeID {
			withID++
			ids = append(ids, r.freeID)
		}
	}
	t.Logf("M1 CENSUS i10 (object-parent-state) sur %d lectures ti=41 :", len(reads))
	t.Logf("   branche attachee : %d · branche libre (en vol) : %d · id transmis (porte ouverte) : %d",
		attached, free, withID)
	if withID == 0 {
		t.Logf("   -> AUCUN id d'owner transmis en vol : le projectile ne replique pas son tireur par i10.")
	} else {
		distinct := projOwnerDistinct(ids)
		t.Logf("   ids transmis : %d valeurs distinctes (owner attendu <= roster) ; top :%s",
			distinct, projOwnerTopVals(ids, 8))
		lives, stable := projOwnerPerLifeStable(reads)
		t.Logf("   stabilite par vie : %d/%d vies (>=2 lectures) portent un id CONSTANT", stable, lives)
		// Resolution en bipede : on cherche la base qui maximise les atterrissages bipede.
		// (Le monde ici est celui de fin de passe ; c'est un indicateur, pas la resolution
		// chrono. Un owner devrait atterrir massivement sur la bande bipede.)
		wReg := projOwnerFinalWorld(t)
		if wReg != nil {
			base, hits := projOwnerBestBaseBiped(wReg, ids)
			t.Logf("   resolution : base %d maximise a %d/%d ids -> bipede (indicatif fin de passe)",
				base, hits, len(ids))
		}
	}
	if attached > 0 {
		t.Logf("   word16 (branche attachee) : %d valeurs distinctes ; top :%s",
			projOwnerDistinct(word16s), projOwnerTopVals(word16s, 8))
	}
	t.Logf("   LECTURE : un champ owner de projectile serait a porte OUVERTE en vol, de faible")
	t.Logf("   cardinalite (<= roster), CONSTANT par vie, et resolvant en bipede. A defaut, i10")
	t.Logf("   ne porte pas l'owner et le lien de champ n'est pas sur le projectile vivant.")
}

// projOwnerM2 : ancre de verite terrain — le tueur (dead-state EnumB) est un champ de faible
// cardinalite, le lien de champ PROUVE a la mort.
func projOwnerM2(t *testing.T, coll projOwnerColl) {
	t.Helper()
	var killers []uint64
	for _, k := range coll.kills {
		killers = append(killers, uint64(k.killer))
	}
	t.Logf("M2 ORACLE dead-state (tueur EnumB = killer-absolute-participant-index) :")
	if len(killers) == 0 {
		t.Logf("   aucun dead-state de bipede mort cleanement decode sur ces %d chunks.", deltaWitnessChunks)
		return
	}
	t.Logf("   %d morts · %d tueurs distincts (attendu <= roster) ; top :%s",
		len(killers), projOwnerDistinct(killers), projOwnerTopVals(killers, 8))
	t.Logf("   LECTURE : ce champ EST le lien projectile->tireur, mais il n'existe QU'A LA MORT.")
}

// projOwnerM3 : peut-on remonter d'un degat explosif au projectile (donc a son owner s'il existe) ?
func projOwnerM3(t *testing.T, coll projOwnerColl) {
	t.Helper()
	t.Logf("M3 PONT degat explosif -> projectile :")
	t.Logf("   %d degats a ref1 non-bipede (candidats explosifs) · %d resolvent ref1 a un slot ti=41 vivant",
		coll.dmgExplo, coll.dmgToTi41)
	if coll.dmgExplo > 0 {
		t.Logf("   -> %.1f %% des degats explosifs pointent une entite projectile encore liee au monde.",
			lot1Pct(coll.dmgToTi41, coll.dmgExplo))
	}
	t.Logf("   LECTURE : ref1 = le PROJECTILE (dom1), pas le tireur ; le projectile est transitoire")
	t.Logf("   (detruit a la detonation), d'ou une resolution faible en fin de passe.")
}

// freeRead : la lecture a-t-elle emprunte la branche libre (en vol) ? On la deduit de hasFreeID
// (porte ouverte) OU d'une branche non-attachee sans id (porte fermee). attached la distingue.
func (r projOwnerRead) freeRead() bool { return !r.attached }

// projOwnerFinalWorld reconstruit le monde de fin de passe (bind de toutes les keyframes) pour
// l'indicateur de resolution de M1. Rendu nil si indisponible (M1 saute alors la resolution).
func projOwnerFinalWorld(t *testing.T) *World {
	t.Helper()
	dir := os.Getenv(lot1TrameFilmEnv)
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return nil
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	w := NewWorld(reg)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
	}
	return w
}
