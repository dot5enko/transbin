package codec

import (
	"fmt"
	"reflect"
	"unsafe"
)

type iWrapper struct {
	iface interface{}
}

func (c *codec) EncodeFull(obj interface{}) []byte {
	return c.encodeInternal(iWrapper{obj}, true)
}
func (c *codec) Encode(obj interface{}) []byte {
	return c.encodeInternal(iWrapper{obj}, false)
}

func (c *codec) encodeInternal(obj iWrapper, full bool) []byte {

	c.reset()

	objWrapper := reflect.ValueOf(obj)
	reflectRaw := reflect.ValueOf(obj.iface)

	o := reflect.Indirect(reflectRaw)
	var objPtr uintptr

	if (reflectRaw.Kind() != reflect.Ptr) {
		objPtr = objWrapper.Field(0).InterfaceData()[1]
	} else {
		objPtr = reflectRaw.Pointer()
	}

	// generate structures
	generalStruct := c.registerStructure(o)


	// data id
	c.mainBuffer.PutUint16(generalStruct.Id)

	// data
	c.writeComplexStructure(generalStruct.Id, objPtr)

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

func (c *codec) writeSimpleFieldData(t codecStructField,v uintptr) {

	switch t.Type {
	case uint16(reflect.Int32):

		result := *(*int32)(unsafe.Pointer(v+t.Offset))

		c.mainBuffer.PutInt32(result)

	case uint16(reflect.Int):

		result := int32(*(*int)(unsafe.Pointer(v+t.Offset)))

		c.mainBuffer.PutInt32(result)
	case uint16(reflect.Float32):
		result := *(*float32)(unsafe.Pointer(v+t.Offset))
		c.mainBuffer.PutFloat32(result)
	case uint16(reflect.Float64):

		result := *(*float64)(unsafe.Pointer(v+t.Offset))

			fmt.Printf(" float64 : %f\n",result)

		c.mainBuffer.PutFloat64(result)
	default:
		panic("no handler for writing simple type ")
	}
}

func (c *codec) writeComplexStructure(id uint16,v uintptr) {
	c.useType(id)

	cf := c.types[id].Fields

	for i := 0; i < int(c.types[id].FieldCount); i++ {

		// todo calc indirect uintptr

		c.writeFieldData(cf[i], v)
	}
}

func (c *codec) writeComplexFieldData(t codecStructField, v uintptr) {

	c.useType(t.Type)

	cf := c.types[t.Type].Fields

	for i := 0; i < int(c.types[t.Type].FieldCount); i++ {

		// todo calc indirect uintptr

		c.writeFieldData(cf[i], v + t.Offset)
	}
}

func (c *codec) writeFieldData(field codecStructField, v uintptr) {

	if field.Type > 26 {
		c.writeComplexFieldData(field, v)
	} else if field.Type == 24 {
		c.writeReferenceFieldData(field, v)
	} else {
		c.writeSimpleFieldData(field,v)
	}
}

func (c *codec) writeReferenceFieldData(t codecStructField, v uintptr) error {

	panic("not implemented!")

	//
	//id, err := c.putReference(v)
	//if err != nil {
	//	return err
	//}
	//
	//c.mainBuffer.PutUint16(id)
	//
	//return nil
}
