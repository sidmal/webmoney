# WebMoney XML interfaces client 

[![Build Status](https://travis-ci.com/sidmal/webmoney.svg?branch=master)](https://travis-ci.com/ProtocolONE/rabbitmq) 
[![codecov](https://codecov.io/gh/sidmal/webmoney/branch/master/graph/badge.svg)](https://codecov.io/gh/sidmal/webmoney)

This library is client for WebMoney payment system XML interfaces.
This library currently realise next WebMoney XML interfaces:

* X2 - transfer money between some wallets (method: **TransferMoney**)
* X3 - check transfer transaction status or get transactions history (method: **GetTransactionsHistory**)
* X9 - retrieving information about wallets balance (method: **GetTransactionsHistory**)
    
More info about WebMoney XML interfaces can be found by follow link:  [WebMoney XML interfaces wiki](https://wiki.wmtransfer.com/projects/webmoney/wiki/XML-interfaces)

## Installation 

`go get github.com/sidmal/webmoney`

## Usage

```go
package main

import (
    "github.com/sidmal/webmoney"
    "log"
)

func main() {
    opts := []webmoney.Option{
        webmoney.WmId("45612378901"),
        webmoney.Key("MTIzNDU2Nzg5MA=="),
        webmoney.Password("kvm_password"),
    }
    wm, err := webmoney.NewWebMoney(opts...)
    
    if err != nil {
        log.Fatal("WebMoney handler initialization failed")
    }
    
    transferMoneyRequest := &webmoney.TransferMoneyRequest{
        TxnId:     1234567890,
        PurseSrc:  "Z123456789012",
        PurseDest: "Z0987654321098",
        Amount:    "10.00",
        Period:    0,
        Desc:      "Тестовая операция",
        PCode:     "",
        WmInvId:   0,
        OnlyAuth:  1,
    }
    transferMoneyResponse, err := wm.TransferMoney(transferMoneyRequest)
    
    if err != nil {
        log.Fatalf("Money transfer failed with error: %s", err)
    }
    
    log.Printf("Money transfered successfully sended. WebMoney transaction ID: %s", transferMoneyResponse.Id)
    
    getTransactionsHistoryRequest := &webmoney.GetTransactionsHistoryRequest{
        Purse:      "Z123456789012",
        TxnId:      1234567890,
        DateStart:  "20060102 15:04:05",
        DateFinish: "20060102 15:04:05",
    }
    getTransactionsHistoryResponse, err := wm.GetTransactionsHistory(getTransactionsHistoryRequest)
    
    if err != nil {
        log.Fatalf("Transaction history receive finished with error: %s", err)
    }
    
    if getTransactionsHistoryResponse.Count == 1 && getTransactionsHistoryResponse.OperationList[0].DateCrt != "" {
        log.Printf("Money transfer ID %d successfully completed", getTransactionsHistoryResponse.OperationList[0].TxnId)
    }
    
    getBalanceRequest := &webmoney.GetBalanceRequest{
        Wmid: "405002833238",
    }
    getBalanceResponse, err := wm.GetBalance(getBalanceRequest)
    
    if err != nil {
        log.Fatalf("Get wallets balance finished with error: %s", err)
    }
    
    for _, val := range getBalanceResponse.PurseList {
        log.Printf("Wallet %s balance is %.2f\n", val.PurseName, val.Amount)
    }
}
```
