# Jetstream VM Inventory Notes

Project: Binance tss-lib Jetstream VM Completion

Controller:
- name: socioty-controller
- public IP: 149.165.171.46
- internal IP: 10.1.20.67
- user: exouser

Known party VMs from Jetstream UI:

| Party | Name | Public IP | Internal IP | Notes |
|---:|---|---|---|---|
| 1 | socioty-party-01 | UI inconsistent / check | UI not copied yet | Need verify |
| 2 | socioty-party-02 | 149.165.171.226 | 10.1.20.78 | Ready |
| 3 | socioty-party-03 | UI inconsistent / check | UI not copied yet | Need verify |
| 4 | socioty-party-04 | 149.165.169.137 | 10.1.20.92 | Ready |
| 5 | socioty-party-05 | 149.165.173.123 | 10.1.20.68 | Ready |
| 6 | socioty-party-06 | 149.165.169.231 | 10.1.20.134 | Ready |
| 7 | socioty-party-07 | 149.165.168.118 | 10.1.20.125 | Ready |
| 8 | socioty-party-08 | UI inconsistent / check | UI not copied yet | Need verify |
| 9 | socioty-party-09 | 149.165.173.163 | 10.1.20.94 | Ready |
| 10 | socioty-party-10 | 149.165.173.250 | 10.1.20.141 | Ready |

Current first target:
- n=3, t=2
- use controller plus parties 2, 4, 5 if party 1/3 IPs are unclear
- later restore canonical party numbering once all internal IPs are confirmed
