package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type itemType int

const leftMeta = "**"
const eof = -1
const itemLeftDelim = 6

type stateFn func(*lexer) stateFn

type lexer struct {
	name          string
	input         string
	leftDelim     string
	rightDelim    string
	state         stateFn
	start         int
	pos           int
	width         int
	items         chan item
	delimiters    map[string]*Delimiter
	lastType      itemType
	paragraphOpen bool
	//consider storing a last "block" hit. different than last emit type, more course grained
}

type item struct {
	typ itemType // The type of this item.
	pos int      // The starting position, in bytes, of this item in the input string.
	val string   // The value of this item.
}

const (
	itemUnset itemType = iota
	itemError          // error occurred; value is text of error
	itemBold           // bold constant
	itemEOF
	itemFreeLink
	itemHeading1
	itemHeading2
	itemHeading3
	itemHeading4
	itemHeading5
	itemHeading6
	itemHorizontalLine
	itemImage
	itemItalics
	itemLink
	itemLineBreak
	itemListUnordered
	itemListOrdered
	itemParagraphStart
	itemParagraphEnd
	itemTable
	itemSingleNewLine
	itemDoubleNewLine // this is also a blank line. maybe rename itemBlankLine, as this essentially means we have a paragraph above.
	itemNoWiki
)

// TODO: modify lex interface to be lex(name, input, options...) options will be a few self referential functions. e.g.
//  addDelimiter(type, left,right), which might be enough for the lexer.

// the facade around the lexer, which is a parser for creole wiki, would use reasonable defaults for creole, but would support extension via:
//  useExtension(name, left,right, function replaceTokens(input string) output string)
func main() {
	var buffer bytes.Buffer
	testValue := "This is the start of a sentence. [[link]] \n== Now a heading w/ **not parsed bold**==\n some words **some bold words** a sentence on two lines\n the other line //italics//"
	l := lex("test", testValue)
	fmt.Println(l)
	for {

		i := l.nextItem()
		fmt.Println(i)
		buffer.WriteString(i.val)
		if l.state == nil {
			break
		}
		//		fmt.Println(item)
	}
	fmt.Println("-------")
	fmt.Println(testValue)
	fmt.Println("-------")
	fmt.Println(buffer.String())
	fmt.Println("-------")
	time.Sleep(5 * time.Second)
	fmt.Println("You're boring; I'm leaving.")

}

func lex(name, input string) *lexer {
	l := &lexer{
		name:       name,
		input:      input,
		state:      lexText,
		items:      make(chan item, 2),
		delimiters: make(map[string]*Delimiter),
	}
	//go l.run()
	return l

}

type Delimiter struct {
	name  string
	delim string
	lexFn stateFn
}

func (l *lexer) addDelimiter(name string, delim string, lexFn stateFn) {
	_, ok := l.delimiters[name]
	if !ok {
		fmt.Println("A delimiter of that name already exists", delim)
		return
	}
	//	var delimiter = Delimiter{left: left, right: right}
	l.delimiters[name] = &Delimiter{name: name, delim: delim, lexFn: lexFn}

}
func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			//	fmt.Println(l.state)
			if l.state != nil {
				l.state = l.state(l)
			}
		}
	}
	panic("bad")
}

//	item := <-l.items
//	l.lastPos = item.pos
//	return item
//}

func lexText(l *lexer) stateFn {
	for {
		//fmt.Println(l.input[l.pos:])

		if strings.HasPrefix(l.input[l.pos:], "**") {
			if l.pos > l.start {
				l.emit(99)
			}
			return lexEmphasis
		}
		if strings.HasPrefix(l.input[l.pos:], "//") {
			if l.pos > l.start {
				l.emit(99)
			}
			return lexItalics
		}
		if strings.HasPrefix(l.input[l.pos:], "\n") {
			if l.pos > l.start {
				l.emit(99)
			}
			return lexNewLine
		}
		if strings.HasPrefix(l.input[l.pos:], "=") {
			if l.pos > l.start {
				l.emit(99)
			}
			return lexHeading
		}
		if strings.HasPrefix(l.input[l.pos:], "[[") {
			if l.pos > l.start {
				l.emit(99)
			}
			return lexLink
		}

		//TODO: this should check for double line break, not single as lastType.
		// which is interesting as how do we prevent a single line emit... will need to peek ahead.
		if l.start == 0 && (l.lastType == itemUnset || l.lastType == itemLineBreak) && isAlphaNumeric(l.peek()) {
			l.emit(itemParagraphStart)
			l.paragraphOpen = true
			l.pos++
			//l.next()
			return lexText
		}
		if l.next() == eof {
			break
		}
	}
	if l.pos > l.start {
		l.emit(1)
	}
	l.emit(itemEOF)
	return nil
}

func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{1, l.start, fmt.Sprintf(format, args...)}
	return nil
}

func lexNewLine(l *lexer) stateFn {
	fmt.Println("lexingNewLine")

	eolCount := 0
	if isEndOfLine(l.peek()) {
		fmt.Println("EOL -yes")
		eolCount++
		l.next()
	}
	//TODO: fire close paragraph on detection of heading, list, blank line (two new lines in a row (perhaps with spaces), hr, table, nowiki
	l.pos += len("\n") * eolCount
	if eolCount > 1 {
		if l.paragraphOpen {
			l.emit(itemParagraphEnd)
		} else {
			l.emit(itemDoubleNewLine)
		}
	} else {
		//		l.pos += len("\n")
	}

	//	if strings.HasPrefix(l.input[l.pos:], leftComment) {
	//		return lexComment
	//	}
	l.emit(itemLineBreak) //TODO: reintroduce if needed
	//l.parenDepth = 0
	return lexText
}

func lexItalics(l *lexer) stateFn {
	fmt.Println("lexingItalics")
	l.pos += len("//")
	//	if strings.HasPrefix(l.input[l.pos:], leftComment) {
	//		return lexComment
	//	}
	l.emit(itemItalics) //TODO: reintroduce if needed
	//l.parenDepth = 0
	return lexText
}
func lexLink(l *lexer) stateFn {
	fmt.Println("lexingLink")

	l.pos += len("[[")

	rightLink := "]]"
	i := strings.Index(l.input[l.pos:], rightLink)
	if i < 0 {
		return l.errorf("unclosed link")
	}
	l.pos += len(rightLink) + i
	l.emit(itemLink)
	return lexText

}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}
func lexHeading(l *lexer) stateFn {
	fmt.Println("lexingHeading")
	//l.next() //get past the line break
	//TODO: THIS MUST BE ON A NEW LINE, and must have a space after the initial heading ==
	headingCount := 0
	fmt.Println("current", l.input[l.pos:l.pos+4])
	fmt.Println("1", string(l.peek()))

	for isHeading(l.peek()) {
		fmt.Println("heading -yes")
		headingCount++
		l.next()
	}
	if headingCount > 6 {
		return lexText
	}
	fmt.Println("2", string(l.peek()))
	if !isSpace(l.peek()) {
		fmt.Println("no space, not a heading")
		l.next()
		return lexText
	} else {

		fmt.Println("hooray . . . . . . . .. . . .")
	}

	l.pos += getHeadingEndPos(l.input, l.pos)
	itemHeading := itemHeading1 - itemType(1) + itemType(headingCount)
	l.emit(itemHeading)
	return lexText
}
func getHeadingEndPos(input string, currentPos int) int {
	i := strings.Index(input[currentPos:], "\n")
	if i >= 0 {
		return i
	} else {
		return len(input)
	}
}
func makeHeading(n int) string {
	var buffer bytes.Buffer
	for i := 0; i < n; i++ {
		buffer.WriteString("=")
	}
	return buffer.String()
}
func isHeading(r rune) bool {
	return r == '='
}
func lexEmphasis(l *lexer) stateFn {
	fmt.Println("lexingEmphasis")
	l.pos += len("**")
	l.emit(itemBold) //TODO: reintroduce if needed
	return lexText
}
func lexLeftDelim(l *lexer) stateFn {
	fmt.Println("lexLeft")
	l.pos += len(l.leftDelim)
	//	if strings.HasPrefix(l.input[l.pos:], leftComment) {
	//		return lexComment
	//	}
	// 	l.emit(itemLeftDelim) //TODO: reintroduce if needed
	//l.parenDepth = 0
	return lexInsideAction
}

func lexRightDelim(l *lexer) stateFn {
	l.pos += len(l.rightDelim)
	l.emit(8)
	return lexText
}

func lexInsideAction(l *lexer) stateFn {
	fmt.Println("lexInsideAction")
	fmt.Println(l.rightDelim)
	if strings.HasPrefix(l.input[l.pos:], l.rightDelim) {
		return lexRightDelim

		fmt.Println("uh oh")
		return l.errorf("unclosed left paren")
	}
	switch r := l.next(); {
	case r == eof || isEndOfLine(r):
		return l.errorf("unclosed action")
	}

	return lexInsideAction
}

func (l *lexer) run() {
	for state := lexText; state != nil; {
		state(l)
	}
	close(l.items)
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
	l.lastType = t
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	//fmt.Println("isSpace", string(r))
	//fmt.Println("rune", r)
	return string(r) == " " || string(r) == "\t"
	//return unicode.IsSpace(r)
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return string(r) == "\r" || string(r) == "\n"
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
