package codec

import (
	"log"
	"reflect"
)

type structDefinition struct {
	Fields     []codecStructField
	FieldCount uint8 // max 255 fields
	Id         uint16
	Name       string
}

type codecStructField struct {
	NameLength uint8
	Name       string
	Type       uint16 // reference to sturct definition
}

func (c *codec) registerStructure(ot reflect.Type) structDefinition {

	name := ot.PkgPath() + "." + ot.Name()
	value, ok := c.typeMap[name]
	if ok {
		return c.types[value]
	} else {

		fieldsCount := ot.NumField()

		c.typesCount += 1

		structDef := structDefinition{
			Fields:     make([]codecStructField, fieldsCount),
			Id:         c.typesCount,
			FieldCount: uint8(fieldsCount),
			Name:       name,
		}

		for i := 0; i < fieldsCount; i++ {

			fData := ot.Field(i)
			ft := fData.Type

			structDef.Fields[i].Name = fData.Name

			actualLenght := len(fData.Name)
			structDef.Fields[i].NameLength = uint8(actualLenght)
			if actualLenght != int(structDef.Fields[i].NameLength) {
				log.Printf("Field name is too long. field name was truncated, which may produce errors on decoding step")
			}

			if ft.Kind() == 25 {
				structDef.Fields[i].Type = c.registerStructure(ft).Id
			} else {
				structDef.Fields[i].Type = uint16(ft.Kind())
			}
		}

		c.types[structDef.Id] = structDef
		c.typeMap[name] = structDef.Id

		return c.types[structDef.Id]
	}
}

func (c *codec) getStructureData() []byte {

	numberOfTypes := c.usedTypes.Length()

	if numberOfTypes > 255 {
		panic("too much types nested")
	}

	// number of types in list
	c.structureBuffer.WriteByte(uint8(numberOfTypes))

	for i := 0; i < c.usedTypes.Length(); i++ {

		v := c.usedTypes.data[i]
		t := c.types[v]

		// type id uint16
		c.structureBuffer.PutUint16(t.Id)

		// number of fields uint8
		c.structureBuffer.WriteByte(t.FieldCount)

		for _, f := range t.Fields {
			// field type
			c.structureBuffer.PutUint16(f.Type)

			// field name
			c.structureBuffer.WriteByte(f.NameLength)
			c.structureBuffer.Write([]byte(f.Name))
		}
	}

	header := c.structureBuffer.Bytes()
	return header
}
