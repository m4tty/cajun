package cajun

import (
	"fmt"
	"testing"
)

var itemName = map[itemType]string{
	itemUnset:          "unset",
	itemError:          "error",
	itemAsterisks:      "asterisks",
	itemBold:           "bold",
	itemEOF:            "EOF",
	itemFreeLink:       "freelink",
	itemHeading1:       "heading1",
	itemHeading2:       "heading2",
	itemHeading3:       "heading3",
	itemHeading4:       "heading4",
	itemHeading5:       "heading5",
	itemHeading6:       "heading6",
	itemHorizontalRule: "heading7",
	itemImage:          "image",
	itemItalics:        "italics",
	itemLink:           "link",
	itemLineBreak:      "linebreak",
	itemListUnordered:  "listunordered",
	itemListOrdered:    "listordered",
	itemTable:          "table",
	itemText:           "text",
	itemNewLine:        "newline",
	itemSpaceRun:       "spaces",
	itemNoWiki:         "nowiki",
	itemWikiLineBreak:  "wikilinebreak",
}

func (i itemType) String() string {
	s := itemName[i]
	if s == "" {
		return fmt.Sprintf("item%d", int(i))
	}
	return s
}

type lexTest struct {
	name  string
	input string
	items []item
}

var (
	tEOF     = item{itemEOF, 0, ""}
	tNewLine = item{itemNewLine, 0, "\n"}
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"spaces", " \t\n", []item{{itemSpaceRun, 0, " \t"}, tNewLine, tEOF}},
	{"new lines", "\n\n\n\n", []item{tNewLine, tNewLine, tNewLine, tNewLine, tEOF}},
	{"text", `now is the time`, []item{{itemText, 0, "now is the time"}, tEOF}},
	//	{"text", `~**now is the time`, []item{{itemText, 0, "now is the time"}, tEOF}},
	{"text with link", "hello-[[blah]]-world", []item{
		{itemText, 0, "hello-"},
		{itemLink, 0, "[[blah]]"},
		{itemText, 0, "-world"},
		tEOF,
	}},
	{"text with bold asterisks", "hello-**blah**-world", []item{
		{itemText, 0, "hello-"},
		{itemBold, 0, "**"},
		{itemText, 0, "blah"},
		{itemBold, 0, "**"},
		{itemText, 0, "-world"},
		tEOF,
	}},
	{"text with asterisks", "**blah**-world", []item{
		{itemBold, 0, "**"},
		{itemText, 0, "blah"},
		{itemBold, 0, "**"},
		{itemText, 0, "-world"},
		tEOF,
	}},
	{"text with asterisks", "* start unordered list\n", []item{
		{itemListUnorderedIncrease, 0, "*"},
		{itemListUnordered, 0, ""},
		{itemText, 0, " start unordered list"},
		tNewLine,
		tEOF,
	}},
	{"text with asterisks", "** start unordered list\n", []item{
		{itemBold, 0, "**"},
		{itemText, 0, " start unordered list"},
		tNewLine,
		tEOF,
	}},
	{"text with asterisks", "*** start unordered list\n", []item{
		{itemBold, 0, "**"},
		{itemText, 0, "* start unordered list"},
		tNewLine,
		tEOF,
	}},
	{"text with pound", "# start ordered list\n", []item{

		{itemListOrderedIncrease, 0, "#"},
		{itemListOrdered, 0, ""},
		{itemText, 0, " start ordered list"},
		tNewLine,
		tEOF,
	}},
	{"text with pound, no space follows", "#start ordered list\n", []item{
		{itemText, 0, "#start ordered list"},
		tNewLine,
		tEOF,
	}},
	{"text with pound, newline, space", "\n   # start ordered list\n text", []item{
		tNewLine,
		{itemSpaceRun, 0, "   "},
		{itemListOrderedIncrease, 0, "#"},
		{itemListOrdered, 0, ""},
		{itemText, 0, " start ordered list"},
		tNewLine,
		{itemText, 0, " text"},
		tEOF,
	}},
	{"text with wiki line break", "wiki wiki\\\\ break\n", []item{
		{itemText, 0, "wiki wiki"},
		{itemWikiLineBreak, 0, "\\\\"},
		{itemText, 0, " break"},
		tNewLine,
		tEOF,
	}},
	{"text with wiki horizontal rule", "wiki wiki\n----\n break\n", []item{
		{itemText, 0, "wiki wiki"},
		tNewLine,
		{itemHorizontalRule, 0, "----"},
		tNewLine,
		{itemText, 0, " break"},
		tNewLine,
		tEOF,
	}},
	{"text with wiki horizontal rule spaces around", "wiki wiki\n   ----   \n break\n", []item{
		{itemText, 0, "wiki wiki"},
		tNewLine,
		{itemSpaceRun, 0, "   "},
		{itemHorizontalRule, 0, "----"},
		{itemSpaceRun, 0, "   "},
		tNewLine,
		{itemText, 0, " break"},
		tNewLine,
		tEOF,
	}},
	{"text with text before horizontal rule", "wiki wiki ---- \n", []item{
		{itemText, 0, "wiki wiki ---- "},
		tNewLine,
		tEOF,
	}},
	{"text with italics", "hello-//blah//-world", []item{
		{itemText, 0, "hello-"},
		{itemItalics, 0, "//"},
		{itemText, 0, "blah"},
		{itemItalics, 0, "//"},
		{itemText, 0, "-world"},
		tEOF,
	}},
	{"text with heading", "= start heading\n", []item{
		{itemHeading1, 0, "="},
		{itemText, 0, " start heading"},
		tNewLine,
		tEOF,
	}},
	{"text with heading", "== start heading\n", []item{
		{itemHeading2, 0, "=="},
		{itemText, 0, " start heading"},
		tNewLine,
		tEOF,
	}},
	{"text with heading open and close", "== start heading==\n", []item{
		{itemHeading2, 0, "=="},
		{itemText, 0, " start heading"},
		{itemHeadingCloseRun, 0, "=="},
		tNewLine,
		tEOF,
	}},
	{"text with heading open and close", "== start heading== \n", []item{
		{itemHeading2, 0, "=="},
		{itemText, 0, " start heading"},
		{itemHeadingCloseRun, 0, "=="},
		{itemText, 0, " "},
		tNewLine,
		tEOF,
	}},
	{"text with free link", "hello-http://www.blah.com/whatever?asdf, -world", []item{
		{itemText, 0, "hello-"},
		{itemFreeLink, 0, "http://www.blah.com/whatever?asdf"},
		{itemText, 0, ", -world"},
		tEOF,
	}},
	{"text with link", "hello- [[http://www.blah.com/whatever?asdf]], -world", []item{
		{itemText, 0, "hello- "},
		{itemLink, 0, "[[http://www.blah.com/whatever?asdf]]"},
		{itemText, 0, ", -world"},
		tEOF,
	}},
	{"text with link", "hello- [[http://www.blah.com/whatever?asdf|blah whatever]], -world", []item{
		{itemText, 0, "hello- "},
		{itemLink, 0, "[[http://www.blah.com/whatever?asdf|blah whatever]]"},
		{itemText, 0, ", -world"},
		tEOF,
	}},
	{"text with link", "hello- [[somepage|blah whatever]], -world", []item{
		{itemText, 0, "hello- "},
		{itemLink, 0, "[[somepage|blah whatever]]"},
		{itemText, 0, ", -world"},
		tEOF,
	}},
	{"text with image", "hello- {{http://www.blah.com/whatever?asdf}}, -world", []item{
		{itemText, 0, "hello- "},
		{itemImage, 0, "{{http://www.blah.com/whatever?asdf}}"},
		{itemText, 0, ", -world"},
		tEOF,
	}},
	{"text with image w/ alt", "hello- {{somepage|blah whatever}}, -world", []item{
		{itemText, 0, "hello- "},
		{itemImage, 0, "{{somepage|blah whatever}}"},
		{itemText, 0, ", -world"},
		tEOF,
	}},
	//	{"text with image", "hello- {{http://www.blah.com/whatever?asdf}}, -world", []item{
	//		{itemText, 0, "hello- "},
	//		{itemImageLeftDelimiter, 0, "{{"},
	//		{itemImageLocation, 0, "http://www.blah.com/whatever?asdf"},
	//		{itemImageRightDelimiter, 0, "}}"},
	//		{itemText, 0, ", -world"},
	//		tEOF,
	//	}},
	//	{"text with image full src with alt", "hello- {{http://www.blah.com/whatever?asdf|blah whatever}}, -world", []item{
	//		{itemText, 0, "hello- "},
	//		{itemImageLeftDelimiter, 0, "{{"},
	//		{itemImageLocation, 0, "http://www.blah.com/whatever?asdf"},
	//		{itemImageDelimiter, 0, "|"},
	//		{itemImageText, 0, "blah whatever"},
	//		{itemImageRightDelimiter, 0, "}}"},
	//		{itemText, 0, ", -world"},
	//		tEOF,
	//	}},
	{"no wiki", "hello- {{{ test ** blah ** test }}} -world", []item{
		{itemText, 0, "hello- "},
		{itemNoWikiOpen, 0, "{{{"},
		{itemNoWikiText, 0, " test ** blah ** test "},
		{itemNoWikiClose, 0, "}}}"},
		{itemText, 0, " -world"},
		tEOF,
	}},
	{"no wiki", "hello- {{{ test \n test }}} -world", []item{
		{itemText, 0, "hello- "},
		{itemNoWikiOpen, 0, "{{{"},
		{itemNoWikiText, 0, " test \n test "},
		{itemNoWikiClose, 0, "}}}"},
		{itemText, 0, " -world"},
		tEOF,
	}},
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest, left, right string) (items []item) {
	l := lex(t.name, t.input)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
	}
	return true
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test, "", "")
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}

//// Some easy cases from above, but with delimiters $$ and @@
//var lexDelimTests = []lexTest{
//	{"punctuation", "$$,@%{{}}@@", []item{
//		tLeftDelim,
//		{itemChar, 0, ","},
//		{itemChar, 0, "@"},
//		{itemChar, 0, "%"},
//		{itemChar, 0, "{"},
//		{itemChar, 0, "{"},
//		{itemChar, 0, "}"},
//		{itemChar, 0, "}"},
//		tRightDelim,
//		tEOF,
//	}},
//	{"empty action", `$$@@`, []item{tLeftDelim, tRightDelim, tEOF}},
//	{"for", `$$for@@`, []item{tLeftDelim, tFor, tRightDelim, tEOF}},
//	{"quote", `$$"abc \n\t\" "@@`, []item{tLeftDelim, tQuote, tRightDelim, tEOF}},
//	{"raw quote", "$$" + raw + "@@", []item{tLeftDelim, tRawQuote, tRightDelim, tEOF}},
//}
//
//var (
//	tLeftDelim  = item{itemLeftDelim, 0, "$$"}
//	tRightDelim = item{itemRightDelim, 0, "@@"}
//)
//
//func TestDelims(t *testing.T) {
//	for _, test := range lexDelimTests {
//		items := collect(&test, "$$", "@@")
//		if !equal(items, test.items, false) {
//			t.Errorf("%s: got\n\t%v\nexpected\n\t%v", test.name, items, test.items)
//		}
//	}
//}
//
//var lexPosTests = []lexTest{
//	{"empty", "", []item{tEOF}},
//	{"punctuation", "{{,@%#}}", []item{
//		{itemLeftDelim, 0, "{{"},
//		{itemChar, 2, ","},
//		{itemChar, 3, "@"},
//		{itemChar, 4, "%"},
//		{itemChar, 5, "#"},
//		{itemRightDelim, 6, "}}"},
//		{itemEOF, 8, ""},
//	}},
//	{"sample", "0123{{hello}}xyz", []item{
//		{itemText, 0, "0123"},
//		{itemLeftDelim, 4, "{{"},
//		{itemIdentifier, 6, "hello"},
//		{itemRightDelim, 11, "}}"},
//		{itemText, 13, "xyz"},
//		{itemEOF, 16, ""},
//	}},
//}
//
//// The other tests don't check position, to make the test cases easier to construct.
//// This one does.
//func TestPos(t *testing.T) {
//	for _, test := range lexPosTests {
//		items := collect(&test, "", "")
//		if !equal(items, test.items, true) {
//			t.Errorf("%s: got\n\t%v\nexpected\n\t%v", test.name, items, test.items)
//			if len(items) == len(test.items) {
//				// Detailed print; avoid item.String() to expose the position value.
//				for i := range items {
//					if !equal(items[i:i+1], test.items[i:i+1], true) {
//						i1 := items[i]
//						i2 := test.items[i]
//						t.Errorf("\t#%d: got {%v %d %q} expected  {%v %d %q}", i, i1.typ, i1.pos, i1.val, i2.typ, i2.pos, i2.val)
//					}
//				}
//			}
//		}
//	}
//}
