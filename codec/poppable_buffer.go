package codec

type buff_allocator struct {
	data []byte
}
type pop_buff struct {
	allocator *buff_allocator

	pos  int
	size int
}

