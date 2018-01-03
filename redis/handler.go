package redis

type Handler interface {
	CheckShield(string) bool
}

type HashSubChannels map[string][]*ChannelWriter

type RedisHandler struct {
	Shield map[string]bool
	Config map[string]interface{}
	SubChannels HashSubChannels
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

func (obj *RedisHandler) Initiation() bool {
	if obj.SubChannels == nil {
		obj.SubChannels = make(HashSubChannels)
	}

	obj.SetShield("Init")
	obj.SetShield("Shutdown")
	obj.SetShield("Lock")
	obj.SetShield("Unlock")
	obj.SetShield("SetShield")
	obj.SetShield("SetConfig")
	obj.SetShield("CheckShield")
	obj.SetShield("Initiation")
	obj.SetShield("ClearSubscribe")

	return true
}

func (obj *RedisHandler) ClearSubscribe(name string) bool {
	if len(obj.SubChannels) == 0 {
		return false
	}

	for n,l := range obj.SubChannels {
		if len(l) == 0 {
			delete(obj.SubChannels, n)
			continue
		}

		leftChannelWriters := make([]*ChannelWriter, 0)
		for _,v := range l {
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