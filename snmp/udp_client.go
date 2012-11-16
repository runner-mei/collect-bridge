package snmp

// #include "bsnmp/config.h"
// #include "bsnmp/asn1.h"
// #include "bsnmp/snmp.h"
// #include "bsnmp/gobindings.h"
import "C"
import (
	"encoding/hex"
	"errors"
	"flag"
	"log"
	"net"
	"time"
	"unsafe"
)

var maxPDUSize = flag.Int("maxPDUSize", 10240, "set max size of pdu")
var logPdu = flag.Bool("log.pdu", true, "log pdu content?")

type Request struct {
	pdu      PDU
	ctx      InvokedContext
	callback func(PDU, SnmpCodeError)
}

func (req *Request) reply(result PDU, err SnmpCodeError) {

	if nil != req.ctx {

		if *logPdu {
			if nil != err {
				log.Printf("snmp - recv pdu failed, %v", err)
			} else {
				log.Printf("snmp - recv pdu success, %v", result)
			}
		}

		req.ctx.Reply(result, err)
	} else {
		req.callback(result, err)
	}
}

type UdpClient struct {
	requestId int
	host      string
	engine    snmpEngine
	conn      *net.UDPConn
	pendings  map[int]*Request
	Svc
}

func NewSnmpClient(host string) (Client, SnmpCodeError) {
	cl := &UdpClient{host: NormalizeAddress(host)}
	cl.pendings = make(map[int]*Request)
	return cl, nil
}

func (client *UdpClient) CreatePDU(op SnmpType, version SnmpVersion) (PDU, SnmpCodeError) {
	if op < 0 || SNMP_PDU_REPORT < op {
		return nil, Errorf(SNMP_CODE_FAILED, "unsupported pdu type: %d", op)
	}

	switch version {
	case SNMP_V1, SNMP_V2C:
		return &V2CPDU{op: op, client: client, version: version}, nil
	case SNMP_V3:
		return &V3PDU{op: op, client: client}, nil
	}
	return nil, Errorf(SNMP_CODE_FAILED, "unsupported version: %d", version)
}

func handleRemoveRequest(client *UdpClient, id int) {
	delete(client.pendings, id)
}

func toSnmpCodeError(e error) SnmpCodeError {
	if err, ok := e.(SnmpCodeError); ok {
		return err
	}
	return newError(SNMP_CODE_FAILED, e, "")
}

func (client *UdpClient) SendAndRecv(req PDU, timeout time.Duration) (pdu PDU, err SnmpCodeError) {

	defer func() {
		if 0 != req.GetRequestID() {
			client.Send(handleRemoveRequest, client, req.GetRequestID())
		}
	}()

	if *logPdu {
		log.Printf("snmp - invoke begin.")
	}

	values := client.SafelyCall(timeout, func(ctx InvokedContext) { client.handleSend(ctx, req) })

	if *logPdu {
		log.Printf("snmp - invoke end.")
	}

	switch len(values) {
	case 0:
		err = Error(SNMP_CODE_FAILED, "return empty.")
	case 1:
		if nil != values[0] {
			err = toSnmpCodeError(values[0].(error))
		}
	case 2:
		if nil != values[0] {
			pdu = values[0].(PDU)
		}
		if nil != values[1] {
			err = toSnmpCodeError(values[1].(error))
		}
	default:
		err = Error(SNMP_CODE_FAILED, "num of return value is error.")
	}
	return
}

func (client *UdpClient) createConnect() SnmpCodeError {
	if nil != client.conn {
		return nil
	}
	addr, err := net.ResolveUDPAddr("udp", client.host)
	if nil != err {
		return newError(SNMP_CODE_FAILED, err, "parse address failed")
	}
	client.conn, err = net.DialUDP("udp", nil, addr)
	if nil != err {
		return newError(SNMP_CODE_FAILED, err, "bind udp port failed")
	}

	go client.readUDP(client.conn)
	return nil
}

func (client *UdpClient) discoverEngine(fn func(PDU, SnmpCodeError)) {
	usm := &USM{auth_proto: SNMP_AUTH_NOAUTH, priv_proto: SNMP_PRIV_NOPRIV}
	pdu := &V3PDU{op: SNMP_PDU_GET, target: client.host, securityModel: usm}
	client.sendPdu(pdu, nil, fn)
}

func (client *UdpClient) sendV3PDU(ctx InvokedContext, pdu *V3PDU, autoDiscoverEngine bool) {
	if !pdu.securityModel.IsLocalize() {
		if nil == pdu.engine {
			if *logPdu {
				log.Printf("snmp - send failed, " + pdu.String())
			}
			ctx.Reply(nil, Error(SNMP_CODE_FAILED, "nil == pdu.engine"))
			return
		}
		pdu.securityModel.Localize(pdu.engine.engine_id)
	}
	if autoDiscoverEngine {
		client.sendPdu(pdu, nil, func(resp PDU, err SnmpCodeError) {
			if nil != err {
				switch err.Code() {
				case SNMP_CODE_NOTINTIME, SNMP_CODE_BADENGINE:

					if nil != pdu.engine {
						pdu.engine.engine_id = nil
					}
					client.engine.engine_id = nil
					client.discoverEngineAndSend(ctx, pdu)
					return
				}
			}

			if *logPdu {
				if nil != err {
					log.Printf("snmp - recv pdu failed, %v", err)
				} else {
					log.Printf("snmp - recv pdu success, %v", resp)
				}
			}

			ctx.Reply(resp, err)
		})
	} else {
		client.sendPdu(pdu, ctx, nil)
	}
}

func (client *UdpClient) discoverEngineAndSend(ctx InvokedContext, pdu *V3PDU) {

	if nil != pdu.engine && nil != pdu.engine.engine_id && 0 != len(pdu.engine.engine_id) {
		client.sendV3PDU(ctx, pdu, false)
		return
	}

	if nil != client.engine.engine_id && 0 != len(client.engine.engine_id) {

		if nil == pdu.engine {
			pdu.engine = &client.engine
		} else {
			pdu.engine.CopyFrom(&client.engine)
		}
		client.sendV3PDU(ctx, pdu, true)
		return
	}

	client.discoverEngine(func(resp PDU, err SnmpCodeError) {
		if nil == resp {

			if nil != err {
				err = newError(err.Code(), err, "discover engine failed")
			} else {
				err = Error(SNMP_CODE_FAILED, "discover engine failed - return nil pdu")
			}

			if *logPdu {
				log.Printf("snmp - recv pdu, " + err.Error())
			}
			ctx.Reply(nil, err)
			return
		}
		v3, ok := resp.(*V3PDU)
		if !ok {

			if nil != err {
				err = newError(err.Code(), err, "discover engine failed - oooooooooooo! it is not v3pdu")
			} else {
				err = Error(SNMP_CODE_FAILED, "discover engine failed - oooooooooooo! it is not v3pdu")
			}

			if *logPdu {
				log.Printf("snmp - recv pdu, " + err.Error())
			}

			ctx.Reply(nil, err)
			return
		}

		client.engine.CopyFrom(v3.engine)
		if nil == pdu.engine {
			pdu.engine = &client.engine
		} else {
			pdu.engine.engine_id = client.engine.engine_id
		}
		client.sendV3PDU(ctx, pdu, false)
	})
}

func (client *UdpClient) readUDP(conn *net.UDPConn) {
	var err error

	defer func() {
		client.conn = nil
		if err := recover(); nil != err {
			log.Printf("%v\r\n", err)
		}
		conn.Close()
	}()
	for {
		var length int
		var bytes []byte

		bytes = make([]byte, *maxPDUSize)
		length, err = conn.Read(bytes)
		if nil != err {
			client.Logger.Println(err)
			break
		}

		if *logPdu {
			log.Printf("snmp - read ok")
			log.Println(hex.EncodeToString(bytes[:length]))
		}

		func(buf []byte) {
			client.Send(func() { client.handleRecv(buf) })
		}(bytes[:length])
	}

	client.Send(func() { client.handleDisConnection(err) })
}

func (client *UdpClient) handleDisConnection(err error) {
	e := newError(SNMP_CODE_BADNET, err, "")

	for _, req := range client.pendings {
		req.reply(nil, e)
	}
}

func (client *UdpClient) handleRecv(bytes []byte) {

	var buffer C.asn_buf_t
	var pdu C.snmp_pdu_t
	var result PDU
	var req *Request
	var ok bool

	C.set_asn_u_ptr(&buffer.asn_u, (*C.char)(unsafe.Pointer(&bytes[0])))
	buffer.asn_len = C.size_t(len(bytes))

	err := DecodePDUHeader(&buffer, &pdu)
	if nil != err {
		client.Logger.Println(err)
		return
	}
	defer C.snmp_pdu_free(&pdu)

	if uint32(SNMP_V3) == pdu.version {

		req, ok = client.pendings[int(pdu.identifier)]
		if !ok {
			client.Logger.Printf("not found request with requestId = %d.\r\n", int(pdu.identifier))
			return
		}

		v3old, ok := req.pdu.(*V3PDU)
		if !ok {
			err = Error(SNMP_CODE_FAILED, "receive pdu is a v3 pdu.")
			goto complete
		}
		usm, ok := v3old.securityModel.(*USM)
		if !ok {
			err = Error(SNMP_CODE_FAILED, "receive pdu is not usm.")
			goto complete
		}
		err = FillUser(&pdu, usm.auth_proto, usm.localization_auth_key,
			usm.priv_proto, usm.localization_priv_key)
		if nil != err {
			log.Printf("%s\r\n", err.Error())
			goto complete
		}

		err = DecodePDUBody(&buffer, &pdu)
		if nil != err {
			log.Printf("%s\r\n", err.Error())
			goto complete
		}
		var v3 V3PDU
		_, err = v3.decodePDU(&pdu)
		result = &v3
	} else {
		err = DecodePDUBody(&buffer, &pdu)
		if nil != err {
			log.Printf("%s\r\n", err.Error())
			return
		}

		req, ok = client.pendings[int(pdu.request_id)]
		if !ok {
			client.Logger.Printf("not found request with requestId = %d.\r\n", int(pdu.request_id))
			return
		}

		var v2 V2CPDU
		_, err = v2.decodePDU(&pdu)
		result = &v2
	}

complete:
	req.reply(result, err)
}

func (client *UdpClient) handleSend(ctx InvokedContext, pdu PDU) {
	var err error = nil
	if nil == client.conn {
		err = client.createConnect()
		if nil != err {
			goto failed
		}
	}

	if SNMP_V3 == pdu.GetVersion() {
		v3, ok := pdu.(*V3PDU)
		if !ok {
			err = errors.New("oooooooooooo! it is not v3pdu.")
			goto failed
		}

		client.discoverEngineAndSend(ctx, v3)
		return
	}

	client.sendPdu(pdu, ctx, nil)
	return
failed:

	if *logPdu {
		log.Printf("snmp - send failed, " + pdu.String())
	}

	ctx.Reply(nil, err)
	return
}

func (client *UdpClient) safelyKillConnection() {
	defer func() {
		client.conn = nil
		if err := recover(); nil != err {
			client.Logger.Printf("%v\r\n", err)
		}
	}()
	if nil != client.conn {
		client.conn.Close()
		client.conn = nil
	}
}

func (client *UdpClient) sendPdu(pdu PDU, ctx InvokedContext, callback func(PDU, SnmpCodeError)) {
	if nil == ctx {
		if nil == callback {
			panic("'ctx' and 'callback' is nil")
		}
	} else if nil != callback {
		panic("'ctx' and 'callback' is not nil")
	}

	var bytes []byte = nil
	var err SnmpCodeError = nil
	var e error = nil
	client.requestId++
	pdu.SetRequestID(client.requestId)

	_, ok := client.pendings[pdu.GetRequestID()]
	if ok {
		err = Error(SNMP_CODE_FAILED, "identifier is repected.")
		goto failed
	}

	bytes, err = EncodePDU(pdu)
	if nil != err {
		err = newError(err.Code(), err, "encode pdu failed")
		goto failed
	}

	client.pendings[pdu.GetRequestID()] = &Request{pdu: pdu, ctx: ctx, callback: callback}

	_, e = client.conn.Write(bytes)
	if nil != e {
		client.safelyKillConnection()
		err = newError(SNMP_CODE_BADNET, e, "send pdu failed")
		goto failed
	}

	if *logPdu {
		log.Printf("snmp - send success, " + pdu.String())
	}

	return
failed:

	if *logPdu {
		log.Printf("snmp - send failed, " + err.Error())
	}

	if nil != callback {
		callback(nil, err)
	} else {
		ctx.Reply(nil, err)
	}
	return
}

func (client *UdpClient) FreePDU(pdus ...PDU) {

}
