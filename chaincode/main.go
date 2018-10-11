package main

import (
    "fmt"
    "github.com/hyperledger/fabric/core/chaincode/shim"
    pb "github.com/hyperledger/fabric/protos/peer"
    "github.com/hyperledger/fabric/protos/msp"
    "github.com/golang/protobuf/proto"
    "encoding/json"
    "crypto/x509"
    "encoding/pem"
)


type SamTestChaincode struct {
}

func (t *SamTestChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
    fmt.Println("##### SamTestChaincode Init #####")

    function, _ := stub.GetFunctionAndParameters()

    if function != "init" {
        return shim.Error("Unknown function call")
    }
    err := stub.PutState("hello", []byte("world"))
    if err != nil {
        return shim.Error(err.Error())
    }
    return shim.Success(nil)
}

func (t *SamTestChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
    fmt.Println("##### SamTestChaincode Invoke #####")

    function, args := stub.GetFunctionAndParameters()

    if function != "invoke" {
        return shim.Error("Unknown funtion call")
    }
    
    if len(args) < 1 {
        return shim.Error("invalid arguments, expected at least 1")
    }

    if args[0] == "query" {
        return t.query(stub, args)
    }
    if args[0] == "invoke" {
        return t.invoke(stub, args)
    }
    if args[0] == "joinPlatoon" {
        return t.joinPlatoon(stub, args)
    }
    if args[0] == "leavePlatoon" {
        return t.leavePlatoon(stub, args)
    }

    return shim.Error("Failed to invoke, check argument 1")
}

func (t *SamTestChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    fmt.Println("##### SamTestChaincode query #####")
    if len(args) < 2 {
        return shim.Error("invalid number of arguments for query, need at least 2")
    }
    state, err := stub.GetState(args[1])
    if err != nil {
        return shim.Error("couldn't get state of argument")
    }
    return shim.Success(state)
}

func (t *SamTestChaincode) joinPlatoon(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    if len(args) < 3 {
        return shim.Error("invalid number of arguments for joinPlatoon, need at least 3")
    }

    serializedID, err := stub.GetCreator()
    if err != nil {
        return shim.Error("could not get creator: " + err.Error())
    }
    sId := &msp.SerializedIdentity{}
    err = proto.Unmarshal(serializedID, sId)
    if err != nil {
        return shim.Error("could not deserialize the sId: " + err.Error())
    }
    bl, _ := pem.Decode(sId.IdBytes)
    if bl == nil {
        return shim.Error("could not decode PEM")
    }
    cert, err := x509.ParseCertificate(bl.Bytes)
    if err != nil {
        return shim.Error("could not parse cert: " + err.Error())
    }
    // Get the client ID object
    //user_id, err := cid.GetID(stub)
    //user_id, err := stub.GetCreator()
    user_id := string(cert.RawSubject)
    //user_id := args[2]
    //if err != nil {
    //    return shim.Error("couldn't get userID: " + err.Error())
    //}
    state, err := stub.GetState(args[1])
    if err != nil {
        return shim.Error("couldn't get state of platoon: " + err.Error())
    }
    var platoonArray []string
    if string(state) != "" {
        err = json.Unmarshal(state, &platoonArray)
        if err != nil {
            return shim.Error("Error decoding JSON data: " + err.Error())
        }
        for _, test := range platoonArray{
            if test == user_id {
                return shim.Error("value already in platoon")
            }
        }
    }
    platoonArray = append(platoonArray, user_id)
    state, err = json.Marshal(platoonArray)
    if err != nil {
        return shim.Error("Error encoding JSON data: " + err.Error())
    }

    err = stub.PutState(args[1], []byte(state))
    if err != nil {
        return shim.Error("failed to update state: " + err.Error())
    }

    err = stub.SetEvent("eventInvoke", []byte{})
    if err != nil {
        return shim.Error(err.Error())
    }
    return shim.Success(nil)

}

func (t *SamTestChaincode) leavePlatoon(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    if len(args) < 3 {
        return shim.Error("invalid number of arguments for lavePlatoon, need at least 3")
    }
    state, err := stub.GetState(args[1])
    if err != nil {
        return shim.Error("couldn't get state of platoon")
    }
    if string(state) == "" {
        return shim.Error("platoon is empty")
    }
    var platoonArray []string
    err = json.Unmarshal(state, &platoonArray)
    if err != nil {
        return shim.Error("Error decoding JSON data: " + err.Error())
    }
    inPlat := false
    for index, test := range platoonArray{
        if test == args[2] {
            platoonArray = append(platoonArray[:index], platoonArray[index+1:]...)
            inPlat = true
            break
        }
    }
    if inPlat != true {
        return shim.Error("Value not in platoon, cannot leave")
    }
    state, err = json.Marshal(platoonArray)
    if err != nil {
        return shim.Error("Error encoding JSON data")
    }

    err = stub.PutState(args[1], []byte(state))
    if err != nil {
        return shim.Error("failed to update state")
    }

    err = stub.SetEvent("eventInvoke", []byte{})
    if err != nil {
        return shim.Error(err.Error())
    }
    return shim.Success(nil)
}

func (t *SamTestChaincode) invoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    fmt.Println("##### SamTestChaincode invoke #####")

    if len(args) < 3 {
        return shim.Error("invalid number of arguments for invoke, need at least 3")
    }

    err := stub.PutState(args[1], []byte(args[2]))
    if err != nil {
        return shim.Error("failed to update state")
    }

    err = stub.SetEvent("eventInvoke", []byte{})
    if err != nil {
        return shim.Error(err.Error())
    }
    return shim.Success(nil)

}

func main() {
    err := shim.Start(new(SamTestChaincode))
    if err != nil {
        fmt.Printf("Error starting SamTest ChainCode: %s", err)
    }
}
