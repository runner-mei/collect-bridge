/*
 * Copyright (c) 2001-2003
 *	Fraunhofer Institute for Open Communication Systems (FhG Fokus).
 *	All rights reserved.
 *
 * Author: Harti Brandt <harti@freebsd.org>
 * 
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 * 
 * THIS SOFTWARE IS PROVIDED BY AUTHOR AND CONTRIBUTORS ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL AUTHOR OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 *
 * $Begemot: bsnmp/lib/snmppriv.h,v 1.9 2004/08/06 08:46:58 brandt Exp $
 *
 * Private functions.
 */

enum asn_err snmp_binding_encode(asn_buf_t *, const snmp_value_t *);
enum snmp_code snmp_pdu_encode_header(asn_buf_t *, snmp_pdu_t *);
enum snmp_code snmp_fix_encoding(asn_buf_t *, snmp_pdu_t *);
enum asn_err snmp_parse_pdus_hdr(asn_buf_t *b, snmp_pdu_t *pdu,
    asn_len_t *lenp);

enum snmp_code snmp_pdu_calc_digest(const snmp_pdu_t *, uint8_t *);
enum snmp_code snmp_pdu_encrypt(const snmp_pdu_t *);
enum snmp_code snmp_pdu_decrypt(const snmp_pdu_t *);

#define DEFAULT_HOST "localhost"
#define DEFAULT_PORT "snmp"
#define DEFAULT_LOCAL "/var/run/snmp.sock"
