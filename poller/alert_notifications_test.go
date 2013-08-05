package poller

import (
	"carrier"
	"commons"
	"commons/types"
	ds "data_store"
	"database/sql"
	"flag"
	"github.com/garyburd/redigo/redis"
	"github.com/runner-mei/delayed_job"
	"net"
	"strings"
	"testing"
	"time"
)

var delayed_job_table_name = flag.String("delayed_job_table_name_test", "delayed_jobs", "the table name of delayed job")

// <class name="RedisCommand" base="Action">
//   <property name="command">
//     <restriction base="string">
//       <maxLength>10</maxLength>
//     </restriction>
//   </property>
//   <property name="arg0">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>
//   <property name="arg1">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>
//   <property name="arg2">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>
//   <property name="arg3">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>
//   <property name="arg4">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>
// </class>

// <class name="DbCommand" base="Action">
//   <property name="drv">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>
//   <property name="url">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>
//   <property name="script">
//     <restriction base="string">
//       <maxLength>2000</maxLength>
//     </restriction>
//   </property>
// </class>

// <class name="Mail" base="Action">
//   <property name="from_address">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>

//   <property name="to_address">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//       <required />
//     </restriction>
//   </property>

//   <property name="cc_address">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>

//   <property name="bcc_address">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>

//   <property name="subject">
//     <restriction base="string">
//       <maxLength>200</maxLength>
//       <required />
//     </restriction>
//   </property>

//   <property name="content_type">
//     <restriction base="string">
//       <!-- html, text -->
//       <maxLength>200</maxLength>
//     </restriction>
//   </property>

//   <property name="content">
//     <restriction base="string">
//       <maxLength>2000</maxLength>
//       <required />
//     </restriction>
//   </property>
// </class>

// <class name="Syslog" base="Action">
//   <property name="facility">
//     <restriction base="string">
//       <maxLength>250</maxLength>
//       <enumeration>
//         <value>kernel</value>
//         <value>user</value>
//         <value>user</value>
//         <value>mail</value>
//         <value>daemon</value>
//         <value>auth</value>
//         <value>syslog</value>
//         <value>lpr</value>
//         <value>news</value>
//         <value>uucp</value>
//         <value>cron</value>
//         <value>authpriv</value>
//         <value>system0</value>
//         <value>system1</value>
//         <value>system2</value>
//         <value>system3</value>
//         <value>system4</value>
//         <value>local0</value>
//         <value>local1</value>
//         <value>local2</value>
//         <value>local3</value>
//         <value>local4</value>
//         <value>local5</value>
//         <value>local6</value>
//         <value>local7</value>
//       </enumeration>
//     </restriction>
//   </property>

//   <property name="severity">
//     <restriction base="string">
//       <maxLength>250</maxLength>
//       <enumeration>
//         <value>emerg</value>
//         <value>alert</value>
//         <value>crit</value>
//         <value>err</value>
//         <value>waining</value>
//         <value>notice</value>
//         <value>info</value>
//         <value>debug</value>
//       </enumeration>
//     </restriction>
//   </property>

//   <property name="tag">
//     <restriction base="string">
//       <maxLength>250</maxLength>
//     </restriction>
//   </property>

//   <property name="content">
//     <restriction base="string">
//       <maxLength>250</maxLength>
//       <required />
//     </restriction>
//   </property>
// </class>

// <class name="ExecCommand" base="Action">
//   <property name="work_directory">
//     <restriction base="string">
//       <maxLength>512</maxLength>
//     </restriction>
//   </property>
//   <property name="prompt">
//     <restriction base="string">
//       <maxLength>250</maxLength>
//     </restriction>
//   </property>
//   <property name="command">
//     <restriction base="string">
//       <maxLength>250</maxLength>
//       <required />
//     </restriction>
//   </property>
// </class>

var redis_test_attributes = map[string]interface{}{"name": "test1",
	"description": "",
	"type":        "redis_command",
	"parent_type": "alert",
	"parent_id":   "1",
	"created_at":  "2013-07-13T14:13:28.7024412+08:00",
	"updated_at":  "2013-07-13T14:13:28.7024412+08:00",
	"command":     "SET",
	"arg0":        "$action_id",
	"arg1":        "$current_value"}

var db_command_test_attributes = map[string]interface{}{"name": "test1",
	"description": "",
	"type":        "db_command",
	"parent_type": "alert",
	"parent_id":   "1",
	"created_at":  "2013-07-13T14:13:28.7024412+08:00",
	"updated_at":  "2013-07-13T14:13:28.7024412+08:00",
	"drv":         "postgres",
	"url":         "host=127.0.0.1 dbname=tpt_data user=tpt password=extreme sslmode=disable",
	"script":      "insert into tpt_test_for_handler(priority, queue) values(12, '{{.current_value}}')"}

var exec_command_test_attributes = map[string]interface{}{"name": "test1",
	"description":    "",
	"type":           "exec_command",
	"parent_type":    "alert",
	"parent_id":      "1",
	"created_at":     "2013-07-13T14:13:28.7024412+08:00",
	"updated_at":     "2013-07-13T14:13:28.7024412+08:00",
	"command":        "cmd /c echo {{.current_value}}",
	"prompt":         "{a:13}",
	"work_directory": "c:\\windows\\"}

var syslog_test_attributes = map[string]interface{}{"name": "test1",
	"description": "",
	"type":        "syslog",
	"parent_type": "alert",
	"parent_id":   "1",
	"created_at":  "2013-07-13T14:13:28.7024412+08:00",
	"updated_at":  "2013-07-13T14:13:28.7024412+08:00",
	"facility":    "user",
	"severity":    "alert",
	"tag":         "abc",
	"content":     "aaaaa {{.current_value}}"}

var mail_test_attributes = map[string]interface{}{"name": "test1",
	"description":  "",
	"type":         "mail",
	"parent_type":  "alert",
	"parent_id":    "1",
	"created_at":   "2013-07-13T14:13:28.7024412+08:00",
	"updated_at":   "2013-07-13T14:13:28.7024412+08:00",
	"subject":      "subject {{.current_value}}",
	"content_type": "text",
	"content":      "aaaaa {{.current_value}}"}

var notifications = []map[string]interface{}{redis_test_attributes,
	db_command_test_attributes,
	exec_command_test_attributes,
	syslog_test_attributes,
	mail_test_attributes}

func TestNotificationsForRedis(t *testing.T) {
	redisTest(t, func(redis_channel chan []string, c redis.Conn) {
		srvTest(t, func(client *ds.Client, definitions *types.TableDefinitions) {
			delayed_job.WorkTest(t, func(worker *delayed_job.TestWorker) {
				carrier.SrvTest(t, func(db *sql.DB, url string) {
					is_test = true
					*foreignUrl = url
					Runforever()
					if nil == server_test {
						t.Error("load trigger failed.")
						return
					}

					notification_group_id := ds.CreateItForTest(t, client, "notification_group", map[string]interface{}{"name": "aaa"})
					ds.CreateItByParentForTest(t, client, "notification_group", notification_group_id, "redis_command", redis_test_attributes)

					action, e := newAlertAction(map[string]interface{}{
						"id":                    "123",
						"name":                  "this is a test alert",
						"notification_group_id": notification_group_id,
						"delay_times":           0,
						"expression_style":      "json",
						"expression_code": map[string]interface{}{
							"attribute": "a",
							"operator":  ">",
							"value":     "12"}},
						map[string]interface{}{"managed_id": 1213},
						server_test.ctx)

					if nil != e {
						t.Error(e)
						return
					}

					//alert := action.(*alertAction)
					for i := 0; i < 10; i++ {
						e = action.Run(time.Now(), commons.Return(map[string]interface{}{"a": "13"}))
						if nil != e {
							t.Error(e)
							return
						}
					}

					i, j, e := worker.WorkOff(1)
					if nil != e {
						t.Error(e)
						return
					}

					if i != 1 {
						t.Log("success is", i, "failed is", j)
						t.Error("excepted job count is 1, excepted is", i)
						return
					}

					clearRedis(t, c, "123")

					i, j, e = worker.WorkOff(1)
					if nil != e {
						t.Error(e)
						return
					}

					if i != 1 {
						t.Log("success is", i, "failed is", j)
						t.Error("excepted job count is 1, excepted is", i)
						return
					}

					containsResult(t, c, "GET", "123", `{"a":"13"}`)
				})
			})
		})
	})
}

func assertCount(t *testing.T, db *sql.DB, sql string, excepted int64) {
	count := int64(-1)
	e := db.QueryRow(sql).Scan(&count)
	if nil != e {
		t.Error(e)
		return
	}

	if count != excepted {
		t.Error("excepted \"", sql, "\" is ", excepted, ", actual is ", count)
	}
}

func TestNotificationsForDb(t *testing.T) {
	srvTest(t, func(client *ds.Client, definitions *types.TableDefinitions) {
		delayed_job.WorkTest(t, func(worker *delayed_job.TestWorker) {
			carrier.SrvTest(t, func(db *sql.DB, url string) {
				_, e := db.Exec(`
					DROP TABLE IF EXISTS tpt_test_for_handler;

					CREATE TABLE IF NOT EXISTS tpt_test_for_handler (
					  id                BIGSERIAL  PRIMARY KEY,
					  priority          int DEFAULT 0,
					  queue             varchar(200)
					);`)

				if nil != e {
					t.Error(e)
					return
				}

				is_test = true
				*foreignUrl = url
				Runforever()
				if nil == server_test {
					t.Error("load trigger failed.")
					return
				}

				notification_group_id := ds.CreateItForTest(t, client, "notification_group", map[string]interface{}{"name": "aaa"})
				ds.CreateItByParentForTest(t, client, "notification_group", notification_group_id, "db_command", db_command_test_attributes)

				action, e := newAlertAction(map[string]interface{}{
					"id":                    "123",
					"name":                  "this is a test alert",
					"notification_group_id": notification_group_id,
					"delay_times":           0,
					"expression_style":      "json",
					"expression_code": map[string]interface{}{
						"attribute": "a",
						"operator":  ">",
						"value":     "12"}},
					map[string]interface{}{"managed_id": 1213},
					server_test.ctx)

				if nil != e {
					t.Error(e)
					return
				}

				//alert := action.(*alertAction)
				for i := 0; i < 10; i++ {
					e = action.Run(time.Now(), commons.Return(map[string]interface{}{"a": "13"}))
					if nil != e {
						t.Error(e)
						return
					}
				}

				i, j, e := worker.WorkOff(1)
				if nil != e {
					t.Error(e)
					return
				}

				if i != 1 {
					t.Log("success is", i, "failed is", j)
					t.Error("excepted job count is 1, excepted is", i)
					return
				}

				i, j, e = worker.WorkOff(1)
				if nil != e {
					t.Error(e)
					return
				}

				if i != 1 {
					t.Log("success is", i, "failed is", j)
					t.Error("excepted job count is 1, excepted is", i)
					return
				}

				assertCount(t, db, "SELECT count(*) FROM tpt_test_for_handler WHERE priority = 12 and queue like '%{\"a\":\"13\"}%'", 1)
			})
		})
	})
}

func TestNotificationsForExec(t *testing.T) {
	srvTest(t, func(client *ds.Client, definitions *types.TableDefinitions) {
		delayed_job.WorkTest(t, func(worker *delayed_job.TestWorker) {
			carrier.SrvTest(t, func(db *sql.DB, url string) {
				_, e := db.Exec(`
					DROP TABLE IF EXISTS tpt_test_for_handler;

					CREATE TABLE IF NOT EXISTS tpt_test_for_handler (
					  id                BIGSERIAL  PRIMARY KEY,
					  priority          int DEFAULT 0,
					  queue             varchar(200)
					);`)

				if nil != e {
					t.Error(e)
					return
				}

				is_test = true
				*foreignUrl = url
				Runforever()
				if nil == server_test {
					t.Error("load trigger failed.")
					return
				}

				notification_group_id := ds.CreateItForTest(t, client, "notification_group", map[string]interface{}{"name": "aaa"})
				ds.CreateItByParentForTest(t, client, "notification_group", notification_group_id, "exec_command", exec_command_test_attributes)

				action, e := newAlertAction(map[string]interface{}{
					"id":                    "123",
					"name":                  "this is a test alert",
					"notification_group_id": notification_group_id,
					"delay_times":           0,
					"expression_style":      "json",
					"expression_code": map[string]interface{}{
						"attribute": "a",
						"operator":  ">",
						"value":     "12"}},
					map[string]interface{}{"managed_id": 1213},
					server_test.ctx)

				if nil != e {
					t.Error(e)
					return
				}

				//alert := action.(*alertAction)
				for i := 0; i < 10; i++ {
					e = action.Run(time.Now(), commons.Return(map[string]interface{}{"a": "13"}))
					if nil != e {
						t.Error(e)
						return
					}
				}

				i, j, e := worker.WorkOff(1)
				if nil != e {
					t.Error(e)
					return
				}

				if i != 1 {
					t.Log("success is", i, "failed is", j)
					t.Error("excepted job count is 1, excepted is", i)
					return
				}

				i, j, e = worker.WorkOff(1)
				if nil != e {
					t.Error(e)
					return
				}

				if i != 1 {
					t.Log("success is", i, "failed is", j)
					t.Error("excepted job count is 1, excepted is", i)
					return
				}

			})
		})
	})
}

func syslogTest(t *testing.T, cb func(client net.PacketConn, port string, c chan string)) {
	client, err := net.ListenPacket("udp", ":0")
	if nil != err {
		t.Error(err)
		return
	}
	defer client.Close()

	c := make(chan string, 100)
	go func() {
		for {
			bs := make([]byte, 1024)
			n, _, e := client.ReadFrom(bs)
			if nil != e {
				break
			}
			c <- string(bs[0:n])
		}
	}()

	laddr := client.LocalAddr().String()
	ar := strings.Split(laddr, ":")

	cb(client, ar[len(ar)-1], c)

	client.Close()
}

func TestNotificationsForSyslog(t *testing.T) {
	syslogTest(t, func(client net.PacketConn, port string, c chan string) {
		notifications[3]["to_address"] = "127.0.0.1:" + port
		srvTest(t, func(client *ds.Client, definitions *types.TableDefinitions) {
			delayed_job.WorkTest(t, func(worker *delayed_job.TestWorker) {
				carrier.SrvTest(t, func(db *sql.DB, url string) {
					is_test = true
					*foreignUrl = url
					Runforever()
					if nil == server_test {
						t.Error("load trigger failed.")
						return
					}

					notification_group_id := ds.CreateItForTest(t, client, "notification_group", map[string]interface{}{"name": "aaa"})
					ds.CreateItByParentForTest(t, client, "notification_group", notification_group_id, "syslog", syslog_test_attributes)

					action, e := newAlertAction(map[string]interface{}{
						"id":                    "123",
						"name":                  "this is a test alert",
						"notification_group_id": notification_group_id,
						"delay_times":           0,
						"expression_style":      "json",
						"expression_code": map[string]interface{}{
							"attribute": "a",
							"operator":  ">",
							"value":     "12"}},
						map[string]interface{}{"managed_id": 1213},
						server_test.ctx)

					if nil != e {
						t.Error(e)
						return
					}

					//alert := action.(*alertAction)
					for i := 0; i < 10; i++ {
						e = action.Run(time.Now(), commons.Return(map[string]interface{}{"a": "13"}))
						if nil != e {
							t.Error(e)
							return
						}
					}

					i, j, e := worker.WorkOff(1)
					if nil != e {
						t.Error(e)
						return
					}

					if i != 1 {
						t.Log("success is", i, "failed is", j)
						t.Error("excepted job count is 1, excepted is", i)
						return
					}

					i, j, e = worker.WorkOff(1)
					if nil != e {
						t.Error(e)
						return
					}

					if i != 1 {
						t.Log("success is", i, "failed is", j)
						t.Error("excepted job count is 1, excepted is", i)
						return
					}

					select {
					case s := <-c:
						if !strings.Contains(s, ` {"value":{"a":"13"},`) {
							t.Error(`excepted message contains [ {"value":{"a":"13"},], but actual is`, s)
						}
					case <-time.After(10 * time.Microsecond):
						t.Error("recv syslog time out")
					}
				})
			})
		})
	})
}

var test_mail_to = flag.String("test.notification.mail_to", "", "the address of mail")

func TestNotificationsForMail(t *testing.T) {
	if "" == *test_mail_to {
		t.Skip("please set 'test.mail_to', 'test.mail_from' and 'test.smtp_server'")
		return
	}

	mail_test_attributes["to_address"] = *test_mail_to
	srvTest(t, func(client *ds.Client, definitions *types.TableDefinitions) {
		delayed_job.WorkTest(t, func(worker *delayed_job.TestWorker) {
			carrier.SrvTest(t, func(db *sql.DB, url string) {
				is_test = true
				*foreignUrl = url
				Runforever()
				if nil == server_test {
					t.Error("load trigger failed.")
					return
				}

				notification_group_id := ds.CreateItForTest(t, client, "notification_group", map[string]interface{}{"name": "aaa"})
				ds.CreateItByParentForTest(t, client, "notification_group", notification_group_id, "mail", mail_test_attributes)

				action, e := newAlertAction(map[string]interface{}{
					"id":                    "123",
					"name":                  "this is a test alert",
					"notification_group_id": notification_group_id,
					"delay_times":           0,
					"expression_style":      "json",
					"expression_code": map[string]interface{}{
						"attribute": "a",
						"operator":  ">",
						"value":     "12"}},
					map[string]interface{}{"managed_id": 1213},
					server_test.ctx)

				if nil != e {
					t.Error(e)
					return
				}

				//alert := action.(*alertAction)
				for i := 0; i < 10; i++ {
					e = action.Run(time.Now(), commons.Return(map[string]interface{}{"a": "13"}))
					if nil != e {
						t.Error(e)
						return
					}
				}

				i, j, e := worker.WorkOff(1)
				if nil != e {
					t.Error(e)
					return
				}

				if i != 1 {
					t.Log("success is", i, "failed is", j)
					t.Error("excepted job count is 1, excepted is", i)
					return
				}

				i, j, e = worker.WorkOff(1)
				if nil != e {
					t.Error(e)
					return
				}

				if i != 1 {
					t.Log("success is", i, "failed is", j)
					t.Error("excepted job count is 1, excepted is", i)
					return
				}
			})
		})
	})
}