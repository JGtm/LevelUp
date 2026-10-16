package assets

import (
	"fmt"
	"strings"
)

// Validate dit si la Ref peut désigner un fichier sous la racine d'un stock : c'est la
// condition de toute lecture, écriture ou récupération (LocalFSStore.Path, DefaultResolver,
// Gate). Ses composants viennent en partie d'une requête HTTP (titre, identifiant,
// variante) et entrent dans un chemin disque.
//
// Refusés (ErrInvalidRef) :
//   - un caractère de contrôle (octet nul compris), une barre oblique inverse ou un
//     deux-points dans un composant (séparateur, lecteur ou flux alterné sous Windows) ;
//   - une barre oblique dans le kind, le titre ou la variante (un seul élément de chemin) ;
//   - un élément « . » ou « .. » : kind, titre ou variante entiers, ou élément de
//     l'identifiant (un identifiant GameCMS est un chemin relatif à barres obliques ; ses
//     éléments vides sont tolérés, filepath.Join les absorbe).
func (r Ref) Validate() error {
	for _, c := range []struct{ name, value string }{
		{"kind", string(r.Kind)}, {"title", r.TitleID}, {"variant", r.Variant},
	} {
		if err := validatePathElement(c.name, c.value); err != nil {
			return err
		}
	}
	if err := validateComponentChars("id", r.ID); err != nil {
		return err
	}
	for _, seg := range strings.Split(r.ID, "/") {
		if seg == "." || seg == ".." {
			return fmt.Errorf("%w: id %q contient un élément %q", ErrInvalidRef, r.ID, seg)
		}
	}
	return nil
}

// validatePathElement : value est vide ou un seul élément de chemin ordinaire.
func validatePathElement(name, value string) error {
	if err := validateComponentChars(name, value); err != nil {
		return err
	}
	if strings.Contains(value, "/") || value == "." || value == ".." {
		return fmt.Errorf("%w: %s %q n'est pas un élément de chemin", ErrInvalidRef, name, value)
	}
	return nil
}

// validateComponentChars refuse les caractères de contrôle, la barre oblique inverse et le
// deux-points.
func validateComponentChars(name, value string) error {
	for _, c := range value {
		if c < 0x20 || c == 0x7f || c == '\\' || c == ':' {
			return fmt.Errorf("%w: %s %q contient un caractère interdit (%U)", ErrInvalidRef, name, value, c)
		}
	}
	return nil
}
