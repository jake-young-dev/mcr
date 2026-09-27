#!/usr/bin/env bash

go test -run=^$ -bench="BenchmarkSendAndRecv|BenchmarkSendNoRes" -memprofile=mem -timeout 5s