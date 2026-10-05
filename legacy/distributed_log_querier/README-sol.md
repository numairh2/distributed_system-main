# Machine Programming 1 – Distributed Log Querier

## Description
This tool can be used to query logs on many different machines. It is fault-tolerant, because it can handle when VMs are unable to be reached (simulated by crashing machine processes). It works by sending log commands via RPC, performing local `grep`s, and sending back results via RPC.

## Report
A one-page report describing this tool can be found [here](https://docs.google.com/document/d/1Z9jDQY07XTp9Nh0F0Xh1Il7eV4QC_AWjnXIp48rGRqg/edit?usp=sharing).

## Installation
Install go 1.21.0.
If running distributed tests, install python 3.6.0.

## Usage
First, SSH into each VM and start at the project root. \
On each of these VMs, begin the `machine` process: `go run main.go machine`. \
Then, perform the query by running a `caller` process on any one of the VMs: `go run main.go caller grep <flags> <pattern>`. \
The line counts will be outputted, and the results will be saved under result/output.i.txt.

## Tests
Unit tests can be run from project root:
```
cd caller
go test
```
```
cd machine
go test
```
Distributed tests can also be run, after starting a `machine` process on each VM:
```
python distributed_test.py
``` 

## Project status
Completed and submitted.