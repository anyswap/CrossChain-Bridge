FROM ubuntu:18.04
COPY build/bin/swapserver build/bin/swaporacle build/bin/swapscan build/bin/swapadmin build/bin/swaptools build/bin/riskctrl /usr/local/bin/
COPY build/bin/config-example.toml build/bin/config-tokenpair-example.toml /usr/local/bin/

##include 1st and 2nd
##cp Dockerfile dcrm6; cd dcrm6
#FROM golang:1.13.5 AS builder
#WORKDIR /build
#COPY . .
#RUN make all
#
#FROM ubuntu:18.04
#COPY build/bin/swapserver build/bin/swaporacle build/bin/swapscan build/bin/swapadmin build/bin/swaptools build/bin/riskctrl /usr/local/bin/

