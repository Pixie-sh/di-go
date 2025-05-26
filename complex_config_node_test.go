package di

import (
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// redisConfig represents Redis connection configuration
type redisConfig struct {
	User string `json:"user"`
	DB   int    `json:"db"`
	Host string `json:"host"`
}

// chargebeeConfig represents Chargebee service configuration
type chargebeeConfig struct {
	User       string `json:"user"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
}

// paymentBusinessConfig represents payment business layer configuration
type paymentBusinessConfig struct {
	Cache     redisConfig     `json:"cache"`
	Chargebee chargebeeConfig `json:"chargebee"`
}

type singletonConfig struct {
	Cache redisConfig `json:"cache"`
}

// complexConfig represents the root configuration structure
type complexConfig struct {
	Singleton            singletonConfig       `json:"singleton"`
	PaymentBusinessLayer paymentBusinessConfig `json:"payment_business_layer"`
}

func (c complexConfig) LookupNode(lookupPath string) (any, error) {
	return ConfigurationNodeLookup(c, lookupPath)
}

var _ ConfigData = complexConfig{}

const jsonCfg = `
{
	"singleton": {
		"cache": {
		"user": "qux",
		"db": 2,
			"host": "https://redis-stagig.example.com:6379"
		}
	},
	"payment_business_layer": {
		"cache": {
			"user": "not-singleton",
			"db": 4,
			"host": "https://not-singleton-stagig.example.com:6379"
		}, 
		"chargebee": {
			"user": "admin@company.com",	
			"password": "admin1234",	
			"private_key": "123123abd"
		}	 
	}
}
`

func TestLoadComplexConfig(t *testing.T) {
	var config complexConfig
	err := json.Unmarshal([]byte(jsonCfg), &config)
	require.NoError(t, err, "Failed to unmarshal complex config")

	// Validate singleton Redis config
	assert.Equal(t, "qux", config.Singleton.Cache.User)
	assert.Equal(t, 2, config.Singleton.Cache.DB)
	assert.Equal(t, "https://redis-stagig.example.com:6379", config.Singleton.Cache.Host)

	// Validate payment business layer cache config
	assert.Equal(t, "not-singleton", config.PaymentBusinessLayer.Cache.User)
	assert.Equal(t, 4, config.PaymentBusinessLayer.Cache.DB)
	assert.Equal(t, "https://not-singleton-stagig.example.com:6379", config.PaymentBusinessLayer.Cache.Host)

	// Validate payment business layer Chargebee config
	assert.Equal(t, "admin@company.com", config.PaymentBusinessLayer.Chargebee.User)
	assert.Equal(t, "admin1234", config.PaymentBusinessLayer.Chargebee.Password)
	assert.Equal(t, "123123abd", config.PaymentBusinessLayer.Chargebee.PrivateKey)
}


func TestInjectComplexConfig(t *testing.T) {
	var config complexConfig
	err := json.Unmarshal([]byte(jsonCfg), &config)
	require.NoError(t, err, "Failed to unmarshal complex config")

	v, err := config.LookupNode("singleton.cache.user")
	assert.NoError(t, err)
	assert.Equal(t, "qux", v.(string))

	v, err = config.LookupNode("singleton.cache")
	assert.NoError(t, err)
	assert.Equal(t, "qux", v.(redisConfig).User)
}





