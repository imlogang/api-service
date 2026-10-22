package setup

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// spanFields builds the map the sampler sees: ex converts each span attribute
// with Value.AsInterface, so integers arrive as int64.
func spanFields(attrs ...attribute.KeyValue) map[string]any {
	fields := map[string]any{}
	for _, a := range attrs {
		fields[string(a.Key)] = a.Value.AsInterface()
	}
	return fields
}

func TestDefaultSampleFn(t *testing.T) {
	for name, tc := range map[string]struct {
		fields map[string]any
		want   string
	}{
		"error is always kept": {
			fields: spanFields(attribute.String("error", "boom"), attribute.String("name", "GetScoreHandler")),
			want:   includeAll,
		},
		"5xx is always kept": {
			fields: spanFields(
				attribute.String("http.server_name", "internalapi"),
				attribute.String("http.route", "/api/private/get_round"),
				attribute.Int("http.status_code", 500),
			),
			want: includeAll,
		},
		"503 is always kept": {
			fields: spanFields(attribute.Int("http.status_code", 503)),
			want:   includeAll,
		},
		"warning is kept often": {
			fields: spanFields(attribute.String("warning", "slow"), attribute.String("name", "x")),
			want:   includeMany,
		},
		"requests are keyed by route and status": {
			fields: spanFields(
				attribute.String("http.server_name", "internalapi"),
				attribute.String("http.route", "/api/private/leaderboard"),
				attribute.Int("http.status_code", 200),
			),
			want: "/api/private/leaderboard 200",
		},
		"4xx is keyed like any other request": {
			fields: spanFields(
				attribute.String("http.server_name", "internalapi"),
				attribute.String("http.route", "/api/private/get_round"),
				attribute.Int("http.status_code", 400),
			),
			want: "/api/private/get_round 400",
		},
		"db spans are kept often": {
			fields: spanFields(attribute.String("name", "db: query")),
			want:   includeMany,
		},
		"other spans are keyed by name": {
			fields: spanFields(attribute.String("name", "GetRoundHandler")),
			want:   "GetRoundHandler",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Check(t, cmp.Equal(defaultSampleFn(tc.fields), tc.want))
		})
	}
}

func TestIntField(t *testing.T) {
	fields := map[string]any{"int": 500, "int64": int64(502), "string": "500"}
	assert.Check(t, cmp.Equal(intField(fields, "int"), 500))
	assert.Check(t, cmp.Equal(intField(fields, "int64"), 502))
	assert.Check(t, cmp.Equal(intField(fields, "string"), 0))
	assert.Check(t, cmp.Equal(intField(fields, "missing"), 0))
}
