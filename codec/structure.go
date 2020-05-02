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

	// not used
	Offsets uint8

	// size of codec structure
	Size int
}

type codecStructField struct {
	NameLength uint8
	Name       string
	Type       uint16 // reference to sturct definition
	Offset     uintptr
	Size       int
}

func getTypeCode(ot reflect.Type) string {
	return ot.PkgPath() + "." + ot.Name()
}

func isArrayType(typeId uint16) bool {
	return ((typeId >> 15) & 1) == 1
}
func setArrayTypeFlag(typeId uint16) uint16 {
	typeId |= (1 << 15)
	return typeId
}
func getArrayElementType(typeId uint16) uint16 {
	mask := ^(1 << 15)
	typeId &= uint16(mask)
	return typeId
}

func (c *codec) registerStructure(ot reflect.Type) (*structDefinition, error) {

	if (ot.Kind() == reflect.Ptr) {
		ot = ot.Elem()
	}

	name := getTypeCode(ot)
	value, ok := c.typeMap[name]
	if ok {
		return c.types[value], nil
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

			sf := &structDef.Fields[i]

			fData := ot.Field(i)
			ft := fData.Type

			sf.Name = fData.Name

			actualLenght := len(fData.Name)
			sf.NameLength = uint8(actualLenght)
			if actualLenght != int(sf.NameLength) {
				log.Printf("Field name is too long. field name was truncated, which may produce errors on decoding step")
			}

			var err error

			switch ft.Kind() {
			case reflect.Struct:

				// could produce npe
				nested, err := c.registerStructure(ot.Field(i).Type)
				if err != nil {
					return nil, err
				}
				sf.Type = nested.Id
				sf.Size = nested.Size
			case reflect.Slice:

				sliceElem := ft.Elem()

				// unroll pointers
				for {
					if sliceElem.Kind() == reflect.Ptr {
						sliceElem = sliceElem.Elem()
					} else {
						break
					}
				}

				var typeWithArrayFlag uint16 = 0

				switch sliceElem.Kind() {
				case reflect.Struct:
					ok = false
					typeWithArrayFlag, ok = c.typeMap[getTypeCode(sliceElem)]
					if !ok {
						nested,err := c.registerStructure(sliceElem)
						if (err != nil) {
							return nil, err
						}

						typeWithArrayFlag = nested.Id

					}
				default:
					typeWithArrayFlag = uint16(sliceElem.Kind())
				}

				// set array bit flag
				typeWithArrayFlag = setArrayTypeFlag(typeWithArrayFlag)
				sf.Type = typeWithArrayFlag
				sf.Size = 2 // reference
			default:
				sf.Type = uint16(ft.Kind())
				sf.Size, err = c.getTypeSize(sf.Type)
				if err != nil {
					return nil, err
				}
			}

			structDef.Size += sf.Size
		}

		c.types[structDef.Id] = &structDef
		c.typeMap[name] = structDef.Id

		return c.types[structDef.Id], nil
	}
}

func (c *codec) getStructureData() []byte {

	numberOfTypes := c.usedTypes.Length()

	if numberOfTypes > 255 {
		panic("too much types nested")
	}

	// number of types in list
	c.structureBuffer.WriteByte(uint8(numberOfTypes))

	//
	c.usedTypes.Sort()

	for i := c.usedTypes.Length() - 1; i >= 0; i-- {

		v := c.usedTypes.data[i]
		t := c.types[uint16(v)]

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
