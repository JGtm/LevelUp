package filmdec

// victime_degat_recu_research_test.go — LOT 1 : le bipède (ti=35) réplique-t-il, CÔTÉ VICTIME,
// un champ « dernier dégât reçu / dernier attaquant » — à la Unit_GetReceivedDamage_WeakDamageOwner
// du moteur — mis à jour à CHAQUE coup (y compris explosif) et NON seulement à la mort ?
//
// POURQUOI CETTE QUESTION. Le moteur EXPOSE bien une telle notion : Ghidra donne une famille
// entière d'accesseurs de reflection/script (table de noms bâtie par FUN_140e61df8), lisibles
// bruts en .rdata à 0x143c58e10..0x143c58fc8 :
//
//	Unit_GetReceivedDamage_WeakDamageOwnerObject   (0x143c58f78)  <- l'INSTIGATEUR (objet)
//	Unit_GetReceivedDamage_WeakDamageOwnerPlayer   (0x143c58e10)  <- l'INSTIGATEUR (joueur)
//	Unit_GetReceivedDamage_WeakDamageSourceObject  (0x143c58e40)  <- la SOURCE (arme/projectile)
//	Unit_GetReceivedDamage_TimeStamp               (0x143c58ee8)
//	Unit_GetReceivedDamage_Normalized              (0x143c58f50)
//	Unit_GetReceivedDamage_Body                    (0x143c58fa8)
//	Unit_GetReceivedDamage_Shield                  (0x143c58fc8)
//
// C'est EXACTEMENT le struct rêvé : un « dernier dégât » qui distingue l'OWNER (le tireur) de la
// SOURCE (l'arme ou le projectile) — donc capable d'attribuer une touche explosive non-fatale à
// son tireur, sans passer par le projectile. MAIS ces noms sont des GETTERS de RUNTIME (HUD,
// télémétrie, médailles) ; leur existence ne prouve PAS que le struct est RÉPLIQUÉ dans le film.
//
// CE QUE CET INSTRUMENT ÉTABLIT, en deux parties, contre la vérité terrain que réclame la garde :
//
//	A. SCHÉMA (déterministe, lu dans le registre du film lui-même) : l'archétype bipède réplique
//	   EXACTEMENT 64 composants (i0..i63) ; on les énumère et on montre qu'AUCUN ne porte un nom
//	   de « received-damage / damage-owner / last-attacker / recent-damage / aftermath ». Les
//	   seuls composants liés au dégât sont i4 (santé), i5 (bouclier), i6 (régions), i7 (sections)
//	   et i11 (dead-state) — et i11 est le SEUL à porter une référence d'entité, celle de LA MORT.
//
//	B. FRÉQUENCE (corpus) : le composant dead-state i11 — unique porteur d'un attaquant sur le
//	   bipède — est-il présent dans le record PER-TICK (celui qui porte position/vitalité/visée),
//	   ou seulement à la mort ? On balaie les records bipèdes et on mesure la présence d'i11 dans
//	   leur masque, comparée au nombre de touches (événements damage_aftermath 0xC0 type 0). Si
//	   i11 est quasi absent des records per-tick alors que les touches sont nombreuses, le bipède
//	   ne réplique AUCUN attaquant non-fatal : l'attribution non-fatale reste hors du record répliqué.
//
// DISTINCTION CLÉ (garde-fou) : « composant répliqué » (dans les 64 du bipède, décodable image
// par image) vs « valeur runtime » (le struct Unit_GetReceivedDamage, vivant en RAM, sérialisé
// dans le film seulement comme l'ÉVÉNEMENT damage_aftermath, jamais comme un champ du bipède).
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borné à
// deltaWitnessChunks (12). Lancer une fois par film (000d5950, 01e1f945, 00502e52).

import (
	"os"
	"regexp"
	"testing"
)

// Indices de composant du bipède liés au dégât (registre ti=35, cf. testdata/ecs_table.tsv).
const (
	vdrBodyVitalityIdx   = 4  // object-body-vitality-component : la SANTE (valeur seule)
	vdrShieldVitalityIdx = 5  // object-shield-vitality-component : le BOUCLIER (valeur seule)
	vdrRegionStateIdx    = 6  // object-region-state-component : quelle REGION du corps abimee
	vdrDamageSectionsIdx = 7  // object-damage-sections-component : niveaux de degat par SECTION
	vdrDeadStateIdx      = 11 // object-dead-state-component : LA MORT + reference vers le tueur
)

// vdrAttackerNameRe repère un nom de composant qui dénoterait un champ « dégât reçu / attaquant /
// instigateur ». La liste des motifs est celle des noms Ghidra (received-damage, damage-owner,
// weak-damage-owner) ÉLARGIE aux synonymes plausibles du schéma de réplication.
var vdrAttackerNameRe = regexp.MustCompile(
	`received.?damage|damage.?owner|weak.?damage|last.?attacker|recent.?damage|aftermath|instigator|attacker`)

// TestVictimeSchemaAucunChampAttaquant (partie A) énumère les composants répliqués du bipède et
// prouve qu'aucun n'est un champ « dernier attaquant ». DÉTERMINISTE : ne dépend que du registre.
func TestVictimeSchemaAucunChampAttaquant(t *testing.T) {
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
	arch, ok := reg.Archetype(BipedTypeIndex)
	if !ok {
		t.Fatalf("archetype bipede (ti=%d) absent du registre (%d archetypes)", BipedTypeIndex, len(reg.Archetypes))
	}
	t.Logf("== SCHEMA bipede ti=%d : %d composants repliques (registre : %d archetypes) ==",
		BipedTypeIndex, len(arch.Components), len(reg.Archetypes))

	// (1) Aucun composant ne porte un nom d'attaquant / dégât reçu.
	var flagged []string
	for i, name := range arch.Components {
		if vdrAttackerNameRe.MatchString(name) {
			flagged = append(flagged, name)
			t.Logf("  ATTENTION i%-2d %s : nom evoquant un attaquant/degat-recu", i, name)
		}
	}
	if len(flagged) != 0 {
		t.Errorf("REFUTATION du negatif : %d composant(s) au nom d'attaquant : %v", len(flagged), flagged)
	} else {
		t.Logf("  AUCUN composant sur %d ne porte un nom received-damage/damage-owner/last-attacker",
			len(arch.Components))
	}

	// (2) Les composants liés au dégât sont exactement les 5 attendus, à leurs index, et un seul
	//     (i11) porte une référence — le tueur, à la mort.
	vdrAssertDamageComponent(t, arch, vdrBodyVitalityIdx, "object-body-vitality-component", "santé (valeur)")
	vdrAssertDamageComponent(t, arch, vdrShieldVitalityIdx, "object-shield-vitality-component", "bouclier (valeur)")
	vdrAssertDamageComponent(t, arch, vdrRegionStateIdx, "object-region-state-component", "régions (aucune réf)")
	vdrAssertDamageComponent(t, arch, vdrDamageSectionsIdx, "object-damage-sections-component", "sections (aucune réf)")
	vdrAssertDamageComponent(t, arch, vdrDeadStateIdx, "object-dead-state-component", "MORT : réf tueur (i11)")

	t.Logf("  CONCLUSION (schema) : le bipede ne replique AUCUN champ 'dernier degat recu / owner'.")
	t.Logf("  La famille Unit_GetReceivedDamage_WeakDamageOwner* est un struct RUNTIME (getters de")
	t.Logf("  reflection/script), non present dans les 64 composants repliques. Le seul porteur d'un")
	t.Logf("  attaquant est i11 dead-state, et c'est l'evenement de MORT — pas un champ per-coup.")
}

// vdrAssertDamageComponent vérifie qu'à l'index attendu le composant porte bien le nom attendu.
func vdrAssertDamageComponent(t *testing.T, arch Archetype, idx int, want, role string) {
	t.Helper()
	if idx >= len(arch.Components) {
		t.Errorf("i%d hors bornes (%d composants)", idx, len(arch.Components))
		return
	}
	got := arch.Components[idx]
	mark := "OK"
	if got != want {
		mark = "DIVERGENCE"
		t.Errorf("i%d : registre='%s' attendu='%s'", idx, got, want)
	}
	t.Logf("    i%-2d  %-40s  %-24s  [%s]", idx, got, role, mark)
}

// vdrMaskStats agrège la présence des composants dans les masques des records bipèdes.
type vdrMaskStats struct {
	records               int
	hasBody, hasShield    int
	hasRegion, hasSection int
	hasDeadState          int // i11 présent dans le masque d'un record per-tick
	deadWithVitality      int // i11 ET (i4 ou i5) dans le même record
	maskOver              int
}

// vdrScanBipedMasks balaie les records bipèdes per-tick (position + vitalité) et compte la
// présence de chaque composant-dégât dans leur masque. C'est l'infrastructure existante
// (ScanFilmBipedPositions/CaptureDirs) : MaskBits voyage dans le record, aucune lecture nouvelle.
func vdrScanBipedMasks(t *testing.T, dir string, chunks []int) vdrMaskStats {
	t.Helper()
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly = true // pas besoin des bornes de carte : on ne lit que les masques
	opt.CaptureDirs = true
	opt.IsolationGapMS = 0
	opt.Chunks = chunks
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("balayage biped impossible : %v", err)
	}
	var s vdrMaskStats
	for _, p := range pos {
		s.records++
		if p.MaskOver {
			s.maskOver++
		}
		body := p.MaskBits&(uint64(1)<<vdrBodyVitalityIdx) != 0
		shield := p.MaskBits&(uint64(1)<<vdrShieldVitalityIdx) != 0
		dead := p.MaskBits&(uint64(1)<<vdrDeadStateIdx) != 0
		if body {
			s.hasBody++
		}
		if shield {
			s.hasShield++
		}
		if p.MaskBits&(uint64(1)<<vdrRegionStateIdx) != 0 {
			s.hasRegion++
		}
		if p.MaskBits&(uint64(1)<<vdrDamageSectionsIdx) != 0 {
			s.hasSection++
		}
		if dead {
			s.hasDeadState++
			if body || shield {
				s.deadWithVitality++
			}
		}
	}
	return s
}

// vdrCountDamageEvents compte les touches : événements damage_aftermath (0xC0 type 0), sur les
// mêmes chunks. C'est l'ORACLE de « coup » : chaque touche non-fatale doit y figurer.
func vdrCountDamageEvents(t *testing.T, dir string, n int) int {
	t.Helper()
	hits := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xC0 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 0 {
				continue // type 1 (damage_section_response), pas type 0
			}
			hits++
		}
	}
	return hits
}

// TestVictimeDeadStateEstMortPasCoup (partie B) mesure, sur le corpus, la présence du dead-state
// i11 dans les records bipèdes per-tick, contre le nombre de touches. Vérité terrain : les
// touches (damage_aftermath) sont l'oracle du « coup » ; i11 est l'unique porteur d'attaquant.
func TestVictimeDeadStateEstMortPasCoup(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	chunks := make([]int, 0, n)
	for c := 1; c <= n; c++ {
		chunks = append(chunks, c)
	}

	s := vdrScanBipedMasks(t, dir, chunks)
	hits := vdrCountDamageEvents(t, dir, n)

	t.Logf("== CORPUS %s · %d chunks · %d records bipedes per-tick ==", vdrFilmID(dir), n, s.records)
	if s.records == 0 {
		t.Skipf("aucun record biped reconnu : mesure impossible")
	}
	t.Logf("  presence dans le masque du record per-tick (denominateur = %d records) :", s.records)
	t.Logf("    i4  sante     : %6d (%5.2f %%)", s.hasBody, vdrPct(s.hasBody, s.records))
	t.Logf("    i5  bouclier  : %6d (%5.2f %%)", s.hasShield, vdrPct(s.hasShield, s.records))
	t.Logf("    i6  regions   : %6d (%5.2f %%)", s.hasRegion, vdrPct(s.hasRegion, s.records))
	t.Logf("    i7  sections  : %6d (%5.2f %%)", s.hasSection, vdrPct(s.hasSection, s.records))
	t.Logf("    i11 dead-state: %6d (%5.2f %%)  <- UNIQUE porteur d'attaquant", s.hasDeadState, vdrPct(s.hasDeadState, s.records))
	t.Logf("        dont avec vitalite (i4/i5) dans le meme record : %d", s.deadWithVitality)
	if s.maskOver > 0 {
		t.Logf("  (masque tronque sur %d records — index >= 64, hors grammaire bipede)", s.maskOver)
	}
	t.Logf("  TOUCHES (oracle du coup) : %d evenements damage_aftermath (0xC0 type 0)", hits)

	// LECTURE. Si i11 (dead-state) est quasi absent des records per-tick alors que les touches sont
	// nombreuses, le bipede ne porte AUCUN attaquant a l'echelle du coup : l'unique champ porteur
	// d'un instigateur (i11) est un evenement de MORT, pas un etat replique per-coup. Le pendant
	// non-fatal recherche N'EXISTE PAS dans le record du bipede.
	t.Logf("  RAPPORT i11(per-tick) / touches : %d / %d", s.hasDeadState, hits)
	negatifProuve := hits >= 20 && s.hasDeadState <= hits/10
	t.Logf("  VERDICT (dead-state per-tick negligeable vs touches, <= 10 %% des touches) : %s",
		vdrVerdict(negatifProuve))
	t.Logf("  Attribution non-fatale cote victime : le film ne la porte que par l'EVENEMENT")
	t.Logf("  damage_aftermath (dont le 'responsable' pour un explosif est le PROJECTILE, cf.")
	t.Logf("  explo_touches), jamais par un champ replique du bipede. La mort reste couverte par i11.")
}

// vdrFilmID rend le dernier segment du chemin (l'identifiant du film) pour le journal.
func vdrFilmID(dir string) string {
	for i := len(dir) - 1; i >= 0; i-- {
		if dir[i] == '/' || dir[i] == '\\' {
			return dir[i+1:]
		}
	}
	return dir
}

// vdrPct rend num/den en pourcentage (0 si den nul).
func vdrPct(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

// vdrVerdict rend un libellé de verdict lisible au journal.
func vdrVerdict(ok bool) string {
	if ok {
		return "NEGATIF CONFIRME (aucun champ attaquant non-fatal replique)"
	}
	return "NON TRANCHE (echantillon insuffisant ou signal inattendu — lire les comptes)"
}
