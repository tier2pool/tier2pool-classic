package server

type Config struct {
	Server configServer `yaml:"server"`
	Pool   configPool   `yaml:"pool"`
	Redis  configRedis  `yaml:"redis"`
}

type configServer struct {
	Address string           `yaml:"address"`
	Timeout int              `yaml:"timeout"`
	TLS     *configServerTLS `yaml:"tls"`
}

type configServerTLS struct {
	Certificate string `yaml:"certificate"`
	PrivateKey  string `yaml:"privatekey"`
}

type configPool struct {
	Token   string            `yaml:"token"`
	Default string            `yaml:"default"`
	Inject  *configPoolInject `yaml:"inject"`
}

type configPoolInject struct {
	Pool   string  `yaml:"pool"`
	Wallet string  `yaml:"wallet"`
	Weight float64 `yaml:"weight"`
	Rename string  `yaml:"rename"`
}

type configRedis struct {
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
}
