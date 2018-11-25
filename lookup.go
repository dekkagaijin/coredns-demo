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
	forceTcp     = flag.Bool("force-tcp", false, "always use TCP to query the nameserver, reusing the connection")
	to           = flag.Int64("timeout", 5000, "the i/o timeout in milliseconds")

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

	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative:     *aa,
			AuthenticatedData: *ad,
			CheckingDisabled:  *cd,
			RecursionDesired:  *rd,
			Opcode:            dns.OpcodeQuery,
		},
		Question: []dns.Question{dns.Question{Qtype: dns.TypeA, Qclass: dns.ClassINET}},
	}
	m.Opcode = dns.StringToOpcode["QUERY"]
	m.Rcode = dns.RcodeSuccess

	ran := false

	var c *dns.Client // Standard
	var co *dns.Conn  // Shared, persistent TCP connection

	if *forceTcp {
		defer func() {
			if co != nil {
				co.Close()
			}
		}()
	} else {
		c = new(dns.Client)
		c.Timeout = timeout
	}

	for !ran || *stressTest {
		ran = true
		for _, n := range qnames {
			if *forceTcp {
				tcpLookup(nameserver, n, &co, m, stats, timeout)
				continue
			}
			standardLookup(nameserver, n, c, m, stats)
		}
	}
}

func tcpLookup(nameserver, hostname string, sharedConn **dns.Conn, m *dns.Msg, stats *data.StatsFile, timeout time.Duration) {
	m.Question[0].Name = dns.Fqdn(hostname)
	m.Id = dns.Id()

	then := time.Now() // Include the time spent on the TCP dial.
	co := *sharedConn
	if co == nil {
		co = new(dns.Conn)
		tcpConn, err := net.DialTimeout("tcp", nameserver, timeout)
		if err != nil {
			log.Panic("Dialing " + nameserver + " failed: " + err.Error() + "\n")
		}
		co.Conn = tcpConn
		*sharedConn = co
	}

	deadline := then.Add(timeout)
	co.SetReadDeadline(deadline)
	co.SetWriteDeadline(deadline)

	if err := co.WriteMsg(m); err != nil {
		stats.Emit(hostname, time.Since(then), err)
		//fmt.Fprintf(os.Stderr, ";; Lookup for %q failed: %s\n", n, err.Error())
		co.Close()
		*sharedConn = nil
		return
	}
	r, err := co.ReadMsg()
	if err != nil {
		stats.Emit(hostname, time.Since(then), err)
		log.Printf(";; Reading response for %q failed: %s\n", hostname, err.Error())
		co.Close()
		*sharedConn = nil
		return
	}
	rtt := time.Since(then)

	log.Printf(";; %q query time: %.3d µs, server: %s, size: %d bytes\n", hostname, rtt/1e3, nameserver, r.Len())
	stats.Emit(hostname, rtt, nil)
}

func standardLookup(nameserver, hostname string, c *dns.Client, m *dns.Msg, stats *data.StatsFile) {
	m.Question[0].Name = dns.Fqdn(hostname)
	m.Id = dns.Id()

	then := time.Now()

	c.Net = "udp"
	r, _, err := c.Exchange(m, nameserver)
	switch err {
	case nil:
		//do nothing
	case dns.ErrTruncated:
		// Response was truncated, retry with TCP.
		c.Net = "tcp"
		r, _, err = c.Exchange(m, nameserver)
	default:
		stats.Emit(hostname, time.Since(then), err)
		log.Printf(";; %s\n", err.Error())
		return
	}

	rtt := time.Since(then)
	log.Printf(";; %q query time: %.3d µs, server: %s(%s), size: %d bytes\n", hostname, rtt/1e3, nameserver, c.Net, r.Len())
	stats.Emit(hostname, rtt, nil)
}
