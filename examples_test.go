package goparsify

import (
	"fmt"
)

func ExampleCut() {
	// without a cut if the close tag is left out the parser will backtrack and ignore the rest of the string
	alpha := Chars("a-z")
	nocut := OneOrMore(Any(Seq("<", alpha, ">"), alpha))
	_, err := Run(nocut, "asdf <foo")
	fmt.Println(err.Error())

	// with a cut, once we see the open tag we know there must be a close tag that matches it, so the parser will error
	cut := OneOrMore(Any(Seq("<", Cut(), alpha, ">"), alpha))
	_, err = Run(cut, "asdf <foo")
	fmt.Println(err.Error())

	// Output:
	// left unparsed: <foo
	// offset 9: expected >
}
