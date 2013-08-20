package main

import (
	"bytes"
	"code.google.com/p/mahonia"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"unicode"
	//"unicode/utf16"
	"commons"
	"errors"
	"unicode/utf8"
)

type VB struct {
	Oid   string
	Value string
}

var (
	proxy = flag.String("proxy", "127.0.0.1:7072", "the address of proxy server, default: 127.0.0.1:7070")
	//target          = flag.String("target", "127.0.0.1,161", "the address of snmp agent, default: 127.0.0.1,161")
	community       = flag.String("community", "public", "the community of snmp agent, default: public")
	action          = flag.String("action", "walk", "the action, default: walk")
	version         = flag.String("version", "2c", "the version of snmp protocal, default: 2c")
	secret_name     = flag.String("name", "", "the name, default: \"\"")
	auth_passphrase = flag.String("auth", "", "the auth passphrase, default: \"\"")
	priv_passphrase = flag.String("priv", "", "the priv passphrase, default: \"\"")
	//started_oid     = flag.String("oid", "1.3.6", "the start oid, default: 1.3.6")
	from_charset = flag.String("charset", "GB18030", "the charset of octet string, default: GB18030")
	columns      = flag.String("columns", "", "the columns of table, default: \"\"")
	//help            = flag.Bool("h", false, "print help")

	decoder mahonia.Decoder
	out     io.Writer

	target      = "127.0.0.1,161"
	started_oid = "1.3.6"
)

type Column struct {
	Index       int
	Description string
}
type Table struct {
	Name        string
	Oid         string
	Description string
	Columns     []Column
}

var (
	tables = []Table{
		Table{Name: "system",
			Oid: "1.3.6.1.2.1.1"},
		Table{Name: "interface_snmp",
			Oid: "1.3.6.1.2.1.2.2.1"},
		Table{Name: "arp_snmp",
			Oid: "1.3.6.1.2.1.4.22.1"},
		Table{Name: "ip_snmp",
			Oid: "1.3.6.1.2.1.4.20.1"},
		Table{Name: "mac_snmp",
			Oid: "unsupported"}, // ?
		Table{Name: "route_snmp",
			Oid: "1.3.6.1.2.1.4.21.1"},
		Table{Name: "cdp_snmp",
			Oid: "1.3.6.1.4.1.9.9.23.1.2.1.1;4,6,7,12"},

		Table{Name: "HuaweiDP", // cdp of huawei
			Oid: ".1.3.6.1.4.1.2011.6.7.5.6.1",
			//".1.3.6.1.4.1.25506.8.7.5.6.1;1,2,3,|.1.3.6.1.4.1.2011.10.2.8.7.5.6.1;1,2,3,|.1.3.6.1.4.1.2011.6.7.5.6.1;1,2,3,";
			Columns: []Column{
			// DeviceMAC string   //对方设备的MAC地址
			// DevicePort string  //对方设备的连接端口，格式"FastEthernet0/1"
			// DeviceID string    //对方设备描述，格式 "S3928E-SI"
			// ifIndex int        //索引项第一项
			//			1, 2, 3,
			}},

		Table{Name: "CabletronDP", // cdp of huawei
			Oid: "1.3.6.1.4.1.52.4.1.2.19.1.3.1",
			//".1.3.6.1.4.1.25506.8.7.5.6.1;1,2,3,|.1.3.6.1.4.1.2011.10.2.8.7.5.6.1;1,2,3,|.1.3.6.1.4.1.2011.6.7.5.6.1;1,2,3,";
			Columns: []Column{
			// DeviceMAC string   //对方设备的MAC地址
			// DevicePort string  //对方设备的连接端口，格式"FastEthernet0/1"
			// DeviceID string    //对方设备描述，格式 "S3928E-SI"
			// ifIndex int        //索引项第一项
			//			2, 3, 4,
			}},
	}
)

func IsAsciiAndPrintable(bs []byte) bool {
	for _, c := range bs {
		if c >= unicode.MaxASCII {
			return false
		}

		if !unicode.IsPrint(rune(c)) {
			return false
		}
	}
	return true
}

func IsUtf8AndPrintable(bs []byte) bool {
	for 0 != len(bs) {
		c, l := utf8.DecodeRune(bs)
		if utf8.RuneError == c {
			return false
		}

		if !unicode.IsPrint(c) {
			return false
		}
		bs = bs[l:]
	}
	return true
}

func IsUtf16AndPrintable(bs []byte) bool {
	if 0 != len(bs)%2 {
		return false
	}

	for i := 0; i < len(bs); i += 2 {
		u16 := binary.LittleEndian.Uint16(bs[i:])
		if !unicode.IsPrint(rune(u16)) {
			return false
		}
	}
	return true
}

func IsUtf32AndPrintable(bs []byte) bool {
	if 0 != len(bs)%4 {
		return false
	}

	for i := 0; i < len(bs); i += 4 {
		u32 := binary.LittleEndian.Uint32(bs[i:])
		if !unicode.IsPrint(rune(u32)) {
			return false
		}
	}
	return true
}

func printValue(value string) {

	if !strings.HasPrefix(value, "[octets") {
		fmt.Println(value)
		return
	}

	bs, err := hex.DecodeString(value[8:])
	if nil != err {
		fmt.Println(value)
		return
	}

	if nil != decoder {
		fmt.Println(value)

		for 0 != len(bs) {
			c, length, status := decoder(bs)
			switch status {
			case mahonia.SUCCESS:
				if unicode.IsPrint(c) {
					out.Write(bs[0:length])
				} else {
					for i := 0; i < length; i++ {
						out.Write([]byte{'.'})
					}
				}
				bs = bs[length:]
			case mahonia.INVALID_CHAR:
				out.Write([]byte{'.'})
				bs = bs[1:]
			case mahonia.NO_ROOM:
				out.Write([]byte{'.'})
				bs = bs[0:0]
			case mahonia.STATE_ONLY:
				bs = bs[length:]
			}
		}
		out.Write([]byte{'\n'})
		return
	}

	if IsUtf8AndPrintable(bs) {
		fmt.Println(string(bs))
		return
	}

	if IsUtf16AndPrintable(bs) {
		rr := make([]rune, len(bs)/2)
		for i := 0; i < len(bs); i += 2 {
			rr[i/2] = rune(binary.LittleEndian.Uint16(bs[i:]))
		}
		fmt.Println(string(rr))
		return
	}
	if IsUtf32AndPrintable(bs) {
		rr := make([]rune, len(bs)/4)
		for i := 0; i < len(bs); i += 4 {
			rr[i/4] = rune(binary.LittleEndian.Uint32(bs[i:]))
		}
		fmt.Println(string(rr))
		return
	}
	fmt.Println(value)

	for _, c := range bs {
		if c >= unicode.MaxASCII {
			fmt.Print(".")
		} else {
			fmt.Print(string(c))
		}
	}
	fmt.Println()
}

func main() {
	flag.Parse()
	targets := flag.Args()
	if nil != targets {
		switch len(targets) {
		case 0:
		case 1:
			target = targets[0]
		case 2:
			target = targets[0]
			started_oid = targets[1]
		default:
			flag.Usage()
			fmt.Println("Examples:")
			fmt.Println("\tsnmptools 127.0.0.1")
			fmt.Println("\tsnmptools 127.0.0.1,161")
			fmt.Println("\tsnmptools 127.0.0.1 interface")
			fmt.Println("\tsnmptools 127.0.0.1 1.3.6.1.2.1.2.2.1")
			fmt.Println("\tsnmptools -action=table -columns=1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22 127.0.0.1 1.3.6.1.2.1.2.2.1")
			return
		}
	}

	//target          = flag.String("target", "127.0.0.1,161", "the address of snmp agent, default: 127.0.0.1,161")
	//started_oid     = flag.String("oid", "1.3.6", "the start oid, default: 1.3.6")

	out = os.Stdout

	if "guess" != *from_charset {
		decoder = mahonia.NewDecoder(*from_charset)
	}

	for _, t := range tables {
		if t.Name == started_oid {
			started_oid = t.Oid
			*action = "table"
			break
		}
	}

	switch started_oid {
	case "system":
		started_oid = "1.3.6.1.2.1.1"
		*action = "table"
	case "system.descr", "system.description":
		started_oid = "1.3.6.1.2.1.1.1.0"
		*action = "get"
	}
	_, e := commons.ConvertToIntList(started_oid, ".")
	if nil == e {
		switch *action {
		case "walk":
			walk()
		case "next":
			next()
		case "get":
			get()
		case "table":
			table()
		default:
			fmt.Println("unsupported action - " + *action)
		}
	} else {

		switch *action {
		case "walk":
			*action = "get"
		}
		e := metric_invoke()
		if nil != e {
			fmt.Println(e)
		}
	}
}

func get() {
	_, err := invoke("snmp_get", started_oid)
	if nil != err {
		fmt.Println(err.Error())
	}
}

func next() {
	_, err := invoke("snmp_next", started_oid)
	if nil != err {
		fmt.Println(err.Error())
	}
}

func walk() {
	var err error = nil
	oid := started_oid
	for {
		oid, err = invoke("snmp_next", oid)
		if nil != err {
			fmt.Println(err.Error())
			break
		}
	}
}
func table() {
	err := invokeTable("snmp_table", started_oid)
	if nil != err {
		fmt.Println(err.Error())
	}

	// var err error = nil
	// oid := *started_oid
	// for {
	//	oid, err = invoke("next", oid)
	//	if nil != err {
	//		fmt.Println(err.Error())
	//		break
	//	}

	//	if !strings.HasPrefix(oid, *started_oid) {
	//		break
	//	}
	// }
}

func metric_createUrl() (string, error) {
	var url string
	switch *version {
	case "2", "2c", "v2", "v2c", "1", "v1":
		url = fmt.Sprintf("http://%s/%s/%s?snmp.version=v2c&snmp.read_community=%s&charset=%s", *proxy, target, started_oid, *community, *from_charset)
	case "3", "v3":
		url = fmt.Sprintf("http://%s/%s/%s?snmp.version=3&snmp.secmodel=usm&snmp.secname=%s&charset=%s", *proxy, target, started_oid, *secret_name, *from_charset)
		if "" != *auth_passphrase {
			url = url + "&auth_pass=" + *auth_passphrase
			if "" != *priv_passphrase {
				url = url + "&priv_pass=" + *priv_passphrase
			}
		}
	default:
		return "", errors.New("version is error.")
	}
	return url, nil
}

func metric_invoke() error {
	url, err := metric_createUrl()
	if nil != err {
		return err
	}
	var resp *http.Response = nil
	switch *action {
	case "get":
		fmt.Println("Get " + url)
		resp, err = http.Get(url)
		if nil != err {
			return fmt.Errorf("get failed - " + err.Error())
		}
	// case "put":
	// 	fmt.Println("Put " + url)
	// 	resp, err = http.Put(url)
	// 	if nil != err {
	// 		return "", fmt.Errorf("get failed - " + err.Error())
	// 	}
	// case "create", "post":
	// 	fmt.Println("Get " + url)
	// 	resp, err = http.Post(url)
	// 	if nil != err {
	// 		return "", fmt.Errorf("get failed - " + err.Error())
	// 	}
	// case "delete", "post":
	// 	fmt.Println("DELETE " + url)
	// 	resp, err = http.Post(url)
	// 	if nil != err {
	// 		return "", fmt.Errorf("get failed - " + err.Error())
	// 	}
	default:
		return errors.New("unsupported action - " + *action)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return fmt.Errorf("read body failed - " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(bs))
	}

	returnObject := commons.SimpleResult{}
	decoder := json.NewDecoder(bytes.NewBuffer(bs))
	decoder.UseNumber()
	err = decoder.Decode(&returnObject)
	if nil != err {
		return errors.New("unmarshal failed - " + err.Error() + "\n" + string(bs))
	}
	value := returnObject.InterfaceValue()
	if nil == value {
		return errors.New("result is empty.\n" + string(bs))
	}
	// vbs := value.(map[string]interface{})
	// if 0 == len(vbs) {
	// 	return errors.New("result is empty." + "\n" + string(bs))
	// }

	bs, err = json.MarshalIndent(value, "", "  ")
	if nil != err {
		return errors.New("marshal failed - " + err.Error() + "\n" + string(bs))
	}
	fmt.Println(string(bs))
	return nil
}

func createUrl(action, oid string) (string, error) {
	var columns_s string = ""
	if "" != *columns && "table" == action {
		columns_s = "&snmp.columns=" + *columns
	}
	var url string
	switch *version {
	case "2", "2c", "v2", "v2c", "1", "v1":
		url = fmt.Sprintf("http://%s/%s/%s?snmp.oid=%s&snmp.version=v2c&snmp.read_community=%s%s", *proxy, target, action, strings.Replace(oid, ".", "_", -1), *community, columns_s)
	case "3", "v3":
		url = fmt.Sprintf("http://%s/%s/%s?snmp.oid=%s&snmp.version=3&snmp.secmodel=usm&snmp.secname=%s%s", *proxy, target, action, strings.Replace(oid, ".", "_", -1), *secret_name, columns_s)
		if "" != *auth_passphrase {
			url = url + "&snmp.auth_pass=" + *auth_passphrase
			if "" != *priv_passphrase {
				url = url + "&snmp.priv_pass=" + *priv_passphrase
			}
		}
	default:
		return "", errors.New("version is error.")
	}
	return url, nil
}

func invoke(action, oid string) (string, error) {
	var err error

	url, err := createUrl(action, oid)
	if nil != err {
		return "", err
	}

	fmt.Println("Get " + url)
	resp, err := http.Get(url)
	if nil != err {
		return "", fmt.Errorf("get failed - " + err.Error())
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return "", fmt.Errorf("read body failed - " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(bs))
	}

	returnObject := commons.SimpleResult{}
	decoder := json.NewDecoder(bytes.NewBuffer(bs))
	decoder.UseNumber()
	err = decoder.Decode(&returnObject)
	if nil != err {
		return "", errors.New("unmarshal failed - " + err.Error() + "\n" + string(bs))
	}
	vbs, err := returnObject.Value().AsObject()
	if nil != err {
		return "", errors.New("type failed - " + err.Error() + "\n" + string(bs))
	}

	if 0 == len(vbs) {
		return "", errors.New("result is empty." + "\n" + string(bs))
	}

	err = nil
	var next_oid string
	for key, v := range vbs {
		value := fmt.Sprint(v)
		if strings.HasPrefix(value, "[error") {
			if !strings.HasPrefix(value, "[error:11]") {
				err = fmt.Errorf("invalid value - %v", value)
			} else {
				err = errors.New("walk end.")
			}
			return "", err
		}

		next_oid = key
		printValue(value)
	}

	return next_oid, nil
}

func invokeTable(action, oid string) error {
	var err error

	url, err := createUrl(action, oid)
	if nil != err {
		return err
	}

	fmt.Println("Get " + url)
	resp, err := http.Get(url)
	if nil != err {
		return fmt.Errorf("get failed - " + err.Error())
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return fmt.Errorf("read body failed - " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(string(bs))
	}

	returnObject := commons.SimpleResult{}
	decoder := json.NewDecoder(bytes.NewBuffer(bs))
	decoder.UseNumber()
	err = decoder.Decode(&returnObject)
	if nil != err {
		return errors.New("unmarshal failed - " + err.Error() + "\n" + string(bs))
	}
	value := returnObject.InterfaceValue()
	if nil == value {
		return errors.New("result is empty." + "\n" + string(bs))
	}

	bytes_indent, err := json.MarshalIndent(value, "", "  ")
	if nil != err {
		return errors.New("marshal failed - " + err.Error() + "\n" + string(bs))
	}

	fmt.Println(string(bytes_indent))
	return nil
}