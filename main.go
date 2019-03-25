package main

import (
    "fmt"
    "PlatoonTest/blockchain"
    "PlatoonTest/web"
    "PlatoonTest/web/controllers"
    "os"
    "math/rand"
    "time"
    //"sync"
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
    //var apps []*controllers.Application
    for i := 1; i < 16; i++ {
        cSetup, err := fSetup.InitializeUser(fmt.Sprintf("User%d", i))
        if err != nil {
            fmt.Printf("Unable to create new user {User%s}: %v\n", i, err)
            return
        }
        cSetups = append(cSetups, cSetup)
    }
    /*
    var funcs []blockchain.FuncDef
    funcs = append(funcs, blockchain.FuncDef{Function:"joinPlatoon", Arg1:"Plat1", Arg2:""})
    funcs = append(funcs, blockchain.FuncDef{Function:"changeSpeed", Arg1:"60", Arg2:""})
    funcs = append(funcs, blockchain.FuncDef{Function:"leavePlatoon", Arg1:"", Arg2:""})

    numUsers := 2

    var wg sync.WaitGroup
    wg.Add(numUsers)
    for i := 0; i < numUsers; i++ {
        go func(client blockchain.ClientSetup, funcs []blockchain.FuncDef, numLoops int) {
            defer wg.Done()
            client.LoadTest(funcs, numLoops)
        }(cSetups[i], funcs, numLoops)
    }
    wg.Wait()
    */
    rand.Seed(time.Now().UnixNano())
    numLoops := -1
    for i := 0; i < numLoops || numLoops == -1; i++ {
        id, err := cSetups[rand.Intn(len(cSetups))].InvokeRandomValid()
        if err != nil {
            fmt.Printf("Error doing random transaction: %v", err)
            return
        }
        fmt.Printf("Did Transaction: %s", id)
    }
    for i, curr := range cSetups {
        app := controllers.Application {
            Fabric: curr,
        }
        web.Serve(app, 8000+i)
        //apps = append(apps, app)
    }
    //for _, curr := range apps {
    //    fmt.Printf("User: %s\n", curr.Fabric.UserName)
    //}
    fmt.Scanf("%s")

}


