package goparsify

import (
	"strconv"
	"testing"

	"os"

	"github.com/stretchr/testify/require"
)

func TestSeq(t *testing.T) {
	parser := Seq("hello", "world")

	t.Run("matches sequence", func(t *testing.T) {
		node, p2 := runParser("hello world", parser)
		assertSequence(t, node, "hello", "world")
		require.Equal(t, "", p2.Get())
	})

	t.Run("returns errors", func(t *testing.T) {
		_, p2 := runParser("hello there", parser)
		require.Equal(t, "world", p2.Error.expected)
		require.Equal(t, 6, p2.Error.pos)
		require.Equal(t, 0, p2.Pos)
	})
}

func TestMaybe(t *testing.T) {
	t.Run("matches sequence", func(t *testing.T) {
		node, p2 := runParser("hello world", Maybe("hello"))
		require.Equal(t, "hello", node.Token)
		require.Equal(t, " world", p2.Get())
	})

	t.Run("returns no errors", func(t *testing.T) {
		node, p3 := runParser("hello world", Maybe("world"))
		require.Equal(t, Result{}, node)
		require.False(t, p3.Errored())
		require.Equal(t, 0, p3.Pos)
	})
}

func TestAny(t *testing.T) {
	t.Run("Matches any", func(t *testing.T) {
		node, p2 := runParser("hello world!", Any("hello", "world"))
		require.Equal(t, "hello", node.Token)
		require.Equal(t, 5, p2.Pos)
	})

	t.Run("Returns longest error", func(t *testing.T) {
		_, p2 := runParser("hello world!", Any(
			"nope",
			Seq("hello", "world", "."),
			Seq("hello", "brother"),
		))
		require.Equal(t, "offset 11: expected nope or .", p2.Error.Error())
		require.Equal(t, 11, p2.Error.Pos())
		require.Equal(t, 0, p2.Pos)
	})

	t.Run("Accepts nil matches", func(t *testing.T) {
		node, p2 := runParser("hello world!", Any(Exact("ffffff")))
		require.Equal(t, Result{}, node)
		require.Equal(t, 0, p2.Pos)
	})

	t.Run("overlapping longest match", func(t *testing.T) {
		EnableLogging(os.Stdout)
		p := OneOrMore(Any("ab", "a"))

		t.Run("a ab", func(t *testing.T) {
			node, ps := runParser("a ab", p)

			require.False(t, ps.Errored())
			require.Equal(t, "a", node.Child[0].Token)
			require.Equal(t, "ab", node.Child[1].Token)
		})

		t.Run("ab a", func(t *testing.T) {
			node, ps := runParser("ab a", p)

			require.False(t, ps.Errored())
			require.Equal(t, "ab", node.Child[0].Token)
			require.Equal(t, "a", node.Child[1].Token)

		})
	})

	// see https://github.com/vektah/goparsify/issues/3
	t.Run("doesn't succeed early after caller error", func(t *testing.T) {
		str := "str"
		str1 := Seq("str", "1")
		str2 := Any("str2")
		p := Any(str1, str2, str)
		_, ps := runParser("str", p)
		require.False(t, ps.Errored())
		require.Equal(t, "", ps.Get())
	})
}

func TestZeroOrMore(t *testing.T) {
	t.Run("Matches sequence with sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", ZeroOrMore(Chars("a-g"), ","))
		require.False(t, p2.Errored())
		assertSequence(t, node, "a", "b", "c", "d", "e")
		require.Equal(t, 10, p2.Pos)
	})

	t.Run("Matches sequence without trailing sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e1111", ZeroOrMore(Chars("a-g"), ","))
		require.False(t, p2.Errored())
		assertSequence(t, node, "a", "b", "c", "d", "e")
		require.Equal(t, "1111", p2.Get())
	})

	t.Run("Matches sequence without sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", ZeroOrMore(Any(Chars("a-g"), ",")))
		assertSequence(t, node, "a", ",", "b", ",", "c", ",", "d", ",", "e", ",")
		require.Equal(t, 10, p2.Pos)
	})

	t.Run("splits words automatically on space", func(t *testing.T) {
		node, p2 := runParser("hello world", ZeroOrMore(Chars("a-z")))
		assertSequence(t, node, "hello", "world")
		require.Equal(t, "", p2.Get())
	})

	t.Run("Stops on error", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", ZeroOrMore(Chars("a-c"), ","))
		assertSequence(t, node, "a", "b", "c")
		require.Equal(t, 6, p2.Pos)
		require.Equal(t, "d,e,", p2.Get())
	})
}

func TestOneOrMore(t *testing.T) {
	t.Run("Matches sequence with sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", OneOrMore(Chars("a-g"), Exact(",")))
		assertSequence(t, node, "a", "b", "c", "d", "e")
		require.Equal(t, 10, p2.Pos)
	})

	t.Run("Matches sequence without sep", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", OneOrMore(Any(Chars("abcdefg"), Exact(","))))
		assertSequence(t, node, "a", ",", "b", ",", "c", ",", "d", ",", "e", ",")
		require.Equal(t, 10, p2.Pos)
	})

	t.Run("Stops on error", func(t *testing.T) {
		node, p2 := runParser("a,b,c,d,e,", OneOrMore(Chars("abc"), Exact(",")))
		assertSequence(t, node, "a", "b", "c")
		require.Equal(t, 6, p2.Pos)
		require.Equal(t, "d,e,", p2.Get())
	})

	t.Run("Returns error if nothing matches", func(t *testing.T) {
		_, p2 := runParser("a,b,c,d,e,", OneOrMore(Chars("def"), Exact(",")))
		require.Equal(t, "offset 0: expected def", p2.Error.Error())
		require.Equal(t, "a,b,c,d,e,", p2.Get())
	})
}

type htmlTag struct {
	Name string
}

func TestMap(t *testing.T) {
	parser := Seq("<", Chars("a-zA-Z0-9"), ">").Map(func(n *Result) {
		n.Result = htmlTag{n.Child[1].Token}
	})

	t.Run("success", func(t *testing.T) {
		result, _ := runParser("<html>", parser)
		require.Equal(t, htmlTag{"html"}, result.Result)
	})

	t.Run("error", func(t *testing.T) {
		_, ps := runParser("<html", parser)
		require.Equal(t, "offset 5: expected >", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})
}

func TestChain(t *testing.T) {
	type data struct {
		typ string
		val any
	}

	// taken from https://youtu.be/nqNzWgSRCeg?t=237
	parser := Chain(
		Seq(Any("string", "number", "diceroll"), ":"),
		func(prevN *Result) Parserish {
			typ := prevN.Child[0].Token
			switch typ {
			case "string":
				return Chars("a-zA-Z0-9").Map(func(n *Result) {
					n.Result = data{
						typ: typ,
						val: n.Token,
					}
				})
			case "number":
				return NumberLit().Map(func(n *Result) {
					n.Result = data{
						typ: typ,
						val: n.Result,
					}
				})
			case "diceroll":
				num := Chars("0-9").Map(func(n *Result) {
					n.Result, _ = strconv.Atoi(n.Token)
				})
				return Seq(num, "d", num).Map(func(n *Result) {
					a := n.Child[0].Result.(int)
					b := n.Child[2].Result.(int)

					n.Result = data{
						typ: typ,
						val: []int{a, b},
					}
				})
			default:
				panic("case for type `" + typ + "` not covered")
			}
		})

	t.Run("success", func(t *testing.T) {
		result, _ := runParser("string:something", parser)
		require.Equal(t, "string", result.Result.(data).typ)
		require.Equal(t, "something", result.Result.(data).val)
	})

	t.Run("error", func(t *testing.T) {
		_, ps := runParser("number:&*%", parser)
		require.Equal(t, "offset 7: expected number", ps.Error.Error())
		require.Equal(t, 7, ps.Pos)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, ps := runParser("dunno:&*%", parser)
		require.Equal(t, "offset 0: expected string or number or diceroll", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})
}

func TestBind(t *testing.T) {
	parser := Bind("true", true)

	t.Run("success", func(t *testing.T) {
		result, _ := runParser("true", parser)
		require.Equal(t, true, result.Result)
	})

	t.Run("error", func(t *testing.T) {
		result, ps := runParser("nil", parser)
		require.Nil(t, result.Result)
		require.Equal(t, "offset 0: expected true", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})

}

func TestCut(t *testing.T) {
	t.Run("test any", func(t *testing.T) {
		_, ps := runParser("var world", Any(Seq("var", Cut(), "hello"), "var world"))
		require.Equal(t, "offset 4: expected hello", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})

	t.Run("test one or more", func(t *testing.T) {
		_, ps := runParser("hello <world", OneOrMore(Any(Seq("<", Cut(), Chars("a-z"), ">"), Chars("a-z"))))
		require.Equal(t, "offset 12: expected >", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})

	t.Run("test maybe", func(t *testing.T) {
		_, ps := runParser("var", Maybe(Seq("var", Cut(), "hello")))
		require.Equal(t, "offset 3: expected hello", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})
}

func TestMerge(t *testing.T) {
	var bracer Parser
	bracer = Seq("(", Maybe(&bracer), ")")
	parser := Merge(bracer)

	t.Run("success", func(t *testing.T) {
		result, _ := runParser("((()))", parser)
		require.Equal(t, "((()))", result.Token)
	})

	t.Run("error", func(t *testing.T) {
		_, ps := runParser("((())", parser)
		require.Equal(t, "offset 5: expected )", ps.Error.Error())
		require.Equal(t, 0, ps.Pos)
	})
}

func TestMapShorthand(t *testing.T) {
	Chars("a-z").Map(func(n *Result) {
		n.Result = n.Token
	})
}

func assertSequence(t *testing.T, node Result, expected ...string) {
	require.NotNil(t, node)
	actual := []string{}

	for _, child := range node.Child {
		actual = append(actual, child.Token)
	}

	require.Equal(t, expected, actual)
}
