package cmd

type ServerConfig struct {
	IP       string `yaml:"ip"`
	User     string `yaml:"user"`
	SSHKey   string `yaml:"ssh_key"`
	Password string `yaml:"password"`
}

type Config struct {
	Service string `yaml:"service"` // Top level service name
	Image   struct {
		Name     string `yaml:"name"`
		Registry struct {
			Server   string   `yaml:"server"`
			Username string   `yaml:"username"`
			Password []string `yaml:"password"`
		} `yaml:"registry"`
	} `yaml:"image"`
	Server  ServerConfig `yaml:"server"`
	Traffic struct {
		Domain      string `yaml:"domain"`
		TSL         bool   `yaml:"tsl"`
		RedirectWWW bool   `yaml:"redirect_www"`
		Email       string `yaml:"email"`
		Port        int    `yaml:"port"`
	} `yaml:"traffic"`
	Env struct {
		Clear   interface{} `yaml:"clear"`
		Secrets []string    `yaml:"secrets"`
	} `yaml:"env"`
}
