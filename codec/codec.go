package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/dot5enko/transbin/utils"
	"reflect"
)

const defaultBufferSize int = 512
const internalTypesCount uint16 = 26

type codec struct {
	typesCount uint16
	types      map[uint16]*structDefinition
	order      binary.ByteOrder
	typeMap    map[string]uint16
}

func (c *codec) get_free_ebuffer(initialSize int) encode_buffer {
	return NewEncodeBuffer(initialSize, c.order)
}

func (c *codec) get_free_dbuffer() *decode_buffer {
	return NewDecodeBuffer(c.order)
}

func NewCodec(order binary.ByteOrder) (*codec, error) {
	result := &codec{}

	// in order to not interfer with internal types

	result.typesCount = 27
	result.types = make(map[uint16]*structDefinition)
	result.order = order
	result.typeMap = make(map[string]uint16)

	return result, nil
}

func (c codec) cacheReflectionData(typeId uint16, t reflect.Type) error {
	tData, ok := c.types[typeId]
	if !ok {
		return errors.New("unable to found a type declaration in codec")
	}

	if tData.Offsets > 0 {
		return nil
	}

	for i := 0; i < int(tData.FieldCount); i++ {
		f := &tData.Fields[i]

		refField, found := t.FieldByName(f.Name)

		if !found {
			return errors.New(fmt.Sprintf("field %s not found", f.Name))
		}

		f.Offset = refField.Offset
	}

	tData.Offsets = 1
	return nil
}

func unrollPrt(p reflect.Type) reflect.Type {
	for {
		if p.Kind() == reflect.Ptr {
			p = p.Elem()
		} else {
			break
		}
	}

	return p
}
func (c codec) getType(p reflect.Type) (uint16, error) {

	p = unrollPrt(p)

	t := uint16(p.Kind())

	switch reflect.Kind(t) {

	case reflect.Struct:
		tref, ok := c.typeMap[getTypeCode(p)]

		if !ok {
			return 0, utils.Error("Unable to found a size for type %d", t)
		}

		return tref, nil
	case reflect.Slice:
		t, err := c.getType(p.Elem())
		if err != nil {
			return 0, err
		}
		return setArrayTypeFlag(t), nil
	default:

		return t, nil

		//return 0, utils.Error("Unable to detect internal type for %s", p.Kind().String())
	}
}

func (c codec) getTypeSize(t uint16) (int, error) {

	if isArrayType(t) {
		return 2, nil
	}

	if t > internalTypesCount {

		tref, ok := c.types[t]

		if !ok {
			return 0, utils.Error("Unable to found a size for type %d", t)
		}

		return tref.Size, nil
	} else {
		switch reflect.Kind(t) {
		case reflect.Bool, reflect.Uint8:
			return 1, nil
		case reflect.String, reflect.Slice: // reference types
			return 2, nil
		case reflect.Uint16:
			return 2, nil
		case reflect.Float32, reflect.Int32, reflect.Uint, reflect.Int, reflect.Uint32, reflect.Uintptr, reflect.Interface:
			return 4, nil
		case reflect.Float64, reflect.Int64, reflect.Uint64:
			return 8, nil
		case reflect.Map:
			return 2 /* reference to data id*/ + 2 /*element type*/ + 2 /*elements count*/, nil
		default:
			return 0, fmt.Errorf("Unable to get a type length for type %s: %d", reflect.Kind(t).String(), t)
		}
	}
}
