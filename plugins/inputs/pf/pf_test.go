package pf

import (
	"log"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/require"
)

type measurementResult struct {
	tags   map[string]string
	fields map[string]any
}

func TestPfctlInvocation(t *testing.T) {
	type pfctlInvocationTestCase struct {
		config PF
		cmd    string
		args   []string
	}

	var testCases = []pfctlInvocationTestCase{
		// 0: no sudo
		{
			config: PF{UseSudo: false},
			cmd:    "fakepfctl",
			args:   []string{"-s", "info"},
		},
		// 1: with sudo
		{
			config: PF{UseSudo: true},
			cmd:    "fakesudo",
			args:   []string{"fakepfctl", "-s", "info"},
		},
	}

	for i, tt := range testCases {
		execLookPath = func(cmd string) (string, error) { return "fake" + cmd, nil }
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			log.Printf("running #%d\n", i)
			cmd, args, err := tt.config.buildPfctlCmd()
			if err != nil {
				t.Fatalf("error when running buildPfctlCmd: %s", err)
			}
			if tt.cmd != cmd || !reflect.DeepEqual(tt.args, args) {
				t.Errorf("%d: expected %s - %#v got %s - %#v", i, tt.cmd, tt.args, cmd, args)
			}
		})
	}
}

func TestPfMeasurements(t *testing.T) {
	type pfTestCase struct {
		TestInput    string
		err          error
		measurements []measurementResult
	}

	testCases := []pfTestCase{
		// 0: nil input should raise an error
		{TestInput: "", err: errParseHeader},
		// 1: changes to pfctl output should raise an error
		{TestInput: `Status: Enabled for 161 days 21:24:45         Debug: Urgent

Interface Stats for re1               IPv4             IPv6
  Bytes In                   2585823744614    1059233657221
  Bytes Out                  1227266932673    3274698578875
  Packets In
    Passed                      2289953086       1945437219
    Blocked                      392835739            48609
  Packets Out
    Passed                      1649146326       2605569054
    Blocked                            107                0

State Table                          Total             Rate
  Current Entrys                       649
  searches                     18421725761         1317.0/s
  inserts                        156762508           11.2/s
  removals                       156761859           11.2/s
Counters
  match                          473002784           33.8/s
  bad-offset                             0            0.0/s
  fragment                            2729            0.0/s
  short                                107            0.0/s
  normalize                           1685            0.0/s
  memory                               101            0.0/s
  bad-timestamp                          0            0.0/s
  congestion                             0            0.0/s
  ip-option                         152301            0.0/s
  proto-cksum                          108            0.0/s
  state-mismatch                     24393            0.0/s
  state-insert                          92            0.0/s
  state-limit                            0            0.0/s
  src-limit                              0            0.0/s
  synproxy                               0            0.0/s
`,
			err: errMissingData("current entries"),
		},
		// 2: bad numbers should raise an error
		{TestInput: `Status: Enabled for 0 days 00:26:05           Debug: Urgent

State Table                          Total             Rate
  current entries                      -23
  searches                           11325            7.2/s
  inserts                                5            0.0/s
  removals                               3            0.0/s
Counters
  match                              11226            7.2/s
  bad-offset                             0            0.0/s
  fragment                               0            0.0/s
  short                                  0            0.0/s
  normalize                              0            0.0/s
  memory                                 0            0.0/s
  bad-timestamp                          0            0.0/s
  congestion                             0            0.0/s
  ip-option                              0            0.0/s
  proto-cksum                            0            0.0/s
  state-mismatch                         0            0.0/s
  state-insert                           0            0.0/s
  state-limit                            0            0.0/s
  src-limit                              0            0.0/s
  synproxy                               0            0.0/s
`,
			err: errMissingData("current entries"),
		},
		{TestInput: `Status: Enabled for 0 days 00:26:05           Debug: Urgent

State Table                          Total             Rate
  current entries                        2
  searches                           11325            7.2/s
  inserts                                5            0.0/s
  removals                               3            0.0/s
Counters
  match                              11226            7.2/s
  bad-offset                             0            0.0/s
  fragment                               0            0.0/s
  short                                  0            0.0/s
  normalize                              0            0.0/s
  memory                                 0            0.0/s
  bad-timestamp                          0            0.0/s
  congestion                             0            0.0/s
  ip-option                              0            0.0/s
  proto-cksum                            0            0.0/s
  state-mismatch                         0            0.0/s
  state-insert                           0            0.0/s
  state-limit                            0            0.0/s
  src-limit                              0            0.0/s
  synproxy                               0            0.0/s
`,
			measurements: []measurementResult{
				{
					fields: map[string]any{
						"entries":        int64(2),
						"searches":       int64(11325),
						"inserts":        int64(5),
						"removals":       int64(3),
						"match":          int64(11226),
						"bad-offset":     int64(0),
						"fragment":       int64(0),
						"short":          int64(0),
						"normalize":      int64(0),
						"memory":         int64(0),
						"bad-timestamp":  int64(0),
						"congestion":     int64(0),
						"ip-option":      int64(0),
						"proto-cksum":    int64(0),
						"state-mismatch": int64(0),
						"state-insert":   int64(0),
						"state-limit":    int64(0),
						"src-limit":      int64(0),
						"synproxy":       int64(0)},
					tags: map[string]string{},
				},
			},
		},
		{TestInput: `Status: Enabled for 161 days 21:24:45         Debug: Urgent

Interface Stats for re1               IPv4             IPv6
  Bytes In                   2585823744614    1059233657221
  Bytes Out                  1227266932673    3274698578875
  Packets In
    Passed                      2289953086       1945437219
    Blocked                      392835739            48609
  Packets Out
    Passed                      1649146326       2605569054
    Blocked                            107                0

State Table                          Total             Rate
  current entries                      649
  searches                     18421725761         1317.0/s
  inserts                        156762508           11.2/s
  removals                       156761859           11.2/s
Counters
  match                          473002784           33.8/s
  bad-offset                             0            0.0/s
  fragment                            2729            0.0/s
  short                                107            0.0/s
  normalize                           1685            0.0/s
  memory                               101            0.0/s
  bad-timestamp                          0            0.0/s
  congestion                             0            0.0/s
  ip-option                         152301            0.0/s
  proto-cksum                          108            0.0/s
  state-mismatch                     24393            0.0/s
  state-insert                          92            0.0/s
  state-limit                            0            0.0/s
  src-limit                              0            0.0/s
  synproxy                               0            0.0/s
`,
			measurements: []measurementResult{
				{
					fields: map[string]any{
						"entries":        int64(649),
						"searches":       int64(18421725761),
						"inserts":        int64(156762508),
						"removals":       int64(156761859),
						"match":          int64(473002784),
						"bad-offset":     int64(0),
						"fragment":       int64(2729),
						"short":          int64(107),
						"normalize":      int64(1685),
						"memory":         int64(101),
						"bad-timestamp":  int64(0),
						"congestion":     int64(0),
						"ip-option":      int64(152301),
						"proto-cksum":    int64(108),
						"state-mismatch": int64(24393),
						"state-insert":   int64(92),
						"state-limit":    int64(0),
						"src-limit":      int64(0),
						"synproxy":       int64(0)},
					tags: map[string]string{},
				},
			},
		},
	}

	for i, tt := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			log.Printf("running #%d\n", i)
			pf := &PF{
				infoFunc: func() (string, error) {
					return tt.TestInput, nil
				},
			}
			acc := new(testutil.Accumulator)
			err := acc.GatherError(pf.Gather)
			if !reflect.DeepEqual(tt.err, err) {
				t.Errorf("%d: expected error '%#v' got '%#v'", i, tt.err, err)
			}
			n := 0
			for j, v := range tt.measurements {
				if len(acc.Metrics) < n+1 {
					t.Errorf("%d: expected at least %d values got %d", i, n+1, len(acc.Metrics))
					break
				}
				m := acc.Metrics[n]
				if !reflect.DeepEqual(m.Measurement, measurement) {
					t.Errorf("%d %d: expected measurement '%#v' got '%#v'\n", i, j, measurement, m.Measurement)
				}
				if !reflect.DeepEqual(m.Tags, v.tags) {
					t.Errorf("%d %d: expected tags\n%#v got\n%#v\n", i, j, v.tags, m.Tags)
				}
				if !reflect.DeepEqual(m.Fields, v.fields) {
					t.Errorf("%d %d: expected fields\n%#v got\n%#v\n", i, j, v.fields, m.Fields)
				}
				n++
			}
		})
	}
}

func TestPfMeasurementsIndentation(t *testing.T) {
	// pfctl may indent the state-table and counter fields differently depending
	// on the platform or version, so make sure the parser accepts any leading
	// whitespace. See https://github.com/influxdata/telegraf/issues/19780.
	testCases := []struct {
		name   string
		indent string
	}{
		{"single-space", " "},
		{"two-spaces", "  "},
		{"four-spaces", "    "},
		{"tab", "\t"},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			b.WriteString("Status: Enabled for 0 days 00:26:05           Debug: Urgent\n\n")
			b.WriteString("State Table                          Total             Rate\n")
			b.WriteString(tt.indent + "current entries                      250\n")
			b.WriteString(tt.indent + "searches                        92810970         1789.5/s\n")
			b.WriteString(tt.indent + "inserts                            64126            1.2/s\n")
			b.WriteString(tt.indent + "removals                           63876            1.2/s\n")
			b.WriteString("Counters\n")
			b.WriteString(tt.indent + "match                          473002784           33.8/s\n")
			b.WriteString(tt.indent + "bad-offset                             0            0.0/s\n")
			b.WriteString(tt.indent + "fragment                            2729            0.0/s\n")
			b.WriteString(tt.indent + "short                                107            0.0/s\n")
			b.WriteString(tt.indent + "normalize                           1685            0.0/s\n")
			b.WriteString(tt.indent + "memory                               101            0.0/s\n")
			b.WriteString(tt.indent + "bad-timestamp                          0            0.0/s\n")
			b.WriteString(tt.indent + "congestion                             0            0.0/s\n")
			b.WriteString(tt.indent + "ip-option                         152301            0.0/s\n")
			b.WriteString(tt.indent + "proto-cksum                          108            0.0/s\n")
			b.WriteString(tt.indent + "state-mismatch                     24393            0.0/s\n")
			b.WriteString(tt.indent + "state-insert                          92            0.0/s\n")
			b.WriteString(tt.indent + "state-limit                            0            0.0/s\n")
			b.WriteString(tt.indent + "src-limit                              0            0.0/s\n")
			b.WriteString(tt.indent + "synproxy                               0            0.0/s\n")

			acc := new(testutil.Accumulator)
			require.NoError(t, parsePfctlOutput(b.String(), acc))
			require.Len(t, acc.Metrics, 1)
			require.Equal(t, map[string]any{
				"entries":        int64(250),
				"searches":       int64(92810970),
				"inserts":        int64(64126),
				"removals":       int64(63876),
				"match":          int64(473002784),
				"bad-offset":     int64(0),
				"fragment":       int64(2729),
				"short":          int64(107),
				"normalize":      int64(1685),
				"memory":         int64(101),
				"bad-timestamp":  int64(0),
				"congestion":     int64(0),
				"ip-option":      int64(152301),
				"proto-cksum":    int64(108),
				"state-mismatch": int64(24393),
				"state-insert":   int64(92),
				"state-limit":    int64(0),
				"src-limit":      int64(0),
				"synproxy":       int64(0),
			}, acc.Metrics[0].Fields)
		})
	}
}
