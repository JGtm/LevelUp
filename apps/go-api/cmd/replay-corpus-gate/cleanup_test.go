package main

import "testing"

// TestNettoyeurComposeVoitLesAjoutsPosterieursAuDefer — REPRODUIT LE BUG C1 : un `defer` arme
// sur la METHODE (jamais sur une valeur de fonction figee) doit voir TOUS les ajouts faits
// avant qu'il ne s'execute, meme ceux survenus APRES que le `defer` a ete pose — exactement le
// cas de `preparerReferenceBase`, appelee bien apres `defer func() { nettoyeur.Executer() }()`
// dans `executer`.
func TestNettoyeurComposeVoitLesAjoutsPosterieursAuDefer(t *testing.T) {
	var ordre []string
	func() {
		n := &nettoyeurCompose{}
		defer func() { n.Executer() }() // arme AVANT le second Ajouter — le point exact du bug C1
		n.Ajouter(func() { ordre = append(ordre, "a") })
		n.Ajouter(func() { ordre = append(ordre, "b") }) // ajout POSTERIEUR au defer
	}()

	if len(ordre) != 2 {
		t.Fatalf("attendu 2 actions executees (l'ajout posterieur au defer doit etre vu), obtenu %d : %v",
			len(ordre), ordre)
	}
	if ordre[0] != "b" || ordre[1] != "a" {
		t.Fatalf("ordre LIFO attendu [b a] (dernier ajoute, premier execute), obtenu %v", ordre)
	}
}

// TestNettoyeurComposeEstIdempotent — un signal d'interruption et le defer normal de fin de
// executer() ne doivent jamais nettoyer deux fois la meme ressource (un second
// `git worktree remove` sur un chemin deja retire, par exemple, rendrait une erreur bruyante
// pour rien).
func TestNettoyeurComposeEstIdempotent(t *testing.T) {
	n := &nettoyeurCompose{}
	compte := 0
	n.Ajouter(func() { compte++ })

	n.Executer()
	n.Executer()

	if compte != 1 {
		t.Fatalf("attendu 1 execution (idempotence), obtenu %d", compte)
	}
}

// TestNettoyeurComposeSansActionNePanicPas — le cas d'une execution qui echoue avant la
// premiere ressource creee (racine de travail introuvable, par exemple) : Executer() sur un
// nettoyeur vide ne doit jamais paniquer.
func TestNettoyeurComposeSansActionNePanicPas(t *testing.T) {
	n := &nettoyeurCompose{}
	n.Executer()
}
