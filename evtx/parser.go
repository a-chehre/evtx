package evtx

import (
	"fmt"
	"io"
	"rawsec-evtx/log"
)

func checkParsingError(err error, e Element) {
	if err != nil {
		log.DontPanicf("%s: parsing %T", err, e)
	}
}

type ErrUnknownToken struct {
	Token uint8
}

func (e ErrUnknownToken) Error() string {
	return fmt.Sprintf("Unknown Token: 0x%02x", e.Token)
}

func Parse(reader io.ReadSeeker, c *Chunk, tiFlag bool) (Element, error) {
	var token [1]byte
	var err error
	read, err := reader.Read(token[:])
	if read != 1 || err != nil {
		return EmptyElement{}, err
	}
	_, err = reader.Seek(-1, io.SeekCurrent)
	if err != nil {
		return EmptyElement{}, err
	}
	switch token[0] {
	case FragmentHeaderToken:
		f := Fragment{}
		err = f.Parse(reader)
		f.BinXMLElement, err = Parse(reader, c, tiFlag)
		if err != nil {
			return &f, err
		}
		if _, ok := f.BinXMLElement.(*ElementStart); ok {
			var e Element
			var ti TemplateInstance
			ti.Definition.Data.Elements = make([]Element, 0)

			ti.Definition.Data.Elements = append(ti.Definition.Data.Elements, f.BinXMLElement.(*ElementStart))
			for e, err = Parse(reader, c, tiFlag); err == nil; e, err = Parse(reader, c, tiFlag) {
				ti.Definition.Data.Elements = append(ti.Definition.Data.Elements, e)
				if _, ok := e.(*BinXMLEOF); ok {
					break
				}
			}
			f.BinXMLElement = &ti
		}
		return &f, err
	case TokenOpenStartElementTag1, TokenOpenStartElementTag2:
		es := ElementStart{IsTemplateInstance: tiFlag}
		err = es.Parse(reader)
		return &es, err
	case TokenNormalSubstitution:
		ns := NormalSubstitution{}
		err = ns.Parse(reader)
		checkParsingError(err, &ns)
		return &ns, err
	case TokenOptionalSubstitution:
		oz := OptionalSubstitution{}
		err = oz.Parse(reader)
		checkParsingError(err, &oz)
		return &oz, err
	case TokenCharRef1:
		tcr := CharEntityRef{}
		err = tcr.Parse(reader)
		checkParsingError(err, &tcr)
		return &tcr, err
	case TokenTemplateInstance:
		var offset int32
		ti := TemplateInstance{}
		if c != nil {
			offset, err = ti.DataOffset(reader)
			if err != nil {
				return nil, err
			}
			if t, ok := c.TemplateTable[offset]; ok {
				err = ti.ParseTemplateDefinitionHeader(reader)
				if err != nil {
					return nil, err
				}
				ti.Definition.Data = t
				if int64(offset) == BackupSeeker(reader) {
					RelGoToSeeker(reader, int64(ti.Definition.Data.Size)+24)
				}
				err = ti.Data.Parse(reader)
				if err != nil {
					return nil, err
				}
				return &ti, nil
			}
		}
		err = ti.Parse(reader)
		if c != nil {
			c.TemplateTable[ti.Definition.Header.DataOffset] = ti.Definition.Data
		}
		checkParsingError(err, &ti)
		return &ti, err
	case TokenValue1, TokenValue2:
		vt := ValueText{}
		err = vt.Parse(reader)
		checkParsingError(err, &vt)
		return &vt, err

	case TokenEntityRef1, TokenEntityRef2:
		e := BinXMLEntityReference{}
		err = e.Parse(reader)
		checkParsingError(err, &e)
		return &e, err

	case TokenEndElementTag:
		b := BinXMLEndElementTag{}
		err = b.Parse(reader)
		checkParsingError(err, &b)
		return &b, err
	case TokenCloseStartElementTag:
		t := BinXMLCloseStartElementTag{}
		err = t.Parse(reader)
		checkParsingError(err, &t)
		return &t, err
	case TokenCloseEmptyElementTag:
		t := BinXMLCloseEmptyElementTag{}
		err = t.Parse(reader)
		checkParsingError(err, &t)
		return &t, err
	case TokenEOF:
		b := BinXMLEOF{}
		err = b.Parse(reader)
		checkParsingError(err, &b)
		return &b, nil
	}
	return EmptyElement{}, ErrUnknownToken{token[0]}
}

func ParseValueReader(vd ValueDescriptor, reader io.ReadSeeker) (Element, error) {
	var err error
	t := vd.ValType
	switch {
	case t.IsType(NullType):
		n := ValueNull{Size: vd.Size}
		_ = n.Parse(reader)
		return &n, err
	case t.IsType(StringType):
		str := ValueString{Size: vd.Size}
		err = str.Parse(reader)
		return &str, err
	case t.IsType(AnsiStringType):
		astring := AnsiString{Size: vd.Size}
		err = astring.Parse(reader)
		return &astring, err
	case t.IsType(Int8Type):
		i := ValueInt8{}
		err = i.Parse(reader)
		return &i, err
	case t.IsType(UInt8Type):
		u := ValueUInt8{}
		err = u.Parse(reader)
		return &u, err
	case t.IsType(Int16Type):
		i := ValueInt16{}
		err = i.Parse(reader)
		return &i, err
	case t.IsType(UInt16Type):
		u := ValueUInt16{}
		err = u.Parse(reader)
		return &u, err
	case t.IsType(Int32Type):
		i := ValueInt32{}
		err = i.Parse(reader)
		return &i, err
	case t.IsType(UInt32Type):
		u := ValueUInt32{}
		err = u.Parse(reader)
		return &u, err
	case t.IsType(Int64Type):
		i := ValueInt64{}
		err = i.Parse(reader)
		return &i, err
	case t.IsType(UInt64Type):
		u := ValueUInt64{}
		err = u.Parse(reader)
		return &u, err
	case t.IsType(Real64Type):
		r := ValueReal64{}
		err = r.Parse(reader)
		return &r, err
	case t.IsType(BoolType):
		b := ValueBool{}
		err = b.Parse(reader)
		return &b, err
	case t.IsType(BinaryType):
		binary := ValueBinary{Size: vd.Size}
		err = binary.Parse(reader)
		return &binary, err
	case t.IsType(GuidType):
		var guid ValueGUID
		err = guid.Parse(reader)
		return &guid, err
	case t.IsType(FileTimeType):
		filetime := ValueFileTime{}
		err = filetime.Parse(reader)
		return &filetime, err
	case t.IsType(SysTimeType):
		systime := ValueSysTime{}
		err = systime.Parse(reader)
		return &systime, err
	case t.IsType(SidType):
		var sid ValueSID
		err = sid.Parse(reader)
		return &sid, err
	case t.IsType(HexInt32Type):
		hi := ValueHexInt32{}
		err = hi.Parse(reader)
		return &hi, err
	case t.IsType(HexInt64Type):
		hi := ValueHexInt64{}
		err = hi.Parse(reader)
		return &hi, err
	case t.IsType(BinXmlType):
		var elt Element
		elt, err = Parse(reader, nil, true)
		if err != nil {
			log.Error(err)
		}
		return elt, err
	case t.IsArrayOf(StringType):
		st := ValueStringTable{Size: vd.Size}
		err = st.Parse(reader)
		return &st, err
	case t.IsArrayOf(UInt16Type):
		a := ValueArrayUInt16{Size: vd.Size}
		err = a.Parse(reader)
		return &a, err
	case t.IsArrayOf(UInt64Type):
		a := ValueArrayUInt64{Size: vd.Size}
		err = a.Parse(reader)
		return &a, err
	default:
		uv := UnkVal{BackupSeeker(reader), t, vd}
		_, err = reader.Seek(int64(vd.Size), io.SeekCurrent)
		if err != nil {
			panic(err)
		}
		return &uv, nil
	}
}
