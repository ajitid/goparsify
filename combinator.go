package goparsify

import (
	"bytes"
	"strings"
)

// Seq matches all of the given parsers in order and returns their result as .Child[n]
func Seq(parsers ...Parserish) Parser {
	parserfied := ParsifyAll(parsers...)

	return NewParser("Seq()", func(ps *State, node *Result) {
		node.Child = make([]Result, len(parserfied))
		startpos := ps.Pos
		for i, parser := range parserfied {
			node.Child[i].Input = node.Input
			parser(ps, &node.Child[i])
			if ps.Errored() {
				ps.Pos = startpos
				return
			}
		}
		node.Start = startpos
		node.End = ps.Pos
	})
}

// NoAutoWS disables automatically ignoring whitespace between tokens for all parsers underneath
func NoAutoWS(parser Parserish) Parser {
	parserfied := Parsify(parser)
	return func(ps *State, node *Result) {
		oldWS := ps.WS
		ps.WS = NoWhitespace
		startpos := ps.Pos
		parserfied(ps, node)
		node.Start = startpos
		node.End = ps.Pos
		ps.WS = oldWS
	}
}

// Any matches the first successful parser and returns its result
func Any(parsers ...Parserish) Parser {
	parserfied := ParsifyAll(parsers...)

	return NewParser("Any()", func(ps *State, node *Result) {
		ps.WS(ps)
		if ps.Pos >= len(ps.Input) {
			ps.ErrorHere("!EOF")
			return
		}
		startpos := ps.Pos

		var longestError Error
		expected := []string{}
		for _, parser := range parserfied {
			parser(ps, node)
			if ps.Errored() {
				if ps.Error.pos >= longestError.pos {
					longestError = ps.Error
					expected = append(expected, ps.Error.expected)
				}
				if ps.Cut > startpos {
					break
				}
				ps.Recover()
				continue
			}
			node.Start = startpos
			node.End = ps.Pos
			return
		}

		ps.Error = Error{
			pos:      longestError.pos,
			expected: strings.Join(expected, " or "),
		}
		ps.Pos = startpos
	})
}

// ZeroOrMore matches zero or more parsers and returns the value as .Child[n]
// an optional separator can be provided and that value will be consumed
// but not returned. Only one separator can be provided.
func ZeroOrMore(parser Parserish, separator ...Parserish) Parser {
	return NewParser("ZeroOrMore()", manyImpl(0, parser, separator...))
}

// OneOrMore matches one or more parsers and returns the value as .Child[n]
// an optional separator can be provided and that value will be consumed
// but not returned. Only one separator can be provided.
func OneOrMore(parser Parserish, separator ...Parserish) Parser {
	return NewParser("OneOrMore()", manyImpl(1, parser, separator...))
}

func manyImpl(min int, op Parserish, sep ...Parserish) Parser {
	var opParser = Parsify(op)
	var sepParser Parser
	if len(sep) > 0 {
		sepParser = Parsify(sep[0])
	}

	return func(ps *State, node *Result) {
		node.Child = make([]Result, 0, 5)
		startpos := ps.Pos
		for {
			node.Child = append(node.Child, Result{Input: node.Input})
			opParser(ps, &node.Child[len(node.Child)-1])
			if ps.Errored() {
				if len(node.Child)-1 < min || ps.Cut > ps.Pos {
					ps.Pos = startpos
					return
				}
				ps.Recover()
				node.Child = node.Child[0 : len(node.Child)-1]
				return
			}

			if sepParser != nil {
				sepParser(ps, TrashResult)
				if ps.Errored() {
					ps.Recover()
					return
				}
			}
		}
		node.Start = startpos
		node.End = ps.Pos
	}
}

// Maybe will 0 or 1 of the parser
func Maybe(parser Parserish) Parser {
	parserfied := Parsify(parser)

	return NewParser("Maybe()", func(ps *State, node *Result) {
		startpos := ps.Pos
		parserfied(ps, node)
		if ps.Errored() && ps.Cut <= startpos {
			ps.Recover()
		}
		node.Start = startpos
		node.End = ps.Pos
	})
}

// Bind will set the node .Result when the given parser matches
// This is useful for giving a value to keywords and constant literals
// like true and false. See the json parser for an example.
func Bind(parser Parserish, val interface{}) Parser {
	p := Parsify(parser)

	return func(ps *State, node *Result) {
		startpos := ps.Pos
		p(ps, node)
		if ps.Errored() {
			return
		}
		node.Result = val
		node.Start = startpos
		node.End = ps.Pos
	}
}

// Map applies the callback if the parser matches. This is used to set the Result
// based on the matched result.
func Map(parser Parserish, f func(n *Result)) Parser {
	p := Parsify(parser)

	return func(ps *State, node *Result) {
		startpos := ps.Pos
		p(ps, node)
		if ps.Errored() {
			return
		}
		node.Start = startpos
		node.End = ps.Pos
		f(node)
	}
}

// Chain lets you choose which parser to call on the basis of the result of
// previous parser.
//
// Chain's second argument is a function that takes in the result of the first
// parser, and with this knowledge, lets you return a successive parser.
//
// Result of this successive parser is considered as the result of the Chain.
func Chain(parser Parserish, getNextParser func(n *Result) Parserish) Parser {
	p1 := Parsify(parser)

	return func(ps *State, node *Result) {
		startpos := ps.Pos

		r1 := NewResult(node.Input)
		p1(ps, r1)
		if ps.Errored() {
			copyResult(node, r1)
			return
		}

		p2 := Parsify(getNextParser(r1))
		p2(ps, node)
		if ps.Errored() {
			return
		}

		node.Start = startpos
		node.End = ps.Pos
	}
}

func flatten(n *Result) {
	if len(n.Child) > 0 {
		sbuf := &bytes.Buffer{}
		for _, child := range n.Child {
			flatten(&child)
			sbuf.WriteString(child.Token)
		}
		n.Token = sbuf.String()
	}
}

// Merge all child Tokens together recursively
func Merge(parser Parserish) Parser {
	return Map(parser, flatten)
}
