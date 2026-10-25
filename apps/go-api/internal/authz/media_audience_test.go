package authz

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
)

func set(xuids ...string) map[string]bool {
	out := map[string]bool{}
	for _, x := range xuids {
		out[x] = true
	}
	return out
}

func TestMediaViewerScope(t *testing.T) {
	admin := &domain.User{Role: domain.RoleAdmin, XUID: "111"}
	alice := &domain.User{Role: domain.RoleUser, XUID: "222"}
	unlinked := &domain.User{Role: domain.RoleUser}

	cases := []struct {
		name     string
		enforced bool
		user     *domain.User
		co       map[string]bool
		want     map[string]bool
	}{
		{"enforcement désactivé : aucune restriction", false, alice, nil, nil},
		{"admin : aucune restriction", true, admin, nil, nil},
		{"sans groupe : soi seul", true, alice, nil, set("222")},
		{"groupe : soi et co-membres", true, alice, set("222", "333"), set("222", "333")},
		{"sans identité : personne", true, unlinked, set("333"), set()},
		{"sans session : personne", true, nil, nil, set()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MediaViewerScope(tc.enforced, tc.user, tc.co)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("MediaViewerScope = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMediaAuthorXUIDs(t *testing.T) {
	cases := []struct {
		name   string
		page   string
		pageCo map[string]bool
		viewer map[string]bool
		want   map[string]bool
	}{
		{"sans groupe : ses médias seuls", "222", nil, nil, set("222")},
		{"groupe : le groupe entier", "222", set("222", "333", "444"), nil, set("222", "333", "444")},
		{"lecteur restreint : autres groupes de l'ami retirés", "333", set("222", "333", "444"), set("222", "333"), set("222", "333")},
		{"joueur de la page toujours présent", "333", nil, set("222"), set("333")},
		{"page sans xuid : personne", "", set("333"), nil, set()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MediaAuthorXUIDs(tc.page, tc.pageCo, tc.viewer)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("MediaAuthorXUIDs = %v, want %v", got, tc.want)
			}
		})
	}
}
