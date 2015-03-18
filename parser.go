package cajun

import (
	"bytes"
	"fmt"
	"strings"
)

// itemTokens is a map containing the start and ending html tags required to process creole
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
	itemImageLeftDelimiter:      []string{"<img", ""},
	itemImageLocation:           []string{"src=\"", "\""},
	itemImageText:               []string{"alt=\"", "\""},
	itemImageRightDelimiter:     []string{"/>", ""},
	itemItalics:                 []string{"<em>", "</em>"},
	itemLink:                    []string{"<a href=\"{{location}}\">{{text}}</a>", ""},
	itemLineBreak:               []string{"<br />", ""},
	itemListUnordered:           []string{"<li>", "</li>"},
	itemListUnorderedIncrease:   []string{"<ul>", "</ul>"},
	itemListUnorderedSameAsLast: []string{"<li>", "</li>"},
	itemListUnorderedDecrease:   []string{"<li>", "</li>"},
	itemListOrdered:             []string{"<li>", "</li>"},
	itemListOrderedIncrease:     []string{"<ol>", "</ol>"},
	itemListOrderedSameAsLast:   []string{"<li>", "</li>"},
	itemListOrderedDecrease:     []string{"<li>", "</li>"},
	itemTable:                   []string{"<table>", "</table>"},
	itemTableRow:                []string{"<tr>", "</tr>"},
	itemTableHeaderItem:         []string{"<th>", "</th>"},
	itemText:                    []string{"<p>", "</p>"},
	itemTableItem:               []string{"<td>", "</td>"},
	itemNewLine:                 []string{"</br>", ""},
	//itemParagraph:       []string{"<p>", "</p>"},
	itemSpaceRun:      []string{"", ""},
	itemNoWiki:        []string{"<pre>", "</pre>"},
	itemWikiLineBreak: []string{"</br>", ""},
}

type FreeLinkFormatter interface {
	FreeLink(href string, text string) string
}

type WikiLinkFormatter interface {
	WikiLink(href string, text string) string
}

//Cajun is for parser options
type Cajun struct {
	FreeLink FreeLinkFormatter
	WikiLink WikiLinkFormatter
}

func New() *Cajun {
	return &Cajun{}
}

//parser keeps track of input processing
type parser struct {
	name           string
	input          string
	openList       map[itemType]int //maybe an int instead of bool, to count the open items ++/--
	preClosedList  map[itemType]int //maybe an int instead of bool, to count the open items ++/--
	openItemsStack *openItems
	items          []item
	lex            *lexer
	depth          int
	cajun          *Cajun
}

//isOpen checks if this item is in the openList
func (p *parser) isOpen(typ itemType) bool {
	if val, ok := p.openList[typ]; ok {
		return val > 0
	} else {
		return false
	}
}

//wasPreClosed checks if something is in the preClosedList indicating it was closed early for some reason.
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

//closeOthers returns all closing tags up to the intended close target itemType
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

// closeSpecific will only close the target itemType and will only search back according to limit
func (p *parser) closeSpecific(typ itemType, limit int) string {
	var buffer bytes.Buffer
	var limitCount = 0
	var addBacks openItems
	for p.openItemsStack.Len() > 0 {
		if limitCount == limit {
			break
		}
		t := p.openItemsStack.Pop()
		if val, ok := itemTokens[t]; ok {
			if t == typ {
				buffer.WriteString(val[1])
				limitCount++
				p.openList[t]--
			} else {
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

	return buffer.String()
}

//closeAtDoubleLineBreak will close everything that is open
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

func (c *Cajun) Transform(input string) (output string, terror error) {
	p := parser{}
	p.cajun = c
	p.openList = make(map[itemType]int)
	p.preClosedList = make(map[itemType]int)
	p.input = input
	p.lex = lex("creole", input)
	p.items = p.items[:0]
	p.openItemsStack = new(openItems)
	//TODO: refactor this long switch
	return p.transform()
}

//Transform processes an input string of creole markdown and returns html or error
func Transform(input string) (output string, terror error) {
	p := parser{}
	p.openList = make(map[itemType]int)
	p.preClosedList = make(map[itemType]int)
	p.input = input
	p.lex = lex("creole", input)
	p.items = p.items[:0]
	p.openItemsStack = new(openItems)
	//TODO: refactor this long switch
	return p.transform()
}

func (p parser) transform() (output string, terror error) {
	var buffer bytes.Buffer
Done:
	for {
		item := p.lex.nextItem()
		p.items = append(p.items, item)

		if p.isFollowingDoubleLineBreak(item) {
			buffer.WriteString(p.closeAtDoubleLineBreak())
		}
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
		case itemListUnordered, itemListUnorderedIncrease, itemListUnorderedSameAsLast, itemListUnorderedDecrease:
			var listLength = len(item.val)
			if item.typ == itemListUnorderedSameAsLast {
				var closed = false
				closeSame := p.closeSpecific(itemListUnorderedSameAsLast, 1)
				if closeSame != "" {
					buffer.WriteString(closeSame)
					closed = true
				}
				if !closed {
					closeItem := p.closeSpecific(itemListUnordered, 1)
					if closeItem != "" {
						buffer.WriteString(closeItem)
						closed = true
					}
				}
				if !closed {
					closeDecrease := p.closeSpecific(itemListUnorderedDecrease, 1)
					buffer.WriteString(closeDecrease)
				}
			}
			if item.typ == itemListUnorderedDecrease {
				closing := p.closeOthers(itemListUnorderedIncrease)
				buffer.WriteString(closing)
				//this is hacky

				closingUnordered := p.closeSpecific(itemListUnordered, 1)
				buffer.WriteString(closingUnordered)
			}
			if item.typ == itemListUnorderedIncrease {
			}
			if val, ok := itemTokens[item.typ]; ok {
				buffer.WriteString(val[0])
				p.openItemsStack.Push(item.typ)
				p.openList[item.typ]++
			} else {
				fmt.Errorf("Can not find item token")
			}

			p.depth = listLength //set to current depth
			break

		case itemListOrdered, itemListOrderedIncrease, itemListOrderedSameAsLast, itemListOrderedDecrease:
			var listLength = len(item.val)
			if item.typ == itemListOrderedSameAsLast {
				var closed = false
				closeSame := p.closeSpecific(itemListOrderedSameAsLast, 1)
				if closeSame != "" {
					buffer.WriteString(closeSame)
					closed = true
				}
				if !closed {
					closeItem := p.closeSpecific(itemListOrdered, 1)
					if closeItem != "" {
						buffer.WriteString(closeItem)
						closed = true
					}
				}
				if !closed {
					closeDecrease := p.closeSpecific(itemListOrderedDecrease, 1)
					buffer.WriteString(closeDecrease)
				}
			}
			if item.typ == itemListOrderedDecrease {
				closing := p.closeOthers(itemListOrderedIncrease)
				buffer.WriteString(closing)

				closingOrdered := p.closeSpecific(itemListOrdered, 1)
				buffer.WriteString(closingOrdered)
			}
			if item.typ == itemListOrderedIncrease {
			}
			if val, ok := itemTokens[item.typ]; ok {
				buffer.WriteString(val[0])
				p.openItemsStack.Push(item.typ)
				p.openList[item.typ]++
			} else {
				fmt.Errorf("Can not find item token")
			}
			p.depth = listLength //set to current depth
			break
		case itemTableRowStart, itemTableRowEnd, itemTableHeaderItem, itemTableItem:
			if item.typ == itemTableRowStart {
				if !p.isOpen(itemTable) {
					buffer.WriteString(itemTokens[itemTable][0])
					p.openItemsStack.Push(itemTable)
					p.openList[itemTable]++
				}
				buffer.WriteString(itemTokens[itemTableRow][0])
				p.openList[itemTableRow]++
			}
			//explicit row end
			if item.typ == itemTableRowEnd {
				if p.isOpen(itemTableItem) {
					buffer.WriteString(itemTokens[itemTableItem][1])
					p.openList[itemTableItem]--
				}
				if p.isOpen(itemTableHeaderItem) {
					buffer.WriteString(itemTokens[itemTableHeaderItem][1])
					p.openList[itemTableHeaderItem]--
				}

				buffer.WriteString(itemTokens[itemTableRow][1])

				p.openList[itemTableRow]--
			}
			if item.typ == itemTableHeaderItem || item.typ == itemTableItem {
				if val, ok := itemTokens[item.typ]; ok {
					if p.isOpen(item.typ) {
						buffer.WriteString(val[1])

						p.openList[item.typ]--

					}
					buffer.WriteString(val[0])
					p.openList[item.typ]++
				} else {
					fmt.Errorf("Can not find item token")
				}
			}
			break
		case itemImage:
			imageHtml := p.translateWikiImageToHtml(item.val)
			buffer.WriteString(imageHtml)
			break
		case itemLink:
			linkHtml := p.translateWikiLinkToHtml(item.val)
			buffer.WriteString(linkHtml)
			break
		case itemFreeLink:
			linkHtml := p.makeHtmlLink(item.val, item.val)
			buffer.WriteString(linkHtml)
			break
		case itemHorizontalRule:
			buffer.WriteString("<hr>")
			break
		case itemWikiLineBreak:
			buffer.WriteString("<br />")
			break

		case itemNoWikiOpen:
			buffer.WriteString("<pre>")
			//TODO: what to do if nowiki is not closed. do we track hanging nowiki tags?
			break
		case itemNoWikiClose:
			buffer.WriteString("</pre>")
			break
		case itemEscape:
			//don't do anything with the itemEscape, we just want to make sure we don't write it (~) out
			break
		case itemEscapeText:
			buffer.WriteString(item.val)
			break
		case itemNewLine:
			//TODO: anything here?
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
	}
	return buffer.String(), nil
}

//translateWikiImageToHtml will given this {{src|alt}}
//returns this <img src="src" alt="alt" />
func (p *parser) translateWikiImageToHtml(wikiImage string) string {
	wikiImage = strings.TrimPrefix(wikiImage, "{{")
	wikiImage = strings.TrimSuffix(wikiImage, "}}")
	var imageParts = strings.Split(wikiImage, "|")
	var alt = ""
	if len(imageParts) == 2 {
		alt = imageParts[1]
	}
	return "<img src=\"" + imageParts[0] + "\" alt=\"" + alt + "\" />"
}

//translateWikiLinkToHtml will given this [[href|text]]
//returns this <a href="href"/>text</a>
func (p *parser) translateWikiLinkToHtml(wikiLink string) string {
	wikiLink = strings.TrimPrefix(wikiLink, "[[")
	wikiLink = strings.TrimSuffix(wikiLink, "]]")
	var linkParts = strings.Split(wikiLink, "|")
	var text = linkParts[0]
	if len(linkParts) == 2 {
		text = linkParts[1]
	}
	if p.cajun.WikiLink != nil {
		return p.cajun.WikiLink.WikiLink(linkParts[0], text)
	}
	return p.makeHtmlLink(linkParts[0], text)
}

//makeHtmlLink fabricates an simple html link
func (p *parser) makeHtmlLink(href string, text string) string {
	if p.cajun.FreeLink != nil {
		return p.cajun.FreeLink.FreeLink(href, text)
	}
	return "<a href=\"" + href + "\" />" + text + "</a>"
}

//isFollowingDoubleLineBreak checks if the current item follows a double line break
func (p *parser) isFollowingDoubleLineBreak(current item) bool {
	if len(p.items) == 1 {
		//at the start of the input.
		return true
	}
	newLineCount := 0
	for i := len(p.items) - 2; i >= 0; i-- {
		precedingItem := p.items[i]

		if precedingItem.typ == itemNewLine {
			newLineCount++
			if newLineCount > 1 {
				return true
			}
			continue
		}
		if precedingItem.typ != itemSpaceRun {
			break
		}
	}
	return false
}

//isParagraphStart checks if the current item is at the start of a paragraph
func (p *parser) isParagraphStart(current item) bool {

	if current.typ == itemText {
		if len(p.items) == 1 {
			//at the start of the input.
			return true
		}
		newLineCount := 0
		for i := len(p.items) - 2; i >= 0; i-- {
			precedingItem := p.items[i]

			if precedingItem.typ == itemNewLine {
				newLineCount++
				if newLineCount > 1 {
					return true
				}
				continue
			}
			if precedingItem.typ != itemSpaceRun {
				break
			}
		}
	}
	return false
}

//nextNonSpace scans forward until the nextNonSpace
// TODO: remove?
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

//openItems represents a stack LIFO for holding the currently open items
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
