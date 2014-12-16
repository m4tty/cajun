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
	{"text with bold", "hello-**blah**-world", "hello-<strong>blah</strong>-world"},
	{"text with italics", "hello-//blah//-world", "hello-<em>blah</em>-world"},
	{"text with bad order", "hello-**//blah**//-world", "hello-<strong><em>blah</em></strong>-world"},
	{"text with bad order, twice", "hello-**//blah**//-world**//blah**// this is a **test**", "hello-<strong><em>blah</em></strong>-world<strong><em>blah</em></strong> this is a <strong>test</strong>"},
	{"text with bad order, twice", "hello-**//blah**//**//blah**//", "hello-<strong><em>blah</em></strong><strong><em>blah</em></strong>"},
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
