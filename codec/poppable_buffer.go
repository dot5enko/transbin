package codec

type pop_buff struct {
	allocator *buff_allocator

	pos  int

	states    []BufferState
	statesPos int
}

type BufferState struct {
	pos  int
	size int
	ref  []byte
}

type buff_allocator struct {
	data []byte
	size int
}

func (this *encode_buffer) Branch(areaSize int) encode_buffer {
	copy := this.BranchParalel()
	this.pos += areaSize
	return copy
}

func (this encode_buffer) BranchParalel() encode_buffer {
	return this
}

