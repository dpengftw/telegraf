package parser

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/parsers"
	"github.com/influxdata/telegraf/plugins/parsers/binary"
	"github.com/influxdata/telegraf/plugins/parsers/grok"
	"github.com/influxdata/telegraf/plugins/parsers/influx"
	"github.com/influxdata/telegraf/plugins/parsers/json"
	"github.com/influxdata/telegraf/plugins/parsers/logfmt"
	"github.com/influxdata/telegraf/plugins/parsers/value"
	"github.com/influxdata/telegraf/testutil"
)

func TestApply(t *testing.T) {
	tests := []struct {
		name         string
		parseFields  []string
		parseTags    []string
		parseBase64  []string
		parser       telegraf.Parser
		dropOriginal bool
		merge        string
		input        telegraf.Metric
		expected     []telegraf.Metric
	}{
		{
			name:         "parse one field drop original",
			parseFields:  []string{"sample"},
			dropOriginal: true,
			parser: &json.Parser{
				TagKeys: []string{
					"ts",
					"lvl",
					"msg",
					"method",
				},
			},
			input: metric.New(
				"singleField",
				map[string]string{
					"some": "tag",
				},
				map[string]interface{}{
					"sample": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"singleField",
					map[string]string{
						"ts":     "2018-07-24T19:43:40.275Z",
						"lvl":    "info",
						"msg":    "http request",
						"method": "POST",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse one field with merge",
			parseFields:  []string{"sample"},
			dropOriginal: false,
			merge:        "override",
			parser: &json.Parser{
				TagKeys: []string{
					"ts",
					"lvl",
					"msg",
					"method",
				},
			},
			input: metric.New(
				"singleField",
				map[string]string{
					"some": "tag",
				},
				map[string]interface{}{
					"sample": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"singleField",
					map[string]string{
						"some":   "tag",
						"ts":     "2018-07-24T19:43:40.275Z",
						"lvl":    "info",
						"msg":    "http request",
						"method": "POST",
					},
					map[string]interface{}{
						"sample": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse one field keep",
			parseFields:  []string{"sample"},
			dropOriginal: false,
			parser: &json.Parser{
				TagKeys: []string{
					"ts",
					"lvl",
					"msg",
					"method",
				},
			},
			input: metric.New(
				"singleField",
				map[string]string{
					"some": "tag",
				},
				map[string]interface{}{
					"sample": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"singleField",
					map[string]string{
						"some": "tag",
					},
					map[string]interface{}{
						"sample": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
					},
					time.Unix(0, 0)),
				metric.New(
					"singleField",
					map[string]string{
						"ts":     "2018-07-24T19:43:40.275Z",
						"lvl":    "info",
						"msg":    "http request",
						"method": "POST",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse one field keep with measurement name",
			parseFields:  []string{"message"},
			parser:       &influx.Parser{},
			dropOriginal: false,
			input: metric.New(
				"influxField",
				map[string]string{},
				map[string]interface{}{
					"message": "deal,computer_name=hosta message=\"stuff\" 1530654676316265790",
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"influxField",
					map[string]string{},
					map[string]interface{}{
						"message": "deal,computer_name=hosta message=\"stuff\" 1530654676316265790",
					},
					time.Unix(0, 0)),
				metric.New(
					"deal",
					map[string]string{
						"computer_name": "hosta",
					},
					map[string]interface{}{
						"message": "stuff",
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse one field override replaces name",
			parseFields:  []string{"message"},
			dropOriginal: false,
			merge:        "override",
			parser:       &influx.Parser{},
			input: metric.New(
				"influxField",
				map[string]string{
					"some": "tag",
				},
				map[string]interface{}{
					"message": "deal,computer_name=hosta message=\"stuff\" 1530654676316265790",
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"deal",
					map[string]string{
						"computer_name": "hosta",
						"some":          "tag",
					},
					map[string]interface{}{
						"message": "stuff",
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse grok field",
			parseFields:  []string{"grokSample"},
			dropOriginal: true,
			parser: &grok.Parser{
				Patterns: []string{"%{COMBINED_LOG_FORMAT}"},
			},
			input: metric.New(
				"success",
				map[string]string{},
				map[string]interface{}{
					"grokSample": "127.0.0.1 - - [11/Dec/2013:00:01:45 -0800] \"" +
						"GET /xampp/status.php HTTP/1.1\" 200 3891 \"http://cadenza/xampp/navi.php\" \"Mozilla/5.0 (Macintosh; " +
						"Intel Mac OS X 10.9; rv:25.0) Gecko/20100101 Firefox/25.0\"",
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"success",
					map[string]string{
						"resp_code": "200",
						"verb":      "GET",
					},
					map[string]interface{}{
						"resp_bytes":   int64(3891),
						"auth":         "-",
						"request":      "/xampp/status.php",
						"referrer":     "http://cadenza/xampp/navi.php",
						"agent":        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:25.0) Gecko/20100101 Firefox/25.0",
						"client_ip":    "127.0.0.1",
						"ident":        "-",
						"http_version": float64(1.1),
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse two fields [replace]",
			parseFields:  []string{"field_1", "field_2"},
			dropOriginal: true,
			parser: &json.Parser{
				TagKeys: []string{"lvl", "err"},
			},
			input: metric.New(
				"bigMeasure",
				map[string]string{},
				map[string]interface{}{
					"field_1": `{"lvl":"info","msg":"http request"}`,
					"field_2": `{"err":"fatal","fatal":"security threat"}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"bigMeasure",
					map[string]string{
						"lvl": "info",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
				metric.New(
					"bigMeasure",
					map[string]string{
						"err": "fatal",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse two fields [merge]",
			parseFields:  []string{"field_1", "field_2"},
			dropOriginal: false,
			merge:        "override",
			parser: &json.Parser{
				TagKeys: []string{"lvl", "msg", "err", "fatal"},
			},
			input: metric.New(
				"bigMeasure",
				map[string]string{},
				map[string]interface{}{
					"field_1": `{"lvl":"info","msg":"http request"}`,
					"field_2": `{"err":"fatal","fatal":"security threat"}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"bigMeasure",
					map[string]string{
						"lvl":   "info",
						"msg":   "http request",
						"err":   "fatal",
						"fatal": "security threat",
					},
					map[string]interface{}{
						"field_1": `{"lvl":"info","msg":"http request"}`,
						"field_2": `{"err":"fatal","fatal":"security threat"}`,
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse two fields [keep]",
			parseFields:  []string{"field_1", "field_2"},
			dropOriginal: false,
			parser: &json.Parser{
				TagKeys: []string{"lvl", "msg", "err", "fatal"},
			},
			input: metric.New(
				"bigMeasure",
				map[string]string{},
				map[string]interface{}{
					"field_1": `{"lvl":"info","msg":"http request"}`,
					"field_2": `{"err":"fatal","fatal":"security threat"}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"bigMeasure",
					map[string]string{},
					map[string]interface{}{
						"field_1": `{"lvl":"info","msg":"http request"}`,
						"field_2": `{"err":"fatal","fatal":"security threat"}`,
					},
					time.Unix(0, 0)),
				metric.New(
					"bigMeasure",
					map[string]string{
						"lvl": "info",
						"msg": "http request",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
				metric.New(
					"bigMeasure",
					map[string]string{
						"err":   "fatal",
						"fatal": "security threat",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse one tag drop original",
			parseTags:    []string{"sample"},
			dropOriginal: true,
			parser:       &logfmt.Parser{},
			input: metric.New(
				"singleTag",
				map[string]string{
					"some":   "tag",
					"sample": `ts=2018-07-24T19:43:40.275Z`,
				},
				map[string]interface{}{},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"singleTag",
					map[string]string{},
					map[string]interface{}{
						"ts": "2018-07-24T19:43:40.275Z",
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse one tag with merge",
			parseTags:    []string{"sample"},
			dropOriginal: false,
			merge:        "override",
			parser:       &logfmt.Parser{},
			input: metric.New(
				"singleTag",
				map[string]string{
					"some":   "tag",
					"sample": `ts=2018-07-24T19:43:40.275Z`,
				},
				map[string]interface{}{},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"singleTag",
					map[string]string{
						"some":   "tag",
						"sample": `ts=2018-07-24T19:43:40.275Z`,
					},
					map[string]interface{}{
						"ts": "2018-07-24T19:43:40.275Z",
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse one tag keep",
			parseTags:    []string{"sample"},
			dropOriginal: false,
			parser:       &logfmt.Parser{},
			input: metric.New(
				"singleTag",
				map[string]string{
					"some":   "tag",
					"sample": `ts=2018-07-24T19:43:40.275Z`,
				},
				map[string]interface{}{},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"singleTag",
					map[string]string{
						"some":   "tag",
						"sample": `ts=2018-07-24T19:43:40.275Z`,
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
				metric.New(
					"singleTag",
					map[string]string{},
					map[string]interface{}{
						"ts": "2018-07-24T19:43:40.275Z",
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "Fail to parse one field but parses other [keep]",
			parseFields:  []string{"good", "bad"},
			dropOriginal: false,
			parser: &json.Parser{
				TagKeys: []string{"lvl"},
			},
			input: metric.New(
				"success",
				map[string]string{},
				map[string]interface{}{
					"good": `{"lvl":"info"}`,
					"bad":  "why",
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"success",
					map[string]string{},
					map[string]interface{}{
						"good": `{"lvl":"info"}`,
						"bad":  "why",
					},
					time.Unix(0, 0)),
				metric.New(
					"success",
					map[string]string{
						"lvl": "info",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "Fail to parse one field but parses other [keep] v2",
			parseFields:  []string{"bad", "good", "ok"},
			dropOriginal: false,
			parser: &json.Parser{
				TagKeys: []string{"lvl", "thing"},
			},
			input: metric.New(
				"success",
				map[string]string{},
				map[string]interface{}{
					"bad":  "why",
					"good": `{"lvl":"info"}`,
					"ok":   `{"thing":"thang"}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"success",
					map[string]string{},
					map[string]interface{}{
						"bad":  "why",
						"good": `{"lvl":"info"}`,
						"ok":   `{"thing":"thang"}`,
					},
					time.Unix(0, 0)),
				metric.New(
					"success",
					map[string]string{
						"lvl": "info",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
				metric.New(
					"success",
					map[string]string{
						"thing": "thang",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "Fail to parse one field but parses other [merge]",
			parseFields:  []string{"good", "bad"},
			dropOriginal: false,
			merge:        "override",
			parser: &json.Parser{
				TagKeys: []string{"lvl"},
			},
			input: metric.New(
				"success",
				map[string]string{
					"a": "tag",
				},
				map[string]interface{}{
					"good": `{"lvl":"info"}`,
					"bad":  "why",
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"success",
					map[string]string{
						"a":   "tag",
						"lvl": "info",
					},
					map[string]interface{}{
						"good": `{"lvl":"info"}`,
						"bad":  "why",
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "Fail to parse one field but parses other [replace]",
			parseFields:  []string{"good", "bad"},
			dropOriginal: true,
			parser: &json.Parser{
				TagKeys: []string{"lvl"},
			},
			input: metric.New(
				"success",
				map[string]string{
					"thing": "tag",
				},
				map[string]interface{}{
					"good": `{"lvl":"info"}`,
					"bad":  "why",
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"success",
					map[string]string{
						"lvl": "info",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:        "parser without metric name (issue #12115)",
			parseFields: []string{"value"},
			merge:       "override",
			// Create parser the config way with the name of the parent plugin.
			parser: func() telegraf.Parser {
				p := parsers.Parsers["value"]("parser")
				vp := p.(*value.Parser)
				vp.DataType = "float"
				vp.FieldName = "value"
				return vp
			}(),
			input: metric.New(
				"myname",
				map[string]string{},
				map[string]interface{}{"value": "7.2"},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"myname",
					map[string]string{},
					map[string]interface{}{"value": float64(7.2)},
					time.Unix(0, 0)),
			},
		},
		{
			name:        "parser with metric name (issue #12115)",
			parseFields: []string{"value"},
			merge:       "override",
			// Create parser the config way with the name of the parent plugin.
			parser: parsers.Parsers["influx"]("parser"),
			input: metric.New(
				"myname",
				map[string]string{},
				map[string]interface{}{"value": "test value=7.2"},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"test",
					map[string]string{},
					map[string]interface{}{"value": float64(7.2)},
					time.Unix(0, 0)),
			},
		},
		{
			name:        "override with timestamp",
			parseFields: []string{"value"},
			merge:       "override-with-timestamp",
			parser: &json.Parser{
				TimeKey:    "timestamp",
				TimeFormat: "2006-01-02 15:04:05",
			},
			input: metric.New(
				"myname",
				map[string]string{},
				map[string]interface{}{
					"value": `{"timestamp": "2020-06-27 19:43:40", "value": 42.1}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"myname",
					map[string]string{},
					map[string]interface{}{
						"value": float64(42.1),
					},
					time.Unix(1593287020, 0)),
			},
		},
		{
			name:        "non-string field with binary parser",
			parseFields: []string{"value"},
			merge:       "override",
			parser: &binary.Parser{
				Configs: []binary.Config{
					{
						MetricName: "parser",
						Entries: []binary.Entry{
							{
								Name: "alarm_0",
								Type: "bool",
								Bits: 1,
							},
							{
								Name: "alarm_1",
								Type: "bool",
								Bits: 1,
							},
							{
								Name: "alarm_2",
								Type: "bool",
								Bits: 1,
							},
							{
								Name: "alarm_3",
								Type: "bool",
								Bits: 1,
							},
							{
								Name: "alarm_4",
								Type: "bool",
								Bits: 1,
							},
							{
								Name: "alarm_5",
								Type: "bool",
								Bits: 1,
							},
							{
								Name: "alarm_6",
								Type: "bool",
								Bits: 1,
							},
							{
								Name: "alarm_7",
								Type: "bool",
								Bits: 1,
							},
						},
					},
				},
			},
			input: metric.New(
				"myname",
				map[string]string{},
				map[string]interface{}{
					"value": uint8(13),
				},
				time.Unix(1593287020, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"myname",
					map[string]string{},
					map[string]interface{}{
						"value":   uint8(13),
						"alarm_0": false,
						"alarm_1": false,
						"alarm_2": false,
						"alarm_3": false,
						"alarm_4": true,
						"alarm_5": true,
						"alarm_6": false,
						"alarm_7": true,
					},
					time.Unix(1593287020, 0)),
			},
		},
		{
			name:         "test base 64 field single",
			parseBase64:  []string{"sample"},
			dropOriginal: true,
			parser: &json.Parser{
				TagKeys: []string{
					"text",
				},
			},
			input: metric.New(
				"singleField",
				map[string]string{
					"some": "tag",
				},
				map[string]interface{}{
					"sample": `eyJ0ZXh0IjogInRlc3QgYmFzZTY0In0=`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"singleField",
					map[string]string{
						"text": "test base64",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse two base64 fields",
			parseBase64:  []string{"field_1", "field_2"},
			dropOriginal: true,
			parser: &json.Parser{
				TagKeys: []string{"lvl", "msg", "err", "fatal"},
			},
			input: metric.New(
				"bigMeasure",
				map[string]string{},
				map[string]interface{}{
					"field_1": `eyJsdmwiOiJpbmZvIiwibXNnIjoiaHR0cCByZXF1ZXN0In0=`,
					"field_2": `eyJlcnIiOiJmYXRhbCIsImZhdGFsIjoic2VjdXJpdHkgdGhyZWF0In0=`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"bigMeasure",
					map[string]string{
						"lvl": "info",
						"msg": "http request",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
				metric.New(
					"bigMeasure",
					map[string]string{
						"err":   "fatal",
						"fatal": "security threat",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
		{
			name:         "parse two fields, one base64",
			parseFields:  []string{"field_2"},
			parseBase64:  []string{"field_1"},
			dropOriginal: true,
			parser: &json.Parser{
				TagKeys: []string{"lvl", "msg", "err", "fatal"},
			},
			input: metric.New(
				"bigMeasure",
				map[string]string{},
				map[string]interface{}{
					"field_1": `eyJsdmwiOiJpbmZvIiwibXNnIjoiaHR0cCByZXF1ZXN0In0=`,
					"field_2": `{"err":"fatal","fatal":"security threat"}`,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"bigMeasure",
					map[string]string{
						"lvl": "info",
						"msg": "http request",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
				metric.New(
					"bigMeasure",
					map[string]string{
						"err":   "fatal",
						"fatal": "security threat",
					},
					map[string]interface{}{},
					time.Unix(0, 0)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if p, ok := tt.parser.(telegraf.Initializer); ok {
				require.NoError(t, p.Init())
			}
			plugin := Parser{
				ParseFields:  tt.parseFields,
				ParseTags:    tt.parseTags,
				Base64Fields: tt.parseBase64,
				DropOriginal: tt.dropOriginal,
				Merge:        tt.merge,
				Log:          testutil.Logger{Name: "processor.parser"},
			}
			plugin.SetParser(tt.parser)

			output := plugin.Apply(tt.input)
			t.Logf("Testing: %s", tt.name)

			// check timestamp when using with-timestamp merge type
			if tt.merge == "override-with-timestamp" {
				testutil.RequireMetricsEqual(t, tt.expected, output, testutil.SortMetrics())
			} else {
				testutil.RequireMetricsEqual(t, tt.expected, output, testutil.SortMetrics(), testutil.IgnoreTime())
			}
		})
	}
}

func TestInvalidMerge(t *testing.T) {
	plugin := Parser{Merge: "fake"}
	require.Error(t, plugin.Init())
}

func TestBadApply(t *testing.T) {
	tests := []struct {
		name        string
		parseFields []string
		parser      telegraf.Parser
		input       telegraf.Metric
		expected    []telegraf.Metric
	}{
		{
			name:        "field not found",
			parseFields: []string{"bad_field"},
			parser:      &json.Parser{},
			input: metric.New(
				"bad",
				map[string]string{},
				map[string]interface{}{
					"some_field": 5,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"bad",
					map[string]string{},
					map[string]interface{}{
						"some_field": 5,
					},
					time.Unix(0, 0)),
			},
		},
		{
			name:        "non string field",
			parseFields: []string{"some_field"},
			parser:      &json.Parser{},
			input: metric.New(
				"bad",
				map[string]string{},
				map[string]interface{}{
					"some_field": 5,
				},
				time.Unix(0, 0)),
			expected: []telegraf.Metric{
				metric.New(
					"bad",
					map[string]string{},
					map[string]interface{}{
						"some_field": 5,
					},
					time.Unix(0, 0)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if p, ok := tt.parser.(telegraf.Initializer); ok {
				require.NoError(t, p.Init())
			}

			plugin := Parser{
				ParseFields: tt.parseFields,
				Log:         testutil.Logger{Name: "processor.parser"},
			}
			plugin.SetParser(tt.parser)

			output := plugin.Apply(tt.input)
			testutil.RequireMetricsEqual(t, tt.expected, output, testutil.IgnoreTime())
		})
	}
}

func TestBase64FieldValidation(t *testing.T) {
	testMetric := metric.New(
		"test",
		map[string]string{},
		map[string]interface{}{
			"b": `eyJsdmwiOiJpbmZvIiwibXNnIjoiaHR0cCByZXF1ZXN0In0=`,
		},
		time.Unix(0, 0))

	testLogger := &testutil.CaptureLogger{}
	plugin := &Parser{
		ParseFields:  []string{"a"},
		Base64Fields: []string{"b"},
		Log:          testLogger,
	}
	plugin.SetParser(&json.Parser{})
	require.NoError(t, plugin.Init())
	plugin.Apply(testMetric)
	require.Empty(t, testLogger.Errors())

	plugin = &Parser{
		ParseFields:  []string{"b"},
		Base64Fields: []string{"b"},
		Log:          testLogger,
	}
	plugin.SetParser(&json.Parser{})
	require.NoError(t, plugin.Init())
	plugin.Apply(testMetric)
	require.NotEmpty(t, testLogger.Errors())
}

func TestTracking(t *testing.T) {
	var testCases = []struct {
		name       string
		numMetrics int
		parser     Parser
		payload    string
	}{
		{
			name:       "keep all",
			numMetrics: 2,
			parser: Parser{
				DropOriginal: false,
				ParseFields:  []string{"payload"},
				parser:       &json.Parser{},
			},
			payload: `{"value": 1}`,
		},
		{
			name:       "drop original",
			numMetrics: 1,
			parser: Parser{
				DropOriginal: true,
				ParseFields:  []string{"payload"},
				parser:       &json.Parser{},
			},
			payload: `{"value": 1}`,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create a tracking metric and tap the delivery information
			var mu sync.Mutex
			delivered := make([]telegraf.DeliveryInfo, 0, 1)
			notify := func(di telegraf.DeliveryInfo) {
				mu.Lock()
				defer mu.Unlock()
				delivered = append(delivered, di)
			}

			// Configure the plugin
			plugin := tt.parser
			require.NoError(t, plugin.Init())

			// Process expected metrics and compare with resulting metrics
			testMetric := metric.New(
				"test",
				map[string]string{},
				map[string]interface{}{
					"payload": tt.payload,
				},
				time.Unix(0, 0),
			)

			input, _ := metric.WithTracking(testMetric, notify)
			result := plugin.Apply(input)

			// Ensure we get back the correct number of metrics
			require.Len(t, result, tt.numMetrics)
			for _, m := range result {
				m.Accept()
			}

			// Simulate output acknowledging delivery of metrics and check delivery
			require.Eventuallyf(t, func() bool {
				mu.Lock()
				defer mu.Unlock()
				return len(delivered) == 1
			}, 1*time.Second, 100*time.Millisecond, "original metric not delivered")
		})
	}
}

// Benchmarks

func getMetricFields(m telegraf.Metric) interface{} {
	key := "field3"
	if v, ok := m.Fields()[key]; ok {
		return v
	}
	return nil
}

func getMetricFieldList(m telegraf.Metric) interface{} {
	key := "field3"
	fields := m.FieldList()
	for _, field := range fields {
		if field.Key == key {
			return field.Value
		}
	}
	return nil
}

func BenchmarkFieldListing(b *testing.B) {
	m := metric.New(
		"test",
		map[string]string{
			"some": "tag",
		},
		map[string]interface{}{
			"field0": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field1": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field2": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field3": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field4": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field5": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field6": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
		},
		time.Unix(0, 0))

	for n := 0; n < b.N; n++ {
		getMetricFieldList(m)
	}
}

func BenchmarkFields(b *testing.B) {
	m := metric.New(
		"test",
		map[string]string{
			"some": "tag",
		},
		map[string]interface{}{
			"field0": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field1": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field2": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field3": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field4": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field5": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
			"field6": `{"ts":"2018-07-24T19:43:40.275Z","lvl":"info","msg":"http request","method":"POST"}`,
		},
		time.Unix(0, 0))

	for n := 0; n < b.N; n++ {
		getMetricFields(m)
	}
}
