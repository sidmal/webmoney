# WebMoney XML interfaces client 

[![Build Status](https://travis-ci.com/ProtocolONE/rabbitmq.svg?branch=v1)](https://travis-ci.com/ProtocolONE/rabbitmq) [![codecov](https://codecov.io/gh/ProtocolONE/rabbitmq/branch/v1/graph/badge.svg)](https://codecov.io/gh/ProtocolONE/rabbitmq)

This library is client for WebMoney payment system XML interfaces.
This library currently realise next WebMoney XML interfaces:
    * X2 - transfer money between some wallets (method: *TransferMoney*)
    * X3 - check transfer transaction status or get transactions history (method: *GetTransactionsHistory*)
    * X9 - retrieving information about wallets balance (method: *GetTransactionsHistory*)
    
More info about WebMoney XML interfaces can be found by follow link:  [![WebMoney XML interfaces wiki](https://wiki.wmtransfer.com/projects/webmoney/wiki/XML-interfaces)](https://wiki.wmtransfer.com/projects/webmoney/wiki/XML-interfaces)

## Installation 

`go get github.com/sidmal/webmoney`

## Usage

```go
package main

import (
	"fmt"
	"github.com/sidmal/webmoney"
	"github.com/ProtocolONE/rabbitmq/pkg"
	"github.com/streadway/amqp"
	"log"
	"math/rand"
)

func main() {
	opts = []Option{
    		WmId(TestWmId),
    		Key(TestKey),
    		Password(TestPassword),
    		HttpClient(mocks.NewTransportStatusOk()),
    	}
    	wm, err := NewWebMoney(suite.defaultOptions...)
}
```
