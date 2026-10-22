package db

import (
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestConfig_ConnString(t *testing.T) {
	for name, c := range map[string]Config{
		"plain":                {Host: "db.internal", Port: "5432", User: "bee", Password: "hunter2", DB: "scores"},
		"separators in secret": {Host: "db.internal", Port: "5432", User: "bee", Password: "p@ss/w:rd?#%", DB: "scores"},
		"spaces":               {Host: "db.internal", Port: "5432", User: "bee movie", Password: "two words", DB: "my scores"},
		"ipv6 host":            {Host: "::1", Port: "6432", User: "bee", Password: "hunter2", DB: "scores"},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := pgxpool.ParseConfig(c.connString())
			assert.NilError(t, err)
			cc := cfg.ConnConfig
			assert.Check(t, cmp.Equal(cc.Host, c.Host))
			assert.Check(t, cmp.Equal(strconv.Itoa(int(cc.Port)), c.Port))
			assert.Check(t, cmp.Equal(cc.User, c.User))
			assert.Check(t, cmp.Equal(cc.Password, c.Password))
			assert.Check(t, cmp.Equal(cc.Database, c.DB))
		})
	}
}
