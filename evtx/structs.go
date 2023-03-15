package evtx

import (
	"fmt"
	"io"

	"rawsec-evtx/encoding"
	"rawsec-evtx/log"
)

type EventIDType int64

type Element interface {
	Parse(reader io.ReadSeeker) error
}

type FragmentHeader struct {
	Token      int8
	MajVersion int8
	MinVersion int8
	Flags      int8
}

func (fh *FragmentHeader) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, fh, Endianness)
	if fh.Token != FragmentHeaderToken {
		return fmt.Errorf("bad fragment header token (0x%02x) instead of 0x%02x", fh.Token, FragmentHeaderToken)
	}
	return err
}

func (fh FragmentHeader) String() string {
	return fmt.Sprintf("%T: %s", fh, string(ToJSON(fh)))
}

type Fragment struct {
	Offset        int64
	Header        FragmentHeader
	BinXMLElement Element
}

func (f *Fragment) GoEvtxMap() *GoEvtxMap {
	switch f.BinXMLElement.(type) {
	case *TemplateInstance:
		pgem := f.BinXMLElement.(*TemplateInstance).GoEvtxMap()
		pgem.DelXmlns()
		return pgem
	}
	return nil
}

func (f *Fragment) Parse(reader io.ReadSeeker) error {
	f.Offset = BackupSeeker(reader)
	err := f.Header.Parse(reader)
	if err != nil {
		return err
	}
	return err
}

func (f Fragment) String() string {
	return fmt.Sprintf("%T: %s", f, string(ToJSON(f)))
}

type ElementStart struct {
	Offset             int64
	IsTemplateInstance bool
	Token              int8
	DepID              int16
	Size               int32
	NameOffset         int32
	Name               Name
	AttributeList      AttributeList
	EOESToken          uint8
}

func (es *ElementStart) Parse(reader io.ReadSeeker) (err error) {
	es.Offset = BackupSeeker(reader)
	es.NameOffset = DefaultNameOffset
	err = encoding.Unmarshal(reader, &es.Token, Endianness)
	if err != nil {
		return err
	}

	if es.IsTemplateInstance {
		err = encoding.Unmarshal(reader, &es.DepID, Endianness)
		if err != nil {
			return err
		}
	}

	err = encoding.Unmarshal(reader, &es.Size, Endianness)
	if err != nil {
		return err
	}

	err = encoding.Unmarshal(reader, &es.NameOffset, Endianness)
	if err != nil {
		return err
	}

	backup := BackupSeeker(reader)
	if backup != int64(es.NameOffset) {
		GoToSeeker(reader, int64(es.NameOffset))
	}
	err = es.Name.Parse(reader)
	if err != nil {
		return err
	}
	if backup != int64(es.NameOffset) {
		GoToSeeker(reader, backup)
	}
	if es.Token == TokenOpenStartElementTag2 {
		err = es.AttributeList.Parse(reader)
		if err != nil {
			return err
		}
	}

	err = encoding.Unmarshal(reader, &es.EOESToken, Endianness)
	if err != nil {
		return err
	}

	if es.EOESToken != TokenCloseStartElementTag && es.EOESToken != TokenCloseEmptyElementTag {
		if es.Token == TokenOpenStartElementTag1 {
			return fmt.Errorf("bad close element token (0x%02x) instead of 0x%02x", es.EOESToken, TokenCloseEmptyElementTag)
		} else {
			return fmt.Errorf("bad close element token (0x%02x) instead of 0x%02x", es.EOESToken, TokenCloseStartElementTag)
		}
	}
	RelGoToSeeker(reader, -1)
	return err
}

func (es ElementStart) String() string {
	return fmt.Sprintf("%T: %s", es, string(ToJSON(es)))
}

type NormalSubstitution struct {
	Token   int8
	SubID   int16
	ValType int8
}

type OptionalSubstitution struct {
	NormalSubstitution
}

func (n *NormalSubstitution) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &n.Token, Endianness)
	if err != nil {
		return err
	}
	err = encoding.Unmarshal(reader, &n.SubID, Endianness)
	if err != nil {
		return err
	}
	return encoding.Unmarshal(reader, &n.ValType, Endianness)
}

func (n *NormalSubstitution) String() string {
	return fmt.Sprintf("%T: %[1]v", *n)
}

type Attribute struct {
	Token         int8
	NameOffset    int32
	Name          Name
	AttributeData Element
}

func (a *Attribute) IsLast() bool {
	return a.Token == TokenAttribute1
}

func (a *Attribute) Parse(reader io.ReadSeeker) error {
	var err error
	err = encoding.Unmarshal(reader, &a.Token, Endianness)
	if err != nil {
		return err
	}
	if a.Token != TokenAttribute1 && a.Token != TokenAttribute2 {
		return fmt.Errorf("bad attribute Token : 0x%02x", uint8(a.Token))
	}
	err = encoding.Unmarshal(reader, &a.NameOffset, Endianness)
	if err != nil {
		return err
	}
	cursor := BackupSeeker(reader)
	if int64(a.NameOffset) != cursor {
		GoToSeeker(reader, int64(a.NameOffset))
	}
	_ = a.Name.Parse(reader)
	if int64(a.NameOffset) != cursor {
		GoToSeeker(reader, cursor)
	}
	a.AttributeData, err = Parse(reader, nil, false)
	return err
}

type AttributeList struct {
	Size       int32
	Attributes []Attribute
}

func (al *AttributeList) ParseSize(reader io.ReadSeeker) error {
	return encoding.Unmarshal(reader, &al.Size, Endianness)
}

func (al *AttributeList) ParseAttributes(reader io.ReadSeeker) error {
	var err error
	al.Attributes = make([]Attribute, 0)
	for {
		attr := Attribute{}
		err = attr.Parse(reader)
		if err != nil {
			return err
		}
		al.Attributes = append(al.Attributes, attr)
		if attr.IsLast() {
			break
		}
	}
	return err
}

func (al *AttributeList) Parse(reader io.ReadSeeker) error {
	err := al.ParseSize(reader)
	if err != nil {
		return err
	}
	return al.ParseAttributes(reader)
}

type Name struct {
	OffsetPrevString int32
	Hash             uint16
	Size             uint16
	UTF16String      UTF16String
}

func (n *Name) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &n.OffsetPrevString, Endianness)
	if err != nil {
		return err
	}
	err = encoding.Unmarshal(reader, &n.Hash, Endianness)
	if err != nil {
		return err
	}
	err = encoding.Unmarshal(reader, &n.Size, Endianness)
	if err != nil {
		return err
	}

	n.UTF16String = make([]uint16, n.Size+1)

	err = encoding.UnmarshaInitSlice(reader, &n.UTF16String, Endianness)
	return err
}

func (n *Name) String() string {
	return n.UTF16String.ToString()
}

type CharEntityRef struct {
	Token int8
	Value int16
}

func (cer *CharEntityRef) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, cer, Endianness)
	return err
}

type ValueText struct {
	Token   int8
	ValType int8
	Value   UnicodeTextString
}

func (vt *ValueText) String() string {
	return vt.Value.String.ToString()
}

func (vt *ValueText) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &vt.Token, Endianness)
	if err != nil {
		return err
	}
	err = encoding.Unmarshal(reader, &vt.ValType, Endianness)
	if err != nil {
		return err
	}
	if vt.ValType != StringType {
		return fmt.Errorf("bad type, must be (0x%02x) StringType", StringType)
	}
	err = vt.Value.Parse(reader)
	return err
}

type UnicodeTextString struct {
	Size   int16
	String UTF16String
}

func (uts *UnicodeTextString) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &uts.Size, Endianness)
	if err != nil {
		return err
	}

	if uts.Size > 0 {
		uts.String = make(UTF16String, uts.Size)
		err = encoding.UnmarshaInitSlice(reader, &uts.String, Endianness)
	}

	return err
}

type EntityReference struct {
	Token            int8
	EntityNameOffset int32
}

type CDATASection struct {
	Token int8
	Text  UnicodeTextString
}

type PITarget struct {
	Token      int8
	NameOffset int32
}

type PIData struct {
	Token int8
	Text  UnicodeTextString
}

func (ti *TemplateInstance) Root() Node {
	node, _ := NodeTree(ti.Definition.Data.Elements, 0)
	return node
}

func (ti *TemplateInstance) ElementToGoEvtx(elt Element) GoEvtxElement {
	switch elt.(type) {
	case *ValueText:
		return elt.(*ValueText).String()
	case *OptionalSubstitution:
		s := elt.(*OptionalSubstitution)
		switch {
		case int(s.SubID) < len(ti.Data.Values):
			return ti.ElementToGoEvtx(ti.Data.Values[int(s.SubID)])
		default:
			panic("Index out of range")
		}
	case *NormalSubstitution:
		s := elt.(*NormalSubstitution)
		switch {
		case int(s.SubID) < len(ti.Data.Values):
			return ti.ElementToGoEvtx(ti.Data.Values[int(s.SubID)])
		default:
			panic("Index out of range")
		}
	case *Fragment:
		temp := elt.(*Fragment).BinXMLElement.(*TemplateInstance)
		root := temp.Root()
		return temp.NodeToGoEvtx(&root)
	case *TemplateInstance:
		temp := elt.(*TemplateInstance)
		root := temp.Root()
		return temp.NodeToGoEvtx(&root)
	case Value:
		if _, ok := elt.(Value).(*ValueNull); ok {
			return nil
		}
		return elt.(Value).Repr()
	case *BinXMLEntityReference:
		ers := elt.(*BinXMLEntityReference).String()
		if ers == "" {
			err := fmt.Errorf("unknown entity reference: %s", ers)
			panic(err)
		}
		return ers

	default:
		err := fmt.Errorf("don't know how to handle: %T", elt)
		panic(err)
	}
}

func (ti *TemplateInstance) NodeToGoEvtx(n *Node) GoEvtxMap {
	switch {
	case n.Start == nil && len(n.Child) == 1:
		m := make(GoEvtxMap)
		m[n.Child[0].Start.Name.String()] = ti.NodeToGoEvtx(n.Child[0])
		return m

	default:
		m := make(GoEvtxMap, len(n.Child))
		for i, c := range n.Child {
			node := ti.NodeToGoEvtx(c)
			switch {
			case node.HasKeys("Name") && len(node) == 1:
				m[node["Name"].(string)] = ""
			case node.HasKeys("Name", "Value") && len(node) == 2:
				m[node["Name"].(string)] = node["Value"]
			default:
				name := c.Start.Name.String()
				if _, ok := m[name]; ok {
					name = fmt.Sprintf("%s%d", name, i)
				}
				if node.HasKeys("Value") && len(node) == 1 {
					m[name] = node["Value"]
				} else {
					m[name] = node
				}
			}
		}

		for _, e := range n.Element {
			ge := ti.ElementToGoEvtx(e)
			switch ge.(type) {
			case GoEvtxMap:
				other := ge.(GoEvtxMap)
				m.Add(other)
			case string:
				if ge != nil {
					if m["Value"] == nil {
						m["Value"] = ge.(string)
					} else {
						m["Value"] = m["Value"].(string) + ge.(string)
					}
				}
			default:
				m["Value"] = ti.ElementToGoEvtx(n.Element[0])
			}
		}
		if n.Start != nil {
			for _, attr := range n.Start.AttributeList.Attributes {
				gee := ti.ElementToGoEvtx(attr.AttributeData)
				if gee != nil {
					m[attr.Name.String()] = gee
				}
			}
		}
		return m
	}
}

func (ti *TemplateInstance) GoEvtxMap() *GoEvtxMap {
	root := ti.Root()
	gem := ti.NodeToGoEvtx(&root)
	return &gem
}

type TemplateInstance struct {
	Token      int8
	Definition TemplateDefinition
	Data       TemplateInstanceData
}

func (ti *TemplateInstance) DataOffset(reader io.ReadSeeker) (offset int32, err error) {
	backup := BackupSeeker(reader)
	GoToSeeker(reader, backup+6)
	err = encoding.Unmarshal(reader, &offset, Endianness)
	GoToSeeker(reader, backup)
	return
}

func (ti *TemplateInstance) ParseTemplateDefinitionHeader(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &ti.Token, Endianness)
	if err != nil {
		return err
	}
	return ti.Definition.Header.Parse(reader)
}

func (ti *TemplateInstance) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &ti.Token, Endianness)
	if err != nil {
		return err
	}
	err = ti.Definition.Parse(reader)
	if err != nil {
		return err
	}
	err = ti.Data.Parse(reader)
	return err
}

func (ti TemplateInstance) String() string {
	return fmt.Sprintf("%T: %s", ti, string(ToJSON(ti)))
}

type TemplateDefinitionHeader struct {
	Unknown1   int8
	Unknown2   int32
	DataOffset int32
}

func (tdh *TemplateDefinitionHeader) Parse(reader io.ReadSeeker) error {
	return encoding.Unmarshal(reader, tdh, Endianness)
}

type TemplateDefinitionData struct {
	Unknown3   int32
	ID         [16]byte
	Size       int32
	FragHeader FragmentHeader
	Elements   []Element
	EOFToken   int8
}

func (td *TemplateDefinitionData) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &td.Unknown3, Endianness)
	if err != nil {
		return err
	}
	err = encoding.Unmarshal(reader, &td.ID, Endianness)
	if err != nil {
		return err
	}
	err = encoding.Unmarshal(reader, &td.Size, Endianness)
	if err != nil {
		return err
	}

	err = td.FragHeader.Parse(reader)
	if err != nil {
		return err
	}

	td.Elements = make([]Element, 0)
	for {
		var elt Element
		elt, err = Parse(reader, nil, true)
		if err != nil {
			return err
		}
		if _, ok := elt.(*BinXMLEOF); ok {
			td.EOFToken = TokenEOF
			break
		}
		td.Elements = append(td.Elements, elt)
	}
	return nil
}

type TemplateDefinition struct {
	Header TemplateDefinitionHeader
	Data   TemplateDefinitionData
}

func (td *TemplateDefinition) Parse(reader io.ReadSeeker) error {
	err := td.Header.Parse(reader)
	if err != nil {
		return err
	}
	backup := BackupSeeker(reader)
	if int64(td.Header.DataOffset) != backup {
		GoToSeeker(reader, int64(td.Header.DataOffset))
	}
	err = td.Data.Parse(reader)
	if err != nil {
		return err
	}
	if int64(td.Header.DataOffset) != backup {
		GoToSeeker(reader, backup)
	}
	return err
}

func (td TemplateDefinition) String() string {
	return fmt.Sprintf("%T: %s", td, string(ToJSON(td)))
}

type TemplateInstanceData struct {
	NumValues    int32
	ValDescs     []ValueDescriptor
	Values       []Element
	ValueOffsets []int32
}

func (tid *TemplateInstanceData) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &tid.NumValues, Endianness)
	if err != nil {
		return err
	}
	if tid.NumValues < 0 {
		return fmt.Errorf("negative number of values in TemplateInstanceData")
	}
	if tid.NumValues > MaxSliceSize {
		return fmt.Errorf("too many values in TemplateInstanceData")
	}
	tid.Values = make([]Element, tid.NumValues)
	tid.ValueOffsets = make([]int32, tid.NumValues)
	tid.ValDescs = make([]ValueDescriptor, tid.NumValues)
	if tid.NumValues > 0 {
		err = encoding.UnmarshaInitSlice(reader, &tid.ValDescs, Endianness)
		if err != nil {
			return err
		}
	}

	for i := int32(0); i < tid.NumValues; i++ {
		tid.Values[i], err = ParseValueReader(tid.ValDescs[i], reader)
		if err != nil {
			log.Errorf("%v : %s", tid.ValDescs[i], err)
		}
	}
	return err
}

type ValueDescriptor struct {
	Size    uint16
	ValType ValueType
	Unknown int8
}

func (v ValueDescriptor) String() string {
	return fmt.Sprintf("Size: %d ValType: 0x%02x Unk: 0x%02x", v.Size, v.ValType, v.Unknown)
}

type BinXMLEOF struct {
	Token int8
}

func (b *BinXMLEOF) Parse(reader io.ReadSeeker) error {
	return encoding.Unmarshal(reader, &b.Token, Endianness)
}

type BinXMLEntityReference struct {
	Token      int8
	NameOffset uint32
	Name       Name
}

func (e *BinXMLEntityReference) Parse(reader io.ReadSeeker) error {
	err := encoding.Unmarshal(reader, &e.Token, Endianness)
	if err != nil {
		return err
	}
	err = encoding.Unmarshal(reader, &e.NameOffset, Endianness)
	if err != nil {
		return err
	}
	o := BackupSeeker(reader)
	if int64(e.NameOffset) == o {
		return e.Name.Parse(reader)
	}
	GoToSeeker(reader, int64(e.NameOffset))
	err = e.Name.Parse(reader)
	GoToSeeker(reader, o)
	return err
}

func (e *BinXMLEntityReference) String() string {
	switch e.Name.String() {
	case "amp":
		return "&"
	case "lt":
		return "<"
	case "gt":
		return ">"
	case "quot":
		return "\""
	case "apos":
		return "'"
	}
	return ""
}

type Token struct {
	Token int8
}

func (t *Token) Parse(reader io.ReadSeeker) error {
	return encoding.Unmarshal(reader, &t.Token, Endianness)
}

type BinXMLEndElementTag struct {
	Token
}

type BinXMLCloseStartElementTag struct {
	Token
}

type BinXMLCloseEmptyElementTag struct {
	Token
}

type EmptyElement struct {
}

func (EmptyElement) Parse(io.ReadSeeker) error {
	return nil
}
