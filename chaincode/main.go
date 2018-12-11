package main

//
//
//TODO: add changeSpeed
//
//

import (
    "fmt"
    "github.com/hyperledger/fabric/core/chaincode/shim"
    pb "github.com/hyperledger/fabric/protos/peer"
    "github.com/hyperledger/fabric/protos/msp"
    "github.com/golang/protobuf/proto"
    "encoding/json"
    "crypto/x509"
    "encoding/pem"
    "strconv"
)


type SamTestChaincode struct {
    pendingUserChanges []platoonUser
    pendingPlatChanges []platoon
    leaderBonus float64
}

type platoonUser struct {
    ID          string
    CurrPlat    string
    Reputation  float64
    Money       float64
    LastMove    int64
}

type platoon struct {
    ID string
    CurrSpeed int
    //timestamp of last change
    LastMove int64
    //distance (in miles) since the leaer was last payed
    Distance float64
    Members []string
}

func (t *SamTestChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
    fmt.Println("##### SamTestChaincode Init #####")

    function, _ := stub.GetFunctionAndParameters()
    TxTimestamp, err := stub.GetTxTimestamp()
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get Timestamp: %v", err))
    }
    TxTime := TxTimestamp.GetSeconds()

    if function != "init" {
        return shim.Error("Unknown function call")
    }
    err = stub.PutState("hello", []byte("world"))
    if err != nil {
        return shim.Error(err.Error())
    }


    //make the dummy platoons and users
    var fakeUsers []platoonUser
    for i := 2; i < 16; i++ {
        fakeUsers = append(fakeUsers, platoonUser{ID:fmt.Sprintf("User%d@org1.samtest.com", i), LastMove:TxTime})
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
        var currPlat platoon
        currPlat.ID = k
        currPlat.CurrSpeed = 60
        currPlat.Members = fakePlats[k]
        currPlat.LastMove = TxTime
        platList = append(platList, k)
        platState, err := json.Marshal(currPlat)
        if err != nil {
            return shim.Error(fmt.Sprintf("error encoding JSON: %v", err))
        }
        err = stub.PutState(k, platState)
        if err != nil {
            return shim.Error(fmt.Sprintf("Unable to update platoon {%s}: %v", k, err))
        }
    }
    state, err := json.Marshal(platList)
    if err != nil {
        return shim.Error(fmt.Sprintf("error encoding JSON: %v", err))
    }
    err = stub.PutState("platoons", state)
    if err != nil {
        return shim.Error(fmt.Sprintf("Unable to update platoon list: %v", err))
    }
    state, err = json.Marshal(fakeUsers)
    if err != nil {
        return shim.Error(fmt.Sprintf("error encoding JSON: %v", err))
    }
    err = stub.PutState("users", state)
    if err != nil {
        return shim.Error(fmt.Sprintf("Unable to update user list: %v", err))
    }

    return shim.Success(nil)
}

func (t *SamTestChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
    fmt.Println("##### SamTestChaincode Invoke #####")

    function, args := stub.GetFunctionAndParameters()
    if function != "invoke" {
        return shim.Error("Unknown funtion call")
    }
    
    var resp pb.Response
    if len(args) < 1 {
        return shim.Error("invalid arguments, expected at least 1")
    }
    //leader gets a bonus per transaction
    t.leaderBonus = .01

    if args[0] == "query" {
        return t.query(stub, args)
    }else if args[0] == "joinPlatoon" {
        resp = t.joinPlatoon(stub, args)
    }else if args[0] == "leavePlatoon" {
        resp = t.leavePlatoon(stub, args)
    }else if args[0] == "mergePlatoon" {
        resp = t.mergePlatoon(stub, args)
    }else if args[0] == "splitPlatoon" {
        resp = t.splitPlatoon(stub, args)
    }else if args[0] == "changeSpeed" {
        resp = t.changeSpeed(stub, args)
    }else {
        return shim.Error("Failed to invoke, check argument 1")
    }
    err := t.commitChanges(stub)
    if err != nil {
        return shim.Error(fmt.Sprintf("Failed to commit changes: %v", err))
    }
    return resp
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

//This function allows a user to join an existing platoon
//PRE: The user passes the name of the platoon it wishes to join
//POST: The user will be added to the back of the platoon
//      The transaction will fail if any of the following occur:
//          the user is already in a platoon
func (t *SamTestChaincode) joinPlatoon(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    if len(args) < 2 {
        return shim.Error("invalid number of arguments for joinPlatoon, need at least 2")
    }
    if args[1] == "users" || args[1] == "platoons" {
        return shim.Error(fmt.Sprintf("{%s} is a reserved name", args[1]))
    }
    //get the time
    txTimestamp, err := stub.GetTxTimestamp()
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get time: %v", err))
    }
    txTime := txTimestamp.GetSeconds()
    //get the user id
    userID, err := getUfromCert(stub)
    if err != nil {
        return shim.Error(fmt.Sprintf("Error getting userID: %v", err))
    }
    err = t.newPlat(stub, args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("Error creating platoon {%s}: %v", args[1], err))
    }
    currUser, err := t.getUser(stub, userID)
    if err != nil {
        if currUser.ID == "None" {
            t.newUser(stub, platoonUser{ID:userID, CurrPlat:args[1], Reputation:0, LastMove:txTime})
        }else {
            return shim.Error(fmt.Sprintf("Couldn't get user {%s}: %v", userID, err))
        }
    }else {
        //user cannot join a platoon if they are already in another platoon
        if currUser.CurrPlat != "" {
            return shim.Error(fmt.Sprintf("Cannot join platoon {%s}: already in platoon {%s}", args[1], currUser.CurrPlat))
        }
        err = t.setUserPlat(stub, userID, args[1])
        if err != nil {
            return shim.Error(fmt.Sprintf("Couldn't set user {%s} platoon to {%s}: %v", userID, args[1], err))
        }
    }
    //get the platoon
    plat, err := t.getPlat(stub, args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("Couldn't get platoon {%s}: %v", args[1], err))
    }

    //calculate what we need to pay the driver
    leader, err := t.getUser(stub, plat.Members[0])
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get leader of %s: %v", args[1], err))
    }
    var payment float64
    timeSince := txTime - plat.LastMove
    //convert seconds->hours, multiply by mph
    distanceTraveled := float64(timeSince)/3600 * float64(plat.CurrSpeed)
    distanceTraveled += plat.Distance
    if len(plat.Members) > 1 {
        payment = (distanceTraveled + t.leaderBonus)/float64(len(plat.Members)-1)
    }else {
        payment = 0
    }
    for _, test := range plat.Members{
        if test == userID {
            return shim.Error("value already in platoon")
        }
        if test != leader.ID {
            //pay the driver, update timestamp
            err = t.addUserRep(stub, leader.ID, payment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
            err = t.addUserRep(stub, test, -1*payment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
        }
    }
    var newMembers []string
    newMembers = append(plat.Members, userID)
    t.setPlatMembers(stub, args[1], newMembers)
    t.setPlatLastMove(stub, args[1], txTime)

    err = stub.SetEvent("eventInvoke", []byte{})
    if err != nil {
        return shim.Error(err.Error())
    }
    return shim.Success(nil)

}

//This function allows a user to leave their current platoon
//PRE: The user passes the name of the platoon (this requirement should be removed)
//POST: The use will no longer be a part of a platoon
//      the transaction will fail if any of the following will occur:
//          the user is not in a platoon
//          the user is not in the specified platoon
func (t *SamTestChaincode) leavePlatoon(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    if len(args) < 2 {
        return shim.Error("invalid number of arguments for lavePlatoon, need at least 2")
    }
    if args[1] == "users" || args[1] == "platoons" {
        return shim.Error(fmt.Sprintf("{%s} is a reserved name", args[1]))
    }

    txTimestamp, err := stub.GetTxTimestamp()
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get timestamp: %v", err))
    }
    txTime := txTimestamp.GetSeconds()
    //get the user id
    userID, err := getUfromCert(stub)
    if err != nil {
        return shim.Error(fmt.Sprintf("Error getting userID: %v", err))
    }
    //try to get user
    currUser, err := t.getUser(stub, userID)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get user {%s}: %v", userID, err))
    }
    if currUser.CurrPlat != args[1] {
        return shim.Error(fmt.Sprintf("user {%s} cant leave platoon {%s}: not in platoon", userID, args[1]))
    }
    err = t.setUserPlat(stub, userID, "")
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't set user {%s} to leave platoon {%s}: %v", userID, args[1], err))
    }
    plat, err := t.getPlat(stub, args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get platoon {%s}: %v", args[1], err))
    }

    //also pay the driver
    leader, err := t.getUser(stub, plat.Members[0])
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get leader of %s: %v", args[1], err))
    }

    var payment float64
    timeSince := txTime - plat.LastMove
    //convert seconds->hours, multiply by mph
    distanceTraveled := float64(timeSince)/3600 * float64(plat.CurrSpeed)
    distanceTraveled += plat.Distance
    if len(plat.Members) > 1 {
        payment = (distanceTraveled + t.leaderBonus)/float64(len(plat.Members)-1)
    }else {
        payment = 0
    }
    toLeave := -1
    for index, test := range plat.Members{
        if test != leader.ID {
            //pay the driver, update timestamp
            err = t.addUserRep(stub, leader.ID, payment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
            err = t.addUserRep(stub, test, -1*payment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
        }

        if test == userID {
            toLeave = index
        }
    }
    if toLeave == -1 {
        return shim.Error(fmt.Sprintf("Value {%s} not in platoon, cannot leave", userID))
    }
    newMembers := append(plat.Members[:toLeave], plat.Members[toLeave+1:]...)
    t.setPlatMembers(stub, args[1], newMembers)
    t.setPlatLastMove(stub, args[1], txTime)

    err = stub.SetEvent("eventInvoke", []byte{})
    if err != nil {
        return shim.Error(err.Error())
    }
    return shim.Success(nil)
}

//This function allows a platoon to merge with another platoon (appending to the back)
//PRE: The user passes the name of the platoon to merge with
//POST: The user's platoon will be appended to the target platoon, forming one platoon
//      the transaction will fail if any of the following occur:
//          the user is not in a platoon
//          the user is not the leader of a platoon
//          the target platoon doesn't exist
func (t *SamTestChaincode) mergePlatoon(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    if len(args) < 2 {
        return shim.Error("invalid number of arguments for leavePlatoon, need at least 2")
    }
    if args[1] == "users" || args[1] == "platoons" {
        return shim.Error(fmt.Sprintf("{%s} is a reserved name", args[1]))
    }
    //get the time
    txTimestamp, err := stub.GetTxTimestamp()
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get time: %v", err))
    }
    txTime := txTimestamp.GetSeconds()

    //get the user id
    userID, err := getUfromCert(stub)
    if err != nil {
        return shim.Error(fmt.Sprintf("Error getting userID: %v", err))
    }
    //try to get user
    currUser, err := t.getUser(stub, userID)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get user {%s}: %v", userID, err))
    }
    //platB is the platoon the caller is the leader of
    //platA is args[1]
    //platB is caller.CurPlat
    //end result of this function is platA = [platA+platB]; platB = []
    toMerge := currUser.CurrPlat
    platB, err := t.getPlat(stub, toMerge)

    //check if the user is the leader of tomerge
    if currUser.ID != platB.Members[0] {
        return shim.Error(fmt.Sprintf("user {%s} not leader of platoon {%s}", userID, args[1]))
    }
    platA, err := t.getPlat(stub, args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get platoon {%s}: %v", args[1], err))
    }

    //platA memebers need to pay up
    platALeader, err := t.getUser(stub, platA.Members[0])
    if err != nil {
        return shim.Error(fmt.Sprintf("could not get leader of %s: %v", args[1], err))
    }

    var platAPayment float64
    timeSince := txTime - platA.LastMove
    //convert seconds->hours, multiply by mph
    distanceTraveled := float64(timeSince)/3600 * float64(platA.CurrSpeed)
    distanceTraveled += platA.Distance
    if len(platA.Members) > 1 {
        platAPayment = (distanceTraveled + t.leaderBonus)/float64(len(platA.Members)-1)
    }else {
        platAPayment = 0
    }

    for _, user := range platA.Members {
        if user != platALeader.ID {
            //pay the driver, update timestamp
            err = t.addUserRep(stub, platALeader.ID, platAPayment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
            err = t.addUserRep(stub, user, -1*platAPayment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
        }
    }
    err = t.setPlatLastMove(stub, args[1], txTime)
    if err != nil {
        return shim.Error(fmt.Sprintf("Could not set last move of {%s}: %v", args[1], err))
    }
    err = t.setPlatDistance(stub, args[1], 0)
    if err != nil {
        return shim.Error(fmt.Sprintf("Could not set distance of {%s}: %v", args[1], err))
    }
    var curr platoonUser
    //platB memebers need to pay up
    platBLeader, err := t.getUser(stub, platB.Members[0])
    if err != nil {
        return shim.Error(fmt.Sprintf("could not get leader of %s: %v", toMerge, err))
    }

    var platBPayment float64
    timeSince = txTime - platB.LastMove
    //convert seconds->hours, multiply by mph
    distanceTraveled = float64(timeSince)/3600 * float64(platB.CurrSpeed)
    distanceTraveled += platB.Distance
    if len(platB.Members) > 1 {
        platBPayment = (distanceTraveled + t.leaderBonus)/float64(len(platB.Members)-1)
    }else {
        platBPayment = 0
    }
    err = t.setPlatDistance(stub, toMerge, 0)
    if err != nil {
        return shim.Error(fmt.Sprintf("could not set distance of {%s}: %v", toMerge, err))
    }
    newPlatA := platA.Members
    for _, user := range platB.Members {
        //make sure all users exist and are actually in platB
        curr, err = t.getUser(stub, user)
        if err != nil {
            return shim.Error(fmt.Sprintf("couldn't get user {%s}: %v", user, err))
        }
        if curr.CurrPlat != toMerge {
            return shim.Error(fmt.Sprintf("user {%s} not in platoon {%s}, in platoon {%s}", user, toMerge, curr.CurrPlat))
        }
        if user != platBLeader.ID {
            //pay the driver, update timestamp
            err = t.addUserRep(stub, platBLeader.ID, platBPayment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
            err = t.addUserRep(stub, user, -1*platBPayment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
        }

        newPlatA = append(newPlatA, user)
    }
    t.setPlatMembers(stub, args[1], newPlatA)
    t.setPlatLastMove(stub, toMerge, txTime)
    for _, user := range newPlatA {
        t.setUserPlat(stub, user, args[1])
    }

    t.setPlatMembers(stub, toMerge, []string{})

    err = stub.SetEvent("eventInvoke", []byte{})
    if err != nil {
        return shim.Error(err.Error())
    }

    return shim.Success(nil)
}

//This function allows a user to split a platoon into multiple platoons
//PRE: The user passes the name of the new platoon to be created from the split
//POST: The user's current platoon is split, with all vehicles in front of the user remaining in the old platoon
//      and all vehicles behind the user in the new platoon, and the user is the leader of the new platoon
//      The transaction will fail if any of the following occur:
//          the user is not in a platoon
//          the target platoon already exists
func (t *SamTestChaincode) splitPlatoon(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    if len(args) < 2 {
        return shim.Error("invalid number of arguments for leavePlatoon, need at least 2")
    }
    if args[1] == "users" || args[1] == "platoons" {
        return shim.Error(fmt.Sprintf("{%s} is a reserved name", args[1]))
    }
    if args[1] == "" {
        return shim.Error(fmt.Sprintf("need ID for new platoon after split"))
    }
    //get the time
    txTimestamp, err := stub.GetTxTimestamp()
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get time: %v", err))
    }
    txTime := txTimestamp.GetSeconds()

    //get the user id
    userID, err := getUfromCert(stub)
    if err != nil {
        return shim.Error(fmt.Sprintf("Error getting userID: %v", err))
    }
    //try to get user
    currUser, err := t.getUser(stub, userID)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get user {%s}: %v", userID, err))
    }
    //try to create the new plat
    err = t.newPlat(stub, args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("error creating new plat {%s}: %v", args[1], err))
    }

    platBID := currUser.CurrPlat
    if platBID == "" {
        return shim.Error(fmt.Sprintf("user {%s} not in any platoon", currUser.ID))
    }
    //get the platoon
    platB, err := t.getPlat(stub, platBID)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get platoon {%s}: %v", platBID, err))
    }

    //okay we have their platoon
    //we can't split into an existing platoon
    platA, err := t.getPlat(stub, args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get state of {%s}: %v", args[1], err))
    }
    if len(platA.Members) > 0 {
        return shim.Error(fmt.Sprintf("cannot split into existing platoon"))
    }
    //now put everything from platB[currUser:] into platA
    //also pay the leader
    leader, err := t.getUser(stub, platB.Members[0])
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get leader of %s: %v", platBID, err))
    }

    var payment float64
    timeSince := txTime - platB.LastMove
    //convert seconds->hours, multiply by mph
    distanceTraveled := float64(timeSince)/3600 * float64(platB.CurrSpeed)
    distanceTraveled += platB.Distance
    if len(platB.Members) > 1 {
        payment = (distanceTraveled + t.leaderBonus)/float64(len(platB.Members)-1)
    }else {
        payment = 0
    }
    splitInd := -1
    for i, user := range platB.Members {
        if user == currUser.ID {
            splitInd = i
        }
        if user != leader.ID {
            //pay the driver, update timestamp
            err = t.addUserRep(stub, leader.ID, payment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
            err = t.addUserRep(stub, user, -1*payment)
            if err != nil {
                return shim.Error(fmt.Sprintf("couldn't pay leader: %v", err))
            }
        }
    }
    if splitInd == -1 {
        return shim.Error(fmt.Sprintf("Cannot split %s: cannot find user %s", platBID, currUser.ID))
    }
    newPlatA := platB.Members[splitInd:]
    newPlatB := platB.Members[:splitInd]
    t.setPlatMembers(stub, args[1], newPlatA)
    t.setPlatMembers(stub, platBID, newPlatB)
    t.setLastMove(stub, args[1], txTime)
    t.setLastMove(stub, platBID, txTime)
    //now go through every user in platA and update their platoon
    //we now can use my function
    for _, user := range newPlatA {
        err = t.setUserPlat(stub, user, args[1])
        if err != nil {
            return shim.Error(fmt.Sprintf("could not set platoon for user {%s}: %v", user, err))
        }
    }

    err = stub.SetEvent("eventInvoke", []byte{})
    if err != nil {
        return shim.Error(err.Error())
    }

    return shim.Success(nil) 

}

//This function allows a user to change the speed of a platoon
//PRE: The user passes in the new speed of the platoon
//POST: The user's platoon changes to the specified speed
//      The transaction will fail if any of the following occur:
//          the user is not in a platoon
//          the user is not the leader of a platoon
//          speed is less than 0
func (t* SamTestChaincode) changeSpeed(stub shim.ChaincodeStubInterface, args []string) pb.Response {
    if len(args) < 1 {
        return shim.Error("invalid number of arguments for changeSpeed, need at least 1")
    }
    if args[1] == "" {
        return shim.Error(fmt.Sprintf("need value for new speed"))
    }
    //get the time
    txTimestamp, err := stub.GetTxTimestamp()
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get time: %v", err))
    }
    txTime := txTimestamp.GetSeconds()

    //get the user id
    userID, err := getUfromCert(stub)
    if err != nil {
        return shim.Error(fmt.Sprintf("Error getting userID: %v", err))
    }
    //try to get user
    currUser, err := t.getUser(stub, userID)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get user {%s}: %v", userID, err))
    }
    plat, err := t.getPlat(stub, currUser.CurrPlat)
    if err != nil {
        return shim.Error(fmt.Sprintf("couldn't get user's platoon: %v", err))
    }
    if plat.ID != currUser.CurrPlat {
        return shim.Error(fmt.Sprintf("could not get platoon"))
    }
    if plat.Members[0] != userID {
        return shim.Error(fmt.Sprintf("user is not leader of platoon"))
    }
    speed, err := strconv.Atoi(args[1])
    if err != nil {
        return shim.Error(fmt.Sprintf("could not parse speed value: %v", err))
    }
    if speed < 0 {
        return shim.Error(fmt.Sprintf("speed can not be negative"))
    }
    //
    //TODO: find out distance traveled
    //
    timeSince := txTime - plat.LastMove
    //convert seconds->hours, multiply by mph
    distanceTraveled := float64(timeSince)/3600 * float64(plat.CurrSpeed)
    err = t.changePlatDistance(stub, plat.ID, distanceTraveled)
    if err != nil {
        return shim.Error(fmt.Sprintf("error changing platoon distance traveled: %v", err))
    }
    err = t.setPlatSpeed(stub, plat.ID, speed)
    if err != nil {
        return shim.Error(fmt.Sprintf("error setting platoon speed: %v", err))
    }
    err = t.setPlatLastMove(stub, plat.ID, txTime)
    if err != nil {
        return shim.Error(fmt.Sprintf("error setting platoon last move: %v", err))
    }

    err = stub.SetEvent("eventInvoke", []byte{})
    if err != nil {
        return shim.Error(err.Error())
    }

    return shim.Success(nil) 

}

//helper function, not a chaincode function
//creates a new user in the database if it isn't already there
func (t* SamTestChaincode) newUser(stub shim.ChaincodeStubInterface, user platoonUser) error {
    if len(t.pendingUserChanges) == 0 {
        userList, err := stub.GetState("users")
        if err != nil {
            return fmt.Errorf("unable to get list of users: %v", err)
        }
        if string(userList) != "" {
            err = json.Unmarshal(userList, &t.pendingUserChanges)
            if err != nil {
                return fmt.Errorf("error decoding JSON data: %v", err)
            }
        }
    }
    for _, currUser := range t.pendingUserChanges {
        if currUser.ID == user.ID {
            //user exists
            return fmt.Errorf("Couldn't create user: {%s} already exists", user.ID)
        }
    }
    //user doesn't exist, make it
    t.pendingUserChanges = append(t.pendingUserChanges, user)
    return nil
}

//helper function, not a chaincode function
//creates a new platoon in the database if it isn't already there
func (t* SamTestChaincode) newPlat(stub shim.ChaincodeStubInterface, platID string) error {
    platList, err := stub.GetState("platoons")
    if err != nil {
        return fmt.Errorf("unable to get list of platoons: %v", err)
    }
    var platArray []string
    if string(platList) != "" {
        err = json.Unmarshal(platList, &platArray)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err)
        }
        for _, currPlat := range platArray {
            if currPlat == platID {
                return nil
            }
        }
    }
    //plat doesn't exist, make it
    platArray = append(platArray, platID)
    state, err := json.Marshal(platArray)
    if err != nil {
        return fmt.Errorf("error encoding JSON: %v", err)
    }
    err = stub.PutState("platoons", state)
    if err != nil {
        return fmt.Errorf("Unable to update platoon list: %v", err)
    }
    var plat platoon
    plat.ID = platID
    plat.CurrSpeed = 60
    txTimestamp, err := stub.GetTxTimestamp()
    if err != nil {
        return fmt.Errorf("couldn't get time: %v", err)
    }
    txTime := txTimestamp.GetSeconds()
    plat.LastMove = txTime
    plat.Members = nil
    t.pendingPlatChanges = append(t.pendingPlatChanges, plat)
    return nil

}

//helper function, not a chaincode function
//gets a platoon from the database
func (t* SamTestChaincode) getPlat(stub shim.ChaincodeStubInterface, platID string) (platoon, error) {
    for _, currPlat := range t.pendingPlatChanges {
        if currPlat.ID == platID {
            //user exists in pending changes
            return currPlat, nil
        }
    }

    platData, err := stub.GetState(platID)
    if err != nil {
        return platoon{}, fmt.Errorf("unable to get platoon {%s}: %v", platID, err)
    }
    var plat platoon
    if string(platData) != "" {
        err = json.Unmarshal(platData, &plat)
        if err != nil {
            return platoon{}, fmt.Errorf("error decoding JSON data: %v", err)
        }
    }
    t.pendingPlatChanges = append(t.pendingPlatChanges, plat)
    return plat, nil
}

func (t* SamTestChaincode) setPlatSpeed(stub shim.ChaincodeStubInterface, platID string, speed int) error {
    for i, currPlat := range t.pendingPlatChanges {
        if currPlat.ID == platID {
            //plat exists in pending changes
            t.pendingPlatChanges[i].CurrSpeed = speed
            return nil
        }
    }

    platData, err := stub.GetState(platID)
    if err != nil {
        return fmt.Errorf("unable to get platoon {%s}: %v", platID, err)
    }
    var plat platoon
    if string(platData) != "" {
        err = json.Unmarshal(platData, &plat)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err)
        }
    }
    plat.CurrSpeed = speed
    t.pendingPlatChanges = append(t.pendingPlatChanges, plat)
    return nil

}

func (t* SamTestChaincode) setPlatLastMove(stub shim.ChaincodeStubInterface, platID string, time int64) error {
    for i, currPlat := range t.pendingPlatChanges {
        if currPlat.ID == platID {
            //plat exists in pending changes
            t.pendingPlatChanges[i].LastMove = time
            return nil
        }
    }

    platData, err := stub.GetState(platID)
    if err != nil {
        return fmt.Errorf("unable to get platoon {%s}: %v", platID, err)
    }
    var plat platoon
    if string(platData) != "" {
        err = json.Unmarshal(platData, &plat)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err)
        }
    }
    plat.LastMove = time
    t.pendingPlatChanges = append(t.pendingPlatChanges, plat)
    return nil
}

func (t* SamTestChaincode) setPlatDistance(stub shim.ChaincodeStubInterface, platID string, distance float64) error {
    for i, currPlat := range t.pendingPlatChanges {
        if currPlat.ID == platID {
            //plat exists in pending changes
            t.pendingPlatChanges[i].Distance = distance
            return nil
        }
    }

    platData, err := stub.GetState(platID)
    if err != nil {
        return fmt.Errorf("unable to get platoon {%s}: %v", platID, err)
    }
    var plat platoon
    if string(platData) != "" {
        err = json.Unmarshal(platData, &plat)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err)
        }
    }
    plat.Distance = distance
    t.pendingPlatChanges = append(t.pendingPlatChanges, plat)
    return nil
}

func (t* SamTestChaincode) changePlatDistance(stub shim.ChaincodeStubInterface, platID string, distance float64) error {
    for i, currPlat := range t.pendingPlatChanges {
        if currPlat.ID == platID {
            //plat exists in pending changes
            t.pendingPlatChanges[i].Distance += distance
            return nil
        }
    }

    platData, err := stub.GetState(platID)
    if err != nil {
        return fmt.Errorf("unable to get platoon {%s}: %v", platID, err)
    }
    var plat platoon
    if string(platData) != "" {
        err = json.Unmarshal(platData, &plat)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err)
        }
    }
    plat.Distance += distance
    t.pendingPlatChanges = append(t.pendingPlatChanges, plat)
    return nil
}

func (t* SamTestChaincode) setPlatMembers(stub shim.ChaincodeStubInterface, platID string, members []string) error {
    for i, currPlat := range t.pendingPlatChanges {
        if currPlat.ID == platID {
            //plat exists in pending changes
            t.pendingPlatChanges[i].Members = members
            return nil
        }
    }

    platData, err := stub.GetState(platID)
    if err != nil {
        return fmt.Errorf("unable to get platoon {%s}: %v", platID, err)
    }
    var plat platoon
    if string(platData) != "" {
        err = json.Unmarshal(platData, &plat)
        if err != nil {
            return fmt.Errorf("error decoding JSON data: %v", err)
        }
    }
    plat.Members = members
    t.pendingPlatChanges = append(t.pendingPlatChanges, plat)
    return nil

}

//helper function, not a chaincode function
//gets a user from the database
func (t* SamTestChaincode) getUser(stub shim.ChaincodeStubInterface, userID string) (platoonUser, error) {
    if len(t.pendingUserChanges) == 0 {
        userList, err := stub.GetState("users")
        if err != nil {
            return platoonUser{}, fmt.Errorf("unable to get list of users: %v", err)
        }
        if string(userList) != "" {
            err = json.Unmarshal(userList, &t.pendingUserChanges)
            if err != nil {
                return platoonUser{}, fmt.Errorf("error decoding JSON data: %v", err)
            }
        }
    }
    for _, currUser := range t.pendingUserChanges {
        if currUser.ID == userID {
            //user exists
            return currUser, nil
        }
    }
    //user doesn't exist, panic
    return platoonUser{ID:"None"}, fmt.Errorf("unable to find user {%s}", userID)
}

//helper function, not a chaincode function
//gets all user from the database
func (t* SamTestChaincode) getAllUsers(stub shim.ChaincodeStubInterface) ([]platoonUser, error) {
    if len(t.pendingUserChanges) == 0 {
        userList, err := stub.GetState("users")
        if err != nil {
            return []platoonUser{}, fmt.Errorf("unable to get list of users: %v", err)
        }
        if string(userList) != "" {
            err = json.Unmarshal(userList, &t.pendingUserChanges)
            if err != nil {
                return []platoonUser{}, fmt.Errorf("error decoding JSON data: %v", err)
            }
        }
    }
    return t.pendingUserChanges, nil
}

//helper function, not a chaincode function
//sets a specific user's platoon (does not commit)
func (t* SamTestChaincode) setUserPlat(stub shim.ChaincodeStubInterface, userID string, platID string) error {
    if len(t.pendingUserChanges) == 0 {
        userList, err := stub.GetState("users")
        if err != nil {
            return fmt.Errorf("unable to get list of users: %v", err)
        }
        if string(userList) != "" {
            err = json.Unmarshal(userList, &t.pendingUserChanges)
            if err != nil {
                return fmt.Errorf("error decoding JSON data: %v", err)
            }
        }
    }
    for i, currUser := range t.pendingUserChanges {
        if currUser.ID == userID {
            //user exists
            t.pendingUserChanges[i].CurrPlat = platID
            return nil
        }
    }
    //user doesn't exist, panic
    return fmt.Errorf("unable to find user {%s}", userID)

}

//helper function, not a chaincode function
//sets a specific user's reputation (does not commit)
func (t* SamTestChaincode) setUserRep(stub shim.ChaincodeStubInterface, userID string, rep float64) error {
    if len(t.pendingUserChanges) == 0 {
        userList, err := stub.GetState("users")
        if err != nil {
            return fmt.Errorf("unable to get list of users: %v", err)
        }
        if string(userList) != "" {
            err = json.Unmarshal(userList, &t.pendingUserChanges)
            if err != nil {
                return fmt.Errorf("error decoding JSON data: %v", err)
            }
        }
    }
    for i, currUser := range t.pendingUserChanges {
        if currUser.ID == userID {
            //user exists
            t.pendingUserChanges[i].Reputation = rep
            t.pendingUserChanges[i].Money = rep
            return nil
        }
    }
    //user doesn't exist, panic
    return fmt.Errorf("unable to find user {%s}", userID)
}

//helper function, not a chaincode function
//adds an amount to a specific user's reputation (does not commit)
func (t* SamTestChaincode) addUserRep(stub shim.ChaincodeStubInterface, userID string, rep float64) error {
    if len(t.pendingUserChanges) == 0 {
        userList, err := stub.GetState("users")
        if err != nil {
            return fmt.Errorf("unable to get list of users: %v", err)
        }
        if string(userList) != "" {
            err = json.Unmarshal(userList, &t.pendingUserChanges)
            if err != nil {
                return fmt.Errorf("error decoding JSON data: %v", err)
            }
        }
    }
    for i, currUser := range t.pendingUserChanges {
        if currUser.ID == userID {
            //user exists
            t.pendingUserChanges[i].Reputation += rep
            t.pendingUserChanges[i].Money += rep
            return nil
        }
    }
    //user doesn't exist, panic
    return fmt.Errorf("unable to find user {%s}", userID)
}

//helper function, not a chaincode function
//sets a specific user's lastMove timestamp (does not commit)
func (t *SamTestChaincode) setLastMove(stub shim.ChaincodeStubInterface, userID string, txTime int64) error {
    if len(t.pendingUserChanges) == 0 {
        userList, err := stub.GetState("users")
        if err != nil {
            return fmt.Errorf("unable to get list of users: %v", err)
        }
        if string(userList) != "" {
            err = json.Unmarshal(userList, &t.pendingUserChanges)
            if err != nil {
                return fmt.Errorf("error decoding JSON data: %v", err)
            }
        }
    }
    for i, currUser := range t.pendingUserChanges {
        if currUser.ID == userID {
            //user exists
            t.pendingUserChanges[i].LastMove = txTime
            return nil
        }
    }
    //user doesn't exist, panic
    return fmt.Errorf("unable to find user {%s}", userID)

}


//helper function, not a chaincode function
//commits all pending changes to the user database
func (t *SamTestChaincode) commitChanges(stub shim.ChaincodeStubInterface) error {
    if len(t.pendingUserChanges) > 0 {
        //something has tried to change the users, we need to push that change
        state, err := json.Marshal(t.pendingUserChanges)
        if err != nil {
            return fmt.Errorf("unable to encode JSON: %v", err)
        }
        err = stub.PutState("users", state)
        if err != nil {
            return fmt.Errorf("Unable to update user list: %v", err)
        }
    }
    if len(t.pendingPlatChanges) > 0 {
        for _, curr := range t.pendingPlatChanges {
            state, err := json.Marshal(curr)
            if err != nil {
                return fmt.Errorf("unable to encode JSON: %v", err)
            }
            err = stub.PutState(curr.ID, state)
            if err != nil {
                return fmt.Errorf("unable to update platoon {%s}: %v", curr.ID, err)
            }
        }
    }
    return nil
}

//helper function, not a chaincode function
//uses the cert to get the username of whoever invoked this transaction
func getUfromCert(stub shim.ChaincodeStubInterface) (string, error) {
    serializedID, err := stub.GetCreator()
    if err != nil {
        return "", fmt.Errorf("could not get creator: %v", err)
    }
    sId := &msp.SerializedIdentity{}
    err = proto.Unmarshal(serializedID, sId)
    if err != nil {
        return "", fmt.Errorf("could not deserialize the sId: %v", err)
    }
    bl, _ := pem.Decode(sId.IdBytes)
    if bl == nil {
        return "", fmt.Errorf("could not decode PEM")
    }
    cert, err := x509.ParseCertificate(bl.Bytes)
    if err != nil {
        return "", fmt.Errorf("could not parse cert: %v", err)
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

//helper function, not a chaincode function
//check if a string is in a slice of strings
func contains(s []string, e string) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

//main
func main() {
    err := shim.Start(new(SamTestChaincode))
    if err != nil {
        fmt.Printf("Error starting SamTest ChainCode: %s", err)
    }
}
