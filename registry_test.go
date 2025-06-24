package di

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type someType struct {
	cfg someTypeConfig
}
type someTypeConfig struct {
	A string
}

func (s someTypeConfig) LookupNode(lookupPath string) (any, error) {
	panic("implement me")
}

func testRegistry(f Registry) error {
	err := RegisterPair[someType, someTypeConfig](func(context Context, opts *RegistryOpts, config someTypeConfig) (someType, error) {
		return someType{cfg: config}, nil
	}, func(c Context, opts *RegistryOpts) (someTypeConfig, error) {
		return someTypeConfig{"B"}, nil
	})
	if err != nil {
		return err
	}

	err = Register[someType](func(context Context, opts *RegistryOpts) (someType, error) {
		cfg, err := CreateConfiguration[someTypeConfig](context, WithOpts(opts))
		if err != nil {
			return someType{}, err
		}

		return someType{cfg}, nil
	})

	err = RegisterConfiguration[someTypeConfig](func(context Context, opts *RegistryOpts) (someTypeConfig, error) {
		return someTypeConfig{"A"}, nil
	})

	return err
}

func TestCreateNoConfig(t *testing.T) {
	assert.NoError(t, testRegistry(Instance))

	instance, err := CreatePair[someType, someTypeConfig](NewContext())
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Equal(t, "B", instance.cfg.A)

	noCfgInstance, err := Create[someType](NewContext())
	assert.NoError(t, err)
	assert.NotNil(t, noCfgInstance)
	assert.Equal(t, "A", noCfgInstance.cfg.A)
}
