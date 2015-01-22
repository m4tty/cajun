package cajun

import (
	"bytes"
	"fmt"
	_ "io/ioutil"
	"strings"
	"unicode"
	"unicode/utf8"
)

type itemType int

const leftMeta = "**"
const eof = -1
const itemLeftDelim = 6

type stateFn func(*lexer) stateFn

type lexer struct {
	name         string
	input        string
	leftDelim    string
	rightDelim   string
	state        stateFn
	start        int
	pos          int
	width        int
	items        chan item
	delimiters   map[string]*Delimiter
	lastType     itemType
	lastLastType itemType
	listDepth    int
	breakCount   int // a count of \newlines emitted, since last list
	//consider storing a last "block" hit. different than last emit type, more course grained
}

type item struct {
	typ itemType // The type of this item.
	pos int      // The starting position, in bytes, of this item in the input string.
	val string   // The value of this item.
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

const (
	itemUnset itemType = iota
	itemError          // error occurred; value is text of error
	itemAsterisks
	itemBold // bold constant
	itemEOF
	itemFreeLink
	itemHeading1
	itemHeading2
	itemHeading3
	itemHeading4
	itemHeading5
	itemHeading6
	itemHorizontalRule
	itemImage
	itemImageLocationInternal
	itemImageDelimiter
	itemImageText
	itemImageLeftDelimiter
	itemImageLocation
	itemImageRightDelimiter
	itemItalics
	itemLinkLocationInternal
	itemLink
	itemLinkDelimiter
	itemLinkText
	itemLinkLeftDelimiter
	itemLinkLocation
	itemLinkRightDelimiter
	itemLineBreak
	itemListUnordered
	itemListOrdered
	itemTable
	itemTableItem
	itemTableRow
	itemTableHeaderItem
	itemText
	itemNewLine
	itemSpaceRun
	itemNoWiki
	itemWikiLineBreak
)

//TODO: possible option needed to set the URL of the "site" for links like [[SomePage,Go to some page]] which would link to <www.blah.com/blah/>SomePage
// Option: links can be auto closed at a new line, or NOT detected as links
// Option: bold/italics can be auto closed, or NOT detected if not closed.
// Option: the above might be "AutoClose" behavior
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

const (
	italicsDelimToken    = "//"
	wikiLineBreakToken   = "\\\\"
	newLineToken         = "\n"
	headingToken         = "="
	linkDelimLeftToken   = "[["
	imageDelimLeftToken  = "{{"
	linkDelimRightToken  = "]]"
	imageDelimRightToken = "}}"
	boldDelimStartToken  = "**"
	unorderedListToken   = "*"
	horizontalRuleToken  = "----"
)

func lexText(l *lexer) stateFn {
	for {
		//change this to a switch on l.next() which returns the next rune. will be cleaner, but adds some complexity for using hasprefix on multi rune checks
		if strings.HasPrefix(l.input[l.pos:], "//") {
			l.emitAnyPreviousText()
			return lexItalics
		}
		if strings.HasPrefix(l.input[l.pos:], "\\\\") {
			l.emitAnyPreviousText()
			return lexWikiLineBreak
		}
		if strings.HasPrefix(l.input[l.pos:], "\n") || strings.HasPrefix(l.input[l.pos:], "\r") {
			l.emitAnyPreviousText()
			return lexNewLine
		}
		if strings.HasPrefix(l.input[l.pos:], "=") {
			l.emitAnyPreviousText()
			return lexHeading
		}
		if strings.HasPrefix(l.input[l.pos:], "[[") {
			//l.emitAnyPreviousText()
			return lexInsideLink
		}

		if strings.HasPrefix(l.input[l.pos:], "{{") {
			//l.emitAnyPreviousText()
			return lexInsideImage
		}
		if strings.HasPrefix(l.input[l.pos:], "http://") {
			l.emitAnyPreviousText()
			return lexFreeLink
		}
		if strings.HasPrefix(l.input[l.pos:], "*") {
			l.emitAnyPreviousText()
			// one use case that could be a itemText is when a single * shows up in the middle of some text.
			//  not after a new line, not after a new line and spaces.
			return lexAsterisk
		}

		if strings.HasPrefix(l.input[l.pos:], "#") {
			l.emitAnyPreviousText()
			// one use case that could be a itemText is when a single * shows up in the middle of some text.
			//  not after a new line, not after a new line and spaces.
			return lexOrderedList
		}
		if strings.HasPrefix(l.input[l.pos:], horizontalRuleToken) {
			return lexHorizontalRule
		}

		if strings.HasPrefix(l.input[l.pos:], "  ") || strings.HasPrefix(l.input[l.pos:], " \t") || strings.HasPrefix(l.input[l.pos:], "\t") {
			l.emitAnyPreviousText()
			return lexSpace
		}
		//fmt.Println("check EOF, which calls next")
		if l.next() == eof {
			break
		}
	}
	l.emitAnyPreviousText()
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

func lexLinkLocation(l *lexer) stateFn {

	length := getLinkLength(l.input, l.pos, "]]")
	linkParts := strings.Split(l.input[l.pos:l.pos+length], "|")

	linkLocation := linkParts[0]
	linkLocationLength := len(linkLocation)

	if len(linkParts) == 1 {
		l.pos += linkLocationLength
		if strings.HasPrefix(linkLocation, "http://") {
			l.emit(itemLinkLocation)
		} else {
			l.emit(itemLinkLocationInternal)
		}
	} else {
		l.pos += linkLocationLength
		if strings.HasPrefix(linkLocation, "http://") {
			l.emit(itemLinkLocation)
		} else {
			l.emit(itemLinkLocationInternal)
		}
	}
	return lexInsideLink
}

func lexLinkInnerDelimiter(l *lexer) stateFn {

	l.width = len("|")
	l.pos += l.width

	l.emit(itemLinkDelimiter) //TODO: reintroduce if needed
	return lexInsideLink
}
func lexLinkText(l *lexer) stateFn {

	length := getLinkLength(l.input, l.pos, "]]")
	l.width = length
	l.pos += l.width

	l.emit(itemLinkText) //TODO: reintroduce if needed
	return lexInsideLink
}
func lexInsideLink(l *lexer) stateFn {

	closed := isExplicitClose(l.input, l.pos, "]]")
	if closed {
		l.emitAnyPreviousText()

		if strings.HasPrefix(l.input[l.pos:], "[[") {
			return lexLinkLeft
		}
		if strings.HasPrefix(l.input[l.pos:], "]]") {
			return lexLinkRight
		}
		if strings.HasPrefix(l.input[l.pos:], "|") {
			return lexLinkInnerDelimiter
		}
		if l.lastType == itemLinkLeftDelimiter {
			return lexLinkLocation
		}
		if l.lastType == itemLinkDelimiter {
			return lexLinkText
		}

	} else {
		//support implicit close (i.e. close at new line)
		l.next()
	}
	return lexText
}

func lexImageLocation(l *lexer) stateFn {

	length := getLinkLength(l.input, l.pos, "}}")
	imageParts := strings.Split(l.input[l.pos:l.pos+length], "|")
	imageLocation := imageParts[0]
	imageLocationLength := len(imageLocation)
	if len(imageParts) == 1 {
		l.pos += imageLocationLength
		l.emit(itemImageLocation)
	} else {
		l.pos += imageLocationLength
		l.emit(itemImageLocation)
	}
	return lexInsideImage
}

func lexImageInnerDelimiter(l *lexer) stateFn {
	l.width = len("|")
	l.pos += l.width
	l.emit(itemImageDelimiter) //TODO: reintroduce if needed
	return lexInsideImage
}
func lexImageText(l *lexer) stateFn {
	length := getLinkLength(l.input, l.pos, "}}")
	l.width = length
	l.pos += l.width
	l.emit(itemImageText) //TODO: reintroduce if needed
	return lexInsideImage
}
func lexInsideImage(l *lexer) stateFn {

	closed := isExplicitClose(l.input, l.pos, "}}")
	if closed {
		l.emitAnyPreviousText()

		if strings.HasPrefix(l.input[l.pos:], "{{") {
			return lexImageLeft
		}
		if strings.HasPrefix(l.input[l.pos:], "}}") {
			return lexImageRight
		}
		if strings.HasPrefix(l.input[l.pos:], "|") {
			return lexImageInnerDelimiter
		}
		if l.lastType == itemImageLeftDelimiter {
			return lexImageLocation
		}
		if l.lastType == itemImageDelimiter {
			return lexImageText
		}

	} else {
		//support implicit close (i.e. close at new line)
		l.next()
	}
	return lexText
}
func lexImageLeft(l *lexer) stateFn {

	l.pos += len("{{")

	//	rightLink := "]]"
	//	i := strings.Index(l.input[l.pos:], rightLink)
	//	if i < 0 {
	//		return l.errorf("unclosed link")
	//	}
	//	l.pos += len(rightLink) + i
	//	fmt.Println("link pos", l.pos)
	//	fmt.Println("link start", l.start)
	l.emit(itemImageLeftDelimiter)
	return lexInsideImage

}
func lexImageRight(l *lexer) stateFn {
	l.pos += len("}}")
	l.emit(itemImageRightDelimiter)
	return lexText
}
func lexNewLine(l *lexer) stateFn {

	l.width = len("\n")
	l.pos += l.width

	l.emit(itemNewLine) //TODO: reintroduce if needed
	l.incrementBreaksSinceList()
	return lexText
}
func (l *lexer) resetListDepth() {
	l.listDepth = 0
}
func (l *lexer) resetBreaksSinceList() {
	l.breakCount = 0
}
func (l *lexer) incrementBreaksSinceList() {
	l.breakCount++
	if l.breakCount >= 2 {
		l.resetListDepth()
	}
}
func lexFreeLink(l *lexer) stateFn {

	length := getFreeLinkLength(l.input, l.pos)
	l.pos += length

	l.emit(itemFreeLink) //TODO: reintroduce if needed
	return lexText
}
func lexWikiLineBreak(l *lexer) stateFn {
	l.pos += len("\\\\")
	//	if strings.HasPrefix(l.input[l.pos:], leftComment) {
	//		return lexComment
	//	}
	l.emit(itemWikiLineBreak) //TODO: reintroduce if needed
	//l.parenDepth = 0
	return lexText
}

func (l *lexer) isPrecededByWhitespace(startPos int) bool {
	whitespaceOnly := false

	i := startPos
	//handle edge case in which we have just started lexing and we have text e.g. "test test ----"
	if l.lastType == itemUnset {
		if startPos > 0 {
			for i >= 0 {
				if strings.HasPrefix(l.input[i:], " ") || strings.HasPrefix(l.input[i:], "\t") {
					whitespaceOnly = true
				} else {
					if strings.HasPrefix(l.input[i:], "\n") {
						whitespaceOnly = true
					} else {
						whitespaceOnly = false
					}
					break
				}
				i--
			}
		} else {
			return true
		}
	}
	if l.lastType == itemNewLine || (l.lastType == itemSpaceRun && l.lastLastType == itemNewLine) || (l.lastType == itemSpaceRun && l.lastLastType == itemUnset) {
		return true
	}
	return whitespaceOnly
}

func isFollowedByWhiteSpace(input string, currentPos int) bool {
	var justWhiteSpace = false
	var tempPos = currentPos
	for {
		r, w := utf8.DecodeRuneInString(input[tempPos:])
		if isSpace(r) {
			tempPos = tempPos + w
			continue
		}
		if isEndOfLine(r) || tempPos == len(input) {
			justWhiteSpace = true
		}
		break
	}
	return justWhiteSpace
}

func lexHorizontalRule(l *lexer) stateFn {
	tempPos := l.pos + len(horizontalRuleToken)
	var followedByWhiteSpace = false
	followedByWhiteSpace = isFollowedByWhiteSpace(l.input, tempPos)
	if followedByWhiteSpace && l.isPrecededByWhitespace(l.pos) {
		//if l.isPrecededByWhitespace(l.pos) {
		l.emitAnyPreviousText()
		l.pos += len(horizontalRuleToken)
		l.emit(itemHorizontalRule) //TODO: reintroduce if needed
	} else {
		l.next()
	}
	return lexText
}
func lexItalics(l *lexer) stateFn {
	l.pos += len("//")
	//	if strings.HasPrefix(l.input[l.pos:], leftComment) {
	//		return lexComment
	//	}
	l.emit(itemItalics) //TODO: reintroduce if needed
	//l.parenDepth = 0
	return lexText
}
func lexLinkLeft(l *lexer) stateFn {

	l.pos += len("[[")

	//	rightLink := "]]"
	//	i := strings.Index(l.input[l.pos:], rightLink)
	//	if i < 0 {
	//		return l.errorf("unclosed link")
	//	}
	//	l.pos += len(rightLink) + i
	//	fmt.Println("link pos", l.pos)
	//	fmt.Println("link start", l.start)
	l.emit(itemLinkLeftDelimiter)
	return lexInsideLink

}
func lexLinkRight(l *lexer) stateFn {
	l.pos += len("]]")
	l.emit(itemLinkRightDelimiter)
	return lexText
}
func lexOrderedList(l *lexer) stateFn {
	poundCount := 0
	for isPound(l.peek()) {
		poundCount++
		l.next()
	}
	if isSpace(l.peek()) && l.isPrecededByWhitespace(l.pos-poundCount) {
		l.emit(itemListOrdered)
	}
	return lexText
}
func isPound(r rune) bool {
	return string(r) == "#"
}
func isAsterisk(r rune) bool {
	return string(r) == "*"
}

func lexAsterisk(l *lexer) stateFn {
	asteriskCount := 0
	for isAsterisk(l.peek()) {
		asteriskCount++
		l.next()
	}

	if isSpace(l.peek()) && l.isPrecededByWhitespace(l.pos-asteriskCount) {
		if l.listDepth+1 == asteriskCount {
			l.emit(itemListUnordered)
			l.listDepth++
			l.breakCount = 0
		} else {
			//here we have 2 or more asterisks, at the beginning of a line (perhaps w/ whitespace), but out of the blue...
			// so I think we will treat the first two as bold, then let the rest be text?
			//adjust l.pos to be only 2 asterisks and emit bold
			l.pos = l.pos - (asteriskCount - 2)
			l.emit(itemBold)
		}
	} else {
		if asteriskCount == 2 {
			l.emit(itemBold)
		}
	}
	return lexText
}

// interpreting if something is bold or a list is an area where complexity lives. could consider
//  just lexing asterisk counts and don't make a determination.  this ambiguity is a bit of a problem.
func lexAsteriskOld(l *lexer) stateFn {

	asteriskCount := 0
	//	fmt.Println("current", l.input[l.pos:l.pos+4])
	//	fmt.Println("1", string(l.peek()))
	//	if l.lastType == itemLineBreak || l.lastType == itemSpaceRun {
	//		//false alarm, we have a astrisk, but not on a new line.  this is bad as it should have been picked up as empasis
	//		return lexText
	//	}
	for isAsterisk(l.peek()) {
		//fmt.Println("heading -yes")
		asteriskCount++
		l.next()
	}

	l.emit(itemAsterisks)

	//	//this should either lex as unordered list or as emphasis
	//	if l.lastType == itemLineBreak || (l.lastType == itemSpaceRun && l.lastLastType == itemLineBreak) {
	//		//also unordered lists should start w/ *, then **, then ***
	//		return lexUnorderedList
	//	}
	//	if strings.HasPrefix(l.input[l.pos:], "**") {
	//		return lexEmphasis
	//	}
	//	l.next()
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
func (l *lexer) emitAnyPreviousText() {
	if l.pos > l.start {
		l.emit(itemText)
	}

}
func lexHeading(l *lexer) stateFn {
	//l.next() //get past the line break
	//TODO: THIS MUST BE ON A NEW LINE, and must have a space after the initial heading ==
	//TODO: can end with heading close, but a heading close must be followed by whitespace. then any equalssigns will be ignored
	// itemHeadingCloseRun (a heading run, but too many equals, if it begining of line, it is text, at the end of a line, it could be a heading close)
	headingCount := 0
	//	fmt.Println("current", l.input[l.pos:l.pos+4])
	//	fmt.Println("1", string(l.peek()))
	isPrecededByWhiteSpaceOnly := isPrecededByWhitespace(l.pos)
	isFollowedByWhiteSpaceOnly := isFollowedByWhiteSpace(l.input, l.pos)

	if isPrecededByWhiteSpaceOnly || isFollowedByWhiteSpaceOnly {
		for isHeading(l.peek()) {
			//fmt.Println("heading -yes")
			headingCount++
			l.next()
		}
		if headingCount > 6 {
			fmt.Println("headingCount")
			//if more than six probably just return equalsSignRun (not a heading, but should be closed like one).
			if isFollowedByWhiteSpaceOnly {
				fmt.Println("this is an end of line heading, but has too many equals")
				// lets emit something helpful here (e.g. closeHeaderRun)
			}
			return lexText
		}
		itemHeading := itemHeading1 - itemType(1) + itemType(headingCount)
		l.emit(itemHeading)

	}
	return lexText
}

func getHeadingLength(input string, currentPos int) int {
	i := strings.Index(input[currentPos:], "\n")
	if i >= 0 {
		return i
	} else {
		return len(input)
	}
}

func getLinkLength(input string, currentPos int, closeChars string) int {
	i := strings.Index(input[currentPos:], closeChars)
	if i >= 0 {
		return i
	} else {
		return len(input)
	}
}
func isExplicitClose(input string, currentPos int, closeDelim string) bool {
	x := strings.IndexAny(input[currentPos:], "\n\r")
	i := strings.Index(input[currentPos:], closeDelim)
	if i == -1 {
		return false
	}
	if x == -1 {
		return true
	}
	return i < x
}
func getFreeLinkLength(input string, currentPos int) int {
	i := strings.Index(input[currentPos:], " ")
	link := input[currentPos : currentPos+i]
	punctuation := ",.?!:;\"'"
	for _, p := range punctuation {
		if strings.HasSuffix(link, string(p)) {
			i = i - len(string(p))
			break
		}
	}
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
