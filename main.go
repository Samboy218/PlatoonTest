package main

import (
    "time"
    "fmt"
    "PlatoonTest/blockchain"
    "PlatoonTest/web"
    "PlatoonTest/web/controllers"
    "encoding/json"
    "os"
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
type dbState struct {
    ActionPerformed string
    Users []platoonUser
    Platoons []platoon
}

func main() {
    fSetup := blockchain.FabricSetup{
        ChannelID:  "samtest",
        ChannelConfig:  os.Getenv("GOPATH") + "/src/PlatoonTest/artifacts/samtest.channel.tx",

        ChainCodeID: "samtest-service",
        ChainCodeGoPath: os.Getenv("GOPATH"),
        ChainCodePath: "PlatoonTest/chaincode/",

        OrgAdmin:   "Admin",
        OrgName:    "Org1",
        ConfigFile: "config.yaml",

        UserName: "User10",
    }

    err := fSetup.Initialize()
    if err != nil {
        fmt.Printf("Unable to initialize the fabric SDK: %v\n", err)
        return
    }

    err = fSetup.InstallAndInstantiateCC()
    if err != nil {
        fmt.Printf("Unable to install and instantiate the chaincode: %v\n", err)
        return
    }


    var cSetups []blockchain.ClientSetup
    var apps []*controllers.Application
    for i := 1; i < 16; i++ {
        cSetup, err := fSetup.InitializeUser(fmt.Sprintf("User%d", i))
        if err != nil {
            fmt.Printf("Unable to create new user {User%s}: %v\n", i, err)
            return
        }
        cSetups = append(cSetups, cSetup)
    }

    //make a way to quickly run through test cases
    moves := make([][]string, 0)
    //moves[0] = []string{"delay", "user", "function", "args"}
    moves = append(moves, []string{"0", "0", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"0", "1", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"0", "2", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"0", "3", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"0", "3", "joinPlatoon", "plat5", ""}) //invalid
    moves = append(moves, []string{"0", "3", "splitPlatoon", "", ""}) //invalid
    moves = append(moves, []string{"0", "3", "mergePlatoon", "plat5", ""}) //invalid
    moves = append(moves, []string{"0", "4", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"0", "5", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"10", "10", "joinPlatoon", "plat2", ""})
    moves = append(moves, []string{"0", "11", "joinPlatoon", "plat2", ""})
    moves = append(moves, []string{"0", "12", "joinPlatoon", "plat2", ""})
    moves = append(moves, []string{"5", "3", "leavePlatoon", "", ""})
    moves = append(moves, []string{"0", "3", "mergePlatoon", "plat1", ""}) //invalid
    moves = append(moves, []string{"0", "3", "changeSpeed", "0", ""}) //invalid
    moves = append(moves, []string{"0", "3", "splitPlatoon", "plat5", ""}) //invalid
    moves = append(moves, []string{"7", "5", "leavePlatoon", "", ""})
    moves = append(moves, []string{"15", "10", "mergePlatoon", "plat1", ""})
    moves = append(moves, []string{"3", "3", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"10", "11", "splitPlatoon", "plat3", ""})
    moves = append(moves, []string{"5", "1", "leavePlatoon", "plat1", ""})
    moves = append(moves, []string{"0", "1", "joinPlatoon", "plat3", ""})

    /*Example of some invalid transactions
    //cannot join a platoon if you are already in one
    moves = append(moves, []string{"0", "6", "joinPlatoon", "plat1", ""}) //valid
    moves = append(moves, []string{"0", "6", "joinPlatoon", "plat2", ""}) //invalid
    moves = append(moves, []string{"0", "6", "joinPlatoon", "plat3", ""}) //invalid
    moves = append(moves, []string{"0", "6", "joinPlatoon", "plat4", ""}) //invalid
    //cannot merge unless you are leader
    moves = append(moves, []string{"0", "6", "mergePlatoon", "plat3", ""}) //invalid
    //cannot merge into self
    moves = append(moves, []string{"0", "6", "mergePlatoon", "plat1", ""}) //invalid
    moves = append(moves, []string{"0", "6", "leavePlatoon", "", ""}) //valid
    //cannot leave/split/merge if you aren't in a platoon
    moves = append(moves, []string{"0", "6", "leavePlatoon", "", ""}) //invalid
    moves = append(moves, []string{"0", "6", "mergePlatoon", "plat1", ""}) //invalid
    moves = append(moves, []string{"0", "6", "splitPlatoon", "plat1", ""}) //invalid
    */

    var runs []dbState
    startTime := time.Now()
    for _, move := range moves {
        var tempRun dbState
        var ind int
        var delay int
        fmt.Sscan(move[0], &delay)
        fmt.Sscan(move[1], &ind)
        tempRun.ActionPerformed = fmt.Sprintf("%s did %s on %s after %s seconds", cSetups[ind].UserName, move[2], move[3], move[0])
        fmt.Printf("Sleeping %d before doing move {%v}\n", delay, move)
        time.Sleep(time.Duration(delay) * time.Second)
        fmt.Printf("doing move {%v}\n", move)
        ID, err := cSetups[ind].Invoke(move[2], move[3], move[4])
        if err != nil {
            fmt.Printf("Failed to execute move {%v}: %v\n", move, err)
            tempRun.ActionPerformed = fmt.Sprintf("%s: FAILED", tempRun.ActionPerformed)
        }else {
            fmt.Printf("Successful transaction, ID: %s\n", ID)
        }
        //get all of the data and put it into a file
        payload, err := cSetups[ind].QueryVal("users")
        if err != nil {
            fmt.Printf("Couldn't query users: %v\n", err)
            return
        }
        if payload != "" {
            err = json.Unmarshal([]byte(payload), &tempRun.Users)
            if err != nil {
                fmt.Printf("unable to decode JSON response: %v\n", err)
                return
            }
        }
        payload, err = cSetups[ind].QueryVal("platoons")
        if err != nil {
            fmt.Printf("Couldn't query platoons: %v\n", err)
            return
        }
        var platoonIDs []string
        if payload != "" {
            err = json.Unmarshal([]byte(payload), &platoonIDs)
            if err != nil {
                fmt.Printf("unable to decode JSON response: %v\n", err)
                return
            }
        }
        for _, curr := range platoonIDs {
            var temp platoon
            payload, err = cSetups[ind].QueryVal(curr)
            if err != nil {
                fmt.Printf("couldn't query %s: %v\n", curr, err)
            }
            if payload != "" {
                err = json.Unmarshal([]byte(payload), &temp)
                if err != nil {
                    fmt.Printf("unable to decode JSON response: %v\n", err)
                    return
                }
                tempRun.Platoons = append(tempRun.Platoons, temp)
            }
        }
        runs = append(runs, tempRun)
    }
    endTime := time.Now()
    elapsed := endTime.Sub(startTime)
    fmt.Printf("Run took %f seconds\n", elapsed.Seconds())

    dump, err := json.Marshal(runs)
    if err != nil {
        fmt.Printf("Couldn't encode JSON: %v\n", err)
    }

    f, err := os.Create("run.json")
    if err != nil {
        fmt.Printf("Couldn't open file: %v\n", err)
    }
    defer f.Close()
    numBytes, err := f.WriteString(string(dump))
    if err != nil {
        fmt.Printf("Couldn't write to file: %v\n", err)
    }
    fmt.Printf("Wrote %d bytes\n", numBytes)
    f.Sync()

    for i, curr := range cSetups {
        app := &controllers.Application {
            Fabric: &curr,
        }
        web.Serve(app, 8000+i)
        apps = append(apps, app)
    }
    fmt.Scanf("%s")

}

