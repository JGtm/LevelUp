package grammar

// kills_rattrapes.go — LE RATTRAPAGE DES MESSAGES DE KILL QUE LA LECTURE DE LA VUE A N ETABLIT PAS : une
// recuperation nommee et comptee, apres la lecture (ADR 0037 IR-6).
//
// La lecture unique de la vue A ([lireLaVueA]) range chaque message de kill qu elle lit
// ([lecture.VueA.Kills]). Elle est ETABLIE quand la marche fait commencer la vue B a sa fin
// ([lecture.DebutParVueA]) : la classe du film la tient pour une lecture, ou la marche depuis cette
// fin ferme le paquet. Dans toute autre trame a evenements — lecture arretee sur un message qu elle ne
// sait pas lire ou sur un film sans table des genres, terminateur que la marche ne retient pas —, les
// messages de kill que la vue A n a pas lus ne sont lus par personne. Le rattrapage les cherche la, et
// seulement la : entre le premier genre de la vue A et le debut de la vue B (la fin du payload quand
// la liste n est pas localisee), a chaque bit, un bit de continuation a 1 suivi du genre 85, ses
// champs lus sans queue ([lireLeKillDeLaChaine]), tueur et victime presents et distincts, puis une
// chaine d au moins trois evenements derriere lui sous la grammaire de la chaine
// (`chaine_d_evenements.go`). Un message que la vue A a deja lu n est pas rendu une seconde fois.
//
// `gate15`, l etat d execution que le corps du Script lu par la chaine suppose, se tranche par film a
// la premiere trame qui en a besoin ([trancherGate15]).
//
// Le repli se compte au registre (`repli_kill_rattrape_hors_vue_a`, ordre apres la lecture), et les
// chaines arretees sur un code non modelise sous `repli_chaine_evenement_code_non_modelise`.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// minChain : profondeur de chaine exigee pour retenir un candidat. Trois evenements — le seuil
// mesure. NE PAS l abaisser sans re-mesurer : c est lui qui elimine 99.4 % des faux.
const minChain = 3

// maxChainProbe : borne de la marche avant. Au-dela le critere ne discrimine plus rien et le
// cout devient inutile.
const maxChainProbe = 12

// octetsMinimauxDuTranchage : une trame a evenements plus courte n entre pas dans le tranchage de
// `gate15`.
const octetsMinimauxDuTranchage = 64

// KillRattrape est un message de kill que le rattrapage retient : ses champs (Debut : le bit de sa
// continuation) et la longueur de la chaine d evenements qui l a valide.
type KillRattrape struct {
	Kill   lecture.MessageDeKill
	Chaine int
}

// rattrapageDesKills porte le rattrapage d un film : `gate15`, tranche a la premiere trame qui en a
// besoin, et les chaines arretees sur un code non modelise.
type rattrapageDesKills struct {
	film            *source.Film
	gate15, tranche bool
	chainesArretees int
}

// rattraper rend les messages de kill que le rattrapage retient dans la trame marchee `p` : aucun
// quand la vue B commence a la fin de sa vue A lue.
func (r *rattrapageDesKills) rattraper(p *lecture.Paquet) []KillRattrape {
	f, ok := fenetreDuRattrapage(p)
	if !ok {
		return nil
	}
	if !r.tranche {
		r.gate15, r.tranche = trancherGate15(r.film), true
	}
	kills, arretees := killsAvecArrets(p.Payload, f, r.gate15)
	r.chainesArretees += arretees
	return kills
}

// fenetreDeRecherche est ce que la recherche bit a bit parcourt dans un payload : les positions de
// genre de `depuis` (compris) a celles dont la continuation tombe a `jusqua` (exclu), sans rendre les
// messages `dejaLus`.
type fenetreDeRecherche struct {
	depuis, jusqua int
	dejaLus        []lecture.MessageDeKill
}

// fenetreDuRattrapage rend la fenetre du rattrapage dans une trame marchee : du premier genre de sa
// vue A au debut de sa vue B, la fin du payload quand la liste n est pas localisee, sans les messages
// que la vue A a lus. Faux quand la trame n annonce aucune liste, ou que la vue B commence a la fin
// de la vue A lue.
func fenetreDuRattrapage(p *lecture.Paquet) (fenetreDeRecherche, bool) {
	if !listeAnnoncee(&p.VueA) {
		return fenetreDeRecherche{}, false
	}
	f := fenetreDeRecherche{depuis: int(p.VueA.Debut) + 1, jusqua: len(p.Payload) * 8, dejaLus: p.VueA.Kills}
	switch p.Debut {
	case lecture.DebutParVueA:
		return fenetreDeRecherche{}, false
	case lecture.DebutParSignature, lecture.DebutParChaine, lecture.DebutParFermeture, lecture.DebutParFermetureAuBit:
		f.jusqua = int(p.VueB.Debut)
	}
	return f, true
}

// teteDeKill est la tete d un message de kill lue d un coup : la continuation (1), puis le genre 85.
const teteDeKill = 1<<LargeurGenreVueA | GenreJoueurTue

// estAncreDeKill : la position `x` (celle du genre) ouvre-t-elle un message de kill ? Sa tete — le
// bit de continuation a 1, puis le genre 85 — se lit sur les huit bits qui commencent en `x-1`.
// C est le GENERATEUR de candidats — la seule lecture faite a CHAQUE bit, donc sans allocation :
// elle passe par les primitives de position de la couche source.
func estAncreDeKill(pl []byte, x int) bool {
	return source.BitsAt(pl, x-1, 1+LargeurGenreVueA) == teteDeKill
}

// killsAvecArrets rend les messages de kill que la recherche retient dans la fenetre `f` de `pl`, et
// le nombre de chaines ouvertes par un message plausible qui se sont arretees sur un code non
// modelise, gardees ou non.
func killsAvecArrets(pl []byte, f fenetreDeRecherche, gate15 bool) ([]KillRattrape, int) {
	var out []KillRattrape
	arretees := 0
	nb := len(pl) * 8
	for x := max(f.depuis, 1); x+8 <= nb && x-1 < f.jusqua; x++ {
		if !estAncreDeKill(pl, x) || dejaLu(f.dejaLus, x-1) {
			continue
		}
		r := nouveauCurseurEv(pl, x+LargeurGenreVueA)
		if !evPresence(r, GenreJoueurTue) {
			continue
		}
		k, fin := lireLeKillDeLaChaine(pl, r.pos())
		if !killPlausible(k, fin) {
			continue
		}
		n, arretee := evChainLenAvecVerdict(pl, fin, gate15, maxChainProbe)
		arretees += unSi(arretee)
		if n < minChain {
			continue
		}
		k.Debut = uint32(x - 1) //nolint:gosec // position dans un payload
		out = append(out, KillRattrape{Kill: k, Chaine: n})
	}
	return out, arretees
}

// dejaLu dit qu un des messages `ks` commence au bit `debut`.
func dejaLu(ks []lecture.MessageDeKill, debut int) bool {
	for i := range ks {
		if int(ks[i].Debut) == debut {
			return true
		}
	}
	return false
}

// trancherGate15 : `gate15` est un etat d EXECUTION du jeu, absent du flux de bits — il ne se lit
// pas, il se TRANCHE par film. Critere : celui des deux qui enchaine le plus de messages de kill sur
// les trames a evenements du film, recherchees en entier. Ce n est pas un reglage libre : les deux
// valeurs sont essayees et la mesure decide, sur une quantite (longueur de chaine) qui ne regarde
// AUCUN resultat publie.
func trancherGate15(f *source.Film) bool {
	score := [2]int{}
	for _, p := range f.AllPackets() {
		if p.Type != int(PacketTypeDelta) || source.BitAt(p.Payload, 1) == 0 ||
			len(p.Payload) < octetsMinimauxDuTranchage {
			continue
		}
		entiere := fenetreDeRecherche{depuis: 1, jusqua: len(p.Payload) * 8}
		for g := range 2 {
			kills, _ := killsAvecArrets(p.Payload, entiere, g == 1)
			score[g] += len(kills)
		}
	}
	return score[1] > score[0]
}
