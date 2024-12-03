package server

type Config struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}
