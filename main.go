package main

import (
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
    //moves[0] = []string{"user", "function", "args"}
    moves = append(moves, []string{"1", "joinPlatoon", "plat1", ""})
    moves = append(moves, []string{"1", "leavePlatoon", "", ""})
    
    for _, move := range moves {
        var ind int
        fmt.Sscan(move[0], &ind)
        ID, err := cSetups[ind].Invoke(move[1], move[2], move[3])
        if err != nil {
            fmt.Printf("Failed to execute move {%v}: %v", move, err)
            return
        }
        fmt.Printf("Successful transaction, ID: %s", ID)
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

