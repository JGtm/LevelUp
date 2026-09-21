package grammar

// movement_states_jump.go — LE SAUT, DERIVE DE LA PHYSIQUE DU FILM (lot 5.9.4, 2026-09-21).
//
// # CE GENRE EST DERIVE, ET IL LE DIT DANS SON NOM
//
// `types.MovementJumpDerived` vaut `jumpDerived`, et JAMAIS `jump`. Les trois autres genres
// (`crouch`, `slide`, `mobility`) sont des bits LUS dans un composant : le deserialiseur publie,
// ce calque replie. Celui-ci ne lit aucun bit de saut — il INTEGRE la vitesse verticale d `i1`
// et reconnait l episode a sa HAUTEUR. Confondre les deux dans un seul `kind` ferait passer un
// calcul pour une lecture.
//
// LA CHAINE DU DECLENCHEUR N EST PAS ABANDONNEE POUR AUTANT. Le lot 5.9.2 l a remontee de la
// condition d animation `IsAirborne` jusqu au compteur de ticks sans contact `u+0x89b`, dont le
// seul ecrivain (`FUN_1408b19cc`) n est appele que par la resolution de contact du controleur de
// personnage (`FUN_1408b2f90`) : la chaine est NON TROUVEE a cette adresse, pas refutee. Le jour
// ou le champ replique sera nomme, un genre `jump` LU remplacera celui-ci, avec sa propre montee
// de revision. C est la decision de l utilisateur du 2026-09-21.
//
// # LA METHODE EST CELLE DE L ORACLE PHYSIQUE, TELLE QUELLE
//
// `mouvement_5_7_5_oracles_research_test.go` (lot 5.7.5) etiquette les episodes aeriens par
// integration de la vitesse verticale TENUE — `i1` ne voyage que sur changement, donc la vitesse
// d une vie a l instant `t` est celle de sa derniere lecture a ou avant `t`. La distribution des
// hauteurs ainsi obtenues porte un PIC ETROIT, mesure sur DEUX films :
//
//	bfecd02b (snowbound)   H = 0,85 m · montee 0,467 s · pic x 10,7 au-dessus de ses voisins
//	4f77afc1 (flood gulch) H = 0,85 m · montee 0,466 s · pic x  3,9 au-dessus de ses voisins
//
// C est un fait de JEU, pas un seuil d instrument : tous les Spartans sautent la meme hauteur.
// D ou la constante [types.SpartanJumpHeightM] et sa fenetre de 10 %.
//
// # UNE DIFFERENCE AVEC L ORACLE, ET ELLE EST DELIBEREE
//
// L oracle compte aussi l episode encore OUVERT a la fin de la marche (il lui sert d histogramme).
// Ce calque le REFUSE : un episode sans instant de fin mesure n a pas d intervalle a publier, et
// sa hauteur est tronquee par le silence qui la termine. Le compteur `JumpEpisodes` dit combien
// d episodes FERMES ont ete examines, `JumpsDerived` combien sont tombes dans la fenetre.

import (
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// jumpVelSample est UNE lecture de vitesse verticale, datee et localisee dans le film.
type jumpVelSample struct {
	ts     uint64
	vz     float64
	chunk  int
	paquet int
}

// jumpEpisode est UNE montee : de l instant ou la vitesse verticale franchit le seuil a celui ou
// elle repasse dessous, avec la hauteur integree entre les deux.
type jumpEpisode struct {
	slot   uint32
	t0, t1 uint64
	chunk  int
	paquet int
	haut   float64
}

// vitesse capte UNE lecture d `i1` pour la derivation du saut.
//
// LE FILTRE EST LE MEME QUE CELUI DES LECTURES D ETAT (slot lie au bipede), mais il ne compte PAS
// dans `SlotUnbound` : ce compteur publie le prix de l attribution partielle de slot sur les
// TRANSITIONS d etat, et y verser la vitesse — universelle, presente sur 78 % des records —
// rendrait sa valeur illisible.
func (sc *movementStateScanner) vitesse(slot uint32, v []uint64) {
	if len(v) < 4 || v[0] != 0 || v[1] != 0 {
		// mode pleine precision (R(96), non dequantifie ici) ou vitesse ABSENTE de cet instant.
		return
	}
	if ti, ok := sc.monde.ArchetypeForSlot(slot); !ok || ti != BipedTypeIndex {
		return
	}
	vec := DecodeVelocity(v[2], v[3])
	if sc.vit == nil {
		sc.vit = map[uint32][]jumpVelSample{}
	}
	sc.vit[slot] = append(sc.vit[slot], jumpVelSample{ts: sc.ts, vz: float64(vec[2]),
		chunk: sc.chunk, paquet: sc.paquetIndex})
	sc.st.VelocityReads++
}

// deriverLesSauts transforme les lectures de vitesse en transitions `jumpDerived`.
//
// Elle passe par `sc.recevoirDerive`, donc par la MEME table de deduplication et le meme tri que
// les trois genres lus : un saut derive est un intervalle comme les autres une fois publie, et
// seul son `Kind` dit d ou il vient.
func (sc *movementStateScanner) deriverLesSauts() {
	slots := make([]uint32, 0, len(sc.vit))
	for s := range sc.vit {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		for _, e := range episodesDeMontee(s, sc.vit[s]) {
			sc.st.JumpEpisodes++
			if !hauteurDeSaut(e.haut) {
				continue
			}
			sc.st.JumpsDerived++
			sc.recevoirDerive(e.slot, e.chunk, e.paquet, e.t0, true)
			sc.recevoirDerive(e.slot, e.chunk, e.paquet, e.t1, false)
		}
	}
}

// hauteurDeSaut dit si une hauteur integree tombe dans la fenetre du saut du Spartan.
func hauteurDeSaut(h float64) bool {
	return math.Abs(h-types.SpartanJumpHeightM) <= types.SpartanJumpHeightM*types.SpartanJumpHeightTol
}

// episodesDeMontee segmente les lectures d UNE vie en montees FERMEES.
func episodesDeMontee(slot uint32, vs []jumpVelSample) []jumpEpisode {
	vs = vitessesOrdonnees(vs)
	var out []jumpEpisode
	var cur *jumpEpisode
	for i := range vs {
		switch {
		case vs[i].vz >= types.SpartanJumpRiseMinMS && cur == nil:
			cur = &jumpEpisode{slot: slot, t0: vs[i].ts, chunk: vs[i].chunk, paquet: vs[i].paquet}
		case vs[i].vz < types.SpartanJumpRiseMinMS && cur != nil:
			cur.t1 = vs[i].ts
			out = append(out, *cur)
			cur = nil
			continue
		}
		if cur != nil {
			cur.haut += vs[i].vz * dureeTenueS(vs, i)
		}
	}
	// L EPISODE ENCORE OUVERT EST JETE, et l en-tete dit pourquoi : pas d instant de fin mesure.
	return out
}

// vitessesOrdonnees trie les lectures d une vie par instant et ECARTE les doublons d instant.
//
// Le chemin d inference de [DecodeFrameViews] re-parcourt un record quand une chaine de
// transitoires le demande : la meme vitesse est alors publiee deux fois au meme instant. La
// compter deux fois ne changerait pas la hauteur (la duree tenue d un doublon est nulle) mais
// rendrait les bornes d episode dependantes de l ordre de parcours.
func vitessesOrdonnees(vs []jumpVelSample) []jumpVelSample {
	cp := make([]jumpVelSample, len(vs))
	copy(cp, vs)
	sort.SliceStable(cp, func(i, j int) bool { return cp[i].ts < cp[j].ts })
	out := cp[:0]
	var dernier uint64
	for i, x := range cp {
		if i > 0 && x.ts == dernier {
			continue
		}
		dernier = x.ts
		out = append(out, x)
	}
	return out
}

// dureeTenueS rend la duree, en secondes, pendant laquelle la lecture `i` vaut — bornee par
// [types.SpartanJumpHoldMaxUS].
//
// LA BORNE EST LE COEUR DE LA METHODE : `i1` ne voyage que sur changement, donc un silence de dix
// secondes ne veut pas dire « la meme vitesse pendant dix secondes » — il veut dire que la vie n a
// rien transmis, souvent parce qu elle est morte ou hors de la trame localisee. Integrer un tel
// silence FABRIQUERAIT des hauteurs.
func dureeTenueS(vs []jumpVelSample, i int) float64 {
	if i+1 >= len(vs) {
		return 0
	}
	d := vs[i+1].ts - vs[i].ts
	if d > types.SpartanJumpHoldMaxUS {
		return 0
	}
	return float64(d) / 1e6
}

// recevoirDerive pose UNE transition derivee dans la table de deduplication.
func (sc *movementStateScanner) recevoirDerive(slot uint32, chunk, paquet int, ts uint64, on bool) {
	lu := types.MovementStateRead{Slot: slot, Kind: types.MovementJumpDerived, On: on,
		TimestampUS: ts, Chunk: chunk, PacketIndex: paquet}
	k := movementStateKey{slot: slot, kind: lu.Kind, ts: ts}
	if _, deja := sc.vues[k]; deja {
		sc.st.Duplicates++
		return
	}
	sc.vues[k] = lu
	sc.st.Read++
}
