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
	"log"
	"net"
	"strconv"
	"time"

	"github.com/dekkagaijin/coredns-demo/data"
	"github.com/miekg/dns"
)

var (
	hostnameFile = flag.String("hostname-file", "data/hostnames.txt", "The file from which to read hostnames to lookup via dns. The file should have exactly one hostname per line.")
	statsFile    = flag.String("stats-file", "", "The file into which lookup statistics should be emitted.")
	stressTest   = flag.Bool("stress-test", false, "endlessly query the input hostnames")
	to           = flag.Int64("timeout", 5000, "i/o timeout in milliseconds")

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
	timeout := time.Duration(*to) * time.Millisecond

	qnames, err := data.ParseHostnameFile(*hostnameFile)
	if err != nil {
		log.Panic(err)
	}

	var stats *data.StatsFile
	if *statsFile != "" {
		stats, err = data.CreateStatsFile(*statsFile)
		if err != nil {
			log.Panic(err)
		}
		defer stats.Close()
	}

	if nameserver == "" {
		conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			log.Panic(err)
		}
		nameserver = conf.Servers[0]
	}

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

	var co *dns.Conn
	ran := false

	for !ran || *stressTest {
		ran = true
		for _, n := range qnames {
			m.Question[0] = dns.Question{Name: dns.Fqdn(n), Qtype: dns.TypeA, Qclass: dns.ClassINET}
			m.Id = dns.Id()

			then := time.Now() // Include the time spent on the TCP dial.
			if co == nil {
				co = new(dns.Conn)
				if co.Conn, err = net.DialTimeout("tcp", nameserver, timeout); err != nil {
					log.Panic("Dialing " + nameserver + " failed: " + err.Error() + "\n")
				}
				defer co.Close()
			}

			deadline := then.Add(timeout)
			co.SetReadDeadline(deadline)
			co.SetWriteDeadline(deadline)

			if err := co.WriteMsg(m); err != nil {
				if stats != nil {
					stats.Emit(n, time.Since(then), err)
				}
				//fmt.Fprintf(os.Stderr, ";; Lookup for %q failed: %s\n", n, err.Error())
				co.Close()
				co = nil
				continue
			}
			r, err := co.ReadMsg()
			if err != nil {
				if stats != nil {
					stats.Emit(n, time.Since(then), err)
				}
				log.Printf(";; Reading response for %q failed: %s\n", n, err.Error())
				co.Close()
				co = nil
				continue
			}
			rtt := time.Since(then)

			log.Printf("%v", r)
			log.Printf("\n;; query time: %.3d µs, server: %s, size: %d bytes\n", rtt/1e3, nameserver, r.Len())
			if stats != nil {
				stats.Emit(n, rtt, nil)
			}
		}
	}
}
