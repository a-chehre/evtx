package encoding

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type Endianness binary.ByteOrder

var (
	ErrInvalidNilPointer  = errors.New("nil pointer is invalid")
	ErrNoPointerInterface = errors.New("interface expect to be a pointer")
)

func marshalArray(data interface{}, endianness Endianness) ([]byte, error) {
	var out []byte
	val := reflect.ValueOf(data)
	if val.IsNil() {
		return out, ErrInvalidNilPointer
	}
	elem := val.Elem()
	if elem.Kind() != reflect.Array {
		return out, fmt.Errorf("not an Array structure")
	}
	for k := 0; k < elem.Len(); k++ {
		buff, err := Marshal(elem.Index(k).Addr().Interface(), endianness)
		if err != nil {
			return out, err
		}
		out = append(out, buff...)
	}
	return out, nil
}

func marshalSlice(data interface{}, endianness Endianness) ([]byte, error) {
	var out []byte
	val := reflect.ValueOf(data)
	if val.IsNil() {
		return out, ErrInvalidNilPointer
	}
	elem := val.Elem()
	if elem.Kind() != reflect.Slice {
		return out, fmt.Errorf("not a Slice object")
	}
	s := elem
	sliceLen := int64(s.Len())
	buff, err := Marshal(&sliceLen, endianness)
	if err != nil {
		return out, err
	}
	out = append(out, buff...)
	for k := 0; k < s.Len(); k++ {
		buff, err := Marshal(s.Index(k).Addr().Interface(), endianness)
		if err != nil {
			return out, err
		}
		out = append(out, buff...)
	}
	return out, nil
}

func Marshal(data interface{}, endianness Endianness) ([]byte, error) {
	var out []byte
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr {
		return out, ErrNoPointerInterface
	}
	if val.IsNil() {
		return out, ErrInvalidNilPointer
	}
	elem := val.Elem()
	typ := elem.Type()
	switch typ.Kind() {
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			tField := typ.Field(i)
			switch tField.Type.Kind() {
			case reflect.Struct:
				buff, err := Marshal(elem.Field(i).Addr().Interface(), endianness)
				if err != nil {
					return out, err
				}
				out = append(out, buff...)
			case reflect.Array:
				buff, err := marshalArray(elem.Field(i).Addr().Interface(), endianness)
				if err != nil {
					return out, err
				}
				out = append(out, buff...)
			case reflect.Slice:
				buff, err := marshalSlice(elem.Field(i).Addr().Interface(), endianness)
				if err != nil {
					return out, err
				}
				out = append(out, buff...)
			default:
				buff, err := Marshal(elem.Field(i).Addr().Interface(), endianness)
				if err != nil {
					return out, err
				}
				out = append(out, buff...)
			}
		}
	case reflect.Array:
		buff, err := marshalArray(elem.Addr().Interface(), endianness)
		if err != nil {
			return out, err
		}
		out = append(out, buff...)

	case reflect.Slice:
		buff, err := marshalSlice(elem.Addr().Interface(), endianness)
		if err != nil {
			return out, err
		}
		out = append(out, buff...)

	default:
		writter := new(bytes.Buffer)
		if err := binary.Write(writter, endianness, elem.Interface()); err != nil {
			return out, err
		}
		out = append(out, writter.Bytes()...)
	}
	return out, nil
}

func UnmarshaInitSlice(reader io.Reader, data interface{}, endianness Endianness) error {
	val := reflect.ValueOf(data)
	if val.IsNil() {
		return ErrInvalidNilPointer
	}
	slice := val.Elem()
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("not a slice object")
	}
	if slice.Len() == 0 {
		return fmt.Errorf("not initialized slice")
	}
	for k := 0; k < slice.Len(); k++ {
		err := Unmarshal(reader, slice.Index(k).Addr().Interface(), endianness)
		if err != nil {
			return err
		}
	}
	return nil
}

func unmarshalArray(reader io.Reader, data interface{}, endianness Endianness) error {
	val := reflect.ValueOf(data)
	if val.IsNil() {
		return ErrInvalidNilPointer
	}
	array := val.Elem()
	if array.Kind() != reflect.Array {
		return fmt.Errorf("not an Array structure")
	}
	for k := 0; k < array.Len(); k++ {
		err := Unmarshal(reader, array.Index(k).Addr().Interface(), endianness)
		if err != nil {
			return err
		}
	}
	return nil
}

func unmarshalSlice(reader io.Reader, data interface{}, endianness Endianness) error {
	var sliceLen int64
	val := reflect.ValueOf(data)
	if val.IsNil() {
		return ErrInvalidNilPointer
	}
	elem := val.Elem()
	if elem.Kind() != reflect.Slice {
		return fmt.Errorf("not a Slice object")
	}
	err := Unmarshal(reader, &sliceLen, endianness)
	if err != nil {
		return err
	}
	s := elem
	newS := reflect.MakeSlice(s.Type(), int(sliceLen), int(sliceLen))
	s.Set(newS)

	for k := 0; k < s.Len(); k++ {
		err := Unmarshal(reader, s.Index(k).Addr().Interface(), endianness)
		if err != nil {
			return err
		}
	}
	return nil
}

func Unmarshal(reader io.Reader, data interface{}, endianness Endianness) error {
	val := reflect.ValueOf(data)
	if val.IsNil() {
		return ErrInvalidNilPointer
	}
	elem := val.Elem()
	typ := elem.Type()
	switch typ.Kind() {
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			tField := typ.Field(i)
			switch tField.Type.Kind() {
			case reflect.Struct:
				err := Unmarshal(reader, elem.Field(i).Addr().Interface(), endianness)
				if err != nil {
					return err
				}
			case reflect.Array:
				err := unmarshalArray(reader, elem.Field(i).Addr().Interface(), endianness)
				if err != nil {
					return err
				}
			case reflect.Slice:
				err := unmarshalSlice(reader, elem.Field(i).Addr().Interface(), endianness)
				if err != nil {
					return err
				}
			default:
				if err := Unmarshal(reader, elem.Field(i).Addr().Interface(), endianness); err != nil {
					return err
				}
			}
		}

	case reflect.Array:
		err := unmarshalArray(reader, elem.Addr().Interface(), endianness)
		if err != nil {
			return err
		}

	case reflect.Slice:
		err := unmarshalSlice(reader, elem.Addr().Interface(), endianness)
		if err != nil {
			return err
		}

	default:
		if err := binary.Read(reader, endianness, data); err != nil {
			return err
		}
	}
	return nil
}
