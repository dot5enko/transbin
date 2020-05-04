package codec

type pop_buff struct {
	data []byte
	pos  int
	size int

	states    []BufferState
	statesPos int
}

type BufferState struct {
	pos  int
	size int
	ref  []byte
}

func (this *pop_buff) InitStack() {
	this.states = make([]BufferState,10)
	this.statesPos = 0
}

func (this *pop_buff) Reset() {
	this.statesPos = 0
}

func (this *pop_buff) PushState(data []byte, at int) {

	this.states[this.statesPos] = BufferState{pos: this.pos, size: this.size, ref: this.data}

	this.data = data
	this.pos = at
	this.size = len(data)

	this.statesPos++
}

func (this *pop_buff) PopState() {

	this.statesPos--
	prevState := this.states[this.statesPos]

	this.data = prevState.ref
	this.pos = prevState.pos
	this.size = prevState.size
}