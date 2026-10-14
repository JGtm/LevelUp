package grammar

// harnais_portee_test.go — LES SEULS ASSISTANTS DE TEST QUI POSENT LA PORTEE `DAT_144e61ea0`
// ([Lecteur.portee]) OU L ETAT COMPLET ([Lecteur.etatComplet]) SUR UN LECTEUR.
//
// En production, seules la marche d etat complet (`keyframe_fullstate_loop.go`) et la relecture a
// l etendue (`relecture_a_l_etendue.go`) les posent. Un test qui lit un composant SOUS la portee ou
// en etat complet hors de ces deux chemins (un lecteur de site, un lecteur unitaire) les pose par
// ici, jamais en ecrivant le champ : garde-rail `portee_ecriture_guard_test.go`.

// sousLaPortee pose la portee sur `br` et le rend.
func sousLaPortee(br *Lecteur) *Lecteur {
	br.portee = true
	return br
}

// enEtatComplet pose (ou retire) l etat complet sur `br` et le rend.
func enEtatComplet(br *Lecteur, v bool) *Lecteur {
	br.etatComplet = v
	return br
}
