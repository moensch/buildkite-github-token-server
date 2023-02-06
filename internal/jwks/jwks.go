package jwks

import (
	"context"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"go.uber.org/zap"
)

const (
	jwksURL = "https://agent.buildkite.com/.well-known/jwks"
)

type JWKS struct {
	ctx   context.Context
	url   string
	cache *jwk.Cache
}

func New(logger *zap.Logger, u string) (*JWKS, error) {
	ctx := context.Background()
	jwks := &JWKS{
		ctx: ctx,
		url: u,
	}

	jwks.cache = jwk.NewCache(jwks.ctx)

	jwks.cache.Register(jwks.url, jwk.WithMinRefreshInterval(15*time.Minute))

	_, err := jwks.Refresh()
	if err != nil {
		logger.Error("unable to fetch buildkite JWKS",
			zap.Error(err),
			zap.String("jwks_uri", jwksURL),
		)
		return nil, err
	} else {
		logger.Info("successfully fetch Buildkite JWKS",
			zap.String("jwks_uri", jwksURL),
		)
	}

	return jwks, nil
}

func (c *JWKS) Refresh() (jwk.Set, error) {
	return c.cache.Refresh(c.ctx, c.url)
}

func (c *JWKS) Get() (jwk.Set, error) {
	return c.cache.Get(c.ctx, c.url)
}
