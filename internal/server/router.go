package server

import (
	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/internal/account"
	"github.com/432539/gpt2api/internal/accountpool"
	"github.com/432539/gpt2api/internal/accountsource"
	"github.com/432539/gpt2api/internal/apikey"
	"github.com/432539/gpt2api/internal/audit"
	"github.com/432539/gpt2api/internal/auth"
	"github.com/432539/gpt2api/internal/backup"
	"github.com/432539/gpt2api/internal/config"
	"github.com/432539/gpt2api/internal/gateway"
	"github.com/432539/gpt2api/internal/image"
	"github.com/432539/gpt2api/internal/middleware"
	"github.com/432539/gpt2api/internal/model"
	"github.com/432539/gpt2api/internal/proxy"
	"github.com/432539/gpt2api/internal/rbac"
	"github.com/432539/gpt2api/internal/recharge"
	"github.com/432539/gpt2api/internal/settings"
	"github.com/432539/gpt2api/internal/usage"
	"github.com/432539/gpt2api/internal/user"
	pkgjwt "github.com/432539/gpt2api/pkg/jwt"
	"github.com/432539/gpt2api/pkg/resp"
)

// Deps 汇集路由依赖。
type Deps struct {
	Config *config.Config
	JWT    *pkgjwt.Manager

	AuthH *auth.Handler
	UserH *user.Handler

	KeySvc         *apikey.Service
	KeyH           *apikey.Handler
	ProxyH         *proxy.Handler
	AccountH       *account.Handler
	AccountPoolH   *accountpool.Handler
	AccountSourceH *accountsource.Handler

	GatewayH *gateway.Handler
	ImagesH  *gateway.ImagesHandler

	BackupH     *backup.Handler
	AuditH      *audit.Handler
	AuditDAO    *audit.DAO
	AdminUserH  *user.AdminHandler
	AdminGroupH *user.AdminGroupHandler

	AdminModelH *model.AdminHandler
	AdminKeyH   *apikey.AdminHandler
	AdminUsageH *usage.AdminHandler

	// 生成面板:当前用户视角的 usage / image 只读接口
	MeUsageH *usage.MeHandler
	MeImageH *image.MeHandler

	RechargeH      *recharge.Handler
	AdminRechargeH *recharge.AdminHandler

	SettingsH *settings.Handler
}

// New 构建 gin.Engine 并挂载最终保留的最小路由集。
func New(d *Deps) *gin.Engine {
	if d.Config.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Recover(),
		middleware.AccessLog(),
		middleware.CORS(d.Config.Security.CORSOrigins),
	)

	r.GET("/healthz", func(c *gin.Context) { resp.OK(c, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) { resp.OK(c, gin.H{"status": "ok"}) })

	api := r.Group("/api")
	{
		authGrp := api.Group("/auth")
		{
			authGrp.POST("/register", d.AuthH.Register)
			authGrp.POST("/login", d.AuthH.Login)
			authGrp.POST("/refresh", d.AuthH.Refresh)
		}

		authed := api.Group("", middleware.JWTAuth(d.JWT))
		{
			authed.GET("/me", d.UserH.Me)
			authed.GET("/me/menu", d.UserH.Menu)
		}

		adminMW := []gin.HandlerFunc{
			middleware.JWTAuth(d.JWT),
			middleware.RequireAdmin(),
		}
		if d.AuditDAO != nil {
			adminMW = append(adminMW, audit.Middleware(d.AuditDAO))
		}

		admin := api.Group("/admin", adminMW...)
		{
			admin.GET("/ping", func(c *gin.Context) { resp.OK(c, gin.H{"msg": "admin pong"}) })

			if d.ProxyH != nil {
				pg := admin.Group("/proxies", middleware.RequirePerm(rbac.PermProxyRead, rbac.PermProxyWrite))
				{
					pg.POST("", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Create)
					pg.POST("/import", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Import)
					pg.POST("/probe-all", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.ProbeAll)
					pg.GET("", d.ProxyH.List)
					pg.GET("/:id", d.ProxyH.Get)
					pg.POST("/:id/probe", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Probe)
					pg.PATCH("/:id", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Update)
					pg.DELETE("/:id", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Delete)
				}
			}

			if d.AccountH != nil {
				ag := admin.Group("/accounts", middleware.RequirePerm(rbac.PermAccountRead, rbac.PermAccountWrite))
				{
					ag.POST("", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Create)
					ag.POST("/import", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Import)
					ag.POST("/import-tokens", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.ImportTokens)
					ag.POST("/refresh-all", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.RefreshAll)
					ag.POST("/probe-quota-all", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.ProbeQuotaAll)
					ag.POST("/bulk-delete", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.BulkDelete)
					ag.GET("/auto-refresh", d.AccountH.GetAutoRefresh)
					ag.PUT("/auto-refresh", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.SetAutoRefresh)
					ag.GET("", d.AccountH.List)
					ag.GET("/:id", d.AccountH.Get)
					ag.GET("/:id/secrets", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.GetSecrets)
					ag.PATCH("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Update)
					ag.DELETE("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Delete)
					ag.POST("/:id/refresh", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Refresh)
					ag.POST("/:id/probe-quota", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.ProbeQuota)
					ag.POST("/:id/bind-proxy", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.BindProxy)
					ag.DELETE("/:id/bind-proxy", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.UnbindProxy)
				}
			}

			if d.AccountPoolH != nil {
				pg := admin.Group("/account-pools", middleware.RequirePerm(rbac.PermAccountRead, rbac.PermAccountWrite))
				{
					pg.GET("", d.AccountPoolH.ListPools)
					pg.POST("", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.CreatePool)
					pg.GET("/:id", d.AccountPoolH.GetPool)
					pg.PATCH("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.UpdatePool)
					pg.DELETE("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.DeletePool)
					pg.GET("/:id/members", d.AccountPoolH.ListMembers)
					pg.POST("/:id/members", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.UpsertMember)
					pg.PATCH("/:id/members/:memberId", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.UpsertMember)
					pg.DELETE("/:id/members/:memberId", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountPoolH.DeleteMember)
				}

				admin.GET("/account-pool-routes",
					middleware.RequirePerm(rbac.PermAccountRead, rbac.PermAccountWrite),
					d.AccountPoolH.ListRoutes)
				admin.PUT("/account-pool-routes/:modelId",
					middleware.RequirePerm(rbac.PermAccountWrite),
					d.AccountPoolH.PutRoute)
				admin.DELETE("/account-pool-routes/:modelId",
					middleware.RequirePerm(rbac.PermAccountWrite),
					d.AccountPoolH.DeleteRoute)
			}

			if d.AccountSourceH != nil {
				sg := admin.Group("/account-import-sources", middleware.RequirePerm(rbac.PermAccountRead, rbac.PermAccountWrite))
				{
					sg.GET("", d.AccountSourceH.List)
					sg.POST("", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountSourceH.Create)
					sg.GET("/:id", d.AccountSourceH.Get)
					sg.PATCH("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountSourceH.Update)
					sg.DELETE("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountSourceH.Delete)
					sg.GET("/:id/sub2api/groups", d.AccountSourceH.ListSub2APIGroups)
					sg.GET("/:id/sub2api/accounts", d.AccountSourceH.ListSub2APIAccounts)
					sg.GET("/:id/cpa/files", d.AccountSourceH.ListCPAFiles)
					sg.POST("/:id/import", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountSourceH.ImportSelected)
				}
			}

			if d.AdminModelH != nil {
				admin.GET("/models", middleware.RequirePerm(rbac.PermAccountRead), d.AdminModelH.List)
			}

			if d.SettingsH != nil {
				sg := admin.Group("/settings", middleware.RequirePerm(rbac.PermSystemSetting))
				{
					sg.GET("", d.SettingsH.List)
					sg.PATCH("", d.SettingsH.Update)
					sg.POST("/reload", d.SettingsH.Reload)
				}
			}
		}
	}

	token := d.Config.Gateway.StaticBearerToken
	if token == "" {
		token = d.Config.JWT.Secret
	}

	v1 := r.Group("/v1", middleware.StaticBearerAuth(token))
	{
		v1.GET("/models", d.GatewayH.ListModels)
		if d.ImagesH != nil {
			v1.POST("/images/generations", d.ImagesH.ImageGenerations)
		}
	}

	if d.ImagesH != nil {
		r.GET("/p/img/:task_id/:idx", d.ImagesH.ImageProxy)
	}

	mountSPA(r)

	return r
}
