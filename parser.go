package cajun

import (
	"bytes"
	"fmt"
	"strings"
)

var itemTokens = map[itemType][]string{
	itemBold:                    []string{"<strong>", "</strong>"},
	itemEOF:                     []string{"</br>", ""},
	itemFreeLink:                []string{"<a href=\"{{val}}\">{{val}}</a>", ""},
	itemHeading1:                []string{"<h1>", "</h1>"},
	itemHeading2:                []string{"<h2>", "</h2>"},
	itemHeading3:                []string{"<h3>", "</h3>"},
	itemHeading4:                []string{"<h4>", "</h4>"},
	itemHeading5:                []string{"<h5>", "</h5>"},
	itemHeading6:                []string{"<h6>", "</h6>"},
	itemHorizontalRule:          []string{"<hr>", ""},
	itemImage:                   []string{"<img src=\"{{location}}\" alt=\"{{text}}\" />", ""},
	itemItalics:                 []string{"<em>", "</em>"},
	itemLink:                    []string{"<a href=\"{{location}}\">{{text}}</a>", ""},
	itemLineBreak:               []string{"<br />", ""},
	itemListUnordered:           []string{"<li>", "</li>"},
	itemListUnorderedIncrease:   []string{"<ul>", "</ul>"},
	itemListUnorderedSameAsLast: []string{"<li>", "</li>"},
	itemListUnorderedDecrease:   []string{"<li>", "</li>"},
	//itemListUnorderedItem: []string{"<li>", "</li>"},
	itemListOrdered:     []string{"<ol>", "</ol>"},
	itemTable:           []string{"<table>", "</table>"},
	itemTableRow:        []string{"<tr>", "</tr>"},
	itemTableHeaderItem: []string{"<th>", "</th>"},
	itemText:            []string{"<p>", "</p>"},
	itemTableItem:       []string{"<td>", "</td>"},
	itemNewLine:         []string{"</br>", ""},
	//itemParagraph:       []string{"<p>", "</p>"},
	itemSpaceRun:      []string{"", ""},
	itemNoWiki:        []string{"<pre>", "</pre>"},
	itemWikiLineBreak: []string{"</br>", ""},
}

type parser struct {
	name           string
	input          string
	boldOpen       bool
	openList       map[itemType]int //maybe an int instead of bool, to count the open items ++/--
	preClosedList  map[itemType]int //maybe an int instead of bool, to count the open items ++/--
	openItemsStack *openItems
	items          []item
	lex            *lexer
	depth          int
}

func (p *parser) isOpen(typ itemType) bool {
	if val, ok := p.openList[typ]; ok {
		return val > 0
	} else {
		return false
	}
}

func (p *parser) wasPreClosed(typ itemType) bool {
	if val, ok := p.preClosedList[typ]; ok {
		return val > 0
	} else {
		return false
	}
}

// -> **// test **// and **// test2 **//
//   -> isOpen(strong). no. add to open items list.
//	 -> isOpen(italics). no. add to open items list.
//	 -> test
//	 -> isOpen(italics). yes. close all. closing first pop. strong. add to preclose.  closing second pop. italics.
//	 -> isOpen(strong) no. but should not write open tag either.

func (p *parser) closeOthers(typ itemType) string {
	var buffer bytes.Buffer
	//var found = false

	//var addMeBack = make(map[itemType]int)
	for p.openItemsStack.Len() > 0 {
		t := p.openItemsStack.Pop()
		if val, ok := itemTokens[t]; ok {
			buffer.WriteString(val[1])
			p.openList[t]--
			if t == typ {
				//				found = true
				break
			} else {
				//addMeBack[t]++
				// closed early
				p.preClosedList[t]++
			}
		}
	}
	//if we didn't find anything then we don't want to go closing everything
	//	if !found {
	//		for k, _ := range addMeBack {
	//			//a map isn't ordered, this could cause trouble.
	//			p.openItemsStack.Push(k) //deal with multiple of same type? if yes, need to change away from map, as we'll need to maintain order
	//		}
	//	}
	return buffer.String()
}

func (p *parser) closeSpecific(typ itemType, limit int) string {
	var buffer bytes.Buffer
	var limitCount = 0
	var addBacks openItems
	fmt.Println("closing specific", limit)
	for p.openItemsStack.Len() > 0 {
		if limitCount == limit {
			break
		}
		t := p.openItemsStack.Pop()
		if val, ok := itemTokens[t]; ok {
			if t == typ {
				fmt.Println("closing", val[1])
				buffer.WriteString(val[1])
				limitCount++
				p.openList[t]--
			} else {
				fmt.Println("no", val[1], t)
				addBacks.Push(t)
				// closed early
				p.preClosedList[t]++
			}
		}

	}
	for addBacks.Len() > 0 {
		val := addBacks.Pop()
		p.openItemsStack.Push(val) //deal with multiple of same type? if yes, need to change away from map, as we'll need to maintain order
		p.openList[val]++
	}

	fmt.Println("Len", p.openItemsStack.Len())
	return buffer.String()
}

var cantCrossLines = map[itemType]bool{
	itemHeading1:      true,
	itemHeading2:      true,
	itemHeading3:      true,
	itemHeading4:      true,
	itemHeading5:      true,
	itemHeading6:      true,
	itemListUnordered: true,
	itemListOrdered:   true,
}

// In many cases, everything can cross lines, and really doesn't matter.
// what do we do with things that are allowed to cross lines... but have been popped. need to add back?
func (p *parser) closeAtLineEnd() string {
	var buffer bytes.Buffer
	var addMeBack = make(map[itemType]int)

	for p.openItemsStack.Len() > 0 {
		t := p.openItemsStack.Pop()

		if _, cantCross := cantCrossLines[t]; cantCross {
			if val, ok := itemTokens[t]; ok {
				buffer.WriteString(val[1])
				p.openList[t]--
				//		if t == typ {
				//			break
				//		} else {
				//			// closed early
				//			//				p.preClosedList[t]++ //Is this closed "early"?
				//		}
			}
		} else {
			addMeBack[t]++ //might we hit multiple of the same type, that are going to cross lines?
		}

	}
	for k, _ := range addMeBack {
		//a map isn't ordered, this could cause trouble.
		p.openItemsStack.Push(k) //deal with multiple of same type? if yes, need to change away from map, as we'll need to maintain order

	}

	return buffer.String()
}

func (p *parser) closeAtDoubleLineBreak() string {
	var buffer bytes.Buffer
	var addMeBack = make(map[itemType]int)
	for p.openItemsStack.Len() > 0 {
		t := p.openItemsStack.Pop()
		//close everything when we encounter two line breaks. e.g. ending a paragraph
		if val, ok := itemTokens[t]; ok {
			buffer.WriteString(val[1])
			p.openList[t]--
			//do we care about preClosed? I think no.
			//p.preClosedList[t]++ //Is this closed "early"?
		}

	}
	for k, _ := range addMeBack {
		//a map isn't ordered, this could cause trouble.
		p.openItemsStack.Push(k) //deal with multiple of same type? if yes, need to change away from map, as we'll need to maintain order

	}

	return buffer.String()
}

// collect gathers the emitted items into a slice.
func (p *parser) collect(input string) (items []item) {
	p.lex = lex("creole", input)
	for {
		item := p.lex.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return items
}

//maintain an open list.  send writeCloses()

func (p *parser) Transform(input string) (output string, terror error) {
	p.openList = make(map[itemType]int)
	p.preClosedList = make(map[itemType]int)
	p.input = input
	var buffer bytes.Buffer
	p.lex = lex("creole", input)
	p.items = p.items[:0]
	p.openItemsStack = new(openItems)

Done:
	for {
		item := p.lex.nextItem()
		p.items = append(p.items, item)
	ProcessNext:
		switch item.typ {

		case itemText:
			if p.isParagraphStart(item) {
				buffer.WriteString("<p>")
				p.openItemsStack.Push(itemText)
				p.openList[item.typ]++
			}

			buffer.WriteString(item.val)
			break
		case itemBold:
			//**//test**// should be <strong><em>test</em></strong>
			if p.wasPreClosed(itemBold) {
				//ignore this item one time
				p.preClosedList[itemBold]--
			} else {
				if p.isOpen(itemBold) == false {
					buffer.WriteString("<strong>")
					p.openItemsStack.Push(itemBold)
					p.openList[item.typ]++
				} else {
					buffer.WriteString(p.closeOthers(itemBold))
				}
			}
			break
		case itemItalics:
			if p.wasPreClosed(itemItalics) {
				//ignore this item one time
				p.preClosedList[itemItalics]--
			} else {
				if p.isOpen(itemItalics) == false {
					buffer.WriteString("<em>")
					p.openItemsStack.Push(itemItalics)
					p.openList[item.typ]++
				} else {
					buffer.WriteString(p.closeOthers(itemItalics))
				}
			}
			break
		case itemHeading1, itemHeading2, itemHeading3, itemHeading4, itemHeading5, itemHeading6:
			if p.wasPreClosed(item.typ) {
				//ignore this item one time
				p.preClosedList[item.typ]--
			} else {
				if p.isOpen(item.typ) == false {
					if val, ok := itemTokens[item.typ]; ok {
						buffer.WriteString(val[0])
					} else {
						fmt.Errorf("Can not find item token")
					}
					p.openItemsStack.Push(item.typ)
					p.openList[item.typ]++
				} else {
					buffer.WriteString(p.closeOthers(item.typ))
				}
			}
			break
		case itemHeadingCloseRun:
			//TODO: fix this, it is messy
			var closeTag = ""
			closeTag += p.closeOthers(itemHeading1)
			closeTag += p.closeOthers(itemHeading2)
			closeTag += p.closeOthers(itemHeading3)
			closeTag += p.closeOthers(itemHeading4)
			closeTag += p.closeOthers(itemHeading5)
			closeTag += p.closeOthers(itemHeading6)

			if closeTag != "" {
				if !strings.HasPrefix(closeTag, "</h") {
					buffer.WriteString(item.val)
				}
				buffer.WriteString(closeTag)
			} else {
				//TODO: we aren't hitting this because something else could be open (e.g. a <p> tag or something)
				// if close tag is empty or not a heading that was left open
				//	fmt.Println(" NOTHING OPEN============ ", closeTag)
			}
			break
		case itemListUnordered, itemListUnorderedIncrease, itemListUnorderedSameAsLast, itemListUnorderedDecrease, itemListOrdered:

			var listLength = len(item.val)
			if item.typ == itemListUnorderedSameAsLast {
				fmt.Println("LEN-1", p.openItemsStack.Len())

				buffer.WriteString(p.closeSpecific(itemListUnorderedSameAsLast, 1))
				//</li><li>
				//we must close things... but only closing the diff between the last list length
				//fmt.Println("itemSame", closeUnordered)
				//buffer.WriteString(closeUnordered)
				buffer.WriteString(p.closeSpecific(itemListUnordered, 1))

				buffer.WriteString(p.closeSpecific(itemListUnorderedDecrease, 1))

			}
			if item.typ == itemListUnorderedDecrease {
				//</li>
				fmt.Println("LEN-2", p.openItemsStack.Len())
				//buffer.WriteString(p.closeSpecific(itemListUnorderedDecrease, (p.depth - listLength)))
				closing := p.closeOthers(itemListUnorderedIncrease)
				fmt.Println("itemDec", closing)
				//buffer.WriteString(p.closeOthers(itemListUnordered))
				//buffer.WriteString(p.closeOthers(itemListUnorderedSameAsLast))
				buffer.WriteString(closing)
				//this is hacky

				closingUnordered := p.closeSpecific(itemListUnordered, 1)
				fmt.Println("itemDec-", closingUnordered)
				buffer.WriteString(closingUnordered)
				//buffer.WriteString(p.closeSpecific(itemListUnorderedIncrease, (p.depth - listLength)))
				//buffer.WriteString(p.closeSpecific(itemListUnorderedSameAsLast, (p.depth - listLength)))
				//buffer.WriteString(p.closeSpecific(itemListUnorderedDecrease, (p.depth - listLength)))
			}
			if item.typ == itemListUnorderedIncrease {

				fmt.Println("LEN-3", p.openItemsStack.Len())
				fmt.Println("itemInc")
				//buffer.WriteString(p.closeOthers(itemListUnordered))
				//buffer.WriteString(p.closeSpecific(itemListUnorderedSameAsLast, (p.depth - listLength)))
			}
			//if p.isOpen(item.typ) == false {
			if val, ok := itemTokens[item.typ]; ok {
				buffer.WriteString(val[0])
				p.openItemsStack.Push(item.typ)
				p.openList[item.typ]++
			} else {
				fmt.Errorf("Can not find item token")
			}
			//} else {
			//		buffer.WriteString(p.closeOthers(item.typ))
			//	}

			p.depth = listLength //set to current depth
			break
		case itemHorizontalRule:
			buffer.WriteString("<hr>")
			break
		case itemWikiLineBreak:
			buffer.WriteString("<br />")
			break
		case itemNewLine:
			var newLineCount = 1
			item, newLineCount = p.nextNonSpace(item, newLineCount)
			if newLineCount > 1 {
				buffer.WriteString(p.closeAtDoubleLineBreak())
			} else {
				// only close the <li>
				//buffer.WriteString(p.closeOthers(itemListUnordered))
			}
			goto ProcessNext
			break
		case itemEOF:
			buffer.WriteString(p.closeAtDoubleLineBreak())
			break Done
		case itemError:
			break Done
		default:
			buffer.WriteString(item.val)
			break
		}
		//		if p.lex.state == nil {
		//			fmt.Println("state is nil")
		//			break
		//		}
	}
	return buffer.String(), nil
}

func (p *parser) isParagraphStart(current item) bool {
	if current.typ == itemText {

		if len(p.items) == 1 {
			//at the start of the input.
			return true
		}
		for i := len(p.items) - 1; i >= 0; i-- {
			precedingItem := p.items[i]
			if precedingItem.typ == itemNewLine {
				return true
			}
			if precedingItem.typ != itemSpaceRun {
				break
			}
		}
	}
	return false
}

func (p *parser) nextNonSpace(current item, currentBreakCount int) (token item, breakCount int) {
	for {
		current = p.lex.nextItem()
		if current.typ != itemSpaceRun && current.typ != itemNewLine {
			break
		}
		if current.typ == itemNewLine {
			currentBreakCount++
		}
	}
	return current, currentBreakCount
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
