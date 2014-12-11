package cajun

import (
	"bytes"
	"fmt"
)

type parser struct {
	name     string
	input    string
	boldOpen bool
	openList map[itemType]bool //maybe an int instead of bool, to count the open items ++/--
}

func (p *parser) isOpen(typ itemType) bool {
	if val, ok := p.openList[typ]; ok {
		return val
	} else {
		return false
	}
}

//maintain an open list.  send writeCloses()

func (p *parser) Transform(input string) (output string, terror error) {
	p.openList = make(map[itemType]bool)
	var buffer bytes.Buffer
	l := lex("creole", input)
	fmt.Println(l)

	openItemsStack := new(openItems)
	for {
		item := l.nextItem()
		switch item.typ {
		case itemBold:
			openItemsStack.Push(itemBold)
			if p.isOpen(itemBold) == false {
				buffer.WriteString("<strong>")
				p.openList[item.typ] = true
			} else {
				buffer.WriteString("</strong>")
				p.openList[item.typ] = false
			}
			break
		case itemItalics:
			if p.isOpen(itemItalics) == false {
				buffer.WriteString("<em>")
				p.openList[item.typ] = true
			} else {
				buffer.WriteString("</em>")
				p.openList[item.typ] = false
			}
			break
		case itemNewLine:
			// close anything that is open that can't cross lines... which is, i think, everything that can be open
			// should we maintain two lists: one for inter line items (bold, italics, images, links) and a second for major items like open headers/lists
			break

		default:
			buffer.WriteString(item.val)
			break
		}
		fmt.Println(item)
		if l.state == nil {
			break
		}
	}
	return buffer.String(), nil
}

type openItems struct {
	top  *openItem
	size int
}

type openItem struct {
	typ  itemType
	next *openItem
}

func (ois *openItems) Len() int {
	return ois.size
}

func (ois *openItems) Push(typ itemType) {
	ois.top = &openItem{typ, ois.top}
	ois.size++
}

func (ois *openItems) Pop() (typ itemType) {
	if ois.size > 0 {
		typ, ois.top = ois.top.typ, ois.top.next
		ois.size--
		return
	}
	return itemUnset
}
