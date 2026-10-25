package authz

import "levelup/go-api/internal/domain"

// MediaViewerScope rend les xuids dont le lecteur peut voir les médias, toutes pages
// confondues : nil quand rien ne le restreint (enforcement désactivé, admin), sinon
// son propre xuid et ceux des co-membres de ses groupes (`coMembers`, cf.
// groupstore.CoMemberXUIDs). Un lecteur sans identité ne voit personne (ensemble vide,
// non nil).
func MediaViewerScope(enforced bool, user *domain.User, coMembers map[string]bool) map[string]bool {
	if !enforced || (user != nil && user.Role == domain.RoleAdmin) {
		return nil
	}
	scope := map[string]bool{}
	if user == nil || user.XUID == "" {
		return scope
	}
	scope[user.XUID] = true
	for x, ok := range coMembers {
		if ok {
			scope[x] = true
		}
	}
	return scope
}

// MediaAuthorXUIDs rend les auteurs dont les médias s'affichent sur la page du joueur
// `pageXUID` : lui-même et les co-membres de ses groupes (`pageCoMembers`) ; sans
// groupe, lui seul. Quand `viewerScope` n'est pas nil (cf. MediaViewerScope), les
// co-membres hors de la portée du lecteur sont retirés : un ami commun n'ouvre pas les
// médias des autres groupes de cet ami. Le joueur de la page reste toujours : la garde
// de propriété a déjà autorisé sa lecture.
func MediaAuthorXUIDs(pageXUID string, pageCoMembers, viewerScope map[string]bool) map[string]bool {
	out := map[string]bool{}
	if pageXUID == "" {
		return out
	}
	out[pageXUID] = true
	for x, ok := range pageCoMembers {
		if !ok || x == "" {
			continue
		}
		if viewerScope != nil && !viewerScope[x] {
			continue
		}
		out[x] = true
	}
	return out
}
