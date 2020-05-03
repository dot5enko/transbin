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
func (c *codec) encodeToMainBuffer(o reflect.Value) (uint16,error) {

	var writtenType uint16 = 0

	t := o.Type()
	if t.Kind() == reflect.Struct {

		// general case when serializing object is a struct
		generalStruct, err := c.registerStructure(t)
		if err != nil {
			return 0,err
		}

		// data id
		c.mainBuffer.PutUint16(generalStruct.Id)

		// data
		err = c.writeComplexType(generalStruct.Id, o)
		if err != nil {
			return 0,err
		}

		writtenType = generalStruct.Id

	} else if t.Kind() == reflect.Map {
		// map
		c.mainBuffer.PutUint16(uint16(reflect.Map))

		// write map as a ref value cause it have dynamic length and no strict structure
		_, err := c.putReference(uint16(reflect.Map), o)
		if err != nil {
			return 0,err
		}

		writtenType = uint16(reflect.Map)

	} else {

		// its a case for map value

		err := c.writeSimpleFieldData(o)
		if err != nil {
			return 0,err
		}

		writtenType = uint16(t.Kind())

	}

	return writtenType,nil
}

func (c *codec) encodeInternal(obj interface{}, full bool) ([]byte, error) {

	c.reset()

	o := reflect.Indirect(reflect.ValueOf(obj))

	c.encodeToMainBuffer(o)

	// write structure
	if full {
		c.encodeBuffer.Write(c.getStructureData())
	} else {
		c.encodeBuffer.WriteByte(0)
	}

	// actual data
	c.encodeBuffer.Write(c.mainBuffer.Bytes())

	// dynamic length data
	c.encodeBuffer.Write(c.ref.buff.Bytes())

	return c.encodeBuffer.Bytes(), nil
}

func (c *codec) writeSimpleFieldData(v reflect.Value) error {

	switch v.Kind() {
	case reflect.Int32:
		c.mainBuffer.PutInt32(v.Interface().(int32))
	case reflect.Int:
		c.mainBuffer.PutInt32(int32(v.Interface().(int)))
	case reflect.Float32:
		c.mainBuffer.PutFloat32(v.Interface().(float32))
	case reflect.Float64:
		c.mainBuffer.PutFloat64(v.Interface().(float64))
	default:
		return utils.Error("no handler for writing simple type %s", v.Kind().String())
	}

	return nil
}

func (c *codec) writeComplexType(t uint16, v reflect.Value) (err error) {

	c.useType(t)

	cf := c.types[t].Fields

	for i := 0; i < int(c.types[t].FieldCount); i++ {
		err = c.writeFieldData(cf[i], reflect.Indirect(v).Field(i))
		if err != nil {
			return
		}
	}

	return

}

func (c *codec) writeFieldData(field codecStructField, v reflect.Value) (err error) {

	if isArrayType(field.Type) {
		err = c.writeReferenceFieldData(field.Type, v)
		if err != nil {
			return
		}
	} else {
		if field.Type > internalTypesCount {
			err = c.writeComplexType(field.Type, v)
			if err != nil {
				return
			}
		} else {
			switch reflect.Kind(field.Type) {
			case reflect.String, reflect.Map:
				err = c.writeReferenceFieldData(field.Type, v)
				if err != nil {
					return
				}
			default:
				c.writeSimpleFieldData(v)
			}
		}
	}

	return

}
func (c *codec) writeSliceFieldData(t uint16, v reflect.Value) (data []byte, err error) {
	sliceLength := v.Len()

	if sliceLength > 0 {

		data := make([]byte,512);
		c.mainBuffer.PushState(data,0)

		var fakeField codecStructField
		fakeField.Type = t

		for i := 0; i < sliceLength; i++ {
			c.writeFieldData(fakeField, v.Index(i))
		}

		data = data[:c.mainBuffer.pos]
		c.mainBuffer.PopState()
	} else {
		data = []byte("")
	}

	return

}

func (c *codec) writeReferenceFieldData(t uint16, v reflect.Value) error {

	id, err := c.putReference(t, v)
	if err != nil {
		return err
	}

	c.mainBuffer.PutUint16(id)

	return nil
}
