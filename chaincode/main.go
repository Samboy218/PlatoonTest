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
    if len(args) < 2 {
        return shim.Error("invalid number of arguments for joinPlatoon, need at least 2")
    }
    //get the user id
    userID, err := getUfromCert(stub)
    if err != nil {
        return shim.Error("Error getting userID: " + err.Error())
    }
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
            if test == userID {
                return shim.Error("value already in platoon")
            }
        }
    }
    platoonArray = append(platoonArray, userID)
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
    if len(args) < 2 {
        return shim.Error("invalid number of arguments for lavePlatoon, need at least 2")
    }
    //get the user id
    userID, err := getUfromCert(stub)
    if err != nil {
        return shim.Error("Error getting userID: " + err.Error())
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
        if test == userID {
            platoonArray = append(platoonArray[:index], platoonArray[index+1:]...)
            inPlat = true
            break
        }
    }
    if inPlat != true {
        return shim.Error(fmt.Sprintf("Value {%s} not in platoon, cannot leave", userID))
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

func getUfromCert(stub shim.ChaincodeStubInterface) (string, error) {
    serializedID, err := stub.GetCreator()
    if err != nil {
        return "", fmt.Errorf("could not get creator: %v", err.Error())
    }
    sId := &msp.SerializedIdentity{}
    err = proto.Unmarshal(serializedID, sId)
    if err != nil {
        return "", fmt.Errorf("could not deserialize the sId: %v", err.Error())
    }
    bl, _ := pem.Decode(sId.IdBytes)
    if bl == nil {
        return "", fmt.Errorf("could not decode PEM")
    }
    cert, err := x509.ParseCertificate(bl.Bytes)
    if err != nil {
        return "", fmt.Errorf("could not parse cert: %v", err.Error())
    }
    // Get the client ID object
    userID := ""
    for _, currVal := range cert.Subject.ToRDNSequence() {
        if currVal[0].Type.String() == "2.5.4.3" {
            userID = fmt.Sprintf("%v", currVal[0].Value)
            break
        }
    }
    if userID == "" {
        return "", fmt.Errorf("couldn't find userID in RDNSequence")
    }
    return userID, nil
}
