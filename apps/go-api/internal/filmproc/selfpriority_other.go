//go:build !windows

package filmproc

// selfpriority_other.go — sans effet hors Windows, pour la MEME raison que `priority_other.go` :
// les seuls postes de TRAVAIL interactifs du projet sont sous Windows, et c'est l'usage
// interactif que la priorite basse protege. Sur Linux (CI, VPS) il n'y a personne devant
// l'ecran, et un `nice` y couterait une divergence de comportement sans rien proteger.

// LowerOwnPriority : sans effet hors Windows (cf. l'en-tete — c'est une decision, pas un oubli).
func LowerOwnPriority(_ string) {}
