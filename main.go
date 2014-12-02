package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type itemType int

const leftMeta = "**"
const eof = -1
const itemEOF = 5
const itemLeftDelim = 6

type stateFn func(*lexer) stateFn

type lexer struct {
	name       string
	input      string
	leftDelim  string
	rightDelim string
	state      stateFn
	start      int
	pos        int
	width      int
	items      chan item
}

type item struct {
	typ itemType // The type of this item.
	pos int      // The starting position, in bytes, of this item in the input string.
	val string   // The value of this item.
}

func main() {

	fmt.Println("hello")
	l := lex("test", "blah adfasdf lba **hasdf** alb asdfh **ab**lasdf\n blah asdfasdf **asdf**")
	fmt.Println(l)
	for {

		i := l.nextItem()
		fmt.Println("GOT ONE ----v")
		fmt.Println(i.val)
		fmt.Println("GOT ONE ----^")
		if l.state == nil {
			break
		}
		//		fmt.Println(item)
	}
	fmt.Println("done")

	time.Sleep(5 * time.Second)
	fmt.Println("You're boring; I'm leaving.")

}

func lex(name, input string) *lexer {
	l := &lexer{
		name:       name,
		input:      input,
		state:      lexText,
		rightDelim: "**",
		leftDelim:  "**",
		items:      make(chan item, 2),
	}
	//go l.run()
	return l

}

func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			fmt.Println("something was lex-ed, let's return it to the caller")
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
		fmt.Println(l.input[l.pos:])
		if strings.HasPrefix(l.input[l.pos:], leftMeta) {
			fmt.Println("leftMeta hit")
			if l.pos > l.start {
				fmt.Println("l.pos > l.start", l.pos, l.start)
				l.emit(1)
			}

			fmt.Println("leftMeta hit")
			return lexLeftDelim
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
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
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
}
