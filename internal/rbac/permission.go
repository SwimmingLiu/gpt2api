// Package rbac 基于角色的访问控制。
//
// 设计原则(为避免权限漏洞):
//
//  1. **最小权限**:普通用户默认仅拥有 self:* 权限,只能读/写自己的资源。
//  2. **菜单和权限分离**:菜单只是 UI 提示,实际访问每条 API 都必须单独做
//     `middleware.RequirePerm(...)`。前端隐藏菜单 != 后端拒绝访问。
//  3. **高危写操作**要求在对应 handler 内做额外确认逻辑。
//
// 本文件只定义权限常量和预设角色。运行时不依赖数据库存角色绑定,
// 角色→权限映射由代码硬编码,避免运维失误导致提权漏洞。
// 未来需要多租户/自定义角色时再引入 `role_permissions` 表即可平滑升级。
package rbac

// Role 角色常量。和 users.role 字段对齐。
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// Permission 形如 "resource:action"。命名遵守小写 + 冒号分隔。
// 所有自定义权限都在这里集中声明,便于检索。
type Permission string

const (
	// --- 普通用户(self) ---
	PermSelfProfile = Permission("self:profile")

	// --- 管理员(admin) ---
	// 账号池 / 代理池 / 模型
	PermAccountRead  = Permission("account:read")
	PermAccountWrite = Permission("account:write")
	PermProxyRead    = Permission("proxy:read")
	PermProxyWrite   = Permission("proxy:write")
	PermModelRead    = Permission("model:read")
	PermModelWrite   = Permission("model:write")

	PermStatsReadAll = Permission("stats:read_all")

	// 系统
	PermSystemSetting = Permission("system:setting") // 改系统配置
)

// rolePermissions 是角色到权限集合的静态映射。
// 启动时被加载到 `permSet`,所有权限检查走 set-lookup。
var rolePermissions = map[string][]Permission{
	RoleUser: {
		PermSelfProfile,
	},
	RoleAdmin: {
		PermSelfProfile,
		PermAccountRead, PermAccountWrite,
		PermProxyRead, PermProxyWrite,
		PermModelRead, PermModelWrite,
		PermStatsReadAll,
		PermSystemSetting,
	},
}

// permSet 预计算的角色→权限集合(O(1) 查询)。
var permSet map[string]map[Permission]struct{}

func init() {
	permSet = make(map[string]map[Permission]struct{}, len(rolePermissions))
	for role, perms := range rolePermissions {
		set := make(map[Permission]struct{}, len(perms))
		for _, p := range perms {
			set[p] = struct{}{}
		}
		permSet[role] = set
	}
}

// Has 返回 role 是否拥有某权限。未知角色一律返回 false。
func Has(role string, perm Permission) bool {
	set, ok := permSet[role]
	if !ok {
		return false
	}
	_, ok = set[perm]
	return ok
}

// HasAny 任一权限成立即 true。
func HasAny(role string, perms ...Permission) bool {
	for _, p := range perms {
		if Has(role, p) {
			return true
		}
	}
	return false
}

// HasAll 所有权限都成立才 true。
func HasAll(role string, perms ...Permission) bool {
	for _, p := range perms {
		if !Has(role, p) {
			return false
		}
	}
	return true
}

// ListPermissions 返回 role 的权限清单(用于 /api/me 返回前端用作提示)。
// 返回副本,调用方可自由修改。
func ListPermissions(role string) []Permission {
	set, ok := permSet[role]
	if !ok {
		return nil
	}
	out := make([]Permission, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	return out
}

// IsAdmin 语义快捷方式。
func IsAdmin(role string) bool { return role == RoleAdmin }
