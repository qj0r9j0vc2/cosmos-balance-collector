package main

var DEFAULT_TIMEOUT = 3

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	Chains map[string]struct {
		StakingTokenDenom string `yaml:"stakingTokenDenom"`
		RPCUrl            string `yaml:"rpcURL"`
		Timeout           int    `yaml:"timeout"`
		Client            Client
	} `yaml:"chains"`
}

func (c *Config) getChains() []string {
	var result []string
	for k, _ := range c.Chains {
		result = append(result, k)
	}
	return result
}
