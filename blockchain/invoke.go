package blockchain

import (
    "fmt"
    "github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
    "time"
    "math/rand"
    "encoding/json"
    "strings"
    "strconv"
)
type platoonUser struct {
    ID          string
    CurrPlat    string
    Reputation  float64
    Money       float64
    LastMove    int64
    EfficiencyClass string
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


func (setup *FabricSetup) Invoke(function string, key string, value string) (string, error) {
    var args []string
    args = append(args, "invoke")
    args = append(args, function)
    args = append(args, key)
    args = append(args, value)

    eventID := "eventInvoke"

    transientDataMap := make(map[string][]byte)
    transientDataMap["result"] = []byte("Transient data in invoke")

    notifier := make(chan *chclient.CCEvent)
    rce, err := setup.client.RegisterChaincodeEvent(notifier, setup.ChainCodeID, eventID)
    //reg, notifier, err := setup.event.RegisterChaincodeEvent(setup.ChainCodeID, eventID)
    if err != nil {
        return "", fmt.Errorf("failed to register chaincode event: %v", err)
    }
    //defer setup.event.Unregister(reg)

    response, err := setup.client.Execute(chclient.Request{ChaincodeID: setup.ChainCodeID, Fcn: args[0], Args: [][]byte{[]byte(args[1]), []byte(args[2]), []byte(args[3])}, TransientMap: transientDataMap})
    if err != nil {
        return "", fmt.Errorf("Failed to move funds: %v", err)
    }

    select {
    case ccEvent := <-notifier:
        fmt.Printf("Received CC event: %s\n", ccEvent)
    case <-time.After(time.Second*10):
        return "", fmt.Errorf("did NOT receive CC event for eventID(%s)", eventID)
    }
    setup.client.UnregisterChaincodeEvent(rce)
    return response.TransactionID.ID, nil
}

func (setup *ClientSetup) Invoke(function string, key string, value string) (string, error) {
    var args []string
    args = append(args, "invoke")
    args = append(args, function)
    args = append(args, key)
    args = append(args, value)

    //eventID := "eventInvoke"

    transientDataMap := make(map[string][]byte)
    transientDataMap["result"] = []byte("Transient data in invoke")

    //notifier := make(chan *chclient.CCEvent)
    //_, err := setup.client.RegisterChaincodeEvent(notifier, setup.ChainCodeID, eventID)
    //rce, err := setup.client.RegisterChaincodeEvent(notifier, setup.ChainCodeID, eventID)
    //reg, notifier, err := setup.event.RegisterChaincodeEvent(setup.ChainCodeID, eventID)
    /*
    if err != nil {
        return "", fmt.Errorf("failed to register chaincode event: %v", err)
    }
    */
    //defer setup.event.Unregister(reg)

    response, err := setup.client.Execute(chclient.Request{ChaincodeID: setup.ChainCodeID, Fcn: args[0], Args: [][]byte{[]byte(args[1]), []byte(args[2]), []byte(args[3])}, TransientMap: transientDataMap})
    if err != nil {
        //setup.client.UnregisterChaincodeEvent(rce)
        return "", fmt.Errorf("Failed to move funds: %v", err)
    }

    /*
    select {
    case ccEvent := <-notifier:
        fmt.Printf("Received CC event: %s\n", ccEvent)
    case <-time.After(time.Second*10):
        return "timeout", fmt.Errorf("did NOT receive CC event for eventID(%s)", eventID)
    }
    */
    //setup.client.UnregisterChaincodeEvent(rce)
    return response.TransactionID.ID, nil
}

//get current state our our user and the platoon we are in, then take a random valid action
func (setup *ClientSetup) InvokeRandomValid() (string, error) {
    //not in platoon
    //  join platoon
    //in platoon as leader
    //  merge
    //  change speed
    //  leave
    //in platoon as follower
    //  split
    //  leave
    rand.Seed(time.Now().UnixNano())
    //Get all platoons and Users
    var users  []platoonUser
    payload, err := setup.QueryVal("users")
    if payload != "" {
        err = json.Unmarshal([]byte(payload), &users)
        if err != nil {
            return "", fmt.Errorf("unable to decode JSON response: %v", err)
        }
    }
    
    var platoonsList []platoon
    var platoonIDs []string
    payload, err = setup.QueryVal("platoons")
    if payload != "" {
        err = json.Unmarshal([]byte(payload), &platoonIDs)
        if err != nil {
            return "", fmt.Errorf("unable to decode JSON response: %v", err)
        }
        var tempPlat platoon
        for _, id := range platoonIDs {
            payload, err = setup.QueryVal(id)
            if err != nil {
                return "", fmt.Errorf("unable to get platoon {%s}: %v", id, err)
            }
            if payload != "" {
                err = json.Unmarshal([]byte(payload), &tempPlat)
                if err != nil {
                    return "", fmt.Errorf("unable to decode JSON: %v\n%s", err, payload)
                }
            }
            platoonsList = append(platoonsList, tempPlat)
            tempPlat = platoon{}
        }
    }

    //okay now find the current user
    var currUser platoonUser
    for _, user := range users {
        splitID := strings.Split(user.ID, "@")
        if splitID[0] == setup.UserName {
            currUser = user
            break
        }
    }

    if currUser.CurrPlat == "" {
        //not in platoon
        //join a random platoon
        //make a new platoon

        //pick a random number, chance to make a new platoon is about 1/3
        choice := rand.Intn(int(len(platoonIDs) + (len(platoonIDs)/2) + 1))
        var joinID string
        if choice >= len(platoonIDs) {
            //new platoon, make a random string
            const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
            b := make([]byte, 10)
            for i := range b {
                b[i] = letterBytes[rand.Intn(len(letterBytes))]
            }
            joinID = "platoon" + string(b)
        }else {
            joinID = platoonIDs[choice]
        }
        return setup.Invoke("joinPlatoon", joinID, "")
    }else {
        //in platoon
        //check if leader
        isLeader := false
        for _, curr := range platoonsList {
            if curr.ID == currUser.CurrPlat {
                if strings.Split(curr.Members[0], "@")[0] == setup.UserName {
                    isLeader = true
                }
                break
            }
        }
        if isLeader {
            //choose one with about equal probability
            //  merge (don't try to merge if there is only one platoon)
            //  change speed
            //  leave
            choice := rand.Int() % 3
            if choice == 0 && len(platoonIDs) > 1{
                toMerge := rand.Intn(len(platoonIDs))
                //make sure we aren't merging with ourselves
                for platoonIDs[toMerge] == currUser.CurrPlat {
                    toMerge = rand.Intn(len(platoonIDs))
                }
                return setup.Invoke("mergePlatoon", platoonIDs[toMerge], "")
            }else if choice == 1 {
                //random speed between 50 and 70
                newSpeed := rand.Intn(20) + 50
                return setup.Invoke("changeSpeed", strconv.Itoa(newSpeed), "")
            }else {
                return setup.Invoke("leavePlatoon", "", "")
            }

        }else {
            //not leader
            //split into new platoon
            //leave
            choice := rand.Int() % 2
            if choice == 0 {
                const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
                b := make([]byte, 10)
                for i := range b {
                    b[i] = letterBytes[rand.Intn(len(letterBytes))]
                }
                return setup.Invoke("splitPlatoon", "platoon"+string(b), "")
            }else {
                return setup.Invoke("leavePlatoon", "", "")
            }
            
        }
    }
}
