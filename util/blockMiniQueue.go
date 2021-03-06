package util

import (
	"container/list"
	"github.com/eris-ltd/decerver-interfaces/modules"
	"sync"
)

// A concurrent queue implementation for *BlockMiniData objects. This is not a
// performance bottleneck, so working with a list is fine.
type BlockMiniQueue struct {
	mutex *sync.Mutex
	queue *list.List
}

func NewBlockMiniQueue() *BlockMiniQueue {
	bmq := &BlockMiniQueue{}
	bmq.queue = list.New()
	bmq.mutex = &sync.Mutex{}
	return bmq
}

func (bmq *BlockMiniQueue) Pop() *modules.BlockMini {
	bmq.mutex.Lock()
	val := bmq.queue.Front()
	bmq.queue.Remove(val)
	num, _ := val.Value.(*modules.BlockMini)
	bmq.mutex.Unlock()
	return num
}

func (bmq *BlockMiniQueue) Push(bmd *modules.BlockMini) {
	bmq.mutex.Lock()
	bmq.queue.PushBack(bmd)
	bmq.mutex.Unlock()
}

func (bmq *BlockMiniQueue) IsEmpty() bool {
	return bmq.queue.Len() == 0
}
