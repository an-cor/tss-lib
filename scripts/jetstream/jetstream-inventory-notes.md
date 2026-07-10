# Jetstream VM Inventory Notes

Project: Binance tss-lib Jetstream VM Completion

Controller:
- Jetstream display name: socioty-controller
- Linux hostname: dkls-test-2
- public IP: 149.165.171.46
- internal IP: 10.1.20.67
- user: exouser
- SSH key used from controller to parties: ~/.ssh/socioty_controller_key

Important:
- The Linux hostname still says dkls-test-2 because the VM was originally created under that name.
- The Jetstream UI display name was later changed to socioty-controller.
- This name mismatch is harmless.
- For placeholders like <party-internal-ip>, replace the whole placeholder with a real IP such as 10.1.20.228. Do not type the angle brackets.

Party inventory is stored at:
scripts/jetstream/parties.json

Controller-to-party SSH command pattern:
ssh -i ~/.ssh/socioty_controller_key exouser@10.1.20.228

First network benchmark target:
- n=3
- t=2
- use canonical parties 1, 2, and 3:
  - 10.1.20.228
  - 10.1.20.78
  - 10.1.20.149

Full party set:
- party 1: 10.1.20.228
- party 2: 10.1.20.78
- party 3: 10.1.20.149
- party 4: 10.1.20.92
- party 5: 10.1.20.68
- party 6: 10.1.20.134
- party 7: 10.1.20.125
- party 8: 10.1.20.241
- party 9: 10.1.20.94
- party 10: 10.1.20.141
