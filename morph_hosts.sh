#!/bin/bash
head="""@                   3600 IN SOA     ns.stpdns.net. domains.nzt.ventures. (
                                      2018120402   ; serial
                                      5m           ; refresh
                                      5m           ; retry
                                      1w           ; expire
                                      12h    )     ; minimum

@                  86400 IN NS      ns.stpdns.net.
@                  86400 IN NS      ns.stpdns.nl.

;redirect
@                  86400 IN A       35.201.95.240
@                  86400 IN AAAA    2600:1901:0:cdc7:0:0:0:0

_redirect          86400 IN TXT     \"v=txtv0;to=https://about.txtdirect.org;type=host\"

@                    300 IN CAA 0   issue \"letsencrypt.org\"
@                    300 IN CAA 0   issuewild \";\"
@                    300 IN CAA 0   iodef \"mailto:caa@nzt.ventures\"

\$ORIGIN kubecon
*                  86400 IN CNAME   txtd.io.
_redirect          86400 IN TXT     \"v=txtv0;to=https://about.txtdirect.org;type=host\"
"""

echo "$head" > data/critical.software

while read line; do
  line_morphed="${line//./-}"
  echo "$line_morphed     86400 IN CNAME     txtd.io." >> data/critical.software
  echo "_redirect.$line_morphed     42000 IN TXT     \"v=txtv0;type=host;to=https://about.txtdirect.org\"" >> data/critical.software
done <data/hostnames.txt

