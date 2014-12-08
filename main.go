package main

import (
	"bytes"
	"fmt"
	_ "io/ioutil"
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
	lastLastType  itemType
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
	//	itemParagraphStart
	//	itemParagraphEnd
	itemTable
	itemText
	itemSingleNewLine
	itemSpaceRun
	itemDoubleNewLine // this is also a blank line. maybe rename itemBlankLine, as this essentially means we have a paragraph above.
	itemNoWiki
)

// TODO: modify lex interface to be lex(name, input, options...) options will be a few self referential functions. e.g.
//  addDelimiter(type, left,right), which might be enough for the lexer.

// the facade around the lexer, which is a parser for creole wiki, would use reasonable defaults for creole, but would support extension via:
//  useExtension(name, left,right, function replaceTokens(input string) output string)
func main() {

	//dat, _ := ioutil.ReadFile("creole1.0test.txt")
	//testValue := string(dat)

	var buffer bytes.Buffer
	var htmlBuffer bytes.Buffer
	testValue := `This is the start of a sentence. [[link]] \n== Now a heading w/ **not parsed bold**==
	some words **some bold words** a sentence on two lines
	the other line //italics//
	
* test1
** test1 child
* test2
	`
	l := lex("test", testValue)
	fmt.Println(l)
	for {

		i := l.nextItem()
		fmt.Println(i)
		buffer.WriteString(i.val)

		//		if i.typ == itemParagraphStart {
		//			buffer.WriteString("<p>")
		//			buffer.WriteString(i.val)
		//		}
		//		if i.typ == itemParagraphEnd {
		//
		//			buffer.WriteString("</p>")
		//			buffer.WriteString(i.val)
		//		}

		//liststart, listend
		if i.typ == itemHeading2 {

			//api, as it stands, requires understanding that this is a FULL heading.  and you must remove the beginning and end double equals.
			// so it is a bit leaky.
			//TODO: a helper method that returns the "token" for a type. e.g. getToken(typ) e.g. getToken(itemHeading2) would return "==".
			// this would allow easier parse to html. as you'll run strings.hasprefix/suffix (trim space?) and replace w/ token.
		}
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

	fmt.Println(htmlBuffer.String())
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
		//		if strings.HasPrefix(l.input[l.pos:], "**") {
		//			fmt.Println("has **")
		//			if l.pos > l.start {
		//				l.emit(itemText)
		//			}
		//			return lexEmphasis
		//		}
		if strings.HasPrefix(l.input[l.pos:], "//") {
			fmt.Println("has //")
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexItalics
		}
		if strings.HasPrefix(l.input[l.pos:], "\n") {
			fmt.Println("has \\n")
			if l.pos > l.start {
				//	fmt.Println("pos, start", l.pos, ",", l.start)
				l.emit(itemText)
			}
			return lexNewLine
		}
		if strings.HasPrefix(l.input[l.pos:], "=") {
			fmt.Println("has =")
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexHeading
		}
		if strings.HasPrefix(l.input[l.pos:], "[[") {
			fmt.Println("has [[")
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexLinkLeft
		}
		if strings.HasPrefix(l.input[l.pos:], "]]") {
			fmt.Println("has ]]")
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexLinkRight
		}

		//star at beginning of line
		if strings.HasPrefix(l.input[l.pos:], "*") {

			fmt.Println("starts w/ *")
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexAsterisk
		}

		if (strings.HasPrefix(l.input[l.pos:], " ") || strings.HasPrefix(l.input[l.pos:], "\t")) && isSpace(l.peek()) {
			fmt.Println("hit space")
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexSpace
		}
		//TODO: this should check for double line break, not single as lastType.
		// which is interesting as how do we prevent a single line emit... will need to peek ahead.
		//		if l.start == 0 && (l.lastType == itemUnset || l.lastType == itemLineBreak) && isAlphaNumeric(l.peek()) {
		//			fmt.Println("paragraph start")
		//			l.emit(itemParagraphStart)
		//			l.paragraphOpen = true
		//			l.pos++
		//			//l.next()
		//			return lexText
		//		}
		//fmt.Println("check EOF, which calls next")
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
	//fmt.Println("adjusting pos by width", w)
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
		//fmt.Println("EOL mark")
		eolCount++
		//l.next()
	}
	//fmt.Println("lexNewline- before - ", l.pos)
	//TODO: fire close paragraph on detection of heading, list, blank line (two new lines in a row (perhaps with spaces), hr, table, nowiki
	l.pos += l.width * eolCount

	//fmt.Println("lexNewline- after - ", l.pos)
	//	if eolCount > 1 {
	//		if l.paragraphOpen {
	//			l.emit(itemParagraphEnd)
	//		} else {
	//			l.emit(itemDoubleNewLine)
	//		}
	//	} else {
	//		//		l.pos += len("\n")
	//	}

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
func lexLinkLeft(l *lexer) stateFn {
	fmt.Println("lexingLinkLeft")

	l.pos += len("[[")

	//	rightLink := "]]"
	//	i := strings.Index(l.input[l.pos:], rightLink)
	//	if i < 0 {
	//		return l.errorf("unclosed link")
	//	}
	//	l.pos += len(rightLink) + i
	//	fmt.Println("link pos", l.pos)
	//	fmt.Println("link start", l.start)
	l.emit(itemLink)
	return lexText

}
func lexLinkRight(l *lexer) stateFn {
	fmt.Println("lexingLinkRight")
	l.pos += len("]]")
	l.emit(itemLink)
	return lexText
}

func lexAsterisk(l *lexer) stateFn {
	//this should either lex as unordered list or as emphasis
	if l.lastType == itemLineBreak || (l.lastType == itemSpaceRun && l.lastLastType == itemLineBreak) {
		return lexUnorderedList
	}
	if strings.HasPrefix(l.input[l.pos:], "**") {
		return lexEmphasis
	}
	l.next()
	return lexText
}
func lexUnorderedList(l *lexer) stateFn {
	fmt.Println("lexingUnorderedList")
	listCount := 0
	//	fmt.Println("current", l.input[l.pos:l.pos+4])
	//	fmt.Println("1", string(l.peek()))
	//	if l.lastType == itemLineBreak || l.lastType == itemSpaceRun {
	//		//false alarm, we have a astrisk, but not on a new line.  this is bad as it should have been picked up as empasis
	//		return lexText
	//	}
	for isUnorderedList(l.peek()) {
		//fmt.Println("heading -yes")
		listCount++
		l.next()
	}
	if listCount > 6 {
		return lexText
	}

	l.pos += getHeadingEndPos(l.input, l.pos)
	//itemHeading := itemHeading1 - itemType(1) + itemType(headingCount)

	//TODO: need to emit the paragraph end, but with no content, just start pos, paragraph end type, and empty.
	//	if l.paragraphOpen {
	//		l.emitManual(itemParagraphEnd, l.start, "")
	//		l.paragraphOpen = false
	//	}
	l.emit(itemListUnordered)

	return lexText

}

// lexSpace scans a run of space characters.
// One space has already been seen.
func lexSpace(l *lexer) stateFn {
	isRun := false
	for isSpace(l.peek()) {
		l.next()
		isRun = true
	}
	if isRun {
		l.emit(itemSpaceRun)
	} else {
		l.next()
	}
	return lexText
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}
func emitAnyPreviousText() {
	if l.pos > l.start {
		l.emit(itemText)
	}

}
func lexHeading(l *lexer) stateFn {
	fmt.Println("lexingHeading")
	//l.next() //get past the line break
	//TODO: THIS MUST BE ON A NEW LINE, and must have a space after the initial heading ==
	headingCount := 0
	//	fmt.Println("current", l.input[l.pos:l.pos+4])
	//	fmt.Println("1", string(l.peek()))

	for isHeading(l.peek()) {
		//fmt.Println("heading -yes")
		headingCount++
		l.next()
	}
	if headingCount > 6 {
		return lexText
	}
	//fmt.Println("2", string(l.peek()))
	if !isSpace(l.peek()) {
		//	fmt.Println("no space, not a heading")
		l.next()
		return lexText
	} else {

		//	fmt.Println("hooray . . . . . . . .. . . .")
	}

	//IF WE WANT TO GET THE ENTIRE HEADING (making the lexer smarter than it probably should be, but more useful to)
	//	l.pos += getHeadingEndPos(l.input, l.pos)
	itemHeading := itemHeading1 - itemType(1) + itemType(headingCount)

	//TODO: need to emit the paragraph end, but with no content, just start pos, paragraph end type, and empty.
	//	if l.paragraphOpen {
	//		l.emitManual(itemParagraphEnd, l.start, "")
	//		l.paragraphOpen = false
	//	}
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

func isUnorderedList(r rune) bool {
	return r == '*'
}
func lexEmphasis(l *lexer) stateFn {
	fmt.Println("lexingEmphasis")
	l.pos += len("**")
	l.emit(itemBold) //TODO: reintroduce if needed
	return lexText
}
func (l *lexer) run() {
	for state := lexText; state != nil; {
		state(l)
	}
	close(l.items)
}

func (l *lexer) emitManual(t itemType, startPos int, input string) {
	l.items <- item{t, startPos, input}
	l.start = startPos
	l.lastType = t

}
func (l *lexer) emit(t itemType) {
	//fmt.Println("EMIT-", l.pos, l.start)
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
	l.lastLastType = l.lastType
	l.lastType = t

	//fmt.Println("lastlasttype", l.lastLastType)
	//fmt.Println("lastType", l.lastType)
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	//fmt.Println("PEEEEEK")
	//fmt.Println("pre-peek pos", l.pos)
	r := l.next()
	l.backup()
	//fmt.Println("peek pos", l.pos)
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
	//fmt.Println("backup pos", l.pos)
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
