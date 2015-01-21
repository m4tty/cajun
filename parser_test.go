package cajun

import "testing"

type parserTest struct {
	name   string
	input  string
	output string
}

var parserTests = []parserTest{
	{"empty", "", ""},
	{"spaces", "   ", "   "},
	{"hr", "----", "<hr>"},
	{"hr preceeded by space", "  ----", "  <hr>"},
	{"hr preceeded by space, break, then text", "  ----  \n more", "  <hr>  <p> more</p>"},
	{"hr followed by space", "----  ", "<hr>  "},
	{"hr too many dashes", "-----", "<p>-----</p>"},
	{"text", `now is the time`, "<p>now is the time</p>"},
	{"text with bold", "hello-**blah**-world", "<p>hello-<strong>blah</strong>-world</p>"},
	{"text with italics", "hello-//blah//-world", "<p>hello-<em>blah</em>-world</p>"},
	{"text with bad order", "hello-**//blah**//-world", "<p>hello-<strong><em>blah</em></strong>-world</p>"},
	{"text with bad order, twice", "hello-**//blah**//-world**//blah**// this is a **test**", "<p>hello-<strong><em>blah</em></strong>-world<strong><em>blah</em></strong> this is a <strong>test</strong></p>"},
	{"text with bad order, twice", "hello-**//blah**//**//blah**//", "<p>hello-<strong><em>blah</em></strong><strong><em>blah</em></strong></p>"},
	{"test closing bold accross line breaks", "close this ** testing a \n    \n bold... more stuff here", "<p>close this <strong> testing a </strong></p><p> bold... more stuff here</p>"},
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
