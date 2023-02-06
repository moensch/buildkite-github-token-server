package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		cfg, err := NewConfig("../../config.yaml")

		t.Logf("Config: %+v", cfg)
		require.NoError(t, err, "can parse config")
	})

	t.Run("InvalidFile", func(t *testing.T) {
		_, err := NewConfig("doesnotexist.yaml")
		assert.Error(t, err, "should error")
	})
}

func TestHost(t *testing.T) {
	c := Config{
		Applications: []*ConfigApplication{
			{
				Host: "github.com",
			},
			{
				Host: "code.hq.twilio.com",
			},
		},
	}

	t.Run("happyPath", func(t *testing.T) {
		snarf, err := c.AppConfigForHost("github.com")
		require.NoError(t, err)
		assert.Equal(t, c.Applications[0], snarf)
	})
	t.Run("notFound", func(t *testing.T) {
		_, err := c.AppConfigForHost("otherhost")
		require.Error(t, err)
	})
}

func TestInstallationID(t *testing.T) {
	c := ConfigApplication{
		Accounts: []ConfigAccount{
			{
				Name:           "twilio",
				InstallationID: 1234,
			},
			{
				Name:           "sendgrid",
				InstallationID: 7894,
			},
		},
	}

	t.Run("happyPath", func(t *testing.T) {
		installID, err := c.InstallationID("sendgrid")
		require.NoError(t, err)
		assert.Equal(t, int64(7894), installID)
	})

	t.Run("notFound", func(t *testing.T) {
		_, err := c.InstallationID("notfoundorg")
		require.Error(t, err)
	})
}
