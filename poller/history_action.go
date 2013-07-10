package poller

import (
	"commons"
	"errors"
	"time"
)

type historyAction struct {
	id           string
	name         string
	description  string
	metric       interface{}
	managed_id   interface{}
	managed_type interface{}
	trigger_id   interface{}
	channel      chan<- *data_object
	cached_data  *data_object
	attribute    string
}

func (self *historyAction) Run(t time.Time, value interface{}) error {

	created_at := t
	if current, ok := value.(commons.Result); ok {
		created_at = current.CreatedAt()
	}

	currentValue, e := commons.ToSimpleValue(value, self.attribute)
	if nil != e {
		return e
	}

	self.cached_data.attributes = map[string]interface{}{
		"action_id":    self.id,
		"sampling_at":  created_at,
		"metric":       self.metric,
		"managed_type": self.managed_type,
		"managed_id":   self.managed_id,
		"trigger_id":   self.trigger_id,
		"value":        currentValue}

	self.channel <- self.cached_data
	return nil
}

func newHistoryAction(attributes, options, ctx map[string]interface{}) (ExecuteAction, error) {
	id, e := commons.GetString(attributes, "id")
	if nil != e || 0 == len(id) {
		return nil, IdIsRequired
	}

	name, e := commons.GetString(attributes, "name")
	if nil != e {
		return nil, NameIsRequired
	}

	attribute, e := commons.GetString(attributes, "attribute")
	if nil != e {
		return nil, CommandIsRequired
	}

	c := ctx["histories_channel"]
	if nil == c {
		return nil, errors.New("'histories_channel' is nil")
	}
	channel, ok := c.(chan<- *data_object)
	if !ok {
		return nil, errors.New("'histories_channel' is not a chan<- *data_object")
	}

	managed_type := options["managed_type"]
	managed_id := options["managed_id"]
	triggger_id := options["trigger_id"]
	metric := options["metric"]

	return &historyAction{id: id,
		name:         name,
		description:  commons.GetStringWithDefault(attributes, "description", ""),
		channel:      channel,
		cached_data:  &data_object{},
		attribute:    attribute,
		metric:       metric,
		managed_id:   managed_id,
		managed_type: managed_type,
		trigger_id:   triggger_id}, nil
}
