package cajun

import "testing"

type parserTest struct {
	name   string
	input  string
	output string
}

var parserTests = []parserTest{
	{"empty", "", ""},
	{"spaces", " \t\n", " </br>"},
	{"text", `now is the time`, "now is the time"},
	{"text with link", "hello-**blah**-world", "hello-<strong>blah</strong>-world"},
	{"text with link", "hello-//blah//-world", "hello-<em>blah</em>-world"},
	{"text with link", "hello-**//blah**//-world", "hello-<em>blah</em>-world"},
}

func TestParser(t *testing.T) {
	p := parser{}
	for _, test := range parserTests {
		output, _ := p.Transform(test.input)
		if test.output != output {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, output, test.output)
		}

	}
}
