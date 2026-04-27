package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Gateway   GatewayConfig   `mapstructure:"gateway"`
	Log       LogConfig       `mapstructure:"log"`
	MySQL     MySQLConfig     `mapstructure:"mysql"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Crypto    CryptoConfig    `mapstructure:"crypto"`
	Security  SecurityConfig  `mapstructure:"security"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	Upstream  UpstreamConfig  `mapstructure:"upstream"`
	SMTP      SMTPConfig      `mapstructure:"smtp"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Listen  string `mapstructure:"listen"`
	BaseURL string `mapstructure:"base_url"`
}

type GatewayConfig struct {
	StaticBearerToken     string `mapstructure:"static_bearer_token"`
	DefaultImagePoolID    uint64 `mapstructure:"default_image_pool_id"`
	DefaultFallbackPoolID uint64 `mapstructure:"default_fallback_pool_id"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type MySQLConfig struct {
	DSN                string `mapstructure:"dsn"`
	MaxOpenConns       int    `mapstructure:"max_open_conns"`
	MaxIdleConns       int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeSec int    `mapstructure:"conn_max_lifetime_sec"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type JWTConfig struct {
	Secret        string `mapstructure:"secret"`
	AccessTTLSec  int    `mapstructure:"access_ttl_sec"`
	RefreshTTLSec int    `mapstructure:"refresh_ttl_sec"`
	Issuer        string `mapstructure:"issuer"`
}

type CryptoConfig struct {
	AESKey string `mapstructure:"aes_key"`
}

type SecurityConfig struct {
	BcryptCost  int      `mapstructure:"bcrypt_cost"`
	CORSOrigins []string `mapstructure:"cors_origins"`
}

type SchedulerConfig struct {
	MinIntervalSec   int     `mapstructure:"min_interval_sec"`
	DailyUsageRatio  float64 `mapstructure:"daily_usage_ratio"`
	LockTTLSec       int     `mapstructure:"lock_ttl_sec"`
	Cooldown429Sec   int     `mapstructure:"cooldown_429_sec"`
	WarnedPauseHours int     `mapstructure:"warned_pause_hours"`
}

type UpstreamConfig struct {
	BaseURL            string `mapstructure:"base_url"`
	RequestTimeoutSec  int    `mapstructure:"request_timeout_sec"`
	SSEReadTimeoutSec  int    `mapstructure:"sse_read_timeout_sec"`
}

// SMTPConfig 用于注册欢迎邮件通知。
// Host 为空时邮件通道整体关闭,不影响主流程。
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`      // 显示的 From 地址
	FromName string `mapstructure:"from_name"` // 显示名
	UseTLS   bool   `mapstructure:"use_tls"`   // true 隐式 TLS(465),false STARTTLS(587)
}

var (
	global *Config
	once   sync.Once
)

func Load(path string) (*Config, error) {
	var loadErr error
	once.Do(func() {
		v := viper.New()
		v.SetConfigFile(path)
		v.SetEnvPrefix("GPT2API")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()
		if err := v.ReadInConfig(); err != nil {
			loadErr = fmt.Errorf("read config: %w", err)
			return
		}
		var c Config
		if err := v.Unmarshal(&c); err != nil {
			loadErr = fmt.Errorf("unmarshal config: %w", err)
			return
		}
		global = &c
	})
	return global, loadErr
}

// Get 返回全局配置,仅在 Load 之后调用。
func Get() *Config {
	if global == nil {
		panic("config not loaded; call config.Load first")
	}
	return global
}
