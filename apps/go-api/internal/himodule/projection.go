// Package himodule — projection.go : la frontiere entre le lecteur et le systeme de fichiers.
//
// POURQUOI CE FICHIER EXISTE. `Open` faisait `os.ReadFile` : il chargeait l'archive ENTIERE
// dans le tas Go. Mesure du 2026-09-05 (profil `-memprofile` sur un balayage de 4 cartes) :
// 87,7 Go alloues au total, dont 74,6 Go (85 %) par `os.readFileContents`. La raison est
// arithmetique — `himap.NewModuleIndex` ouvre, POUR CHAQUE CARTE, le module de la carte plus
// tous les modules globaux, soit 16,2 Go d'archives (`globals` 7,4 Go + `common` 2,7 Go +
// `multiplayer` 0,9 Go, plus leurs compagnons `_hd1`). Vingt-six cartes, vingt-six fois.
//
// CE QUE CHANGE LA PROJECTION. Une archive projetee n'est plus copiee dans le tas : les pages
// restent celles du cache de fichiers du systeme, partagees entre les projections, et le
// systeme les REPREND quand la memoire manque. Le lecteur ne touche de toute facon qu'une
// petite part de chaque archive (les tables, puis les blocs des tags demandes) : payer la
// copie integrale pour lire quelques megaoctets etait le vrai gaspillage.
//
// CE QUE LA PROJECTION NE CHANGE PAS, ET C'EST LE POINT : les octets. `m.data` reste un
// `[]byte` a acces direct, la meme arithmetique d'offsets s'y applique, et RIEN de ce que le
// paquet rend n'alias la projection — `Extract` decompresse dans un tampon neuf, le chemin
// non compresse fait `copy`, `ResourceBlob` concatene par `append`, `fourCC` construit sa
// chaine. C'est ce qui rend la fermeture sure, et c'est verifie par le harnais d'empreintes
// (`himap.TestEmpreinteLecteurDeModules`) : memes empreintes avant et apres.
package himodule

import "fmt"

// projection est une archive projetee en memoire, et de quoi la relacher.
type projection struct {
	octets []byte
	fermer func() error
}

// ferme relache la projection. Idempotent : un module ferme deux fois n'est pas une erreur.
func (p *projection) ferme() error {
	if p == nil || p.fermer == nil {
		return nil
	}
	f := p.fermer
	p.fermer, p.octets = nil, nil
	return f()
}

// projette ouvre un fichier en lecture seule et rend son contenu.
//
// UN FICHIER VIDE N'EST PAS PROJETE : les deux systemes refusent une projection de taille
// nulle, et l'appelant (`loadHd1`) traite deja le cas « compagnon vide » comme une absence.
func projette(chemin string) (*projection, error) {
	octets, fermer, err := projetteFichier(chemin)
	if err != nil {
		return nil, fmt.Errorf("himodule: projection de %s: %w", chemin, err)
	}
	return &projection{octets: octets, fermer: fermer}, nil
}
