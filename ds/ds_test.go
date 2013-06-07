package ds

import (
	"commons/as"
	"commons/types"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
)

var id_name = "id"

func atoi(s string) int {
	i, e := strconv.Atoi(s)
	if nil != e {
		panic(e.Error())
	}
	return i
}

func readAll(r io.Reader) string {
	bs, e := ioutil.ReadAll(r)
	if nil != e {
		panic(e.Error())
	}
	return string(bs)
}

func createMockMetricRule2(t *testing.T, client *Client, factor string) string {
	return createJson(t, client, "metric_trigger", fmt.Sprintf(`{"name":"%s", "expression":"d%s", "metric":"2%s"}`, factor, factor, factor))
}
func createMockMetricRule(t *testing.T, client *Client, id, factor string) string {
	return createJson(t, client, "metric_trigger", fmt.Sprintf(`{"name":"%s", "expression":"d%s", "metric":"2%s", "managed_object_id":"%s"}`, factor, factor, factor, id))
}

func createMockInterface(t *testing.T, client *Client, id, factor string) string {
	return createJson(t, client, "interface", fmt.Sprintf(`{"name":"if-%s", "ifIndex":%s, "ifDescr":"d%s", "ifType":2%s, "ifMtu":3%s, "ifSpeed":4%s, "device_id":"%s"}`, factor, factor, factor, factor, factor, factor, id))
}

func createMockDevice(t *testing.T, client *Client, factor string) string {
	return createJson(t, client, "device", fmt.Sprintf(`{"name":"dd%s", "type":"switch", "address":"192.168.1.%s", "catalog":%s, "services":2%s, "managed_address":"20.0.8.110"}`, factor, factor, factor, factor))
}

func updateMockDevice(t *testing.T, client *Client, id, factor string) {
	updateJson(t, client, "device", id, fmt.Sprintf(`{"name":"dd%s", "catalog":%s, "services":2%s}`, factor, factor, factor))
}

func getDeviceById(t *testing.T, client *Client, id string) map[string]interface{} {
	return findById(t, client, "device", id)
}

func deviceNotExistsById(t *testing.T, client *Client, id string) {
	if existsById(t, client, "device", id) {
		t.Error("device '" + id + "' is exists")
		t.FailNow()
	}
}

func getDeviceByName(t *testing.T, client *Client, factor string) map[string]interface{} {
	res := findBy(t, client, "device", map[string]string{"name": "dd" + factor})
	if 1 > len(res) {
		return nil
	}
	return res[0]
}

func searchBy(res []map[string]interface{}, f func(r map[string]interface{}) bool) map[string]interface{} {
	for _, r := range res {
		if f(r) {
			return r
		}
	}
	return nil
}

func fetchInt(params map[string]interface{}, key string) int {

	v := params[key]
	if nil == v {
		panic(fmt.Sprintf("value of '"+key+"' is nil in %v", params))
	}
	return int(v.(float64))
}
func validMockDeviceWithId(t *testing.T, factor string, drv map[string]interface{}, id string) {
	if nil == drv[id_name] {
		t.Errorf("excepted id is '%s', actual id is 'nil'", id)
		return
	}
	i, e := as.AsInt(drv[id_name])
	if nil != e {
		t.Errorf("excepted id is a number, actual id is [%T]'%v'", drv[id_name], drv[id_name])
		return
	}
	if id != fmt.Sprint(i) {
		t.Errorf("excepted id is '%s', actual id is '%v'", id, drv[id_name])
		return
	}
}

func validMockDevice(t *testing.T, client *Client, factor string, drv map[string]interface{}) {
	if nil == drv["name"] {
		t.Errorf("excepted name is 'dd%s', actual name is 'nil'", factor)
		return
	}

	if "dd"+factor != drv["name"].(string) {
		t.Errorf("excepted name is 'dd%s', actual name is '%v'", factor, drv["name"])
		return
	}
	if atoi(factor) != fetchInt(drv, "catalog") {
		t.Errorf("excepted catalog is '%s', actual catalog is '%v'", factor, drv["catalog"])
		return
	}
	if atoi("2"+factor) != fetchInt(drv, "services") {
		t.Errorf("excepted services is '2%s', actual services is '%v'", factor, drv["services"])
		return
	}
}
func create(t *testing.T, client *Client, target string, body map[string]interface{}) string {
	id, e := client.Create(target, body)
	if nil != e {
		t.Errorf("create %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
	return id
}

func createJson(t *testing.T, client *Client, target, msg string) string {
	id, e := client.CreateJson("http://127.0.0.1:7071/"+target, []byte(msg))
	if nil != e {
		t.Errorf("create %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
	return id
}

func updateById(t *testing.T, client *Client, target, id string, body map[string]interface{}) {
	e := client.UpdateById(target, id, body)
	if nil != e {
		t.Errorf("update %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
}

func updateBy(t *testing.T, client *Client, target string, params map[string]string, body map[string]interface{}) {
	_, e := client.UpdateBy(target, params, body)
	if nil != e {
		t.Errorf("update %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
}

// func update(t, id string, body map[string]interface{}) error {
//	msg, e := json.Marshal(body)
//	if nil != e {
//		return fmt.Errorf("update %s failed, %v", t, e)
//	}
//	return updateJson(t, id, string(msg))
// }

func HttpPut(endpoint string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("PUT", endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return http.DefaultClient.Do(req)
}
func httpDelete(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func updateJson(t *testing.T, client *Client, target, id, msg string) {
	_, e := client.UpdateJson("http://127.0.0.1:7071/"+target+"/"+id, []byte(msg))
	if nil != e {
		t.Errorf("update %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
}

func existsById(t *testing.T, client *Client, target, id string) bool {
	_, e := client.FindByIdWithIncludes(target, id, "")
	if nil != e {
		if 404 == e.Code() {
			return false
		}
		t.Errorf("find %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
	return true
}

func findById(t *testing.T, client *Client, target, id string) map[string]interface{} {
	return findByIdWithIncludes(t, client, target, id, "")
}

func findByIdWithIncludes(t *testing.T, client *Client, target, id string, includes string) map[string]interface{} {
	res, e := client.FindByIdWithIncludes(target, id, includes)
	if nil != e {
		t.Errorf("find %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
	return res
}

func deleteByIdWhileNotExist(t *testing.T, client *Client, target, id string) {
	e := client.DeleteById(target, id)
	if nil == e {
		t.Errorf("delete not exist %s failed", target)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
	if 404 != e.Code() {
		t.Errorf("delete not exist %s failed, %v", target, e)
		t.FailNow()
	}
}

func deleteById(t *testing.T, client *Client, target, id string) {
	e := client.DeleteById(target, id)
	if nil != e {
		t.Errorf("delete %s with id is '%v' failed, %v", target, id, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
}

func deleteBy(t *testing.T, client *Client, target string, params map[string]string) {
	_, e := client.DeleteBy(target, params)
	if nil != e {
		t.Errorf("delete %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
}

func count(t *testing.T, client *Client, target string, params map[string]string) int64 {
	i, e := client.Count(target, params)
	if nil != e {
		t.Errorf("count %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
	return i
}

func findOne(t *testing.T, client *Client, target string, params map[string]string) map[string]interface{} {
	return findOneByQueryWithIncludes(t, client, target, params, "")
}

func findOneByQueryWithIncludes(t *testing.T, client *Client, target string, params map[string]string, includes string) map[string]interface{} {
	res := findByQueryWithIncludes(t, client, target, params, includes)
	if 0 == len(res) {
		t.Errorf("find %s failed, result is empty", target)
		t.FailNow()
	}
	return res[0]
}

func ExistsInChilren(t *testing.T, attrs map[string]interface{}, target string, params map[string]string) bool {
	a := attrs["$"+target]
	if nil == a {
		return false
	}
	aa, ok := a.([]interface{})
	if !ok {
		t.Error("$" + target + " is not a []interface{} type")
		t.FailNow()
	}
	for _, r := range aa {
		res, ok := r.(map[string]interface{})
		if !ok {
			t.Error("$" + target + " is not a map[string]interface{} type")
			t.FailNow()
		}
		found := true
		for k, v := range params {
			if v != fmt.Sprint(res[k]) {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}

	return false
}

func findOneFrom(t *testing.T, attrs map[string]interface{}, target string, params map[string]string) map[string]interface{} {
	a := attrs["$"+target]
	if nil == a {
		t.Error("$" + target + " is not found")
		t.FailNow()
	}
	aa, ok := a.([]interface{})
	if !ok {
		t.Error("$" + target + " is not a []interface{} type")
		t.FailNow()
	}
	for _, r := range aa {
		res, ok := r.(map[string]interface{})
		if !ok {
			t.Error("$" + target + " is not a map[string]interface{} type")
			t.FailNow()
		}
		found := true
		for k, v := range params {
			if v != fmt.Sprint(res[k]) {
				found = false
				break
			}
		}
		if found {
			return res
		}
	}

	t.Error("$" + target + " is not found with the speacfic params")
	t.FailNow()
	return nil
}

func findBy(t *testing.T, client *Client, target string, params map[string]string) []map[string]interface{} {
	return findByQueryWithIncludes(t, client, target, params, "")
}

func findByQueryWithIncludes(t *testing.T, client *Client,
	target string, params map[string]string,
	includes string) []map[string]interface{} {
	results, e := client.FindByWithIncludes(target, params, includes)
	if nil != e {
		t.Errorf("find %s failed, %v", target, e)
		t.FailNow()
	}
	return results
}

func findByChild(t *testing.T, client *Client, target, child, child_id string) map[string]interface{} {
	res, e := client.Parent(child, child_id, target)
	if nil != e {
		t.Errorf("find %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
	return res
}

func findByParent(t *testing.T, client *Client, parent, parent_id,
	target string, params map[string]string) []map[string]interface{} {
	res, e := client.Children(parent, parent_id, target, params)
	if nil != e {
		t.Errorf("find %s failed, %v", target, e)
		t.FailNow()
	}
	if nil != client.Warnings {
		t.Error(client.Warnings)
	}
	return res
}

func checkMetricRuleCount(t *testing.T, client *Client, id1, id2, id3, id4 string, all, d1, d2, d3, d4 int64) {
	tName := "metric_trigger"
	if c := count(t, client, tName, map[string]string{}); all != c {
		t.Errorf("%d != len(all.rules), actual is %d", all, c)
	}
	if c := count(t, client, tName, map[string]string{"managed_object_id": id1}); d1 != c {
		t.Errorf("%d != len(d1.rules), actual is %d", d1, c)
	}
	if c := count(t, client, tName, map[string]string{"managed_object_id": id2}); d2 != c {
		t.Errorf("%d != len(d2.rules), actual is %d", d2, c)
	}
	if c := count(t, client, tName, map[string]string{"managed_object_id": id3}); d3 != c {
		t.Errorf("%d != len(d3.rules), actual is %d", d3, c)
	}
	if c := count(t, client, tName, map[string]string{"managed_object_id": id4}); d4 != c {
		t.Errorf("%d != len(d4.rules), actual is %d", d4, c)
	}
}
func checkInterfaceCount(t *testing.T, client *Client, id1, id2, id3, id4 string, all, d1, d2, d3, d4 int64) {
	checkCount(t, client, "device_id", "interface", id1, id2, id3, id4, all, d1, d2, d3, d4)
}
func checkCount(t *testing.T, client *Client, field, tName, id1, id2, id3, id4 string, all, d1, d2, d3, d4 int64) {
	if c := count(t, client, tName, map[string]string{}); all != c {
		t.Errorf("%d != len(all.interfaces), actual is %d", all, c)
	}
	if c := count(t, client, tName, map[string]string{field: id1}); d1 != c {
		t.Errorf("%d != len(d1.interfaces), actual is %d", d1, c)
	}
	if c := count(t, client, tName, map[string]string{field: id2}); d2 != c {
		t.Errorf("%d != len(d2.interfaces), actual is %d", d2, c)
	}
	if c := count(t, client, tName, map[string]string{field: id3}); d3 != c {
		t.Errorf("%d != len(d3.interfaces), actual is %d", d3, c)
	}
	if c := count(t, client, tName, map[string]string{field: id4}); d4 != c {
		t.Errorf("%d != len(d4.interfaces), actual is %d", d4, c)
	}
}

func initData(t *testing.T, client *Client) []string {

	deleteBy(t, client, "device", map[string]string{})
	deleteBy(t, client, "interface", map[string]string{})
	deleteBy(t, client, "trigger", map[string]string{})

	id1 := createMockDevice(t, client, "1")
	id2 := createMockDevice(t, client, "2")
	id3 := createMockDevice(t, client, "3")
	id4 := createMockDevice(t, client, "4")

	createMockMetricRule2(t, client, "s")
	createMockInterface(t, client, id1, "10001")
	createMockInterface(t, client, id1, "10002")
	createMockInterface(t, client, id1, "10003")
	createMockInterface(t, client, id1, "10004")
	createMockMetricRule(t, client, id1, "10001")
	createMockMetricRule(t, client, id1, "10002")
	createMockMetricRule(t, client, id1, "10003")
	createMockMetricRule(t, client, id1, "10004")

	createMockInterface(t, client, id2, "20001")
	createMockInterface(t, client, id2, "20002")
	createMockInterface(t, client, id2, "20003")
	createMockInterface(t, client, id2, "20004")
	createMockMetricRule(t, client, id2, "20001")
	createMockMetricRule(t, client, id2, "20002")
	createMockMetricRule(t, client, id2, "20003")
	createMockMetricRule(t, client, id2, "20004")

	createMockInterface(t, client, id3, "30001")
	createMockInterface(t, client, id3, "30002")
	createMockInterface(t, client, id3, "30003")
	createMockInterface(t, client, id3, "30004")
	createMockMetricRule(t, client, id3, "30001")
	createMockMetricRule(t, client, id3, "30002")
	createMockMetricRule(t, client, id3, "30003")
	createMockMetricRule(t, client, id3, "30004")

	createMockInterface(t, client, id4, "40001")
	createMockInterface(t, client, id4, "40002")
	createMockInterface(t, client, id4, "40003")
	createMockInterface(t, client, id4, "40004")
	createMockMetricRule(t, client, id4, "40001")
	createMockMetricRule(t, client, id4, "40002")
	createMockMetricRule(t, client, id4, "40003")
	createMockMetricRule(t, client, id4, "40004")
	return []string{id1, id2, id3, id4}
}

func TestDeviceDeleteCascadeAll(t *testing.T) {
	srvTest2(t, "etc/mj_models.xml", func(client *Client, definitions *types.TableDefinitions) {

		deleteBy(t, client, "device", map[string]string{})
		deleteBy(t, client, "interface", map[string]string{})

		id1 := createMockDevice(t, client, "1")
		id2 := createMockDevice(t, client, "2")
		id3 := createMockDevice(t, client, "3")
		id4 := createMockDevice(t, client, "4")
		if "" == id1 {
			return
		}

		createMockInterface(t, client, id1, "10001")
		createMockInterface(t, client, id1, "10002")
		createMockInterface(t, client, id1, "10003")
		createMockInterface(t, client, id1, "10004")

		createMockInterface(t, client, id2, "20001")
		createMockInterface(t, client, id2, "20002")
		createMockInterface(t, client, id2, "20003")
		createMockInterface(t, client, id2, "20004")

		createMockInterface(t, client, id3, "30001")
		createMockInterface(t, client, id3, "30002")
		createMockInterface(t, client, id3, "30003")
		createMockInterface(t, client, id3, "30004")

		createMockInterface(t, client, id4, "40001")
		createMockInterface(t, client, id4, "40002")
		createMockInterface(t, client, id4, "40003")
		createMockInterface(t, client, id4, "40004")

		deleteBy(t, client, "device", map[string]string{})

		if c := count(t, client, "interface", map[string]string{}); 0 != c {
			t.Errorf("0 != len(all.interfaces), actual is %d", c)
		}
	})
}

func TestDeviceDeleteCascadeByAll(t *testing.T) {
	srvTest2(t, "etc/mj_models.xml", func(client *Client, definitions *types.TableDefinitions) {
		idlist := initData(t, client)
		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 16, 4, 4, 4, 4)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 17, 4, 4, 4, 4)
		deleteBy(t, client, "device", map[string]string{})
		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 0, 0, 0, 0, 0)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 1, 0, 0, 0, 0)
	})
}

func TestDeviceDeleteCascadeByQuery(t *testing.T) {
	srvTest2(t, "etc/mj_models.xml", func(client *Client, definitions *types.TableDefinitions) {
		idlist := initData(t, client)
		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 16, 4, 4, 4, 4)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 17, 4, 4, 4, 4)
		deleteBy(t, client, "device", map[string]string{"catalog": "[gte]3"})
		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 8, 4, 4, 0, 0)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 9, 4, 4, 0, 0)
	})
}

func TestDeviceDeleteCascadeById(t *testing.T) {
	srvTest2(t, "etc/mj_models.xml", func(client *Client, definitions *types.TableDefinitions) {

		idlist := initData(t, client)

		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 16, 4, 4, 4, 4)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 17, 4, 4, 4, 4)
		deleteById(t, client, "device", idlist[0])

		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 12, 0, 4, 4, 4)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 13, 0, 4, 4, 4)
		deleteById(t, client, "device", idlist[1])

		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 8, 0, 0, 4, 4)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 9, 0, 0, 4, 4)
		deleteById(t, client, "device", idlist[2])

		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 4, 0, 0, 0, 4)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 5, 0, 0, 0, 4)
		deleteById(t, client, "device", idlist[3])

		checkInterfaceCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 0, 0, 0, 0, 0)
		checkMetricRuleCount(t, client, idlist[0], idlist[1], idlist[2], idlist[3], 1, 0, 0, 0, 0)
	})
}

func TestDeviceCURD(t *testing.T) {
	srvTest2(t, "etc/mj_models.xml", func(client *Client, definitions *types.TableDefinitions) {
		deleteBy(t, client, "device", map[string]string{})
		deleteBy(t, client, "interface", map[string]string{})
		deleteBy(t, client, "trigger", map[string]string{})

		id1 := createMockDevice(t, client, "1")
		id2 := createMockDevice(t, client, "2")
		id3 := createMockDevice(t, client, "3")
		id4 := createMockDevice(t, client, "4")
		if "" == id1 {
			return
		}

		d1 := getDeviceById(t, client, id1)
		d2 := getDeviceById(t, client, id2)
		d3 := getDeviceById(t, client, id3)
		d4 := getDeviceById(t, client, id4)

		if nil == d1 {
			return
		}

		validMockDeviceWithId(t, "1", d1, id1)
		validMockDeviceWithId(t, "2", d2, id2)
		validMockDeviceWithId(t, "3", d3, id3)
		validMockDeviceWithId(t, "4", d4, id4)

		updateMockDevice(t, client, id1, "11")
		updateMockDevice(t, client, id2, "21")
		updateMockDevice(t, client, id3, "31")
		updateMockDevice(t, client, id4, "41")

		d1 = getDeviceById(t, client, id1)
		d2 = getDeviceById(t, client, id2)
		d3 = getDeviceById(t, client, id3)
		d4 = getDeviceById(t, client, id4)

		if nil == d1 {
			return
		}

		validMockDevice(t, client, "11", d1)
		validMockDevice(t, client, "21", d2)
		validMockDevice(t, client, "31", d3)
		validMockDevice(t, client, "41", d4)

		d1 = getDeviceByName(t, client, "11")
		d2 = getDeviceByName(t, client, "21")
		d3 = getDeviceByName(t, client, "31")
		d4 = getDeviceByName(t, client, "41")

		if nil == d1 {
			return
		}

		validMockDevice(t, client, "11", d1)
		validMockDevice(t, client, "21", d2)
		validMockDevice(t, client, "31", d3)
		validMockDevice(t, client, "41", d4)

		deleteBy(t, client, "device", map[string]string{})

		d1 = getDeviceByName(t, client, "11")
		d2 = getDeviceByName(t, client, "21")
		d3 = getDeviceByName(t, client, "31")
		d4 = getDeviceByName(t, client, "41")

		if nil != d1 || nil != d2 || nil != d3 || nil != d4 {
			t.Errorf("remove all failed")
		}
	})
}

func TestDeviceDeleteById(t *testing.T) {
	srvTest2(t, "etc/mj_models.xml", func(client *Client, definitions *types.TableDefinitions) {
		deleteBy(t, client, "device", map[string]string{})

		id1 := createMockDevice(t, client, "1")
		id2 := createMockDevice(t, client, "2")
		id3 := createMockDevice(t, client, "3")
		id4 := createMockDevice(t, client, "4")
		if "" == id1 {
			return
		}

		deleteById(t, client, "device", id1)
		deleteById(t, client, "device", id2)
		deleteById(t, client, "device", id3)
		deleteById(t, client, "device", id4)

		deviceNotExistsById(t, client, id1)
		deviceNotExistsById(t, client, id2)
		deviceNotExistsById(t, client, id3)
		deviceNotExistsById(t, client, id4)

		deleteByIdWhileNotExist(t, client, "device", "343467")
	})
}

func TestDeviceFindBy(t *testing.T) {
	srvTest2(t, "etc/mj_models.xml", func(client *Client, definitions *types.TableDefinitions) {
		deleteBy(t, client, "device", map[string]string{})

		id1 := createMockDevice(t, client, "1")
		id2 := createMockDevice(t, client, "2")
		id3 := createMockDevice(t, client, "3")
		id4 := createMockDevice(t, client, "4")
		if "" == id1 {
			return
		}
		if "" == id2 {
			return
		}
		if "" == id3 {
			return
		}
		if "" == id4 {
			return
		}

		res := findBy(t, client, "device", map[string]string{"catalog": "[eq]1"})
		validMockDevice(t, client, "1", res[0])

		res = findBy(t, client, "device", map[string]string{"catalog": "[lte]1"})
		validMockDevice(t, client, "1", res[0])

		res = findBy(t, client, "device", map[string]string{"catalog": "[lte]2"})
		if 2 != len(res) {
			t.Errorf("catalog <=2 failed, len(result) is %v", len(res))
			return
		}
		d1 := searchBy(res, func(r map[string]interface{}) bool { return r["name"] == "dd1" })
		d2 := searchBy(res, func(r map[string]interface{}) bool { return r["name"] == "dd2" })
		if nil == d1 {
			t.Errorf("catalog <=2 failed, result is %v", res)
			return
		}
		validMockDevice(t, client, "1", d1)
		validMockDevice(t, client, "2", d2)

		res = findBy(t, client, "device", map[string]string{"catalog": "[lt]2"})
		validMockDevice(t, client, "1", res[0])

		res = findBy(t, client, "device", map[string]string{"catalog": "[gt]3"})
		validMockDevice(t, client, "4", res[0])
		res = findBy(t, client, "device", map[string]string{"catalog": "[gte]3"})
		if 2 != len(res) {
			t.Errorf("catalog <=2 failed, len(result) is %v", len(res))
			return
		}
		d3 := searchBy(res, func(r map[string]interface{}) bool { return r["name"] == "dd3" })
		d4 := searchBy(res, func(r map[string]interface{}) bool { return r["name"] == "dd4" })
		if nil == d3 {
			t.Errorf("catalog <=2 failed, result is %v", res)
			return
		}
		validMockDevice(t, client, "3", d3)
		validMockDevice(t, client, "4", d4)

		res = findBy(t, client, "device", map[string]string{"catalog": "[ne]3"})
		if 3 != len(res) {
			t.Errorf("catalog <=3 failed, len(result) is %v", len(res))
			return
		}
		d1 = searchBy(res, func(r map[string]interface{}) bool { return r["name"] == "dd1" })
		d2 = searchBy(res, func(r map[string]interface{}) bool { return r["name"] == "dd2" })
		d4 = searchBy(res, func(r map[string]interface{}) bool { return r["name"] == "dd4" })
		if nil == d1 {
			t.Errorf("catalog <=2 failed, result is %v", res)
			return
		}
		validMockDevice(t, client, "1", d1)
		validMockDevice(t, client, "2", d2)
		validMockDevice(t, client, "4", d4)
	})
}

func srvTest2(t *testing.T, file string, cb func(client *Client, definitions *types.TableDefinitions)) {
	testBase(t, file, func(conn *sql.DB) {
		sql_str := `
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS redis_commands;
DROP TABLE IF EXISTS actions;
DROP TABLE IF EXISTS metric_triggers;
DROP TABLE IF EXISTS triggers;
DROP TABLE IF EXISTS actions;
DROP TABLE IF EXISTS wbem_params;
DROP TABLE IF EXISTS ssh_params;
DROP TABLE IF EXISTS snmp_params;
DROP TABLE IF EXISTS endpoint_params;
DROP TABLE IF EXISTS access_params;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS interfaces;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS managed_objects;
DROP TABLE IF EXISTS attributes;
DROP SEQUENCE IF EXISTS actions_seq;
DROP SEQUENCE IF EXISTS triggers_seq;
DROP SEQUENCE IF EXISTS managed_object_seq;
DROP SEQUENCE IF EXISTS attributes_seq;


CREATE SEQUENCE managed_object_seq;

CREATE TABLE managed_objects (
  id integer NOT NULL DEFAULT nextval('managed_object_seq')  PRIMARY KEY,
  name varchar(250),
  description varchar(2000),
  created_at timestamp,
  updated_at timestamp
);

CREATE SEQUENCE attributes_seq;

CREATE TABLE attributes (
  id integer NOT NULL DEFAULT nextval('attributes_seq')  PRIMARY KEY,
  managed_object_id integer,
  description varchar(2000)
);

CREATE TABLE devices (
  -- id integer NOT NULL DEFAULT nextval('managed_object_seq')  PRIMARY KEY,
  address       varchar(250),
  manufacturer  integer,
  catalog       integer,
  oid           varchar(250),
  services      integer,
  location      varchar(2000),
  CONSTRAINT devices_pkey PRIMARY KEY (id)
) INHERITS (managed_objects);


CREATE TABLE  interfaces (
  -- id integer NOT NULL DEFAULT nextval('managed_object_seq')  PRIMARY KEY,
  ifIndex integer,
  ifDescr varchar(2000),
  ifType integer,
  ifMtu integer,
  ifSpeed integer,
  ifPhysAddress varchar(50),
  device_id integer,
  CONSTRAINT interfaces_pkey PRIMARY KEY (id)
) INHERITS (managed_objects);

CREATE TABLE addresses (
  -- id integer NOT NULL DEFAULT nextval('managed_object_seq')  PRIMARY KEY,
  address varchar(50),
  ifIndex integer,
  netmask varchar(50),
  bcastAddress integer,
  reasmMaxSize integer,
  device_id integer,
  CONSTRAINT addresses_pkey PRIMARY KEY (id)
) INHERITS (managed_objects);


CREATE TABLE access_params (
  -- id integer NOT NULL DEFAULT nextval('attributes_seq')  PRIMARY KEY,
  CONSTRAINT access_params_pkey PRIMARY KEY (id)
) INHERITS (attributes);


CREATE TABLE endpoint_params (
  -- id integer NOT NULL DEFAULT nextval('attributes_seq')  PRIMARY KEY,
  address varchar(50),
  port integer,
  CONSTRAINT endpoint_params_pkey PRIMARY KEY (id)
) INHERITS (access_params);


CREATE TABLE snmp_params (
  -- id integer NOT NULL DEFAULT nextval('attributes_seq')  PRIMARY KEY,
  version varchar(50),
  community varchar(250),
  CONSTRAINT snmp_params_pkey PRIMARY KEY (id)
) INHERITS (endpoint_params);

CREATE TABLE ssh_params (
  -- id integer NOT NULL DEFAULT nextval('attributes_seq')  PRIMARY KEY,
  user_name varchar(50),
  user_password varchar(250),
  CONSTRAINT ssh_params_pkey PRIMARY KEY (id)
) INHERITS (endpoint_params);

CREATE TABLE wbem_params (
  -- id integer NOT NULL DEFAULT nextval('attributes_seq')  PRIMARY KEY,
  url varchar(2000),
  user_name varchar(50),
  user_password varchar(250),
  CONSTRAINT wbem_params_pkey PRIMARY KEY (id)
) INHERITS (access_params);


CREATE SEQUENCE triggers_seq;
CREATE TABLE triggers (
  id integer NOT NULL DEFAULT nextval('triggers_seq')  PRIMARY KEY,
  name varchar(250),
  expression varchar(250),
  attachment  varchar(2000),
  description   varchar(2000)
);

CREATE TABLE metric_triggers (
  -- id integer NOT NULL DEFAULT nextval('triggers_seq')  PRIMARY KEY,
  metric varchar(250),
  managed_object_id integer,
  CONSTRAINT metric_triggers_pkey PRIMARY KEY (id)
) INHERITS (triggers);

CREATE SEQUENCE actions_seq;
CREATE TABLE actions (
  id integer NOT NULL DEFAULT nextval('actions_seq')  PRIMARY KEY,
  name varchar(250),
  description   varchar(2000),

  parent_type varchar(250),
  parent_id integer
);

CREATE TABLE redis_commands (
  -- id integer NOT NULL DEFAULT nextval('actions_seq')  PRIMARY KEY,
  command varchar(10),
  arg0  varchar(200),
  arg1  varchar(200),
  arg2  varchar(200),
  arg3  varchar(200),
  arg4  varchar(200),
  
  CONSTRAINT redis_commands_pkey PRIMARY KEY (id)
) INHERITS (actions);


CREATE TABLE alerts (
  -- id integer NOT NULL DEFAULT nextval('actions_seq')  PRIMARY KEY,
  max_repeated  integer,
  expression_style varchar(50),
  expression_code  varchar(2000),
  
  CONSTRAINT alerts_pkey PRIMARY KEY (id)
) INHERITS (actions);

`

		_, err := conn.Exec(sql_str)
		if err != nil {
			t.Fatal(err)
			t.FailNow()
			return
		}
	}, cb)
}