package grammar

// keyframe_body_variants_instrument_helpers_test.go — LES HUIT LECTURES DU CORPS D UN RECORD
// D IMAGE-CLE, ET LEUR MARCHEUR : UN INSTRUMENT, PAS UNE LECTURE DE PRODUCTION.
//
// # POURQUOI CE CODE EST DANS UN FICHIER `_test.go` (revue de jalon M1, constat C4, 2026-09-16)
//
// `walkKeyframeBody` a cesse d etre une lecture de production au lot 1.4 (2026-09-14) : le CADRE
// d image-cle d etat complet est devenu LA lecture (D8 du plan), et l option qui faisait varier
// l en-tete de corps a disparu avec `shiftArchetypeLevels`. Son en-tete le disait deja — « ZERO
// appelant de production » — et le lot l avait UNEXPORTE, ce qui reduit la surface sans retirer
// le code du BINAIRE. Regle 7 du depot (« 0 code mort ») : ce qu on debranche du routing sort de
// la production. Le deplacement ici est la seule forme qui garde la MESURE sans garder la dette :
// ses cinq appelants sont tous des instruments `_test.go` DU MEME PAQUET.
//
// # CE QUE CES CINQ INSTRUMENTS FONT, ET POURQUOI ON NE LES SUPPRIME PAS AVEC
//
// Ce sont les comparateurs A/B d autres chantiers — bipede bit-exact, vehicules v5b, grammaire de
// l ecrivain : ils rejouent le corps d un record sous les huit combinaisons de trois bascules
// (etat par defaut, porte, masque) et disent laquelle marche. Un comparateur qui disparait n est
// pas une dette qu on solde, c est une mesure qu on perd.
//
// # PAS DE TAG `research` ICI
//
// Ce fichier ne lit AUCUN film : il ne porte que la table des variantes et le marcheur, qui
// reutilise la boucle de composants de production (`traverseComponentLoop`) et ses
// deserialiseurs d etat par defaut. Ce sont les instruments APPELANTS qui ouvrent des films et
// qui portent leur propre garde ; un helper partage vit dans un fichier NON tague (regle du
// 2026-09-16, §2.3 du plan — modele `filmdec/e191_helpers_test.go`).

// ---------------------------------------------------------------------------------------
// LES VARIANTES DE CORPS — parce qu'aucune n'est supposee, elles sont toutes MESUREES.
//
// Le balayage du decalage (instrument `TestKFGramOffset`) a etabli deux faits : les 6 bits
// de `typeIndex` sont bien a `+58` (415/415 et 2008/2008 records relus corrects, aucun autre
// decalage de 0 a 127 ne fait mieux que le hasard), et AUCUN decalage ne rend une seule
// marche bit-exacte. Le corps d'un record de la table d'image-cle n'est donc PAS le corps
// d'un record NEW, quel que soit l'endroit ou on le pose.
//
// Reste la lecture (a) de la decouverte 3 du lot R3 : l'image-cle porterait un ETAT COMPLET
// — tous les composants de l'archetype, sans masque epars. Les longueurs reelles mesurees
// vont dans ce sens (ti=38 : 39 valeurs distinctes seulement sur 2 008 records, dominante
// 827 bits ; la marche de record NEW n'en consomme qu'environ 40 %). `keyframeBodyVariant`
// expose les trois bascules qui separent ces lectures, et l'instrument les balaie toutes.
// ---------------------------------------------------------------------------------------

// keyframeBodyVariant decrit UNE lecture possible du corps d'un record d'image-cle.
type keyframeBodyVariant struct {
	// DefaultState : jouer le deserialiseur d'etat par defaut de l'archetype (vtable[0x60]).
	DefaultState bool
	// Gate : lire la porte `R(1)` qui precede le masque dans un record NEW.
	Gate bool
	// Mask : lire le masque de presence dans le flux (sinon : TOUS les composants presents).
	Mask bool
}

// String rend une etiquette lisible de la variante.
func (v keyframeBodyVariant) String() string {
	f := func(b bool) string {
		if b {
			return "oui"
		}
		return "non"
	}
	return "etatParDefaut=" + f(v.DefaultState) + " porte=" + f(v.Gate) + " masque=" + f(v.Mask)
}

// keyframeBodyVariants est la matrice des huit lectures probees. La premiere est celle du
// record NEW (celle que `TraverseEntity` joue), la derniere l'etat complet nu.
var keyframeBodyVariants = []keyframeBodyVariant{
	{DefaultState: true, Gate: true, Mask: true},
	{DefaultState: true, Gate: true, Mask: false},
	{DefaultState: true, Gate: false, Mask: true},
	{DefaultState: true, Gate: false, Mask: false},
	{DefaultState: false, Gate: true, Mask: true},
	{DefaultState: false, Gate: true, Mask: false},
	{DefaultState: false, Gate: false, Mask: true},
	{DefaultState: false, Gate: false, Mask: false},
}

// walkKeyframeBody rejoue le corps d'un record d'image-cle sous la variante `v`, en partant
// du debut du record (`recBit`). Il REUTILISE la boucle de composants de production
// (`traverseComponentLoop`) et les deserialiseurs d'etat par defaut de production : rien
// n'est recopie, seule la facon de lire l'en-tete de corps change.
//
// CE N'EST PLUS UNE LECTURE DE PRODUCTION (lot 1.4, 2026-09-14), ET IL N'EST PLUS DANS LA
// PRODUCTION (revue de jalon M1, 2026-09-16). Inventaire sur pieces du 2026-09-14, refait le
// 2026-09-16 (`grep -rn "walkKeyframeBody\|keyframeBodyVariant" --include=*.go internal/ cmd/`) :
// les fichiers qui le citent sont CINQ instruments `_test.go` du paquet et sa propre
// declaration — ZERO appelant de production, aux deux dates. Le lot 1.4 l'avait UNEXPORTE, ce
// qui reduit la surface atteignable sans retirer le code du binaire ; il vit desormais dans un
// fichier de test, ce qui l'en retire.
//
// IL N'EST PAS SUPPRIME parce que ces cinq instruments sont les comparateurs A/B d'autres
// chantiers (bipede bit-exact, vehicules v5b, grammaire d'ecrivain) : un comparateur qui
// disparait n'est pas une dette qu'on solde, c'est une mesure qu'on perd.
//
// RETRAIT (regle 11) : pose le 2026-09-14, deplace le 2026-09-16 ; cible = lot 3.6 (ports de
// composants), quand ces cinq instruments seront rebases sur le cadre d'etat complet ; critere
// mesurable = zero fichier citant `walkKeyframeBody` en dehors de ce fichier.
func walkKeyframeBody(pay []byte, recBit int, reg *Registry, v keyframeBodyVariant) EntityTrace {
	br := LecteurSur(pay)
	br.SetBitPos(recBit + keyframeRecordTIBit)
	t := EntityTrace{DesyncAt: -1}
	t.TypeIndex = uint32(br.ReadBits(6))
	if t.TypeIndex >= objectArchetypeCount {
		t.DesyncAt, t.EndBit = 0, br.BitPos()
		return t
	}
	if v.DefaultState {
		consumeKeyframeDefaultState(br, t.TypeIndex)
	}
	if v.Gate {
		t.Gate = br.ReadBit()
	}
	if v.Mask {
		t.Mask = consumeMask(br)
	} else {
		t.Mask = ^uint64(0) // etat complet : tous les composants presents
	}
	arch, ok := reg.Archetype(int(t.TypeIndex))
	if !ok {
		t.DesyncAt, t.EndBit = 0, br.BitPos()
		return t
	}
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}
