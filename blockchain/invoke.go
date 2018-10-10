package blockchain

import (
    "fmt"
    "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
    "time"
)

func (setup *FabricSetup) Invoke(function string, key string, value string) (string, error) {
    var args []string
    args = append(args, "invoke")
    args = append(args, function)
    args = append(args, key)
    args = append(args, value)

    eventID := "eventInvoke"

    transientDataMap := make(map[string][]byte)
    transientDataMap["result"] = []byte("Transient data in invoke")

    //notifier := make(chan *chclient.CCEvent)
    //rce, err := setup.client.RegisterChaincodeEvent(notifier, setup.ChainCodeID, eventID)
    reg, notifier, err := setup.event.RegisterChaincodeEvent(setup.ChainCodeID, eventID)
    if err != nil {
        return "", fmt.Errorf("failed to register chaincode event: %v", err)
    }
    defer setup.event.Unregister(reg)

    response, err := setup.client.Execute(channel.Request{ChaincodeID: setup.ChainCodeID, Fcn: args[0], Args: [][]byte{[]byte(args[1]), []byte(args[2]), []byte(args[3])}, TransientMap: transientDataMap})
    if err != nil {
        return "", fmt.Errorf("Failed to move funds: %v", err)
    }

    select {
    case ccEvent := <-notifier:
        fmt.Printf("Received CC event: %s\n", ccEvent)
    case <-time.After(time.Second*20):
        return "", fmt.Errorf("did NOT receive CC event for eventID(%s)", eventID)
    }
    //setup.client.UnregisterChaincodeEvent(rce)
    return string(response.TransactionID), nil
}
