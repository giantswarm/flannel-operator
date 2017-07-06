package totp

import (
	"strings"
	"time"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
	cache "github.com/patrickmn/go-cache"
)

func Factory(conf *logical.BackendConfig) (logical.Backend, error) {
	return Backend(conf).Setup(conf)
}

func Backend(conf *logical.BackendConfig) *backend {
	var b backend
	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(backendHelp),

		Paths: []*framework.Path{
			pathListKeys(&b),
			pathKeys(&b),
			pathCode(&b),
		},

		Secrets: []*framework.Secret{},
	}

	b.usedCodes = cache.New(0, 30*time.Second)

	return &b
}

type backend struct {
	*framework.Backend

	usedCodes *cache.Cache
}

const backendHelp = `
The TOTP backend dynamically generates time-based one-time use passwords.
`
