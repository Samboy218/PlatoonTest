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

type platoonUser struct {
    ID          string
    CurrPlat    string
    Reputation  int
    Money       int
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


    //make the dummy platoons and users
    var fakeUsers []platoonUser
    for i := 2; i < 16; i++ {
        fakeUsers = append(fakeUsers, platoonUser{ID:fmt.Sprintf("User%d@org1.samtest.com", i)})
    }
    fakePlats := make(map[string][]string)
    for i, user := range fakeUsers[0:5] {
        fakePlats["plat1"] = append(fakePlats["plat1"], user.ID)
        fakeUsers[i].CurrPlat = "plat1"
    }
    for i, user := range fakeUsers[5:10] {
        fakePlats["plat2"] = append(fakePlats["plat2"], user.ID)
        fakeUsers[i+5].CurrPlat = "plat2"
    }
    for i, user := range fakeUsers[10:len(fakeUsers)] {
        fakePlats["plat3"] = append(fakePlats["plat3"], user.ID)
        fakeUsers[i+10].CurrPlat = "plat3"
    }
    platList := make([]string, 0, len(fakePlats))
    //now do each platoon
    for k := range fakePlats {
        platList = append(platList, k)
        platState, err := json.Marshal(fakePlats[k])
        if err != nil {
            return shim.Error(fmt.Sprintf("error encoding JSON: %v", err.Error()))
        }
        err = stub.PutState(k, platState)
        if err != nil {
            return shim.Error(fmt.Sprintf("Unable to update platoon {%s}: %v", k, err.Error()))
        }
    }
    state, err := json.Marshal(platList)
    if err != nil {
        return shim.Error(fmt.Sprintf("error encoding JSON: %v", err.Error()))
    }
    err = stub.PutState("platoons", state)
    if err != nil {
        return shim.Error(fmt.Sprintf("Unable to update platoon list: %v", err.Error()))
    }
    state, err = json.Marshal(fakeUsers)
    if err != nil {
        return shim.Error(fmt.Sprintf("error encoding JSON: %v", err.Error()))
    }
    err = stub.PutState("users", state)
    if err != nil {
        return shim.Error(fmt.Sprintf("Unable to update user list: %v", err.Error()))
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
    if args[0] == "mergePlatoon" {
        return t.mergePlatoon(stub, args)
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
    err = newPlat(stub, args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("Error creating platoon {%s}: %v", args[1], err.Error()))
    }
    currUser, err := getUser(stub, userID)
    if err != nil {
        if currUser.ID == "None" {
            newUser(stub, platoonUser{ID:userID, CurrPlat:args[1], Reputation:0})
        }else {
            return shim.Error(fmt.Sprintf("Couldn't get user {%s}: %v", userID, err.Error))
        }
    }else {
        //user cannot join a platoon if they are already in another platoon
        if currUser.CurrPlat != "" {
            return shim.Error(fmt.Sprintf("Cannot join platoon {%s}: already in platoon {%s}", args[1], currUser.CurrPlat))
        }
        err = setUserPlat(stub, userID, args[1])
        if err != nil {
            return shim.Error(fmt.Sprintf("Couldn't set user {%s} platoon to {%s}: %v", userID, args[1], err.Error))
        }
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
    //try to get user
    currUser, err := getUser(stub, userID)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get user {%s}: %v", userID, err.Error()))
    }
    if currUser.CurrPlat != args[1] {
        return shim.Error(fmt.Sprintf("user {%s} cant leave platoon {%s}: not in platoon", userID, args[1]))
    }
    err = setUserPlat(stub, userID, "")
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't set user {%s} to leave platoon {%s}: %v", userID, args[1], err.Error()))
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

func (t *SamTestChaincode) mergePlatoon(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    if len(args) < 2 {
        return shim.Error("invalid number of arguments for lavePlatoon, need at least 2")
    }
    //get the user id
    userID, err := getUfromCert(stub)
    if err != nil {
        return shim.Error("Error getting userID: " + err.Error())
    }
    //try to get user
    currUser, err := getUser(stub, userID)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get user {%s}: %v", userID, err.Error()))
    }
    if currUser.CurrPlat != args[1] {
        return shim.Error(fmt.Sprintf("user {%s} not leader of platoon {%s}", userID, args[1]))
    }
    //platB is the platoon the caller is the leader of
    //platA is args[1]
    //platB is caller.CurPlat
    //end result of this function is platA = [platA+platB]; platB = []
    toMerge := currUser.CurrPlat
    state, err := stub.GetState(toMerge)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get platoon {%s}: %v", toMerge, err.Error()))
    }
    if string(state) == "" {
        return shim.Error(fmt.Sprintf("platoon {%s} empty or doesn't exist", toMerge))
    }
    var platB []string
    err = json.Unmarshal(state, &platB)
    if err != nil {
        return shim.Error(fmt.Sprintf("error decoding json: %v", err.Error()))
    }
    var platA []string
    state, err = stub.GetState(args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get platoon {%s}: %v", args[1], err.Error()))
    }
    if string(state) == "" {
        return shim.Error(fmt.Sprintf("platoon {%s} empty or doesn't exist", args[1]))
    }
    err = json.Unmarshal(state, &platA)
    if err != nil {
        return shim.Error(fmt.Sprintf("error decoding json: %v", err.Error()))
    }
    var curr platoonUser
    for _, user := range platB {
        //make sure all users exist and are actually in platB
        curr, err = getUser(stub, user)
        if err != nil {
            return shim.Error(fmt.Sprintf("couldn't get user {%s}: %v", user, err.Error()))
        }
        if curr.CurrPlat != toMerge {
            return shim.Error(fmt.Sprintf("user {%s} not in platoon {%s}, in platoon {%s}", user, toMerge, curr.CurrPlat))
        }
        //everything is good for this user, put them in platA
        err = setUserPlat(stub, user, args[1])
        if err != nil {
            return shim.Error(fmt.Sprintf("error setting user {%s} to platoon {%s}: %v", user, args[1], err.Error()))
        }
        platA = append(platA, user)
    }
    state, err = json.Marshal(platA)
    if err != nil {
        return shim.Error(fmt.Sprintf("error encoding JSON: %v", err.Error()))
    }
    err = stub.PutState(args[1], []byte(state))
    if err != nil {
        return shim.Error(fmt.Sprintf("error putting platoon data: %v", err.Error()))
    }
    err = stub.PutState(toMerge, []byte(""))
    if err != nil {
        return shim.Error(fmt.Sprintf("error putting platoon data: %v", err.Error()))
    }

    return shim.Success(nil)
}

func newUser(stub shim.ChaincodeStubInterface, user platoonUser) error {
    userList, err := stub.GetState("users")
    if err != nil {
        return fmt.Errorf("unable to get list of users: %v", err.Error())
    }
    var userArray []platoonUser
    if string(userList) != "" {
        err = json.Unmarshal(userList, &userArray)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err.Error())
        }
        for _, currUser := range userArray {
            if currUser.ID == user.ID {
                return fmt.Errorf("Couldn't create user: {%s} already exists", user.ID)
            }
        }
    }
    //user doesn't exist, make it
    userArray = append(userArray, user)
    state, err := json.Marshal(userArray)
    if err != nil {
        return fmt.Errorf("error encoding JSON: %v", err.Error())
    }
    err = stub.PutState("users", state)
    if err != nil {
        return fmt.Errorf("Unable to update user list: %v", err.Error())
    }
    return nil
}

func newPlat(stub shim.ChaincodeStubInterface, platID string) error {
    platList, err := stub.GetState("platoons")
    if err != nil {
        return fmt.Errorf("unable to get list of platoons: %v", err.Error())
    }
    var platArray []string
    if string(platList) != "" {
        err = json.Unmarshal(platList, &platArray)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err.Error())
        }
        for _, currPlat := range platArray {
            if currPlat == platID {
                return nil
            }
        }
    }
    //user doesn't exist, make it
    platArray = append(platArray, platID)
    state, err := json.Marshal(platArray)
    if err != nil {
        return fmt.Errorf("error encoding JSON: %v", err.Error())
    }
    err = stub.PutState("platoons", state)
    if err != nil {
        return fmt.Errorf("Unable to update platoon list: %v", err.Error())
    }
    return nil

}

func getUser(stub shim.ChaincodeStubInterface, userID string) (platoonUser, error) {
    userList, err := stub.GetState("users")
    if err != nil {
        return platoonUser{}, fmt.Errorf("unable to get list of users: %v", err.Error())
    }
    var userArray []platoonUser
    if string(userList) != "" {
        err = json.Unmarshal(userList, &userArray)
        if err != nil {
            return platoonUser{}, fmt.Errorf("error decoding JSON data: %v", err.Error())
        }
        for _, currUser := range userArray {
            if currUser.ID == userID {
                //user exists, return
                return currUser, nil
            }
        }
    }
    //user doesn't exist, panic
    return platoonUser{ID:"None"}, fmt.Errorf("unable to find user {%s}", userID)
}

func setUserPlat(stub shim.ChaincodeStubInterface, userID string, platID string) error {
    userList, err := stub.GetState("users")
    if err != nil {
        return fmt.Errorf("unable to get list of users: %v", err.Error())
    }
    var userArray []platoonUser
    if string(userList) != "" {
        err = json.Unmarshal(userList, &userArray)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err.Error())
        }
        for i, currUser := range userArray {
            if currUser.ID == userID {
                //user exists
                userArray[i].CurrPlat = platID
                state, err := json.Marshal(userArray)
                if err != nil {
                    return fmt.Errorf("unable to encode JSON: %v", err.Error())
                }
                err = stub.PutState("users", state)
                if err != nil {
                    return fmt.Errorf("Unable to update user list: %v", err.Error())
                }
                return nil
            }
        }
    }
    //user doesn't exist, panic
    return fmt.Errorf("unable to find user {%s}", userID)

}

func setUserRep(stub shim.ChaincodeStubInterface, userID string, rep int) error {
    userList, err := stub.GetState("users")
    if err != nil {
        return fmt.Errorf("unable to get list of users: %v", err.Error())
    }
    var userArray []platoonUser
    if string(userList) != "" {
        err = json.Unmarshal(userList, &userArray)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err.Error())
        }
        for i, currUser := range userArray {
            if currUser.ID == userID {
                //user exists
                userArray[i].Reputation = rep
                state, err := json.Marshal(userArray)
                if err != nil {
                    return fmt.Errorf("unable to encode JSON: %v", err.Error())
                }
                err = stub.PutState("users", state)
                if err != nil {
                    return fmt.Errorf("Unable to update user list: %v", err.Error())
                }
                return nil
            }
        }
    }
    //user doesn't exist, panic
    return fmt.Errorf("unable to find user {%s}", userID)
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
