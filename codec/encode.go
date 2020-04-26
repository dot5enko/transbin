package codec

import (
	"reflect"
)

func (c *codec) EncodeFull(obj interface{}) []byte {
	return c.encodeInternal(obj, true)
}
func (c *codec) Encode(obj interface{}) []byte {
	return c.encodeInternal(obj, false)
}

func (c *codec) encodeInternal(obj interface{}, full bool) []byte {

	c.reset()

	o := reflect.Indirect(reflect.ValueOf(obj))

	// generate structures
	generalStruct := c.registerStructure(o.Type())

	// data id
	c.mainBuffer.PutUint16(generalStruct.Id)

	// data
	c.writeComplexFieldData(generalStruct.Id, o)

	// write structure
	if full {
		c.encodeBuffer.Write(c.getStructureData())
	} else {
		c.encodeBuffer.WriteByte(0)
	}

	// write data to result
	c.encodeBuffer.Write(c.mainBuffer.Bytes())

	// dynamic length data
	c.encodeBuffer.Write(c.ref.buff.Bytes())

	return c.encodeBuffer.Bytes()
}

func (c *codec) writeSimpleFieldData(v reflect.Value) {

	switch v.Kind() {
	case reflect.Int32:
		c.mainBuffer.PutInt32(v.Interface().(int32))
	case reflect.Int:
		c.mainBuffer.PutInt32(int32(v.Interface().(int)))
	case reflect.Float32:
		c.mainBuffer.PutFloat32(v.Interface().(float32))
	default:
		panic("no handler for writing simple type " + v.Kind().String())
	}
}

func (c *codec) writeComplexFieldData(t uint16, v reflect.Value) {

	c.useType(t)

	cf := c.types[t].Fields

	for i := 0; i < int(c.types[t].FieldCount); i++ {
		c.writeFieldData(cf[i], v.Field(i))
	}
}

func (c *codec) writeFieldData(field codecStructField, v reflect.Value) {
	if field.Type > 26 {
		c.writeComplexFieldData(field.Type, v)
	} else if field.Type == 24 {
		c.writeReferenceFieldData(field.Type, v)
	} else {
		c.writeSimpleFieldData(v)
	}
}

func (c *codec) writeReferenceFieldData(t uint16, v reflect.Value) error {

	id, err := c.putReference(v)
	if err != nil {
		return err
	}

	c.mainBuffer.PutUint16(id)

	return nil
}



