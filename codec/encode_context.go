package codec

import (
	"github.com/dot5enko/transbin/utils"
	"reflect"
)

type encode_context struct {
	global    *codec
	usedTypes *DynamicArray

	ref *references_writer

	result_buffer encode_buffer
	data_buffer   encode_buffer
}

func NewEncodeContext(global *codec) *encode_context {

	result := &encode_context{}

	result.usedTypes = NewDArray(10)

	result.data_buffer = global.get_free_ebuffer(1024)
	result.result_buffer = global.get_free_ebuffer(1024)

	result.ref, _ = NewReferencesWriter(16, global.order)

	result.global = global

	return result
}

func (c encode_context) useType(t uint16) {
	c.usedTypes.Push(t)
}

func (c encode_context) Reset() {
	c.usedTypes.Clear()
	c.data_buffer.Reset()
	c.ref.Reset()

	c.result_buffer.Reset()
}

func (c *encode_context) EncodeFull(obj interface{}) ([]byte, error) {
	return c.encodeInternal(obj, true)
}
func (c *encode_context) Encode(obj interface{}) ([]byte, error) {
	return c.encodeInternal(obj, false)
}
func (c *encode_context) encodeElementToBuffer(buffer encode_buffer, o reflect.Value) (uint16, error) {

	var writtenType uint16 = 0

	t := o.Type()

	switch t.Kind() {

	case reflect.Struct:

		// general case when serializing object is a struct
		generalStruct, err := c.registerStructure(t)
		if err != nil {
			return 0, err
		}

		// data
		err = c.writeComplexType(buffer, generalStruct.Id, o)
		if err != nil {
			return 0, err
		}

		writtenType = generalStruct.Id

	case reflect.Map:

		fakeField := codecStructField{}
		fakeField.Type = uint16(reflect.Map)
		err := c.writeFieldData(buffer, fakeField, o)
		if err != nil {
			return 0, err
		}

		writtenType = uint16(reflect.Map)
	case reflect.Array:
		panic("don't know how to write arary")
	default:
		// its a case for map value
		var err error

		fakeField := codecStructField{}
		fakeField.Type, err = c.global.getType(o.Type())
		if err != nil {
			return 0, err
		}

		err = c.writeFieldData(buffer, fakeField, o)
		if err != nil {
			return 0, err
		}

		writtenType = uint16(t.Kind())

	}

	return writtenType, nil
}

func (c *encode_context) encodeInternal(obj interface{}, full bool) ([]byte, error) {

	c.Reset()

	o := reflect.Indirect(reflect.ValueOf(obj))

	// allocate 2 bytes for element type
	start := c.data_buffer.Branch(2)

	typeId, err := c.encodeElementToBuffer(c.data_buffer, o)

	start.PutUint16(typeId)

	if err != nil {
		return nil, err
	}

	// write structure
	if full {
		c.writeStructureData(c.result_buffer)
	} else {
		c.result_buffer.WriteByte(0)
	}

	// actual data
	c.result_buffer.Write(c.data_buffer.Bytes())

	// dynamic length data
	// todo use same buffer for references
	// just offset a type length
	refsData := c.ref.buff.Bytes()
	c.result_buffer.Write(refsData)

	return c.result_buffer.Bytes(), nil
}

func (c encode_context) writeSimpleFieldData(buffer *encode_buffer, v reflect.Value) error {



	switch v.Kind() {
	case reflect.Int32:
			buffer.PutInt32(int32(v.Int()))
	case reflect.Int:
			buffer.PutInt32(int32(v.Int()))
	case reflect.Float32:
			buffer.PutFloat32(float32(v.Float()))
	case reflect.Float64:
			buffer.PutFloat64(v.Float())
	default:
		return utils.Error("no handler for writing simple type %s", v.Kind().String())
	}

	return nil
}

func (c *encode_context) writeComplexType(buffer encode_buffer, t uint16, v reflect.Value) (err error) {

	c.useType(t)

	cf := c.global.types[t].Fields

	for i := 0; i < int(c.global.types[t].FieldCount); i++ {
		err = c.writeFieldData(buffer, cf[i], reflect.Indirect(v).Field(i))
		if err != nil {
			return
		}
	}

	return

}

func (c *encode_context) writeFieldData(buffer encode_buffer, field codecStructField, v reflect.Value) (err error) {

	if isArrayType(field.Type) {
		err = c.writeReferenceFieldData(buffer, field.Type, v)
		if err != nil {
			return
		}
	} else {
		if field.Type > internalTypesCount {
			err = c.writeComplexType(buffer, field.Type, v)
			if err != nil {
				return
			}
		} else {
			switch reflect.Kind(field.Type) {
			case reflect.String, reflect.Map, reflect.Interface:
				err = c.writeReferenceFieldData(buffer, field.Type, v)
				if err != nil {
					return
				}
			default:
				c.writeSimpleFieldData(&buffer, v)
			}
		}
	}

	return

}

func (c *encode_context) writeReferenceFieldData(buffer encode_buffer, t uint16, v reflect.Value) error {

	id, err := c.putReference(buffer, t, v)
	if err != nil {
		return err
	}

	buffer.PutUint16(id)

	return nil
}
