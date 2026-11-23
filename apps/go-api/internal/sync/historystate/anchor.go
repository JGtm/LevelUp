package historystate

import "strings"

// AnchorMaxPages borne, en nombre de pages de la liste de l'API, la recherche de l'ancre d'un
// marque-page ([Cursor.HeadID]) en haut de la liste : au-delà, l'ancre est tenue pour perdue et la
// passe reprend depuis le début de la liste. Huit pages de 25 entrées couvrent 200 matchs arrivés
// entre deux passes, soit le budget d'une passe ([PassBudget]) : un joueur qui a joué plus que cela
// entre deux passes de rattrapage (une toutes les dix minutes) n'existe pas en pratique, et le coût
// du repli reste le parcours lent mais sûr.
const AnchorMaxPages = 8

// AnchorScanLimit est le nombre d'entrées de tête de liste à lire au plus pour retrouver l'ancre,
// pour des pages de pageSize entrées : [AnchorMaxPages] pages.
func AnchorScanLimit(pageSize int) int { return AnchorMaxPages * max(pageSize, 1) }

// HasAnchor dit si le marque-page porte une ancre ([Cursor.HeadID]). Un marque-page rangé avant
// l'ancre (ou par un appelant qui ne la fournit pas) n'en a pas : l'appelant retombe alors sur sa
// règle sans ancre.
func (c Cursor) HasAnchor() bool { return strings.TrimSpace(c.HeadID) != "" }

// ArrivedSince rend le nombre d'entrées arrivées en tête de la liste de l'API depuis la passe qui
// a rangé le marque-page : la position de l'ancienne tête ([Cursor.HeadID]) dans `top`, qui est
// le début de la liste lu tel quel, de la position 0 à la position len(top)-1, TOUTES entrées
// comptées (connues, inconnues, introuvables, écartées : l'API les place toutes dans la liste, et
// c'est cette liste que le décalage mesure).
//
// found=false : pas d'ancre, ou ancienne tête absente de `top`. L'appelant lit `top` sur au plus
// [AnchorScanLimit] entrées ; une ancre perdue se traite comme un marque-page perdu (reprise depuis
// le début de la liste, journalisée et comptée par l'appelant). n n'a de sens que si found.
func (c Cursor) ArrivedSince(top []string) (n int, found bool) {
	if !c.HasAnchor() {
		return 0, false
	}
	for i, id := range top {
		if id == c.HeadID {
			return i, true
		}
	}
	return 0, false
}

// ResumeOffset rend le décalage de reprise dans la liste de l'API quand `arrived` entrées sont
// arrivées en tête depuis le rangement du marque-page : la prochaine entrée à traiter est à
// Offset+arrived, et l'entrée à ResumeOffset(arrived)-1 doit être [Cursor.LastMatchID] (le
// contrôle de position garde l'appelant d'une ancre qui mentirait : un match disparu de la liste).
func (c Cursor) ResumeOffset(arrived int) int { return c.Offset + max(arrived, 0) }
