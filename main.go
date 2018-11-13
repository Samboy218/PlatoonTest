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

    err = fSetup.NewUser("sam", "sampw", "org1")
    if err != nil {
        fmt.Printf("unable to make new user 'sam': %v", err)
    }

    var cSetups []blockchain.ClientSetup
    var apps []*controllers.Application
    for i := 1; i < 16; i++ {
        cSetup, err := fSetup.InitializeUser(fmt.Sprintf("User%d", i))
        if err != nil {
            fmt.Printf("Unable to create new user {User%s}: %v\n", i, err)
            return
        }
        app := &controllers.Application {
            Fabric: &cSetup,
        }
        cSetups = append(cSetups, cSetup)
        web.Serve(app, 8000+i)
        apps = append(apps, app)
    }
    cSetup, err := fSetup.InitializeUser("sam")
    if err != nil {
        fmt.Printf("Unable to create new user {sam}: %v\n", err)
        return
    }
    app := &controllers.Application {
        Fabric: &cSetup,
    }
    cSetups = append(cSetups, cSetup)
    web.Serve(app, 8016)
    apps = append(apps, app)

    fmt.Scanf("%s")
}

