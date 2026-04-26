package rbac

// Menu 是一棵静态菜单树,前端按当前用户角色可见子集渲染。
//
// 每一项菜单都绑定一个或多个权限;只要用户拥有其中任意一个,该项就出现。
// 未绑定权限的菜单(Perms 为 nil)对所有已登录用户可见。
//
// 重要:菜单只是 UI 暗示,后端接口必须独立做 `middleware.RequirePerm(...)`。
type Menu struct {
	Key      string       `json:"key"`                // 唯一标识,前端路由用
	Title    string       `json:"title"`              // 显示名(简体中文)
	Icon     string       `json:"icon,omitempty"`     // Element Plus 图标名
	Path     string       `json:"path,omitempty"`     // 前端路由路径
	Perms    []Permission `json:"-"`                  // 必需权限(any);空=仅需已登录
	Children []Menu       `json:"children,omitempty"` // 子菜单
}

// menuTree 静态全量菜单树。对应前端路由。
var menuTree = []Menu{
	{
		Key: "admin", Title: "后台管理", Icon: "Setting", Path: "/admin",
		Perms: []Permission{PermAccountRead, PermProxyRead, PermSystemSetting},
		Children: []Menu{
			{Key: "admin.accounts", Title: "GPT账号", Icon: "Connection", Path: "/admin/accounts",
				Perms: []Permission{PermAccountRead}},
			{Key: "admin.proxies", Title: "代理管理", Icon: "Guide", Path: "/admin/proxies",
				Perms: []Permission{PermProxyRead}},
			{Key: "admin.account-pools", Title: "账号池", Icon: "Collection", Path: "/admin/account-pools",
				Perms: []Permission{PermAccountRead}},
			{Key: "admin.account-pool-routes", Title: "池路由", Icon: "Share", Path: "/admin/account-pool-routes",
				Perms: []Permission{PermAccountRead, PermAccountWrite}},
			{Key: "admin.settings", Title: "系统设置", Icon: "Tools", Path: "/admin/settings",
				Perms: []Permission{PermSystemSetting}},
		},
	},
}

// MenuForRole 按角色过滤菜单树。节点没有可见子节点时递归裁剪。
func MenuForRole(role string) []Menu {
	return filterMenus(menuTree, role)
}

func filterMenus(src []Menu, role string) []Menu {
	out := make([]Menu, 0, len(src))
	for _, m := range src {
		// 复制一份避免对源数据做任何写入
		copied := m
		copied.Children = filterMenus(m.Children, role)

		// 可见性规则:
		//   - 无 Perms 限制 → 所有已登录可见(但若有 children,仍要求 children 非空)
		//   - 有 Perms → 必须 role 拥有其中任一权限
		visible := true
		if len(copied.Perms) > 0 {
			visible = HasAny(role, copied.Perms...)
		}
		// 无子节点 + 不可见 → 跳过
		// 有子节点但 children 被裁成 0 → 如果自己也不可见,跳过
		if !visible && len(copied.Children) == 0 {
			continue
		}
		if len(m.Children) > 0 && len(copied.Children) == 0 && !visible {
			continue
		}
		out = append(out, copied)
	}
	return out
}
