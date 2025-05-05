package config

// Config estrutura para configurações da aplicação
type Config struct {
	SessionPath string
	LogLevel    string
}

// NewConfig cria uma nova instância de configuração
func NewConfig() *Config {
	return &Config{
		SessionPath: "./whatsapp-session",
		LogLevel:    "INFO",
	}
}
