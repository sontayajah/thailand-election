package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	DB       DBConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	JWT      JWTConfig
	Centrifugo CentrifugoConfig
	DOPA     DOPAConfig
	App      AppConfig
}

type ServerConfig struct {
	Port    string        `mapstructure:"SERVER_PORT"`
	Timeout time.Duration `mapstructure:"SERVER_TIMEOUT"`
}

type DBConfig struct {
	URL      string `mapstructure:"DATABASE_URL"`
	MaxConns int32  `mapstructure:"DB_MAX_CONNS"`
	MinConns int32  `mapstructure:"DB_MIN_CONNS"`
}

type RedisConfig struct {
	PersistentURL string `mapstructure:"REDIS_PERSISTENT_URL"`
	CacheURL      string `mapstructure:"REDIS_CACHE_URL"`
	AsynqURL      string `mapstructure:"REDIS_ASYNQ_URL"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"KAFKA_BROKERS"`
}

type JWTConfig struct {
	PrivateKeyPath string        `mapstructure:"JWT_PRIVATE_KEY_PATH"`
	PublicKeyPath  string        `mapstructure:"JWT_PUBLIC_KEY_PATH"`
	VoterTTL       time.Duration `mapstructure:"JWT_VOTER_TTL"`
	AdminTTL       time.Duration `mapstructure:"JWT_ADMIN_TTL"`
}

type CentrifugoConfig struct {
	APIEndpoint string `mapstructure:"CENTRIFUGO_API_ENDPOINT"`
	APIKey      string `mapstructure:"CENTRIFUGO_API_KEY"`
	TokenSecret string `mapstructure:"CENTRIFUGO_TOKEN_SECRET"`
}

type DOPAConfig struct {
	BaseURL string `mapstructure:"DOPA_API_URL"`
	Timeout time.Duration `mapstructure:"DOPA_TIMEOUT"`
}

type AppConfig struct {
	Env               string `mapstructure:"APP_ENV"`
	NationalIDPepper  string `mapstructure:"NATIONAL_ID_PEPPER"`
	PhoneEncKey       string `mapstructure:"PHONE_ENCRYPTION_KEY"`
	OTPDevMode        bool   `mapstructure:"OTP_DEV_MODE"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	setDefaults()

	var cfg Config

	cfg.Server.Port = viper.GetString("SERVER_PORT")
	cfg.Server.Timeout = viper.GetDuration("SERVER_TIMEOUT")

	cfg.DB.URL = viper.GetString("DATABASE_URL")
	cfg.DB.MaxConns = int32(viper.GetInt("DB_MAX_CONNS"))
	cfg.DB.MinConns = int32(viper.GetInt("DB_MIN_CONNS"))

	cfg.Redis.PersistentURL = viper.GetString("REDIS_PERSISTENT_URL")
	cfg.Redis.CacheURL = viper.GetString("REDIS_CACHE_URL")
	cfg.Redis.AsynqURL = viper.GetString("REDIS_ASYNQ_URL")

	brokerStr := viper.GetString("KAFKA_BROKERS")
	cfg.Kafka.Brokers = strings.Split(brokerStr, ",")

	cfg.JWT.PrivateKeyPath = viper.GetString("JWT_PRIVATE_KEY_PATH")
	cfg.JWT.PublicKeyPath = viper.GetString("JWT_PUBLIC_KEY_PATH")
	cfg.JWT.VoterTTL = viper.GetDuration("JWT_VOTER_TTL")
	cfg.JWT.AdminTTL = viper.GetDuration("JWT_ADMIN_TTL")

	cfg.Centrifugo.APIEndpoint = viper.GetString("CENTRIFUGO_API_ENDPOINT")
	cfg.Centrifugo.APIKey = viper.GetString("CENTRIFUGO_API_KEY")
	cfg.Centrifugo.TokenSecret = viper.GetString("CENTRIFUGO_TOKEN_SECRET")

	cfg.DOPA.BaseURL = viper.GetString("DOPA_API_URL")
	cfg.DOPA.Timeout = viper.GetDuration("DOPA_TIMEOUT")

	cfg.App.Env = viper.GetString("APP_ENV")
	cfg.App.NationalIDPepper = viper.GetString("NATIONAL_ID_PEPPER")
	cfg.App.PhoneEncKey = viper.GetString("PHONE_ENCRYPTION_KEY")
	cfg.App.OTPDevMode = viper.GetBool("OTP_DEV_MODE")

	return &cfg, nil
}

func (c *Config) IsDevMode() bool {
	return c.App.Env == "development" || c.App.Env == "dev"
}

func setDefaults() {
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("SERVER_TIMEOUT", "30s")
	viper.SetDefault("DB_MAX_CONNS", 20)
	viper.SetDefault("DB_MIN_CONNS", 2)
	viper.SetDefault("JWT_VOTER_TTL", "30m")
	viper.SetDefault("JWT_ADMIN_TTL", "15m")
	viper.SetDefault("DOPA_TIMEOUT", "5s")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("OTP_DEV_MODE", true)
	viper.SetDefault("CENTRIFUGO_API_ENDPOINT", "http://centrifugo:8000/api")
	viper.SetDefault("DOPA_API_URL", "http://dopa-mock:9090")
}
