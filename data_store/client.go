package data_store

import (
	"commons"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Client struct {
	*commons.HttpClient
}

func NewClient(url string) *Client {
	return &Client{HttpClient: &commons.HttpClient{Url: url}}
}

func marshalError(e error) commons.RuntimeError {
	return commons.NewApplicationError(commons.BadRequestCode, "marshal failed, "+e.Error())
}

func (self *Client) Create(target string, body map[string]interface{}) (string, commons.RuntimeError) {
	msg, e := json.Marshal(body)
	if nil != e {
		return "", marshalError(e)
	}
	_, id, err := self.CreateJson(self.CreateUrl().Concat(target).ToUrl(), msg)
	return id, err
}

func (self *Client) CreateByParent(parent_type, parent_id, target string, body map[string]interface{}) (string, commons.RuntimeError) {
	msg, e := json.Marshal(body)
	if nil != e {
		return "", marshalError(e)
	}
	_, id, err := self.CreateJson(self.CreateUrl().Concat(parent_type, parent_id, "children", target).ToUrl(), msg)
	return id, err
}

func (self *Client) SaveBy(target string, params map[string]string,
	body map[string]interface{}) (string, string, commons.RuntimeError) {
	url := self.CreateUrl().
		Concat(target).
		WithQueries(params, "@").
		WithQuery("save", "true").
		ToUrl()

	msg, e := json.Marshal(body)
	if nil != e {
		return "", "unknow", marshalError(e)
	}
	res, id, err := self.CreateJson(url, msg)
	if nil != res && res.HasOptions() {
		if !res.Options().Contains("is_created") {
			return id, "unknow", err
		}
		if res.Options().GetBoolWithDefault("is_created", false) {
			return id, "new", err
		}
		return id, "update", err
	}
	return id, "unknow", err
}

func (self *Client) CreateJson(url string, msg []byte) (commons.Result, string, commons.RuntimeError) {
	res := self.InvokeWithBytes("POST", url, msg, 201)
	if res.HasError() {
		return nil, "", res.Error()
	}

	if nil == res.LastInsertId() {
		return nil, "", commons.NewApplicationError(commons.InternalErrorCode, "lastInsertId is nil")
	}

	result := fmt.Sprint(res.LastInsertId())
	if "-1" == res.LastInsertId() {
		return nil, "", commons.NewApplicationError(commons.InternalErrorCode, "lastInsertId is -1")
	}

	return res, result, nil
}

func (self *Client) UpdateById(target, id string, body map[string]interface{}) commons.RuntimeError {
	msg, e := json.Marshal(body)
	if nil != e {
		return marshalError(e)
	}
	c, err := self.UpdateJson(self.CreateUrl().Concat(target, id).ToUrl(), msg)
	if nil != err {
		return err
	}
	if 1 != c {
		return commons.NewApplicationError(commons.InternalErrorCode, fmt.Sprintf("update row with id is '%s', effected row is %d", id, c))
	}
	return nil
}

func (self *Client) UpdateBy(target string, params map[string]string,
	body map[string]interface{}) (int64, commons.RuntimeError) {
	url := self.CreateUrl().Concat(target).WithQueries(params, "@").ToUrl()
	msg, e := json.Marshal(body)
	if nil != e {
		return -1, marshalError(e)
	}
	return self.UpdateJson(url, msg)
}

func (self *Client) UpdateJson(url string, msg []byte) (int64, commons.RuntimeError) {
	res := self.InvokeWithBytes("PUT", url, msg, 200)
	if res.HasError() {
		return -1, res.Error()
	}
	result := res.Effected()
	if -1 == result {
		return -1, commons.NewApplicationError(commons.InternalErrorCode, "effected rows is -1")
	}
	return result, nil
}

func (self *Client) DeleteById(target, id string) commons.RuntimeError {
	res := self.InvokeWithBytes("DELETE", self.CreateUrl().Concat(target, id).ToUrl(), nil, 200)
	return res.Error()
}

func (self *Client) DeleteBy(target string, params map[string]string) (int64, commons.RuntimeError) {
	url := self.CreateUrl().Concat(target).WithQueries(params, "@").ToUrl()
	res := self.InvokeWithBytes("DELETE", url, nil, 200)
	if res.HasError() {
		return -1, res.Error()
	}
	result := res.Effected()
	if -1 == result {
		return -1, commons.NewApplicationError(commons.InternalErrorCode, "effected rows is -1")
	}
	return result, nil
}

func (self *Client) Count(target string, params map[string]string) (int64, commons.RuntimeError) {
	url := self.CreateUrl().Concat(target, "@count").WithQueries(params, "@").ToUrl()
	res := self.InvokeWithBytes("GET", url, nil, 200)
	if res.HasError() {
		return -1, res.Error()
	}

	result, err := res.Value().AsInt64()
	if nil != err {
		return -1, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
	}
	return result, nil
}

func (self *Client) FindById(target, id string) (map[string]interface{},
	commons.RuntimeError) {
	return self.FindByIdWithIncludes(target, id, "")
}

type RecordVersion struct {
	Id        int       `json:"id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

func GetRecordVersionFrom(values map[string]interface{}) (*RecordVersion, error) {
	t1 := values["created_at"]
	t2 := values["updated_at"]
	if nil == t1 && nil == t2 {
		return nil, nil
	}
	var e error
	version := &RecordVersion{}
	version.CreatedAt, e = commons.AsTime(t1)
	if nil != e {
		return nil, fmt.Errorf("get 'created_at' failed, %v", e)
	}
	version.UpdatedAt, e = commons.AsTime(t2)
	if nil != e {
		return nil, fmt.Errorf("get 'updated_at' failed, %v", e)
	}
	return version, nil
}

func Diff(new_snapshots, old_snapshots map[string]*RecordVersion) (newed, updated, deleted []string) {
	//newed = make([]string, 0, len(old_snapshots))
	//updated = make([]string, 0, len(old_snapshots))
	//deleted = make([]string, 0, len(old_snapshots))
	for id, version := range new_snapshots {
		old_version, ok := old_snapshots[id]
		if !ok {
			//fmt.Println("not found, skip", id)
			newed = append(newed, id)
			continue
		}

		delete(old_snapshots, id)
		if nil == old_version {
			//fmt.Println("old version is nil, reload", id)
			//delete(c.objects, id)
			newed = append(newed, id)
			continue
		}
		if nil == version {
			//fmt.Println("version is nil, skip", id)
			continue
		}

		if version.UpdatedAt.After(old_version.UpdatedAt) {
			//fmt.Println("after, reload", id)
			updated = append(updated, id)
		} // else {
		//	fmt.Println("not after, skip", id, version.UpdatedAt, old_version.UpdatedAt)
		//}
	}

	for id, _ := range old_snapshots {
		deleted = append(deleted, id)
		//delete(c.objects, id)
		// fmt.Println("delete", id)
	}

	return
}

type SnapshotResult struct {
	Eid             interface{}               `json:"request_id,omitempty"`
	Eerror          *commons.ApplicationError `json:"error,omitempty"`
	Ewarnings       interface{}               `json:"warnings,omitempty"`
	Evalue          []*RecordVersion          `json:"value,omitempty"`
	Eeffected       int64                     `json:"effected,omitempty"`
	ElastInsertId   interface{}               `json:"lastInsertId,omitempty"`
	Eoptions        map[string]interface{}    `json:"options,omitempty"`
	Ecreated_at     time.Time                 `json:"created_at,omitempty"`
	Erepresentation string                    `json:"representation,omitempty"`
}

var SnapshotResultIsNil = commons.NewApplicationError(commons.InternalErrorCode, "snapshot result is nil")

func Snapshot(url string, params map[string]string, cached_snapshots []*RecordVersion) (map[string]*RecordVersion,
	commons.RuntimeError) {

	var raw_results SnapshotResult
	raw_results.Evalue = cached_snapshots
	e := commons.InvokeWeb("GET", url, nil, 200, &raw_results)
	if nil != e {
		return nil, e
	}
	if nil != raw_results.Eerror && (0 != raw_results.Eerror.Ecode || 0 != len(raw_results.Eerror.Emessage)) {
		return nil, raw_results.Eerror
	}

	if nil == raw_results.Evalue {
		return nil, SnapshotResultIsNil
	}

	snapshots := make(map[string]*RecordVersion)
	for _, res := range raw_results.Evalue {
		snapshots[fmt.Sprint(res.Id)] = res
		// id := res["id"]
		// if nil == id {
		// 	return nil, commons.NewApplicationError(commons.InternalErrorCode, fmt.Sprintf("'id' is nil in the results[%v]", i))
		// }
		// snapshot, err := GetRecordVersionFrom(res)
		// if nil != err {
		// 	return nil, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
		// }
		// snapshots[fmt.Sprint(id)] = snapshot
	}
	return snapshots, nil
}

func (self *Client) Snapshot(target string, params map[string]string, cached_snapshots []*RecordVersion) (map[string]*RecordVersion,
	commons.RuntimeError) {
	url := self.CreateUrl().
		Concat(target, "@snapshot").
		WithQueries(params, "@").ToUrl()
	return Snapshot(url, params, cached_snapshots)

	// raw_results := self.InvokeWithBytes("GET", url, nil, 200)
	// if raw_results.HasError() {
	// 	return nil, raw_results.Error()
	// }

	// results, err := raw_results.Value().AsObjects()
	// if nil != err {
	// 	return nil, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
	// }

	// snapshots := make(map[string]*RecordVersion)
	// for i, res := range results {
	// 	id := res["id"]
	// 	if nil == id {
	// 		return nil, commons.NewApplicationError(commons.InternalErrorCode, fmt.Sprintf("'id' is nil in the results[%v]", i))
	// 	}
	// 	snapshot, err := GetRecordVersionFrom(res)
	// 	if nil != err {
	// 		return nil, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
	// 	}
	// 	snapshots[fmt.Sprint(id)] = snapshot
	// }
	// return snapshots, nil
}

func (self *Client) FindBy(target string, params map[string]string) ([]map[string]interface{},
	commons.RuntimeError) {
	return self.FindByWithIncludes(target, params, "")
}

func (self *Client) FindByWithIncludes(target string, params map[string]string, includes string) (
	[]map[string]interface{}, commons.RuntimeError) {
	url := self.CreateUrl().
		Concat(target).
		WithQueries(params, "@")
	if 0 != len(includes) {
		url.WithQuery("includes", includes)
	}
	res := self.InvokeWithBytes("GET", url.ToUrl(), nil, 200)
	if res.HasError() {
		return nil, res.Error()
	}
	result, err := res.Value().AsObjects()
	if nil != err {
		return nil, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
	}
	return result, nil
}

func (self *Client) EachByMultIdWithIncludes(target string, id_list []string, includes string, cb func(map[string]interface{})) error {
	offset := 0
	for ; (offset + 100) < len(id_list); offset += 100 {
		results, e := self.FindByMultIdWithIncludes(target, id_list[offset:offset+100], includes)
		if nil != e {
			return e
		}
		for _, res := range results {
			cb(res)
		}
	}

	results, e := self.FindByMultIdWithIncludes(target, id_list[offset:], includes)
	if nil != e {
		return e
	}
	for _, res := range results {
		cb(res)
	}
	return nil
}

func (self *Client) FindByMultIdWithIncludes(target string, id_list []string, includes string) (
	[]map[string]interface{}, commons.RuntimeError) {
	url := self.CreateUrl().
		Concat(target).
		WithQuery("@id", "[in]"+strings.Join(id_list, ","))
	if 0 != len(includes) {
		url.WithQuery("includes", includes)
	}
	res := self.InvokeWithBytes("GET", url.ToUrl(), nil, 200)
	if res.HasError() {
		return nil, res.Error()
	}
	result, err := res.Value().AsObjects()
	if nil != err {
		return nil, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
	}
	return result, nil
}

func (self *Client) FindByIdWithIncludes(target, id string, includes string) (
	map[string]interface{}, commons.RuntimeError) {
	url := self.CreateUrl().Concat(target, id)
	if 0 != len(includes) {
		url.WithQuery("includes", includes)
	}
	res := self.InvokeWithBytes("GET", url.ToUrl(), nil, 200)
	if res.HasError() {
		return nil, res.Error()
	}
	result, err := res.Value().AsObject()
	if nil != err {
		return nil, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
	}
	//fmt.Printf("res = %#v\r\n\r\n", res)
	//fmt.Printf("value = %#v\r\n\r\n", res.InterfaceValue())
	return result, nil
}

func (self *Client) Children(parent, parent_id, target string, params map[string]string) ([]map[string]interface{},
	commons.RuntimeError) {
	url := self.CreateUrl().
		Concat(parent, parent_id, "children", target).
		WithQueries(params, "@").ToUrl()
	res := self.InvokeWithBytes("GET", url, nil, 200)
	if res.HasError() {
		return nil, res.Error()
	}
	result, err := res.Value().AsObjects()
	if nil != err {
		return nil, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
	}
	return result, nil
}

func (self *Client) Parent(child, child_id, target string) (map[string]interface{},
	commons.RuntimeError) {
	url := self.CreateUrl().
		Concat(child, child_id, "parent", target).ToUrl()
	res := self.InvokeWithBytes("GET", url, nil, 200)
	if res.HasError() {
		return nil, res.Error()
	}
	result, err := res.Value().AsObject()
	if nil != err {
		return nil, commons.NewApplicationError(commons.InternalErrorCode, err.Error())
	}
	return result, nil
}

func GetChildrenForm(instance interface{}, matchers map[string]commons.Matcher) []map[string]interface{} {
	if nil == instance {
		return nil
	}
	if result, ok := instance.(map[string]interface{}); ok {
		if nil == matchers || commons.IsMatch(result, matchers) {
			return []map[string]interface{}{result}
		}
		return nil
	}

	var results []map[string]interface{} = nil
	if values, ok := instance.([]interface{}); ok {
		for _, v := range values {
			if result, ok := v.(map[string]interface{}); ok {
				if nil == matchers || commons.IsMatch(result, matchers) {
					results = append(results, result)
				}
			}
		}
		return results
	}

	if values, ok := instance.([]map[string]interface{}); ok {
		if nil == matchers {
			return values
		}

		for _, result := range values {
			if commons.IsMatch(result, matchers) {
				results = append(results, result)
			}
		}
	}
	return nil
}
