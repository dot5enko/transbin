package codec

import (
	"container/list"
)

type pop_buff struct {
	data []byte
	pos  int
	size int

	states *list.List
}

type BufferState struct {
	pos  int
	size int
	ref  []byte
}

func (this *pop_buff) InitStack() {
	this.states = list.New()
}

func (this *pop_buff) PushState(data []byte, at int) {

	this.states.PushBack(BufferState{pos: this.pos, size: this.size, ref: this.data})

	this.data = data
	this.pos = at
	this.size = len(data)
}

func (this *pop_buff) PopState() {

	el := this.states.Back()
	if el != nil {
		this.states.Remove(el)
	}
	prevState := el.Value.(BufferState)

	this.data = prevState.ref
	this.pos = prevState.pos
	this.size = prevState.size
}

func (this *pop_buff) InPushedState() bool {
	return this.states.Len() > 0
}
