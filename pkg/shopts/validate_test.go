package shopts

import "testing"

func TestValidate(t *testing.T) {
	cases := []struct {
		schema, value, want, err string
	}{
		{"long=a, type=int;", "007", "7", ""},
		{"long=a, type=int;", "+5", "5", ""},
		{"long=a, type=int;", "-0", "0", ""},
		{"long=a, type=int;", "1_000", "", "must be a valid integer"},
		{"long=a, type=int;", "0x10", "", "must be a valid integer"},
		{"long=a, type=int;", "abc", "", "must be a valid integer"},
		{"long=a, type=int, min=1, max=65535;", "1", "1", ""},
		{"long=a, type=int, min=1, max=65535;", "65535", "65535", ""},
		{"long=a, type=int, min=1, max=65535;", "0", "", "must be at least 1"},
		{"long=a, type=int, min=1, max=65535;", "65536", "", "must be at most 65535"},
		{"long=a, type=float;", "2.50", "2.50", ""},
		{"long=a, type=float;", "NaN", "", "must be a valid number"},
		{"long=a, type=float;", "Inf", "", "must be a valid number"},
		{"long=a, type=float;", "1_000", "", "must be a valid number"},
		{"long=a, type=float;", "0x1p-2", "", "must be a valid number"},
		{"long=a, type=float;", "1e999", "", "must be a finite number"},
		{"long=a, type=float;", "1e3", "1e3", ""},
		{"long=a, type=float;", ".5", ".5", ""},
		{"long=a, type=float;", "-2.", "-2.", ""},
		{"long=a, type=float;", "x", "", "must be a valid number"},
		{"long=a, type=float, min=0.5, max=1;", "0.4", "", "must be at least 0.5"},
		{"long=a, type=float, min=0.5, max=1;", "1.01", "", "must be at most 1"},
		{"long=a, type=bool;", "T", "true", ""},
		{"long=a, type=bool;", "1", "true", ""},
		{"long=a, type=bool;", "TRUE", "true", ""},
		{"long=a, type=bool;", "f", "false", ""},
		{"long=a, type=bool;", "yes", "", "must be true or false"},
		{`long=a, type=enum, enum="dev,prod";`, "prod", "prod", ""},
		{`long=a, type=enum, enum="dev,prod";`, "qa", "", "must be one of: dev, prod"},
		{"long=a, type=string, minLength=2, maxLength=3;", "héé", "héé", ""},
		{"long=a, type=string, minLength=2, maxLength=3;", "é", "", "must be at least 2 characters long"},
		{"long=a, type=string, minLength=2, maxLength=3;", "abcd", "", "must be no more than 3 characters long"},
		{"long=a, type=string, pattern=[0-9]+;", "abc1", "", "must match the pattern [0-9]+"},
		{"long=a, type=string, pattern=[0-9]+;", "123", "123", ""},
		{"long=a, type=string, pattern=^[0-9]+$;", "123", "123", ""},
		{"long=a, type=string, pattern=a|b;", "ab", "", "must match the pattern a|b"},
		{`long=a, type=string, pattern=\Q1.2.3;`, "1.2.3", "1.2.3", ""},
		{`long=a, type=string, pattern=\Q1.2.3;`, "1x2.3", "", `must match the pattern \Q1.2.3`},
		{"long=a, type=string, pattern=(?i)abc;", "ABC", "ABC", ""},
		{"long=a, type=string, pattern={{ PortNumber }};", "+80", "", "must be a port number from 1 to 65535"},
		{"long=a, type=string, pattern={{ PortNumber }}, failure=custom;", "+80", "", "custom"},
		{"long=a, type=string, pattern=[a-z]+, failure=lowercase only;", "ABC", "", "lowercase only"},
		{"long=a, type=string;", "a\nb", "", "must not contain a newline"},
		{"long=a, type=string;", "a	b", "a	b", ""}, // values may contain the delimiter
	}
	for _, tc := range cases {
		t.Run(tc.schema+" "+tc.value, func(t *testing.T) {
			e := mustParse(t, tc.schema).entries[0]
			got, err := validate(e, tc.value)
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("got error %v, want %q", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
