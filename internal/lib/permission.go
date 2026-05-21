package lib

import "slices"

var RoleHierarchy = map[string]int{
	"admin":      5,
	"director":   4,
	"manager":    3,
	"supervisor": 2,
	"staff":      1,
}

func CanManage(managerRole, targetRole string) bool {
	if managerRole == "admin" && targetRole == "admin" {
		return true
	}
	return RoleHierarchy[managerRole] == RoleHierarchy[targetRole]+1
}

func RoleBelow(role string) string {
	targetLevel := RoleHierarchy[role] - 1
	for r, l := range RoleHierarchy {
		if l == targetLevel {
			return r
		}
	}
	return ""
}

func HasPermission(perms []string, codename string) bool {
	return slices.Contains(perms, codename)
}

func MatchedPermissions(userPerms []string, requiredPerms []string) []string {
	var matched []string
	for _, p := range requiredPerms {
		if slices.Contains(userPerms, p) {
			matched = append(matched, p)
		}
	}
	return matched
}
