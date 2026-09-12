package filmdec

import (
	"fmt"
	"os"
	"testing"
)

// statborg_chain_count_test.go — le CONTROLE CROISE de la decision D1 du plan
// `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md` (lot A, phase 0).
//
// # La question, et ce qu'elle decide
//
// Le score dans le temps a DEUX voies possibles vers les memes octets :
//
//	l'ANCRAGE  `objectiveevents.StatRecords` localise chaque enregistrement d'entite par les
//	           contraintes mesurees de son en-tete, sans traverser la chaine de composants ;
//	la CHAINE  `DecodeFrameRecords` marche la boucle de records du paquet et passe par
//	           `consumeByName`, dont deux cas concernent l'archetype statborg (ti=6 i0 et i57).
//
// D1 tranche pour l'ancrage et propose de SUPPRIMER `filmdec/statborg.go` (0 appelant). Cette
// decision ne tient que si la chaine ne voit pas plus d'enregistrements que l'ancrage : si elle
// en voyait nettement plus, l'ancrage laisserait du score sur la table et D1 serait a revoir.
//
// Ce test compte donc, sur UN film, les records d'archetype statborg atteints par la chaine.
// Le compte de l'ancrage vient de l'instrument du lot A (colonne `meta` de
// `.ai/V7.5/replay2d/registre_film/lotA/<short8>.tsv`) : les deux se comparent au journal.
//
// Garde : `D1_FILM` = dossier de chunks d'UN film. `D1_OUT`, s'il est pose, recoit une ligne
// TSV. Un film par processus (D17).

const (
	// d1FilmEnv porte le dossier de chunks du film a compter.
	d1FilmEnv = "D1_FILM"
	// d1OutEnv porte le fichier TSV cumulatif de sortie (facultatif).
	d1OutEnv = "D1_OUT"
	// statborgComponentName est le composant qui IDENTIFIE l'archetype statborg. L'archetype
	// est resolu par ce nom, jamais par un index cable : un typeIndex est un numero de build.
	statborgComponentName = "statborg-current-round-value-stat-component"
)

// TestD1StatborgChainCount compte les records statborg vus par la voie « chaine ».
func TestD1StatborgChainCount(t *testing.T) {
	dir := os.Getenv(d1FilmEnv)
	if dir == "" {
		t.Skipf("controle D1 non arme (%s vide)", d1FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	reg := registryOf(t, dir)
	ti := archetypeWithComponent(reg, statborgComponentName)
	if ti < 0 {
		t.Fatalf("aucun archetype du registre ne declare %q", statborgComponentName)
	}

	c := chainCount{reg: reg, ti: uint32(ti), world: NewWorld(reg)}
	for i := 1; i <= n; i++ {
		data, err := ReadFilmChunk(dir, i)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			c.packet(pk, data)
		}
	}
	line := fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", shortOf(dir), ti,
		c.frames, c.cleanFrames, c.desyncFrames, c.records, c.statborgRecords,
		c.statborgClean, c.statborgDirty, c.newStatborg)
	t.Logf("D1_CHAIN\tfilm\tti\tpaquets\tpropres\tdesync\trecords\tstatborg_total" +
		"\tstatborg_propres\tstatborg_desync\tnew_statborg")
	t.Logf("D1_CHAIN\t%s", line)
	if out := os.Getenv(d1OutEnv); out != "" {
		appendLine(t, out, line)
	}
}

// chainCount porte les compteurs de la marche (regle des 5 parametres : un etat, pas huit
// arguments).
type chainCount struct {
	reg   *Registry
	ti    uint32
	world *World
	// frames / cleanFrames / desyncFrames comptent les paquets delta marches.
	frames, cleanFrames, desyncFrames int
	// records est le total rendu par la boucle ; statborgRecords ceux dont le slot est lie a
	// l'archetype statborg ; newStatborg les seuls NEW propres de cet archetype.
	records, statborgRecords, newStatborg int
	// statborgClean / statborgDirty separent les records statborg des paquets marches SANS
	// erreur de ceux des paquets qui ont desynchronise. La distinction est le coeur de la
	// mesure : passe une desync, la boucle lit des positions de bits qui ne sont plus des
	// en-tetes de record, et un slot tombe par hasard sur une entite liee au statborg. Compter
	// les deux ensemble ferait passer du BRUIT pour de la couverture.
	statborgClean, statborgDirty int
}

// packet marche UN paquet : un keyframe re-etablit les liaisons du monde, un delta est compte.
func (c *chainCount) packet(pk FilmPacket, data []byte) {
	pay := pk.Payload(data)
	switch pk.Type {
	case PacketTypeKeyframe:
		// Le monde est reconstruit sur l'etat complet : sans liaisons, un record delta n'a
		// pas d'archetype et ne serait attribuable a aucun ti.
		c.world = WorldFromKeyframe(c.reg, pay)
	case PacketTypeDelta:
		c.frames++
		recs, err := DecodeFrameRecords(NewBitReader(pay), c.world, DefaultFrameConfig())
		if err != nil {
			c.desyncFrames++
		} else {
			c.cleanFrames++
		}
		c.records += len(recs)
		here := 0
		for _, r := range recs {
			if r.Type == recNew && r.DesyncAt == -1 && r.TypeIndex == c.ti {
				c.newStatborg++
			}
			// L'archetype d'un record delta se lit dans le monde. La liaison est relue APRES
			// la marche du paquet : une entite qui change d'archetype en cours de paquet
			// serait mal attribuee — cas non observe sur le statborg, qui vit tout le match.
			if a, ok := c.world.ArchetypeForSlot(r.Slot); ok && a == c.ti {
				here++
			}
		}
		c.statborgRecords += here
		if err != nil {
			c.statborgDirty += here
		} else {
			c.statborgClean += here
		}
	}
}

// registryOf charge le registre du film (chunk_00).
func registryOf(t *testing.T, dir string) *Registry {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 (registre) illisible dans %s : %v", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible dans %s : %v", dir, err)
	}
	return reg
}

// archetypeWithComponent rend l'index du premier archetype qui declare le composant nomme,
// ou -1. Resoudre par nom est la regle du depot : l'index est un numero de build.
func archetypeWithComponent(reg *Registry, name string) int {
	for i := range reg.Archetypes {
		if len(reg.Archetypes[i].indicesOf(name)) > 0 {
			return i
		}
	}
	return -1
}

// shortOf rend la forme courte du film depuis son dossier de chunks.
func shortOf(dir string) string {
	for i := len(dir) - 1; i >= 0; i-- {
		if dir[i] == '/' || dir[i] == '\\' {
			return dir[i+1:]
		}
	}
	return dir
}

// appendLine ajoute une ligne au TSV cumulatif de sortie.
func appendLine(t *testing.T, path, line string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("ouverture de %s : %v", path, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatalf("ecriture dans %s : %v", path, err)
	}
}
