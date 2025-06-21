package di

import (
	"github.com/stretchr/testify/assert"
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
	Cache        singletonConfig `json:"cache"`
	SessionCache singletonConfig `json:"session_cache"`
	Chargebee    chargebeeConfig `json:"chargebee"`
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

var _ Configuration = complexConfig{}

type complexSessionCache struct {
	singletonCache redisConfig
}

type complexPaymentBizLayer struct {
	chargebee    chargebeeConfig
	cache        complexSessionCache
	sessionCache complexSessionCache
}

const jsonCfg = `
{
	"$shared": {
		"cache": {
			"user": "qux",
			"db": 2,
				"host": "https://redis-stagig.example.com:6379"
			}
	},
	"singleton": ${di.$shared},
	"payment_business_layer": {
		"session_cache": {
			"cache": "${di.$shared.cache}"
		},
		"cache": {
			"cache": {
				"user": "not-singleton",
				"db": 4,
				"host": "https://not-singleton-stagig.example.com:6379"
			}
		}, 
		"chargebee": {
			"user": "admin@company.com",	
			"password": "admin1234",	
			"private_key": "123123abd"
		}	 
	}
}
`

func TestLoadConfigWithinInjection(t *testing.T) {
	var config complexConfig
	err := UnmarshalJSONWithDIResolution([]byte(jsonCfg), &config)
	assert.NoError(t, err, "Failed to unmarshal complex config")

	_ = RegisterConfiguration[redisConfig](ConfigurationLookup[redisConfig])
	_ = RegisterConfiguration[chargebeeConfig](ConfigurationLookup[chargebeeConfig])

	_ = Register[complexSessionCache](func(c Context, opts *RegistryOpts) (complexSessionCache, error) {
		cache, err := CreateConfiguration[redisConfig](c, WithOpts(opts), WithConfigNodePath("cache"))
		if err != nil {
			return complexSessionCache{}, err
		}

		return complexSessionCache{cache}, nil
	})

	_ = Register[*complexPaymentBizLayer](func(c Context, opts *RegistryOpts) (*complexPaymentBizLayer, error) {
		chargebee, err := CreateConfiguration[chargebeeConfig](c, WithOpts(opts), WithConfigNodePath("chargebee"))
		if err != nil {
			return &complexPaymentBizLayer{}, err
		}

		singletonCache, err := Create[complexSessionCache](c, WithOpts(opts), WithConfigNodePath("session_cache"))
		if err != nil {
			return &complexPaymentBizLayer{}, err
		}

		cache, err := Create[complexSessionCache](c, WithOpts(opts), WithConfigNodePath("cache"))
		if err != nil {
			return &complexPaymentBizLayer{}, err
		}

		cpbl := complexPaymentBizLayer{
			cache:        cache,
			chargebee:    chargebee,
			sessionCache: singletonCache,
		}

		return &cpbl, nil
	}, WithToken("payment_business_layer"))

	diCtx := NewContext(config)
	paymentBizLayer, err := Create[*complexPaymentBizLayer](diCtx, WithToken("payment_business_layer"))
	if err != nil {
		t.Fatal(err)
	}

	paymentBizLayer1, err1 := Create[*complexPaymentBizLayer](diCtx, WithToken("payment_business_layer"))
	if err1 != nil {
		t.Fatal(err1)
	}

	assert.Equal(t, paymentBizLayer, paymentBizLayer1)
	assert.Same(t, paymentBizLayer, paymentBizLayer1)
	t.Log(paymentBizLayer)
}

func TestLoadComplexConfig(t *testing.T) {
	var config complexConfig
	err := UnmarshalJSONWithDIResolution([]byte(jsonCfg), &config)
	assert.NoError(t, err, "Failed to unmarshal complex config")

	// Validate singleton Redis config
	assert.Equal(t, "qux", config.Singleton.Cache.User)
	assert.Equal(t, 2, config.Singleton.Cache.DB)
	assert.Equal(t, "https://redis-stagig.example.com:6379", config.Singleton.Cache.Host)

	// Validate payment business layer cache config
	assert.Equal(t, "not-singleton", config.PaymentBusinessLayer.Cache.Cache.User)
	assert.Equal(t, 4, config.PaymentBusinessLayer.Cache.Cache.DB)
	assert.Equal(t, "https://not-singleton-stagig.example.com:6379", config.PaymentBusinessLayer.Cache.Cache.Host)

	// Validate payment business layer Chargebee config
	assert.Equal(t, "admin@company.com", config.PaymentBusinessLayer.Chargebee.User)
	assert.Equal(t, "admin1234", config.PaymentBusinessLayer.Chargebee.Password)
	assert.Equal(t, "123123abd", config.PaymentBusinessLayer.Chargebee.PrivateKey)
}

func TestInjectComplexConfig(t *testing.T) {
	var config complexConfig
	err := UnmarshalJSONWithDIResolution([]byte(jsonCfg), &config)
	assert.NoError(t, err, "Failed to unmarshal complex config")

	v, err := config.LookupNode("singleton.cache.user")
	assert.NoError(t, err)
	assert.Equal(t, "qux", v.(string))

	v, err = config.LookupNode("singleton.cache")
	assert.NoError(t, err)
	assert.Equal(t, "qux", v.(redisConfig).User)
}

func TestDIResolution(t *testing.T) {
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
		"session_cache": "${di.singleton}",
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

	var config complexConfig
	err := UnmarshalJSONWithDIResolution([]byte(jsonCfg), &config)
	assert.NoError(t, err, "Failed to unmarshal complex config with DI resolution")

	// Test the resolved values
	assert.Equal(t, "qux", config.Singleton.Cache.User)
	assert.Equal(t, "qux", config.PaymentBusinessLayer.SessionCache.Cache.User) // This should now work
	assert.Equal(t, 2, config.PaymentBusinessLayer.SessionCache.Cache.DB)
	assert.Equal(t, "https://redis-stagig.example.com:6379", config.PaymentBusinessLayer.SessionCache.Cache.Host)
}
