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
	{"heading1", "= Level 1 =", "<h1> Level 1 </h1>"},
	{"should not be heading1. no space", "=Level 1 =", "<p>=Level 1 =</p>"},
	{"heading2", "== Level 2 ==", "<h2> Level 2 </h2>"},
	{"heading3", "=== Level 3 ===", "<h3> Level 3 </h3>"},
	{"heading4", "==== Level 4 ====", "<h4> Level 4 </h4>"},
	{"heading5", "===== Level 5 =====", "<h5> Level 5 </h5>"},
	{"heading6", "====== Level 6 ======", "<h6> Level 6 </h6>"},
	{"heading6", "====== Level 6 ========", "<h6> Level 6 </h6>"},
	{"heading: should close h1 as h3", "=== Level =", "<h3> Level </h3>"},
	{"hr", "----", "<hr>"},
	{"hr preceeded by space", "  ----", "  <hr>"},
	{"hr preceeded by space, break, then text", "  ----  \n more", "  <hr>   more"},
	{"hr followed by space", "----  ", "<hr>  "},
	{"hr too many dashes", "-----", "<p>-----</p>"},
	{"text", `now is the time`, "<p>now is the time</p>"},
	{"text with bold", "hello-**blah**-world", "<p>hello-<strong>blah</strong>-world</p>"},
	{"text with italics", "hello-//blah//-world", "<p>hello-<em>blah</em>-world</p>"},
	{"text with bad order", "hello-**//blah**//-world", "<p>hello-<strong><em>blah</em></strong>-world</p>"},
	{"text with bad order, twice", "hello-**//blah**//-world**//blah**// this is a **test**", "<p>hello-<strong><em>blah</em></strong>-world<strong><em>blah</em></strong> this is a <strong>test</strong></p>"},
	{"text with bad order, twice", "hello-**//blah**//**//blah**//", "<p>hello-<strong><em>blah</em></strong><strong><em>blah</em></strong></p>"},
	{"test closing bold accross line breaks", "close this ** testing a \n\n\n\n\n\n bold... more stuff here", "<p>close this <strong> testing a </strong></p><p> bold... more stuff here</p>"},

	{"line break", "line \\\\break", "<p>line <br />break</p>"},
	{"unordered list simple", "* list item\n** child item", "<ul><li> list item<ul><li> child item</li></ul></li></ul>"},
	{"unordered list - one child, in first parent", "* list item\n** child item\n* list item", "<ul><li> list item<ul><li> child item</li></ul></li><li> list item</li></ul>"},
	{"unordered list - two child in first parent", "* item1\n** item1.1\n** item1.2\n* item2", "<ul><li> item1<ul><li> item1.1</li><li> item1.2</li></ul></li><li> item2</li></ul>"},
	{"unordered list - 3 first level, 2 child", "* item1\n** item1.1\n** item1.2\n* item2\n* item3", "<ul><li> item1<ul><li> item1.1</li><li> item1.2</li></ul></li><li> item2</li><li> item3</li></ul>"},
	{"unordered list, ", "* item1\n** item1.1\n** item1.2", "<ul><li> item1<ul><li> item1.1</li><li> item1.2</li></ul></li></ul>"},
	{"3 children", "* item1\n** item1.1\n** item1.2\n** item1.3", "<ul><li> item1<ul><li> item1.1</li><li> item1.2</li><li> item1.3</li></ul></li></ul>"},
	//	{"unordered list - ", "* item1\n** item1.1\n** item1.2\n* item2\n** item2.1\n** item2.2\n*** item2.2.1", "<ul><li> item1<ul><li> item1.1</li><li> item1.2</li></ul></li><li> item2 <ul><li> item2.1</li><li> item2.2<ul><li> item2.2.1</li></ul></li></ul></li></ul>"},
	{"5 levels", "* 1\n** 2\n*** 3\n**** 4\n***** 5", "<ul><li> 1<ul><li> 2<ul><li> 3<ul><li> 4<ul><li> 5</li></ul></li></ul></li></ul></li></ul></li></ul>"},
	{"multiline list items", "* 1\n test\n* 2\n test", "<ul><li> 1 test</li><li> 2 test</li></ul>"},
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
