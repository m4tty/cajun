package cajun

import (
	"bytes"
	"fmt"
)

var itemName = map[itemType][]string{
	itemBold:              []string{"<strong>", "</strong>"},
	itemEOF:               []string{"</br>", ""},
	itemFreeLink:          []string{"<a href=\"{{val}}\">{{val}}</a>", ""},
	itemHeading1:          []string{"<h1>", "</h1>"},
	itemHeading2:          []string{"<h2>", "</h2>"},
	itemHeading3:          []string{"<h3>", "</h3>"},
	itemHeading4:          []string{"<h4>", "</h4>"},
	itemHeading5:          []string{"<h5>", "</h5>"},
	itemHeading6:          []string{"<h6>", "</h6>"},
	iitemHorizontalRule:   []string{"<hr>", ""},
	itemImage:             []string{"<img src=\"{{location}}\" alt=\"{{text}}\" />", ""},
	itemItalics:           []string{"<em>", "</em>"},
	itemLink:              []string{"<a href=\"{{location}}\">{{text}}</a>", ""},
	iitemLineBreak:        []string{"<br />", ""},
	itemImage:             []string{"<img src=\"{{location}}\" alt=\"{{text}}\" />", ""},
	itemListUnordered:     []string{"<ul>", "</ul>"},
	itemListUnorderedItem: []string{"<li>", "</li>"},
	itemListOrdered:       []string{"<ol>", "</ol>"},
	itemTable:             []string{"<table>", "</table>"},
	itemTableRow:          []string{"<tr>", "</tr>"},
	itemTableHeaderItem:   []string{"<th>", "</th>"},
	itemTableItem:         []string{"<td>", "</td>"},
	itemText:              []string{"<td>", "</td>"},
	itemTableItem:         []string{"<td>", "</td>"},
	itemNewLine:           []string{"", ""},
	itemSpaceRun:          []string{"", ""},
	itemNoWiki:            []string{"<pre>", "</pre>"},
	itemWikiLineBreak:     []string{"<pre>", "</pre>"},
}

type parser struct {
	name           string
	input          string
	boldOpen       bool
	openList       map[itemType]int //maybe an int instead of bool, to count the open items ++/--
	openItemsStack *openItems
}

func (p *parser) isOpen(typ itemType) bool {
	if val, ok := p.openList[typ]; ok {
		return val > 0
	} else {
		return false
	}
}

func (p *parser) closeOthers(typ itemType) string {
	for p.openItemsStack.Pop() != typ {

	}

}

//maintain an open list.  send writeCloses()

func (p *parser) Transform(input string) (output string, terror error) {
	p.openList = make(map[itemType]int)
	var buffer bytes.Buffer
	l := lex("creole", input)
	fmt.Println(l)

	p.openItemsStack = new(openItems)
	for {
		item := l.nextItem()
		switch item.typ {
		case itemBold:
			//**//test**// should be <strong><em>test</em></strong>
			if p.isOpen(itemBold) == false {
				buffer.WriteString("<strong>")
				p.openItemsStack.Push(itemBold)
				p.openList[item.typ]++
			} else {
				buffer.WriteString(p.closeOthers(itemBold))
				buffer.WriteString("</strong>")
				p.openList[item.typ]--
			}
			break
		case itemItalics:
			if p.isOpen(itemItalics) == false {
				buffer.WriteString("<em>")
				p.openList[item.typ]++
			} else {
				buffer.WriteString("</em>")
				p.openList[item.typ]--
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

func (p *parser) processItem(item item) (string, error) {

	return "", nil
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
