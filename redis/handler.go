package redis

type Handler interface {
	CheckShield(string) bool
	CallClientStateChangedFunc(client *Client, state int)
}

type HashSubChannels map[string][]*ChannelWriter
type ClientStateChangeFunc func(client *Client, state int)

type RedisHandler struct {
	Shield      map[string]bool
	Config      map[string]interface{}
	SubChannels HashSubChannels
	clientStateChangedFunc ClientStateChangeFunc
}

func (obj *RedisHandler) SetShield(name string) *RedisHandler {
	if obj.Shield == nil {
		obj.Shield = make(map[string]bool)
	}

	obj.Shield[name] = true
	return obj
}

func (obj *RedisHandler) SetConfig(config map[string]interface{}) *RedisHandler {
	if obj.Config == nil {
		obj.Config = make(map[string]interface{})
	}

	obj.Config = config
	return obj
}

func (obj *RedisHandler) CheckShield(name string) bool {
	_, ok := obj.Shield[name]
	return ok
}

func (obj *RedisHandler) CanPublish() bool {
	return len(obj.SubChannels) > 0
}

func (obj *RedisHandler) Command() error {
	return nil
}

func (obj *RedisHandler) RegisterClientStateChangedFunc(fc ClientStateChangeFunc) {
	obj.clientStateChangedFunc = fc
}

func (obj *RedisHandler) CallClientStateChangedFunc(client *Client, state int) {
	if obj.clientStateChangedFunc != nil {
		obj.clientStateChangedFunc(client, state)
	}
}

func (obj *RedisHandler) Initiation(f func()) bool {
	if obj.SubChannels == nil {
		obj.SubChannels = make(HashSubChannels)
	}

	obj.clientStateChangedFunc = nil

	obj.SetShield("Init")
	obj.SetShield("Shutdown")
	obj.SetShield("Lock")
	obj.SetShield("Unlock")
	obj.SetShield("SetShield")
	obj.SetShield("SetConfig")
	obj.SetShield("CheckShield")
	obj.SetShield("Initiation")
	obj.SetShield("ClearSubscribe")
	obj.SetShield("CanPublish")
	obj.SetShield("CallClientStateChangedFunc")
	obj.SetShield("RegisterClientStateChangedFunc")

	if f != nil {
		f()
	}

	return true
}

func (obj *RedisHandler) ClearSubscribe(name string) bool {
	if len(obj.SubChannels) == 0 {
		return false
	}

	for n, l := range obj.SubChannels {
		if len(l) == 0 {
			delete(obj.SubChannels, n)
			continue
		}

		leftChannelWriters := make([]*ChannelWriter, 0)
		for _, v := range l {
			if v.Name != name {
				leftChannelWriters = append(leftChannelWriters, v)
			}
		}

		if len(leftChannelWriters) > 0 {
			obj.SubChannels[n] = leftChannelWriters
		} else {
			delete(obj.SubChannels, n)
		}
	}

	return true
}
