package codec

import (
	"github.com/dot5enko/transbin/utils"
	"reflect"
)

func (c *codec) EncodeFull(obj interface{}) ([]byte, error) {
	return c.encodeInternal(obj, true)
}
func (c *codec) Encode(obj interface{}) ([]byte, error) {
	return c.encodeInternal(obj, false)
}
func (c *codec) encodeElementToBuffer(buffer *encode_buffer, o reflect.Value) (uint16, error) {

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
		fakeField.Type, err = c.getType(o.Type())
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

func (c *codec) encodeInternal(obj interface{}, full bool) ([]byte, error) {

	c.reset()

	o := reflect.Indirect(reflect.ValueOf(obj))

	// allocate 2 bytes for element type
	start := c.mainBuffer.Branch(2)

	typeId, err := c.encodeElementToBuffer(c.mainBuffer, o)

	start.PutUint16(typeId)

	if err != nil {
		return nil, err
	}

	// write structure
	if full {
		c.encodeBuffer.Write(c.getStructureData())
	} else {
		c.encodeBuffer.WriteByte(0)
	}

	// actual data
	c.encodeBuffer.Write(c.mainBuffer.Bytes())

	// dynamic length data
	refsData := c.ref.buff.Bytes()
	c.encodeBuffer.Write(refsData)

	return c.encodeBuffer.Bytes(), nil
}

func (c *codec) writeSimpleFieldData(buffer *encode_buffer, v reflect.Value) error {

	switch v.Kind() {
	case reflect.Int32:
		buffer.PutInt32(v.Interface().(int32))
	case reflect.Int:
		buffer.PutInt32(int32(v.Interface().(int)))
	case reflect.Float32:
		buffer.PutFloat32(v.Interface().(float32))
	case reflect.Float64:
		buffer.PutFloat64(v.Interface().(float64))
	case reflect.Int64:
		buffer.PutInt64(v.Interface().(int64))
	default:
		return utils.Error("no handler for writing simple type %s", v.Kind().String())
	}

	return nil
}

func (c *codec) writeComplexType(buffer *encode_buffer, t uint16, v reflect.Value) (err error) {

	c.useType(t)

	cf := c.types[t].Fields

	for i := 0; i < int(c.types[t].FieldCount); i++ {
		err = c.writeFieldData(buffer, cf[i], reflect.Indirect(v).Field(i))
		if err != nil {
			return
		}
	}

	return

}

func (c *codec) writeFieldData(buffer *encode_buffer, field codecStructField, v reflect.Value) (err error) {

	if isArrayType(field.Type) {
		err = c.writeReferenceFieldData(buffer, field.Type, v)
	} else {
		if field.Type > internalTypesCount {
			err = c.writeComplexType(buffer, field.Type, v)
		} else {
			switch reflect.Kind(field.Type) {
			case reflect.String, reflect.Map, reflect.Interface:
				err = c.writeReferenceFieldData(buffer, field.Type, v)
			default:
				err = c.writeSimpleFieldData(buffer, v)
			}
		}
	}

	return

}

func (c *codec) writeReferenceFieldData(buffer *encode_buffer, t uint16, v reflect.Value) error {

	id, err := c.putReference(buffer, t, v)
	if err != nil {
		return err
	}

	buffer.PutUint16(id)

	return nil
}
