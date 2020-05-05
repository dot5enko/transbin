package codec

type pop_buff struct {
	*buff_allocator
}

type buff_allocator struct {
	data []byte
	size int
	pos int
}

func (this encode_buffer) Branch(areaSize int) encode_buffer {
	copy := this.BranchParalel()
	this.pos += areaSize
	return copy
}

func (this encode_buffer) BranchParalel() encode_buffer {

	oldPos := this.pos
	oldData := this.data
	oldSize := this.size

	this.buff_allocator = new(buff_allocator)
	this.pos = oldPos
	this.size = oldSize
	this.data = oldData

	return this
}
