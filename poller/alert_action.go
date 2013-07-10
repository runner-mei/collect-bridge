package poller

import (
	"commons"
	"encoding/json"
	"errors"
	"time"
)

const MAX_REPEATED = 9999990

var reset_error = errors.New("please reset channel.")

type alertAction struct {
	name         string
	max_repeated int

	options     map[string]interface{}
	result      map[string]interface{}
	channel     chan<- *data_object
	cached_data *data_object

	checker      Checker
	last_status  int
	repeated     int
	already_send bool
}

func (self *alertAction) Run(t time.Time, value interface{}) error {
	current, err := self.checker.Run(value, self.result)
	if nil != err {
		return err
	}

	if current == self.last_status {
		self.repeated++

		if self.repeated >= 9999996 || self.repeated < 0 { // inhebit overflow
			self.repeated = self.max_repeated + 10
		}
	} else {
		self.repeated = 1
		self.last_status = current
		self.already_send = false
	}

	if self.repeated < self.max_repeated {
		return nil
	}

	if self.already_send {
		return nil
	}

	evt := map[string]interface{}{}
	for k, v := range self.result {
		evt[k] = v
	}
	if nil != self.options {
		for k, v := range self.options {
			evt[k] = v
		}
	}

	if _, found := evt["triggered_at"]; !found {
		evt["triggered_at"] = t
	}

	if _, found := evt["current_value"]; !found {
		evt["current_value"] = value
	}

	evt["status"] = current

	err = self.send(evt)
	if nil == err {
		self.already_send = true
		return nil
	}

	if err == reset_error {
		self.cached_data = &data_object{c: make(chan error, 2)}
	}
	return err
}

func (self *alertAction) send(evt map[string]interface{}) error {
	self.cached_data.attributes = evt
	self.channel <- self.cached_data
	return <-self.cached_data.c
}

var (
	ExpressionStyleIsRequired    = commons.IsRequired("expression_style")
	ExpressionCodeIsRequired     = commons.IsRequired("expression_code")
	NotificationChannelIsNil     = errors.New("'alerts_channel' is nil")
	NotificationChannelTypeError = errors.New("'alerts_channel' is not a chan<- *data_object ")
)

func newAlertAction(attributes, options, ctx map[string]interface{}) (ExecuteAction, error) {
	name, e := commons.GetString(attributes, "name")
	if nil != e {
		return nil, NameIsRequired
	}

	c := ctx["alerts_channel"]
	if nil == c {
		return nil, NotificationChannelIsNil
	}
	channel, ok := c.(chan<- *data_object)
	if !ok {
		return nil, NotificationChannelTypeError
	}

	checker, e := makeChecker(attributes, ctx)
	if nil != e {
		return nil, e
	}

	max_repeated := commons.GetIntWithDefault(attributes, "max_repeated", 1)
	if max_repeated <= 0 {
		max_repeated = 1
	}

	if max_repeated >= MAX_REPEATED {
		max_repeated = MAX_REPEATED - 20
	}

	return &alertAction{name: name,
		//description: commons.GetString(attributes, "description", ""),
		already_send: false,
		options:      options,
		max_repeated: max_repeated,
		result:       map[string]interface{}{"name": name},
		channel:      channel,
		cached_data:  &data_object{c: make(chan error, 2)},
		checker:      checker}, nil
}

func makeChecker(attributes, ctx map[string]interface{}) (Checker, error) {
	style, e := commons.GetString(attributes, "expression_style")
	if nil != e {
		return nil, ExpressionStyleIsRequired
	}

	code, e := commons.GetString(attributes, "expression_code")
	if nil != e {
		codeObject, e := commons.GetObject(attributes, "expression_code")
		if nil != e {
			return nil, ExpressionCodeIsRequired
		}

		codeBytes, e := json.Marshal(codeObject)
		if nil != e {
			return nil, ExpressionCodeIsRequired
		}

		code = string(codeBytes)
	}

	switch style {
	case "json":
		return makeJsonChecker(code)
	}
	return nil, errors.New("expression style '" + style + "' is unknown")
}
