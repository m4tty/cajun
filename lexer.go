//Package cajun provide creole processing (lexing and parsing) functionality
package cajun

import (
	"fmt"
	_ "io/ioutil"
	"strings"
	"unicode/utf8"
)

type itemType int

const eof = -1

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
	itemHeadingCloseRun
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
	itemListUnorderedIncrease
	itemListUnorderedDecrease
	itemListUnorderedSameAsLast
	itemListOrdered
	itemListOrderedIncrease
	itemListOrderedDecrease
	itemListOrderedSameAsLast
	itemTable
	itemTableItem
	itemTableRow
	itemTableRowStart
	itemTableRowEnd
	itemTableHeaderItem
	itemText
	itemNewLine
	itemSpaceRun
	itemNoWiki
	itemNoWikiClose
	itemNoWikiOpen
	itemNoWikiText
	itemWikiLineBreak
)

func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		state: lexText,
		items: make(chan item, 2),
	}
	return l

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
			return lexLink
		}
		if strings.HasPrefix(l.input[l.pos:], "{{{") {
			//l.emitAnyPreviousText()
			return lexInsideNoWiki
		}
		if strings.HasPrefix(l.input[l.pos:], "{{") {
			//l.emitAnyPreviousText()
			return lexImage
		}
		if strings.HasPrefix(l.input[l.pos:], "http://") {
			l.emitAnyPreviousText()
			return lexFreeLink
		}
		if strings.HasPrefix(l.input[l.pos:], "*") {
			l.emitAnyPreviousText()
			return lexAsterisk
		}

		if strings.HasPrefix(l.input[l.pos:], "#") {
			l.emitAnyPreviousText()
			return lexOrderedList
		}

		if strings.HasPrefix(l.input[l.pos:], "|") {
			l.emitAnyPreviousText()
			return lexTable
		}
		if strings.HasPrefix(l.input[l.pos:], horizontalRuleToken) {
			return lexHorizontalRule
		}

		if strings.HasPrefix(l.input[l.pos:], "  ") || strings.HasPrefix(l.input[l.pos:], " \t") || strings.HasPrefix(l.input[l.pos:], "\t") {
			l.emitAnyPreviousText()
			return lexSpace
		}
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

func lexImage(l *lexer) stateFn {

	closed := isExplicitClose(l.input, l.pos, "}}")
	if closed {
		l.emitAnyPreviousText()
		length := getTextLength(l.input, l.pos, "}}")
		l.width = length + 2
		l.pos += l.width
		l.emit(itemImage)
	} else {
		//support implicit close (i.e. close at new line)
		l.next()
	}
	return lexText
}
func lexNoWikiText(l *lexer) stateFn {

	length := getTextLength(l.input, l.pos, "}}}")
	l.width = length
	l.pos += l.width
	l.emit(itemNoWikiText)
	return lexInsideNoWiki
}

func lexInsideNoWiki(l *lexer) stateFn {

	closed := isExplicitCloseMultiline(l.input, l.pos, "}}}")
	if closed {
		l.emitAnyPreviousText()

		if strings.HasPrefix(l.input[l.pos:], "{{{") {
			return lexNoWikiLeft
		}
		if strings.HasPrefix(l.input[l.pos:], "}}}") {
			return lexNoWikiRight
		}
		if l.lastType == itemNoWikiOpen {
			return lexNoWikiText
		}
	} else {
		//support implicit close (i.e. close at new line)
		l.next()
	}
	return lexText
}
func lexNoWikiLeft(l *lexer) stateFn {

	l.pos += len("{{{")

	l.emit(itemNoWikiOpen)
	return lexInsideNoWiki

}
func lexNoWikiRight(l *lexer) stateFn {

	//TODO: check for multiple closing braces, and include all that precede it assuming no additional openings.
	// scan for additional closing and additional opening. if additional closing pos is less than additional opening then we continue to lexInsideNoWiki,
	//  else we close the nowiki
	l.pos += len("}}}")
	l.emit(itemNoWikiClose)
	return lexText
}
func lexNewLine(l *lexer) stateFn {

	if l.isPrecededByWhitespace(l.pos) {
		// we just encountered an empty line
		l.resetBreaksSinceList()
		l.resetListDepth()
	}
	l.width = len("\n")
	l.pos += l.width
	l.emit(itemNewLine) //TODO: reintroduce if needed

	//l.incrementBreaksSinceList()
	return lexText
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

func (l *lexer) isFollowedByWhiteSpace(currentPos int) bool {
	var justWhiteSpace = false
	var tempPos = currentPos
	for {
		r, w := utf8.DecodeRuneInString(l.input[tempPos:])
		if isSpace(r) {
			tempPos = tempPos + w
			continue
		}
		if isEndOfLine(r) || tempPos == len(l.input) {
			justWhiteSpace = true
		}
		break
	}
	return justWhiteSpace
}

func lexHorizontalRule(l *lexer) stateFn {
	tempPos := l.pos + len(horizontalRuleToken)
	var followedByWhiteSpace = false
	followedByWhiteSpace = l.isFollowedByWhiteSpace(tempPos)
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

func lexLink(l *lexer) stateFn {
	closed := isExplicitClose(l.input, l.pos, "]]")
	if closed {
		l.emitAnyPreviousText()
		length := getTextLength(l.input, l.pos, "]]")
		l.width = length + 2
		l.pos += l.width
		l.emit(itemLink)
	} else {
		//support implicit close (i.e. close at new line)
		l.next()
	}
	return lexText
}
func lexOrderedList(l *lexer) stateFn {
	poundCount := 0
	for isPound(l.peek()) {
		poundCount++
		l.next()
	}
	if isSpace(l.peek()) && l.isPrecededByWhitespace(l.pos-poundCount) {
		if l.listDepth+1 == poundCount {
			//this is a new list start
			l.emit(itemListOrderedIncrease)
			l.emit(itemListOrdered)
			l.listDepth++
			l.breakCount = 0
		} else if l.listDepth == poundCount {
			l.emit(itemListOrderedSameAsLast)
			l.breakCount = 0
		} else if l.listDepth != 0 && l.listDepth >= poundCount {
			l.listDepth--
			l.emit(itemListOrderedDecrease)
			l.breakCount = 0
		} else {
			l.next()
		}
	}
	return lexText
}

func lexAsterisk(l *lexer) stateFn {

	asteriskCount := 0
	for isAsterisk(l.peek()) {
		asteriskCount++
		l.next()
	}

	//could be a list item begin or a list item
	// if first of a depth (only incrementally from previous length)
	//  then we are starting a new (maybe embedded list)
	if isSpace(l.peek()) && l.isPrecededByWhitespace(l.pos-asteriskCount) {
		if l.listDepth+1 == asteriskCount {
			//this is a new list start
			l.emit(itemListUnorderedIncrease)
			l.emit(itemListUnordered)
			l.listDepth++
			l.breakCount = 0
		} else if l.listDepth == asteriskCount {
			l.emit(itemListUnorderedSameAsLast)
			l.breakCount = 0
		} else if l.listDepth != 0 && l.listDepth >= asteriskCount {
			l.listDepth--
			l.emit(itemListUnorderedDecrease)
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

func lexTable(l *lexer) stateFn {
	if l.lastType == itemNewLine || l.lastType == itemUnset {
		for isPipe(l.peek()) {
			l.next()
		}
		if isEquals(l.peek()) {
			l.next()
			l.emit(itemTableRowStart)
			l.emit(itemTableHeaderItem)
		} else {
			l.emit(itemTableRowStart)
			l.emit(itemTableItem)
		}
	} else {
		l.next()
		if isEquals(l.peek()) {
			l.next()
			l.emit(itemTableHeaderItem)
		} else {
			if l.isFollowedByWhiteSpace(l.pos) {
				l.emit(itemTableRowEnd)
			} else {
				l.emit(itemTableItem)
			}
		}
	}
	return lexText

}

func lexHeading(l *lexer) stateFn {
	headingCount := 0
	isPrecededByWhiteSpaceOnly := l.isPrecededByWhitespace(l.pos)

	for isEquals(l.peek()) {
		//fmt.Println("heading -yes")
		headingCount++
		l.next()

	}
	isFollowedByWhiteSpaceOnly := l.isFollowedByWhiteSpace(l.pos)

	if isFollowedByWhiteSpaceOnly {
		l.emit(itemHeadingCloseRun)
	} else if isPrecededByWhiteSpaceOnly && isSpace(l.peek()) {
		itemHeading := itemHeading1 - itemType(1) + itemType(headingCount)
		l.emit(itemHeading)
	} else {
		l.next()
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

func getTextLength(input string, currentPos int, closeChars string) int {
	i := strings.Index(input[currentPos:], closeChars)
	if i >= 0 {
		return i
	} else {
		return len(input)
	}
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

func isExplicitCloseMultiline(input string, currentPos int, closeDelim string) bool {
	i := strings.Index(input[currentPos:], closeDelim)
	if i == -1 {
		return false
	}
	return true
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
func lexEmphasis(l *lexer) stateFn {
	l.pos += len("**")
	l.emit(itemBold) //TODO: reintroduce if needed
	return lexText
}

func (l *lexer) emit(t itemType) {
	//	fmt.Println("emitting", t, l.start, l.pos)
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
	l.lastLastType = l.lastType
	l.lastType = t
}
func (l *lexer) emitAnyPreviousText() {
	if l.pos > l.start {
		l.emit(itemText)
	}

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

func (l *lexer) resetListDepth() {
	l.listDepth = 0
}
func (l *lexer) resetBreaksSinceList() {
	l.breakCount = 0
}
func (l *lexer) incrementBreaksSinceList() {
	l.breakCount++
	if l.breakCount >= 2 {
		l.resetListDepth() //TODO: is this needed?
	}
}

func isPound(r rune) bool {
	return string(r) == "#"
}
func isAsterisk(r rune) bool {
	return string(r) == "*"
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return string(r) == " " || string(r) == "\t"
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return string(r) == "\r" || string(r) == "\n"
}
func isEquals(r rune) bool {
	return r == '='
}
func isPipe(r rune) bool {
	return r == '|'
}

func isUnorderedList(r rune) bool {
	return r == '*'
}
