package cajun

import (
	"bytes"
	"fmt"
)

type parser struct {
	name  string
	input string
}

func (p *parser) Transform(input string) (output string, terror error) {
	var buffer bytes.Buffer
	l := lex("creole", input)
	fmt.Println(l)
	for {
		item := l.nextItem()

		if item.typ == itemBold {
			fmt.Printf("item.typ %+v\n", item.typ)
		}

		if item.typ == itemItalics {
			fmt.Printf("item.typ", item.typ)
		}

		fmt.Println(item)
		buffer.WriteString(item.val)
		if l.state == nil {
			break
		}
	}
	return buffer.String(), nil
}
