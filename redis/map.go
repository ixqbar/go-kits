package redis

type SeqMap struct {
	Data map[string]interface{}
	Keys []string
}

func NewSeqMap() *SeqMap {
	return &SeqMap{
		Data: make(map[string]interface{}, 0),
		Keys: make([]string, 0),
	}
}

func (this *SeqMap) Len() int {
	return len(this.Keys)
}

func (this *SeqMap) Add(key string, value interface{}) {
	if _, ok := this.Data[key]; ok == false {
		this.Keys = append(this.Keys, key)
	}

	this.Data[key] = value
}

func (this *SeqMap) Delete(key string) {
	if _, ok := this.Data[key]; ok == false {
		return
	}

	var _keys []string
	for _, v := range this.Keys {
		if v != key {
			_keys = append(_keys, v)
		}
	}

	this.Keys = _keys
	delete(this.Data, key)
}
