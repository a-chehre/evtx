package evtx

type Node struct {
	Start   *ElementStart
	Element []Element
	Child   []*Node
}

func NodeTree(es []Element, index int) (Node, int) {
	var n Node
	for index < len(es) {
		e := es[index]
		switch e.(type) {
		case *ElementStart:
			var nn Node
			nn, index = NodeTree(es, index+1)
			nn.Start = e.(*ElementStart)
			n.Child = append(n.Child, &nn)
		case *BinXMLEndElementTag, *BinXMLCloseEmptyElementTag:
			return n, index
		case *BinXMLCloseStartElementTag:
			break
		default:
			n.Element = append(n.Element, e)
		}
		index++
	}
	return n, index
}
