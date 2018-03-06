package list

import (
	"container/list"
	"sync"
)

type List struct {
	mu   sync.RWMutex
	list *list.List
}

func New() *List {
	return &List{list: list.New()}
}

func (obj *List) PushFront(v interface{}) *list.Element {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	return obj.list.PushFront(v)
}

func (obj *List) PushBack(v interface{}) *list.Element {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	return obj.list.PushBack(v)
}

func (obj *List) InsertAfter(v interface{}, mark *list.Element) *list.Element {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	return obj.list.InsertAfter(v, mark)
}

func (obj *List) InsertBefore(v interface{}, mark *list.Element) *list.Element {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	return obj.list.InsertBefore(v, mark)
}

func (obj *List) BatchPushFront(vs []interface{}) {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	for _, item := range vs {
		obj.list.PushFront(item)
	}
}

func (obj *List) PopBack() interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	if elem := obj.list.Back(); elem != nil {
		item := obj.list.Remove(elem)
		return item
	}

	return nil
}

func (obj *List) PopFront() interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	if elem := obj.list.Front(); elem != nil {
		item := obj.list.Remove(elem)
		return item
	}

	return nil
}

func (obj *List) BatchPopBack(max int) []interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	count := obj.list.Len()
	if count == 0 {
		return []interface{}{}
	}

	if count > max {
		count = max
	}

	items := make([]interface{}, count)
	for i := 0; i < count; i++ {
		items[i] = obj.list.Remove(obj.list.Back())
	}

	return items
}

func (obj *List) BatchPopFront(max int) []interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	count := obj.list.Len()
	if count == 0 {
		return []interface{}{}
	}

	if count > max {
		count = max
	}
	items := make([]interface{}, count)
	for i := 0; i < count; i++ {
		items[i] = obj.list.Remove(obj.list.Front())
	}

	return items
}

func (obj *List) PopBackAll() []interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	count := obj.list.Len()
	if count == 0 {
		return []interface{}{}
	}

	items := make([]interface{}, count)
	for i := 0; i < count; i++ {
		items[i] = obj.list.Remove(obj.list.Back())
	}

	return items
}

func (obj *List) PopFrontAll() []interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	count := obj.list.Len()
	if count == 0 {
		return []interface{}{}
	}

	items := make([]interface{}, count)
	for i := 0; i < count; i++ {
		items[i] = obj.list.Remove(obj.list.Front())
	}

	return items
}

func (obj *List) Remove(e *list.Element) interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	return obj.list.Remove(e)
}

func (obj *List) RemoveAll() {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	obj.list = list.New()
}

func (obj *List) FrontAll() []interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	count := obj.list.Len()
	if count == 0 {
		return []interface{}{}
	}

	items := make([]interface{}, 0, count)
	for e := obj.list.Front(); e != nil; e = e.Next() {
		items = append(items, e.Value)
	}

	return items
}

func (obj *List) BackAll() []interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	count := obj.list.Len()
	if count == 0 {
		return []interface{}{}
	}

	items := make([]interface{}, 0, count)
	for e := obj.list.Back(); e != nil; e = e.Prev() {
		items = append(items, e.Value)
	}

	return items
}

func (obj *List) FrontItem() interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	if f := obj.list.Front(); f != nil {
		return f.Value
	}

	return nil
}

func (obj *List) BackItem() interface{} {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	if f := obj.list.Back(); f != nil {
		return f.Value
	}

	return nil
}

func (obj *List) Front() *list.Element {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	return obj.list.Front()
}

func (obj *List) Back() *list.Element {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	return obj.list.Back()
}

func (obj *List) Len() int {
	obj.mu.Lock()
	defer obj.mu.Unlock()

	return obj.list.Len()
}
