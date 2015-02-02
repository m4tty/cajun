cajun
=====

Cajun: a Creole processor in golang

Creole (which is a markdown like format, but simpler and safer) processor.  Takes in creole outputs html.

[![GoDoc] (https://godoc.org/github.com/m4tty/cajun?status.png)](https://godoc.org/github.com/m4tty/cajun)
[![Build Status](https://travis-ci.org/m4tty/cajun.svg?branch=master)](https://travis-ci.org/m4tty/cajun)


Motivation
------

An excuse to write a lexer.  I also like Creole over Markdown.  Because reasons.  This [Why Markdown Is Not My Favourite Language](http://www.wilfred.me.uk/blog/2012/07/30/why-markdown-is-not-my-favourite-language/) covers it pretty well.



Installation
------------

With Go and git installed:

    go get github.com/m4tty/cajun

will download, compile, and install the package into your `$GOPATH`
directory hierarchy. Alternatively, you can achieve the same if you
import it into a project:

		import "github.com/m4tty/cajun"

and `go get` without parameters.


Design
-----
It is a traditional lexer, but followed much of the strategy (a state machine that emits tokens on a channel) explained here by Rob Pike. [Lexical Scanning in Go - Rob Pike] (https://www.youtube.com/watch?v=HxaD_trXwRE)


Example
-----
Transform some input using:

```go
output := cajun.Transform(input)

```
