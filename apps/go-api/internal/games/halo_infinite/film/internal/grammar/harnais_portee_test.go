package grammar

// harnais_portee_test.go — LE SEUL ASSISTANT DE TEST QUI POSE LA PORTEE `DAT_144e61ea0` SUR UN
// LECTEUR ([Lecteur.portee]).
//
// En production, seule la marche d etat complet la pose (`keyframe_fullstate_loop.go`). Un test
// ou un instrument qui lit un composant SOUS la portee hors de cette marche (un lecteur de site,
// une relecture a l etendue) la pose par ici, jamais en ecrivant le champ : garde-rail
// `portee_ecriture_guard_test.go`.

// sousLaPortee pose la portee sur `br` et le rend.
func sousLaPortee(br *Lecteur) *Lecteur {
	br.portee = true
	return br
}
