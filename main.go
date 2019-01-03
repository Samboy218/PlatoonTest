package main

import (
    "time"
    "fmt"
    "PlatoonTest/blockchain"
    "PlatoonTest/web"
    "PlatoonTest/web/controllers"
    "os"
)

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
    moves = append(moves, []string{"0", "4", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"0", "5", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"10", "10", "joinPlatoon", "plat2", ""})
    moves = append(moves, []string{"0", "11", "joinPlatoon", "plat2", ""})
    moves = append(moves, []string{"0", "12", "joinPlatoon", "plat2", ""})
    moves = append(moves, []string{"5", "3", "leavePlatoon", "", ""})
    moves = append(moves, []string{"7", "5", "leavePlatoon", "", ""})
    moves = append(moves, []string{"15", "10", "mergePlatoon", "plat1", ""})
    moves = append(moves, []string{"3", "3", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"10", "11", "splitPlatoon", "plat3", ""})
    moves = append(moves, []string{"5", "1", "leavePlatoon", "plat1", ""})
    moves = append(moves, []string{"0", "1", "joinPlatoon", "plat3", ""})
    
    for _, move := range moves {
        var ind int
        var delay int
        fmt.Sscan(move[0], &delay)
        fmt.Sscan(move[1], &ind)
        fmt.Printf("Sleeping %d before doing move {%v}\n", delay, move)
        time.Sleep(time.Duration(delay) * time.Second)
        fmt.Printf("doing move {%v}\n", move)
        ID, err := cSetups[ind].Invoke(move[2], move[3], move[4])
        if err != nil {
            fmt.Printf("Failed to execute move {%v}: %v\n", move, err)
            return
        }
        fmt.Printf("Successful transaction, ID: %s\n", ID)
    }

    for i, curr := range cSetups {
        app := &controllers.Application {
            Fabric: &curr,
        }
        web.Serve(app, 8000+i)
        apps = append(apps, app)
    }
    fmt.Scanf("%s")

}

