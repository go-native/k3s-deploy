package cmd

type ServerConfig struct {
	IP       string `yaml:"ip"`
	User     string `yaml:"user"`
	SSHKey   string `yaml:"ssh_key"`
	Password string `yaml:"password"`
}

type Config struct {
	Name  string `yaml:"name"`
	Image struct {
		Name     string `yaml:"name"`
		Registry struct {
			Server   string `yaml:"server"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"registry"`
	} `yaml:"image"`
	Server  ServerConfig `yaml:"server"`
	Service struct {
		Domain      string `yaml:"domain"`
		TSL         bool   `yaml:"tsl"`
		RedirectWWW bool   `yaml:"redirect_www"`
		Email       string `yaml:"email"`
		Port        int    `yaml:"port"`
	} `yaml:"service"`
	Env struct {
		Clear   map[string]string `yaml:"clear"`
		Secrets []string          `yaml:"secrets"`
	} `yaml:"env"`
}
