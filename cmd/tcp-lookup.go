// Copyright (c) 2018 Jacob Sanders, Michael Grosser
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/dekkagaijin/coredns-demo/data"
	"github.com/miekg/dns"
)

var (
	hostnameFile = flag.String("hostname-file", "data/hostnames.txt", "The file from which to read hostnames to lookup via dns. The file should have exactly one hostname per line.")

	ns   = flag.String("nameserver", "", "The nameserver to use, e.g. `8.8.8.8`")
	port = flag.Int("port", 53, "port number to use")
	aa   = flag.Bool("aa", false, "set AA (Authoritative) flag in query")
	ad   = flag.Bool("ad", false, "set AD (AuthenticatedData) flag in query")
	cd   = flag.Bool("cd", false, "set CD (CheckingDisabled) flag in query")
	rd   = flag.Bool("rd", true, "set RD (RecursionDesired) flag in query")
)

func main() {
	flag.Parse()

	nameserver := *ns

	qnames, err := data.ParseHostnameFile(*hostnameFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("num hosts: %d", len(qnames))

	if nameserver == "" {
		conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			log.Fatal(err)
		}
		nameserver = conf.Servers[0]
	}
	fmt.Println("nameserver: " + nameserver)

	// /etc/resolv.conf adds [ and ], breaking net.ParseIP.
	if nameserver[0] == '[' && nameserver[len(nameserver)-1] == ']' {
		nameserver = nameserver[1 : len(nameserver)-1]
	}
	if i := net.ParseIP(nameserver); i != nil {
		nameserver = net.JoinHostPort(nameserver, strconv.Itoa(*port))
	} else {
		nameserver = dns.Fqdn(nameserver) + ":" + strconv.Itoa(*port)
	}

	c := new(dns.Client)
	c.Net = "tcp"
	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative:     *aa,
			AuthenticatedData: *ad,
			CheckingDisabled:  *cd,
			RecursionDesired:  *rd,
			Opcode:            dns.OpcodeQuery,
		},
		Question: make([]dns.Question, 1),
	}
	m.Opcode = dns.StringToOpcode["QUERY"]
	m.Rcode = dns.RcodeSuccess

	co := new(dns.Conn)
	tcp := "tcp"
	if co.Conn, err = net.DialTimeout(tcp, nameserver, 2*time.Second); err != nil {
		log.Fatal("Dialing " + nameserver + " failed: " + err.Error() + "\n")
	}
	defer co.Close()

	for _, v := range qnames {
		m.Question[0] = dns.Question{Name: dns.Fqdn(v), Qtype: dns.TypeA, Qclass: dns.ClassINET}
		//m.Question[1] = dns.Question{Name: dns.Fqdn(v), Qtype: dns.TypeAAAA, Qclass: dns.ClassINET}
		m.Id = dns.Id()

		co.SetReadDeadline(time.Now().Add(2 * time.Second))
		co.SetWriteDeadline(time.Now().Add(2 * time.Second))

		then := time.Now()
		if err := co.WriteMsg(m); err != nil {
			fmt.Fprintf(os.Stderr, ";; %s\n", err.Error())
			continue
		}
		r, err := co.ReadMsg()
		if err != nil {
			fmt.Fprintf(os.Stderr, ";; %s\n", err.Error())
			continue
		}
		rtt := time.Since(then)
		if r.Id != m.Id {
			fmt.Fprintf(os.Stderr, "Id mismatch\n")
			continue
		}

		fmt.Printf("%v", r)
		fmt.Printf("\n;; query time: %.3d µs, server: %s, size: %d bytes\n", rtt/1e3, nameserver, r.Len())
	}
}
