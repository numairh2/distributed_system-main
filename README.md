# MP_2_G35

## Code Execution Instructions

1. Once SSH'ed into the desired VM, get into the MP directory with ```cd /home/NETID/mp_1_g35``` and replace NETID with your specific NetId.
2. Enter the MP2 directory with `cd membership-system`.
3. Establish and run a TCP server with `go run main.go machine`
4. Establish and run a UDP connection with `go run main.go listener`
5. Establish a daemon and a membership system with `go run main.go daemon`
6. In the daemon terminal, run the following commands:
    - list_mem: list all nodes in the membership list
    - list_self: list current host
    - join <HOSTNAME>: join a host to the membership list
    - join_all <NUM_MACHINES>: join a user-specific number of machines
    - leave <HOSTNAME>: make a host voluntarily leave the membership list
    - leave_all <NUM_MACHINES>: make a user-specific number of machines voluntarily leave
    - enable_sus/disable_sus: toggle on/off suspicion
    - check_sus: get the current suspicion toggle status
    - exit: end the daemon process and membership system

## MP2 Report

[MP Report Doc](https://docs.google.com/document/d/1AqJVFc-fNjXq7zUwm4aICjH8nfH83K5fQ8scUuEAK90/edit?usp=sharing)
